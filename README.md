# argocd-helm-ext-plugin

An Argo CD plugin to retrieve helm chart from remote helm repository and use values files and custom values on local git repository.

## What can this plugin do?
* Set `spec.source.repoURL` to your-own git repository, which contains value files, and uses a helm chart on the remote Helm chart repository.
* Support multiple values YAML files for a better management experience.
* Set additional values by passing `HELM_VALUES` plugin environment variable via `argocd app set <APP_NAME> --plugin-env "HELM_VALUES=key1=value1;key2=value2"`

## Why do you needs this plugin?
The current Argo CD (v2.4) does not support using a published helm chart on a remote repository and assigning custom value settings in files on another git repository (usually, it is your repo.)
All the value files must be in the same repository as the Helm chart, which has some drawbacks.

* Need to write value settings in `spec.source.helm.values`. It is hard to write and maintain.
* Argo CD will not know when settings changed in the application YAML file because the `repoURL` points to the remote repository, so it needs to use `app-of-apps pattern` for bootstrapping and syncing the changes.

> This plugin provides an alternative way to point to the same git repository URL containing helm value settings.
> Moreover, use the delicate value files to manage Helm chart settings.

## Plugin environment variables

| Name  | Description |
|---|---|
| HELM_REPO_URL | The helm chart repository url, e.g.: `https://charts.bitnami.com/bitnami` |
| HELM_CHART | The helm chart name, e.g.: `nginx` |
| HELM_CHART_VERSION | The helm chart vesion, e.g.: `12.0.0` |
| HELM_VALUE_FILES | The values files, space-separated, e.g.: `values.yaml secrects.yaml` |
| HELM_VALUES | The additional values, semicolon-separated, e.g: `key1=values;key2=values` |

## Installation
Please refer [Argo CD Config Management Plugins Installation](https://argo-cd.readthedocs.io/en/stable/user-guide/config-management-plugins/#option-1-configure-plugins-via-argo-cd-configmap). It recommends to configure this plugin via Argo CD configmap.

**argocd-cm ConfigMap example**
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: argocd-cm
data:
  configManagementPlugins: |-
    - name: argocd-helm-ext-plugin
      generate:
        command: ["argocd-helm-ext-plugin"]
        args: ["--include-crds"]
```

## Plugin arguments
```
Usage of argocd-helm-ext-plugin:
  -d, --debug        debug mode
  -c, --include-crds include custom resource
  -h, --help         prints help information
```

## How to override helm values on the fly?
For some use cases, we need to specify some helm chart's values during deployment, e.g.: we might need to specify the `image.tag` to use specific image version.
For this case we can set `HELM_VALUES` via Argo CD CLI.

Example:
```bash
# set image.tag and image.imagePullPolicy
argocd app set <APP_NAME> --plugin-env HELM_VALUE="image.tag=afeacb7;image.imagePullPolicy=Always"

# It's a argocd weird issue.,we need to hard refresh the application to invalidate manifest cache
# to make app. sync invokes plugin (tested on argocd v2.4.3)
argocd app get <APP_NAME> --hard-refresh

# start to sync app
argocd app sync <APP_NAME>
```
