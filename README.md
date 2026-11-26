# LGTMFaster

A fast, keyboard-driven TUI for reviewing pull requests and monitoring pipelines on Azure DevOps and GitHub.

Manage PRs across multiple accounts, review diffs, leave comments, approve, merge, and work with local repositories — without leaving the terminal.

![Dashboard](docs/img/dashboard.png)

### Review pull requests

Browse diffs with vim motions, search files, comment on lines, approve, request changes, and merge. Filter whitespace changes, switch diff modes, and navigate files with fuzzy search (`<space><space>`).

![Diff view](docs/img/diff.png)

### Monitor pipelines

Browse Azure DevOps pipelines across projects, track runs and stages, manage approvals, and inspect logs with search and follow mode.

![Pipeline logs](docs/img/logs.png)

## Install

```bash
brew install --cask jaforsgren/tap/lgtmfaster
```

Or build from source:

```bash
make build      # build ./lgtmfaster
make install    # install to $GOPATH/bin
```

Or build directly with Go:

```bash
go build -o lgtmfaster ./cmd/lgtmfaster
```

Run with `lgtmfaster`, or use `lgtmfaster --debug /tmp/lgtm.log` to write logs to a file.

## Getting started

1. Launch the app and press `a` to add a personal access token (PAT) for GitHub or Azure DevOps.
2. Press `space` to select one or more accounts, then `enter` to load them.
3. Start reviewing PRs or browsing pipelines.

Selected accounts load automatically on subsequent launches. Use `:session` to restore your previous view.

## Pull requests from the command line

Run from inside a repository whose branch is pushed to `origin` (GitHub or Azure DevOps):

```bash
lgtmfaster -pr                          # create a PR from the current branch into main
lgtmfaster -pr -t develop -a 1234       # into develop, linked to work item 1234
lgtmfaster -pr -de                      # write the description in $EDITOR first
lgtmfaster -pr -i -a "1234,5678"        # open the TUI's new PR form, prefilled
```

| Flag | Meaning |
|---|---|
| `-i`, `--interactive` | Open the new pull request form in the TUI instead of creating it |
| `-a`, `--ado` | Work item numbers to link (`1234` or `"1234,5678"`) |
| `-d`, `--description` | Description text |
| `-df`, `--description-file` | File holding the description |
| `-de`, `--description-editor` | Write the description in `$EDITOR`; save and quit submits, `:cq` aborts |
| `-t`, `--target` | Branch to merge into (default `main`; with `-i`, the repository's default branch) |

The title comes from the branch name (`feature/add-login` → "Add login"). The PR is created with the account whose provider and organization match the remote; `-i` needs that account selected.

## Commands and keys

| Command | Action |
|---|---|
| `:dashboard` | Open dashboard |
| `:pr` | Browse pull requests |
| `:pr <repo>:<branch>` / `pr` | Create a pull request (Azure DevOps) |
| `:pipelines` | Browse Azure DevOps pipelines |
| `:session` | Restore previous view |
| `:settings` | Edit configuration |

| Key | Action |
|---|---|
| `j` / `k`, `enter` | Navigate and open |
| `/` | Search or filter |
| `d` | View PR diff |
| `a` / `r` | Approve / request changes |
| `ctrl+s` | Submit review |
| `q` / `ctrl+c` | Back / quit |

See [all commands and shortcuts](docs/commands-and-keys.md).

## Configuration

Configuration is stored in `~/.lgtmfaster/config.json`. Use `:settings` to edit it in `$EDITOR`.

See [configuration](docs/configuration.md) for available options.

## Development

```bash
make test    # race detector and coverage
make lint    # golangci-lint
make fmt
```