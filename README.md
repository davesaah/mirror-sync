# Mirror Sync

Sync your local gitea repos to github, codeberg and gitlab.

## Interface Design

### mirror-sync

```
Usage:
  mirror-sync [flags]
  mirror-sync [command]

Available Commands:
  begin       Begin mirror syncing
  completion  Generate the autocompletion script for the specified shell
  config      Configure options for mirror sync
  help        Help about any command

Flags:
  -h, --help      Show help
  -v, --version   Show version

Use "mirror-sync [command] --help" for more information about a command.
```

## mirror-sync completion

Generate completion scripts for your shell.

```
Usage:
  mirror-sync completion [command]

Available Commands:
  bash        Generate the autocompletion script for bash
  fish        Generate the autocompletion script for fish
  powershell  Generate the autocompletion script for powershell
  zsh         Generate the autocompletion script for zsh

Flags:
  -h, --help   help for completion

Use "mirror-sync completion [command] --help" for more information about a command.
```

### mirror-sync config

This allows you to configure options and API tokens for mirror-sync to work.

Permissions for access tokens:

- github:
  - admin:repo_hook
  - delete_repo
  - repo
  - workflow
- codeberg:
  - write:organization
  - write:repository
  - write:user
- gitlab:
  - API
  - READ REPOSITORY
  - READ API
  - WRITE REPOSITORY
- local gitea:
  - write:organization
  - write:repository

Config file is found at `$HOME/.config/mirror-sync.json`.

```
Usage:
  mirror-sync config [flags]

Flags:
      --codeberg-token string    Add codeberg access token
      --external-user string     Set username on cloud providers
      --github-token string      Add github access token
      --gitlab-token string      Add gitlab access token
  -h, --help                     help for config
      --local-url string         Set local gitea server url (e.g. <https://gitea.com>)
      --localhost-token string   Add local gitea server access token
```

All these options need to be set for the mirror-sync to work. There are no defaults. Everything is in your hands.

Optionally, you can create the `$HOME/.config/mirror-sync.json` and populate it according to this format:

```json
{
  "codeberg-token": "",
  "external-user": "",
  "github-token": "",
  "gitlab-token": "",
  "local-url": "",
  "localhost-token": ""
}
```

### mirror-sync begin

Start to sync the defined local gitea repository to github, codeberg and gitlab.

```
Usage:
  mirror-sync begin [flags]

Flags:
  -h, --help           help for begin
  -u, --owner string   Local owner (user/organization)
  -p, --private        Set visibility of the repo to private
  -n, --repo string    Name of repository
```

Examples:

Sync to a private repo

```bash
mirror-sync begin --owner learning --repo ai-use-cases --private
```

Sync to a public repo

```bash
mirror-sync begin --owner learning --repo ai-use-cases
```
