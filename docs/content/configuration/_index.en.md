+++
date = '2025-09-21T21:15:47+02:00'
title = "Configuration"
linkTitle = "Configuration"
weight = 20
[params]
  menuPre = "<i class='fa-fw fas fa-gears'></i> "
+++

## Example configuration

To configure token-operator you need to provide a configuration file in `yaml` or `json`.

Here's an example of a full configuration:

```yaml
{{% include file="configuration/full-config.yaml" %}}
```

### Global options

All global option can also be provided on the command line or through environment variables.
For example: `--dry-run` or `DRY_RUN`, see `tocli --help`.

- `dry_run`: check source and vault, but do not change anything.
- `force_rotate`: sets `rotate_before` to more than one year for all tokens to force rotation.
- `license`: an Enterprise license key for HashiCorp Vault or group/project access tokens.
  For an Enterprise license key, please contact us at toop@sickit.eu.
- `source.url`: the API URL of the GitLab instance.
- `vault.type`: `1password` (default) or `hashicorp`.
- `vault.url`: the HashiCorp Vault URL.

### Defining rotation

Rotation parameter can be defined globally with `default_rotation` or per token with `rotation`.

- `rotate_before`: the amount of hours before the token expires when to start rotating the token. `168h` is one week.
- `validity`: for how long a rotated token should be valid, also in hours. `840h` is 5 weeks.

### Token attributes

- `name`: the name of the token that appears in logs.
- `state`: one of `active`, `inactive` or `deleted`. If `deleted`, the GitLab token and vault item will be revoked/deleted respectively.
- `rotation`: see above
- `source`: see below
- `vault`: see below

### Defining source

The source is a GitLab access token.

- `name`: must match the name of a GitLab access token.
- `description`: is used when creating a new group or project access token.
- `type`: must be one of: `personal`, `group` or `project`
- `scopes`: defines the permissions of the token, see 
  https://docs.gitlab.com/user/profile/personal_access_tokens/#personal-access-token-scopes 
  or https://docs.gitlab.com/user/group/settings/group_access_tokens/#scopes-for-a-group-access-token 
  or https://docs.gitlab.com/user/project/settings/project_access_tokens/#scopes-for-a-project-access-token
- `owner`: required for `type: group|project`, the full path of the group or project.
- `role`: required for `type: group|project`, defines the access role of the access token, see
  https://docs.gitlab.com/user/permissions/#roles

### Defining vault

The vault defines a password vault item. The vault type can be defined in the config or via `--vault.type` on the command line.
The vault URL is only required for `vault.type: hashicorp`.

- `path`: the path to the vault.
- `item`: the name of the vault item.
- `field`: the field of the vault item.
- Optional unique identifiers: some password managers use or require unique identifiers, as names are not unique and may change. 
  If they are provided, they are used to identify an item instead of matching the name.
  - `orgID`: organization UUID, required for [Bitwarden](https://bitwarden.com/)
  - `pathID`: vault/project UUID, used to uniquely identify vault/project when provided
  - `itemID`: item/secret UUID, used to uniquely identify item/secret when provided

## Configuring multiple tokens

You can configure as many tokens as you like in one configuration file. 
They will all use the same GitLab and vault token for authentication though.

So if you intend to use token-operator with different GitLab or vault credentials, each of them needs its own configuration file.
