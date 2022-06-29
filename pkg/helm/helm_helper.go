package helm

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/Xfers/argocd-helm-ext-plugin/pkg/config"
	"github.com/Xfers/argocd-helm-ext-plugin/pkg/utils"
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

	h.loadHelmValues()

	return h, nil
}

func helmValuesHistoryPath(revision string) string {
	return fmt.Sprintf("/tmp/values_%s", revision)
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

	helm_template_file, err := os.OpenFile("/tmp/generated_template.yaml", os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		os.Exit(1)
	}
	helm_template_file.Write(out.Bytes())
	helm_template_file.Close()

	fmt.Print(out.String())

	if err := h.saveHelmValues(); err != nil {
		return err
	}

	return nil
}

func (h *HelmCli) loadHelmValues() {
	helm_values := utils.Getenv("HELM_VALUES")

	if len(helm_values) == 0 {
		value_store_file_path := helmValuesHistoryPath(h.AppRevision)
		if _, err := os.Stat(value_store_file_path); err == nil {
			if content, read_err := os.ReadFile(value_store_file_path); read_err == nil {
				h.Values = parseHelmValues(string(content))
				log.Printf("load previous helm values from %s\n", value_store_file_path)
			}
		}
	} else {
		log.Printf("load helm values: %s\n", helm_values)
		h.Values = parseHelmValues(helm_values)
	}
}

func (h *HelmCli) saveHelmValues() error {
	if len(h.Values) == 0 {
		return nil
	}

	var values []string
	for k, v := range h.Values {
		values = append(values, fmt.Sprintf("%s=%s", k, v))
	}

	value_store_file_path := helmValuesHistoryPath(h.AppRevision)
	err := os.WriteFile(value_store_file_path, []byte(strings.Join(values, ";")), 0644)
	if err != nil {
		return errors.New(fmt.Sprintf("store helm values to %s fail", value_store_file_path))
	}
	log.Printf("save helm values to %s\n", value_store_file_path)
	return nil
}
