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
	var opts config.Options

	flag.BoolVar(&dev_mode, "d", false, "debug mode")
	flag.BoolVar(&dev_mode, "debug", false, "debug mode")
	flag.BoolVar(&opts.IncludeCrds, "c", false, "include custom resource")
	flag.BoolVar(&opts.IncludeCrds, "include-crds", false, "include custom resource")

	flag.Usage = func() { fmt.Print(usage) }

	flag.Parse()

	log.SetFlags(log.Ldate | log.Lmicroseconds)

	if !dev_mode {
		log_filename := "/tmp/plugin.log"
		app_name := os.Getenv("ARGOCD_APP_NAME")
		if len(app_name) > 0 {
			log_filename = fmt.Sprintf("/tmp/plugin_%s.log", app_name)
		}
		logFile, err := os.OpenFile(log_filename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			os.Exit(1)
		}
		log.SetOutput(logFile)
	}

	if export_content, err := exec.Command("sh", "-c", "export").Output(); err == nil {
		f, open_err := os.OpenFile("/tmp/export.log", os.O_RDWR|os.O_CREATE, 0644)
		if open_err == nil {
			f.Write(export_content)
			f.Close()
		}
	} else {
		log.Printf("err: %v", err)
	}

	cli, err := helm.New(&opts)
	if err != nil {
		log.Fatalf("create helm cli error: %s\n", err.Error())
	}

	if err := cli.GenerateTemplate(); err != nil {
		log.Fatalf("Generate %s v%s template fail\n", cli.Chart, cli.ChartVersion)
	}
}
