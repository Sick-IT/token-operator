# Token Operator CLI cron Helm Chart: tocli-cron

![Version: 0.0.2](https://img.shields.io/badge/Version-0.0.2-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 0.3.2](https://img.shields.io/badge/AppVersion-0.3.2-informational?style=flat-square)

## What is Token Operator?

[Token Operator](https://gitlab.com/sickit/token-operator) is a tool to automate rotation of your GitLab tokens.

## Prerequisites

## Requirements

- A running Kubernetes cluster
- [Kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) installed and setup to use the cluster
- [Helm](https://helm.sh/) [installed](https://github.com/helm/helm#install) and setup to use the cluster (helm init)

## Deploy Token Operator

The fastest way to install Token Operator using Helm is to deploy it from our public Helm chart repository.
First, add the repository and list its contents with these commands:

```console
helm repo add toop https://gitlab.com/api/v4/projects/sickit%2Ftoken-operator/packages/helm/stable
helm repo update
helm search repo toop
```

Next, install the chart with custom values in the `tocli` namespace:

```console
helm show values toop/tocli-cron > values.yaml
# edit values.yaml and define your token-operator config
helm install tocli-cron toop/tocli-cron --namespace tocli --values values.yaml
```

This will deploy a single Token Operator CLi cronjob instance in the `tocli` namespace with your token-operator configuration.

Unstall with the following commands:

```console
helm uninstall tocli-cron --namespace tocli
```

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| affinity | object | `{}` |  |
| config | object | `{"default_rotation":{"rotate_before":"168h","validity":"888h"},"tokens":[{"name":"mytoken","source":{"description":"describe what you use it for or where you use it","name":"my-token","scopes":["read_api"],"type":"personal"},"state":"active","vault":{"field":"password","item":"my-gitlab-token","path":"my-token-vault"}}]}` | Token-operator configuration, see https://gitlab.com/sickit/token-operator/-/blob/main/pkg/toop/full-config.yaml |
| config.default_rotation.rotate_before | string | `"168h"` | Time in hours when to rotate a token before it expires. Default: 168h (= 1 week). Also supports minutes (m) and seconds (s). |
| config.default_rotation.validity | string | `"888h"` | GitLab token validity in hours when rotating a token. Default: 888h (= 5 weeks). Also supports minutes (m) and seconds (s). |
| config.tokens[0].name | string | `"mytoken"` | Token name, mentioned in logs. |
| config.tokens[0].source.description | string | `"describe what you use it for or where you use it"` | GitLab token description. |
| config.tokens[0].source.name | string | `"my-token"` | GitLab token name. |
| config.tokens[0].source.scopes | list | `["read_api"]` | GitLab token scopes, see https://docs.gitlab.com/user/profile/personal_access_tokens/#personal-access-token-scopes. |
| config.tokens[0].source.type | string | `"personal"` | GitLab token type: personal, project or group. |
| config.tokens[0].state | string | `"active"` | Token state: active, inactive or deleted. |
| config.tokens[0].vault.field | string | `"password"` | Vault item secret field. |
| config.tokens[0].vault.item | string | `"my-gitlab-token"` | Vault item name or ID. |
| config.tokens[0].vault.path | string | `"my-token-vault"` | Vault name/path. |
| failedJobHistoryLimit | int | `3` |  |
| fullnameOverride | string | `""` | This is to override the full name. |
| image.pullPolicy | string | `"IfNotPresent"` | This sets the pull policy for images. |
| image.repository | string | `"registry.gitlab.com/sickit/token-operator"` |  |
| imagePullSecrets | list | `[]` | This is for the secrets for pulling an image from a private repository. More information can be found here: https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/ |
| nameOverride | string | `""` | This is to override the chart name. |
| nodeSelector | object | `{}` |  |
| podAnnotations | object | `{}` |  |
| podLabels | object | `{}` |  |
| podSecurityContext | object | `{}` |  |
| resources | object | `{}` |  |
| schedule | string | `"3 7 * * *"` | Cronjob schedule, see https://kubernetes.io/docs/concepts/workloads/controllers/cron-jobs/#schedule-syntax example: daily at 7:03 |
| securityContext.capabilities.drop[0] | string | `"ALL"` |  |
| securityContext.readOnlyRootFilesystem | bool | `true` |  |
| securityContext.runAsNonRoot | bool | `true` |  |
| securityContext.runAsUser | int | `1000` |  |
| source.existingSecret | object | `{}` | Reference an existing Secret, managed for example with external-secrets. Recommended. |
| source.token | string | `""` | GitLab token with `api` access, plain text. Not recommended. |
| source.url | string | `"https://gitlab.com/api/v4"` | GitLab API URL. |
| successfulJobHistoryLimit | int | `3` |  |
| tolerations | list | `[]` |  |
| vault.existingSecret | object | `{}` | Reference an existing Secret, managed for example with external-secrets. Recommended. |
| vault.token | string | `""` | Vault token, plain text. Not recommended. |
| vault.url | string | `""` | Vault URL, required only for Hashicorp Vault. |
| volumeMounts | list | `[]` |  |
| volumes | list | `[]` |  |

## Testing

Using `chart-testing` to lint, install and test the chart on a local Kubernetes (Minikube, Rancher Desktop, ...)

```shell
ct lint-and-install --all
```
