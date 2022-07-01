package helm

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/Xfers/argocd-helm-ext-plugin/pkg/config"
	"github.com/Xfers/argocd-helm-ext-plugin/pkg/utils"
	"github.com/go-redis/redis/v8"
)

const (
	SIMPLE_VALUE int = iota
	STRING_VALUE
)

// the special helm variable that needs to use `--set-string` to avoid string escaping issue``
var special_helm_variables = map[string]int{
	"image.tag": STRING_VALUE,
}

func getHelmValueType(name string) int {
	helm_val, ok := special_helm_variables[name]
	if ok {
		return helm_val
	}
	return SIMPLE_VALUE
}

// Helm CLI helper to pull chart and generate template
type HelmCli struct {
	// ARGOCD_APP_NAMESPACE environment variable
	AppNamespace string
	// ARGOCD_APP_NAME environment variable
	AppName string
	// ARGOCD_APP_REVISION environment variable
	AppRevision string
	// HELM_REPO_URL environment variable
	RepoUrl string
	// HELM_CHART environment variable
	Chart string
	// HELM_CHART_VERSION environment variable
	ChartVersion string
	// convert HELM_VALUE_FILES environment variable into list of values files
	ValueFiles []string
	// convert HELM_VALUES environment variable into a values map
	Values map[string]string

	opts *config.Options
}

func New(opts *config.Options) (*HelmCli, error) {
	values_files := strings.Split(utils.Getenv("HELM_VALUE_FILES"), " ")
	for i, v := range values_files {
		values_files[i] = strings.TrimSpace(v)
	}

	h := &HelmCli{
		AppNamespace: os.Getenv("ARGOCD_APP_NAMESPACE"),
		AppName:      os.Getenv("ARGOCD_APP_NAME"),
		AppRevision:  os.Getenv("ARGOCD_APP_REVISION"),
		RepoUrl:      utils.Getenv("HELM_REPO_URL"),
		Chart:        utils.Getenv("HELM_CHART"),
		ChartVersion: utils.Getenv("HELM_CHART_VERSION"),
		ValueFiles:   values_files,
		opts:         opts,
	}

	if len(h.AppName) == 0 {
		return nil, errors.New("ARGOCD_APP_NAME is empty")
	}
	if len(h.RepoUrl) == 0 {
		return nil, errors.New("HELM_REPO_URL is empty")
	}
	if len(h.Chart) == 0 {
		return nil, errors.New("HELM_CHART is empty")
	}
	if len(h.ChartVersion) == 0 {
		return nil, errors.New("HELM_CHART_VERSION is empty")
	}

	// initial redis client
	if _, err := getRedisClient(); err != nil {
		return nil, err
	}

	if err := h.loadHelmValues(); err != nil {
		return nil, err
	}

	return h, nil
}

func parseHelmValues(helm_values string) map[string]string {
	values := make(map[string]string)
	values_env_var := strings.Split(helm_values, ";")
	for _, v := range values_env_var {
		token := strings.Split(v, "=")
		if len(token) != 2 {
			continue
		}
		values[strings.TrimSpace(token[0])] = strings.TrimSpace(token[1])
	}
	return values
}

func (h *HelmCli) PullChart() error {
	cmd := exec.Command("helm", "pull", h.Chart, "--untar", "--repo", h.RepoUrl, "--version", h.ChartVersion)
	log.Printf("Execute: %s\n", strings.Join(cmd.Args, " "))

	if err := cmd.Run(); err != nil {
		return errors.New(fmt.Sprintf("pull chart %s error: %s\n", h.Chart, err.Error()))
	}
	return nil
}

func (h *HelmCli) GenerateTemplate() error {
	args := []string{
		"template",
		h.AppName,
		h.Chart,
		"--repo",
		h.RepoUrl,
		"--version",
		h.ChartVersion,
	}

	if h.opts.IncludeCrds {
		args = append(args, "--include-crds")
	}

	for _, value_file := range h.ValueFiles {
		if _, err := os.Stat(value_file); err == nil {
			args = append(args, "-f")
			args = append(args, value_file)
		}
	}

	for key, value := range h.Values {
		var value_type int
		helm_val, ok := special_helm_variables[key]
		if ok {
			value_type = helm_val
		} else {
			value_type = SIMPLE_VALUE
		}

		switch value_type {
		case STRING_VALUE:
			args = append(args, "--set-string")
		default:
			args = append(args, "--set")
		}
		args = append(args, fmt.Sprintf("%s=%s", key, value))
	}

	var out bytes.Buffer
	cmd := exec.Command("helm", args...)
	cmd.Stdout = &out

	log.Printf("Execute(%s): %s\n", h.AppRevision, strings.Join(cmd.Args, " "))

	if err := cmd.Run(); err != nil {
		return errors.New(fmt.Sprintf("pull chart %s error: %s\n", h.Chart, err.Error()))
	}

	manifest_filename := fmt.Sprintf("/tmp/manifest-%s-%s.yaml", h.AppNamespace, h.AppName)
	manifest_file, err := os.OpenFile(manifest_filename, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("can't open manifest file: %s", manifest_filename)
	}
	manifest_file.Write(out.Bytes())
	manifest_file.Close()

	fmt.Print(out.String())

	return nil
}

func (h *HelmCli) helmValueCacheKey() string {
	return fmt.Sprintf("%s/%s/argocd_helm_ext_plugin/helm_values", h.AppNamespace, h.AppName)
}

func (h *HelmCli) loadRedisHelmValues() (string, error) {
	var client *redis.Client
	var err error

	client, err = getRedisClient()
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var helm_cached_values string
	helm_cached_values, err = client.Get(ctx, h.helmValueCacheKey()).Result()
	if err != nil {
		return "", nil
	}
	return helm_cached_values, nil
}

func (h *HelmCli) saveRedisHelmValues(helm_values string) error {
	var client *redis.Client
	var err error

	client, err = getRedisClient()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err = client.Set(ctx, h.helmValueCacheKey(), helm_values, 0).Err(); err != nil {
		return fmt.Errorf("Set helm values to redis failed: %s\n", helm_values)
	}

	return nil
}

func (h *HelmCli) loadHelmValues() error {
	helm_values := utils.Getenv("HELM_VALUES")

	if len(helm_values) == 0 {
		if helm_cached_values, err := h.loadRedisHelmValues(); err != nil {
			return err
		} else if len(helm_cached_values) > 0 {
			h.Values = parseHelmValues(helm_cached_values)
			log.Printf("load previous helm values from redis: %s\n", helm_cached_values)
		} else {
			log.Print("helm values in redis is empty\n")
		}
	} else {
		log.Printf("load helm values from env.: %s\n", helm_values)
		h.Values = parseHelmValues(helm_values)
		if err := h.saveRedisHelmValues(helm_values); err != nil {
			return err
		}
		log.Printf("update helm values to redis: %s\n", helm_values)
	}

	return nil
}
