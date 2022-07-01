package cmd

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/Xfers/argocd-helm-ext-plugin/pkg/config"
	"github.com/Xfers/argocd-helm-ext-plugin/pkg/helm"
)

const usage = `Usage of argocd-helm-ext-plugin:
  -d, --debug        debug mode
  -c, --include-crds include custom resource
  -h, --help         prints help information
`

func ArgoCDHelmExtPlugin() {
	var dev_mode bool
	var log_env bool
	var opts config.Options

	flag.BoolVar(&dev_mode, "d", false, "debug mode")
	flag.BoolVar(&dev_mode, "debug", false, "debug mode")
	flag.BoolVar(&log_env, "e", false, "log environment variables")
	flag.BoolVar(&log_env, "env", false, "log environment variables")
	flag.BoolVar(&opts.IncludeCrds, "c", false, "include custom resource")
	flag.BoolVar(&opts.IncludeCrds, "include-crds", false, "include custom resource")

	flag.Usage = func() { fmt.Print(usage) }

	flag.Parse()

	log.SetFlags(log.Ldate | log.Lmicroseconds)

	app_namespace := os.Getenv("ARGOCD_APP_NAMESPACE")
	app_name := os.Getenv("ARGOCD_APP_NAME")

	if !dev_mode {
		log_filename := fmt.Sprintf("/tmp/plugin-%s-%s.log", app_namespace, app_name)
		logFile, err := os.OpenFile(log_filename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			os.Exit(1)
		}
		log.SetOutput(logFile)
	}

	if log_env {
		if env_content, err := exec.Command("sh", "-c", "export").Output(); err == nil {
			env_content_filename := fmt.Sprintf("/tmp/env-%s-%s.log", app_namespace, app_name)
			os.WriteFile(env_content_filename, env_content, 0644)
		}
	}

	cli, err := helm.New(&opts)
	if err != nil {
		log.Fatalf("[Error]: %s\n", err.Error())
	}

	if err := cli.GenerateTemplate(); err != nil {
		log.Fatalf("[Error] Generate %s v%s template fail\n", cli.Chart, cli.ChartVersion)
	}
}
