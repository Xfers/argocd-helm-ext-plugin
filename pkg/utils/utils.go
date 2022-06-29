package utils

import (
	"fmt"
	"os"
)

func Getenv(name string) string {
	prefixed_env_var := os.Getenv(fmt.Sprintf("ARGOCD_ENV_%s", name))
	if len(prefixed_env_var) < 1 {
		return os.Getenv(name)
	}
	return prefixed_env_var
}
