+++
date = '2025-09-21T21:15:47+02:00'
title = "Initial setup"
weight = 10
+++

For the initial setup we will
1. discover existing access tokens of a GitLab instance.
2. configure and run `tocli`, the token-operator CLI.

### Discovering existing tokens

To discover all existing tokens of a GitLab instance, we provide a small shell script that can also be modified easily.
The output is compatible with token-operator and can be used to define your configuration.

If you use an access token of an account with admin permissions, the script will also list all tokens of users.

```shell
curl -Lo dump-tokens.sh "https://gitlab.com/sickit/token-operator/-/raw/main/scripts/dump-tokens.sh?ref_type=heads"

export GITLAB_HOST=https://gitlab.com
export GITLAB_TOKEN=...
bash ./dump-tokens.sh 2> /dev/null | tee alltokens.yaml
```

### Downloading token-operator

For releases and binaries, please refer to https://gitlab.com/sickit/token-operator/-/releases

```shell
OS=linux
ARCH=amd64
VERSION=0.3.3
curl -Lo tocli https://gitlab.com/sickit/token-operator/-/releases/v${VERSION}/downloads/tocli_${VERSION}_${OS}_${ARCH}
chmod +x tocli
./tocli --help
```

### Running token-operator in a container

You can also run token-operator CLI in a container:

```shell
docker run --rm -it registry.gitlab.com/sickit/token-operator:0.3.3 --help
```

### Configuring token-operator self-rotating GitLab access token

For the initial setup, we will use the GitLab access token you will create below and rotate itself.
It will be stored in a vault named `tocli-setup` as item `tocli-pat`, or adjust the `vault` attributes below to your needs.

Create the file `tocli-initial-setup.yaml` with the following contents:
```yaml
# tocli-initial-setup.yaml
tokens:
  # rotate the token we use to rotate tokens (self-rotate)
  - name: "tocli-setup" # this name appears in logs
    state: "active"
    rotation:
      rotate_before: 168h # 1 week, token-operator will attempt rotation 1 week before it expires
      validity: 840h # 5 weeks
    source:
      name: "tocli-pat"
      description: "Token used by token-operator CLI to rotate tokens"
      type: "personal"
      scopes:
        - "api"
    vault:
      path: "tocli-setup"
      item: "tocli-pat"
      field: "password"
```

### Running token-operator with 1Password vault

Prerequisites

- Create a personal access token in GitLab with scopes `api` for the token-operator called `tocli-pat`.
  The person creating the PAT must have permissions to edit access tokens that are in the configuration.
- [Create a 1Password service account](https://developer.1password.com/docs/service-accounts/get-started/) with read/write access to the vault where you want to store your GitLab tokens.

```shell
tocli --source.token glpat-.... --vault.token ops-ey... \
    --config tocli-initial-setup.yaml --log.format console [--dry-run]
```

### Running token-operator with HashiCorp Vault (Enterprise version)

Prerequisites

- Create a personal access token in GitLab with scopes `api` for the token-operator called `tocli-pat`.
  The person creating the PAT must have permissions to edit access tokens that are in the configuration.
- Provide `--vault.type hashicorp` and `--vault.url` to a HashiCorp Vault instances along with the `--vault.token` that has
  permissions to create and update vault items in the configuration.
- Add `license` to the config or use `--license` on the command line to provide an Enterprise license key.

```shell
tocli --source.token glpat-.... --vault.type hashicorp --vault.url=https://vault... \
  --vault.token ... --config tocli-initial-setup.yaml --log.format console [--dry-run] \
  --license ...
```

For an Enterprise license key, please contact us at toop@sickit.eu.

### Example output

Here is an example console output with `--log.format console` or `LOG_FORMAT=console`:

```console
INFO reconciling token svc=tocli name=tocli-setup type=personal
INFO checking 1password vault item svc=tocli path=op://tocli-setup/tocli-pat/password
INFO skipping rotation, vault item available and token still valid svc=tocli name=tocli-setup secret=T...5 rotateBefore=168h0m0s expireDuration=514h2m54.503865s expireDate=2025-08-28 00:00:00 +0000 UTC
```
