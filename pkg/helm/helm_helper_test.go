package helm_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/Xfers/argocd-helm-ext-plugin/pkg/config"
	"github.com/Xfers/argocd-helm-ext-plugin/pkg/helm"
)

func TestHelmHelper(t *testing.T) {

	t.Run("will return error without required environment variables", func(t *testing.T) {
		var opts config.Options
		var err error

		required_env_vars := map[string]string{
			"ARGOCD_APP_NAME":    "test",
			"HELM_REPO_URL":      "https://test.local",
			"HELM_CHART":         "test_chart",
			"HELM_CHART_VERSION": "1.0.0",
		}

		for k, v := range required_env_vars {
			expected := fmt.Sprintf("%s is empty", k)
			_, err = helm.New(&opts)
			if err == nil || err.Error() != expected {
				t.Errorf("expected \"%s\", got \"%s\"", expected, err.Error())
			}
			os.Setenv(k, v)
		}
	})
}
