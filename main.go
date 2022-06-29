package main

import (
	"os"

	"github.com/Xfers/argocd-helm-ext-plugin/cmd"
)

func main() {
	cmd.ArgoCDHelmExtPlugin()
	os.Exit(0)
}
