# LGTMFaster

Fast, keyboard-driven TUI for reviewing pull requests and following pipelines on Azure DevOps and GitHub.

## Features

- **Pull requests across accounts**: list, filter and inspect PRs from several PATs at once, categorised as yours, assigned to you, or other
- **Reviews**: read diffs (full or changed lines only), comment on lines, approve, request changes, merge, edit the description
- **Create pull requests**: `:pr` suggests the branches you pushed recently and opens a form with target branch, title, draft flag and description
- **Comments**: every comment with the code it is about; votes shown as "approved", "rejected", …; export them all as markdown
- **Pipelines** (Azure DevOps): every pipeline of every project in one foldable list with favorites, runs, stages and approvals, with optional auto-refresh
- **Local repositories**: link a repo to a folder, then check out a PR's branch or pull from the TUI
- **Session resume**: reopens the view, PR, pipeline or run you were on when you quit

### Provider support

| | Azure DevOps | GitHub |
|---|---|---|
| List, inspect and review PRs | ✓ | ✓ |
| Inline comments | ✓ | ✓ |
| Merge, edit description | ✓ | ✓ |
| Create PRs | ✓ | implemented, not yet verified |
| Pipelines | ✓ | – |

## Install

```bash
make build      # builds ./lgtmfaster with version info
make install    # installs to $GOPATH/bin
```

Or with Go directly: `go build -o lgtmfaster ./cmd/lgtmfaster`.

```bash
./lgtmfaster                          # run
./lgtmfaster --debug /tmp/lgtm.log    # also write the log to a file
```

## First-time setup

1. Start the app; it opens on the PAT list.
2. Press `a` to add a PAT and fill in the name, token, provider (`github` or `azuredevops`), your username and, for Azure DevOps, the organization.
3. Press `space` to select the PATs to use (several can be selected; one is primary) and `enter` to load their pull requests.

On later starts the selected PATs load straight away, in the view you were on when you quit.

## The screen

```
 LGTMFaster
 CONTEXT                   │ ACCOUNTS 1/2               │ SHORTCUTS
 📍 Mode: PR Diff          │ ● work-ado (primary)       │ <c> View comments   <i> Inline comment on line
 📦 Repo: proj/payments    │ ○ personal-github          │ <a> Approve PR      <f> Toggle diff view mode
 📋 PR: #1287 [OPEN ✓]     │                            │ …
╭──────────────────────────────────────────────────────────────────────────────╮
│  / Pull Requests                                                             │
│  …                                                                           │
╰──────────────────────────────────────────────────────────────────────────────╯
 :command bar
 status message
 j/k: navigate | enter: select | /: filter | :: command …
```

- **Top bar**: where you are, the accounts in use, and the actions of the current view.
- **Command bar**: `:` opens it; `tab` completes command names (and branches after `:pr `).
- **Status bar**: messages, and the navigation keys of the current view.

## Commands

| Command | Aliases | |
|---|---|---|
| `:pats` | `:p`, `:pat` | Manage PATs |
| `:pr` | `:prs`, `:pulls` | List pull requests |
| `:pr <repository>:<branch>` | | Create a pull request from a branch |
| `:pipelines` | `:pl`, `:pipes` | Pipelines (Azure DevOps) |
| `:merge` | `:m` | Merge the inspected PR |
| `:link [path]` | `:l` | Link the PR's repository to a local folder |
| `:repos` | | Manage linked repositories |
| `:settings` | `:s` | Edit the config file in `$EDITOR` |
| `:logs` | `:log` | Session log |
| `:help` | `:h`, `:?` | List the commands |
| `:quit` | `:q`, `:exit` | Quit |

## Keys

Keys mean the same thing in every list.

| Key | |
|---|---|
| `j`/`k`, `↑`/`↓` | Move |
| `enter` | Open / select |
| `/` | Filter (`esc` clears it) |
| `tab` | Toggle the details pane next to the list |
| `→` / `←` | Focus the details pane (then `j`/`k` scroll it) / back to the list |
| `r` | Refresh |
| `ctrl+o` | Open in the browser |
| `h`, `backspace`, `esc` | Back |
| `q` | Back; quits from the PAT list |
| `ctrl+c` | Quit from anywhere |

### Pull request list

The details pane (`tab`) shows the PR's info and description. Rows are marked `✎` yours, `→` assigned to you, `○` other; `✓`/`✗`/`◯` is the review state.

### Inspecting a pull request

| Key | |
|---|---|
| `d` | Diff (`q` goes back to the description) |
| `n`/`p`, `→`/`←` | Next / previous file |
| `f` | Toggle full file / changed lines only |
| `w` | Hide / show whitespace-only changes |
| `F` | Choose which file extensions the diff shows |
| `y` / `Y` | Copy the shown file's diff / all shown diffs (after the `w` and `F` filters) |
| `i` | Comment on the line under the cursor |
| `enter` | Write a review comment, posting the queued comments without voting |
| `a` / `r` | Approve / request changes, posting the queued comments too |
| `ctrl+s` | Submit the review |
| `C` | Commits; `enter` opens a commit's diff, `q` steps back (commit → commits → PR diff → description) |
| `c` | Comments view; `ctrl+o` there opens all comments as markdown |
| `e` | Edit the description (`ctrl+g` opens `$EDITOR`) |
| `m` | Merge |
| `L` | Link the repository to a local folder |
| `g` / `P` / `o` | Check out the branch / pull / open the folder (linked repositories) |

Inline comments show the file, line and surrounding code while you write them. They are queued (💭 in the diff) and posted with the review.

### Creating a pull request

Type `:pr ` (with a space): the status bar lists the branches you pushed in the last hour that have no open PR yet, and `tab` cycles through them. `enter` opens the form:

| Key | |
|---|---|
| `tab` / `shift+tab` | Next / previous field |
| `↑`/`↓`, `enter` | Pick the target branch (type to filter) |
| `space` | Toggle draft |
| `ctrl+g` | Write the description in `$EDITOR` |
| `ctrl+s` | Create the PR and open it |
| `esc` | Cancel |

### Pipelines (Azure DevOps)

| Key | |
|---|---|
| `enter` | Unfold a group, open a pipeline's runs, or open a run |
| `z` / `Z` | Fold the group / unfold a pipeline's latest runs; fold or unfold everything |
| `f` | Favorite the pipeline, repository, project or run |
| `F` | Show favorites only |
| `0`–`5` | Sort: favorites & recently viewed, project, repository, name, last run, status |
| `R` | Toggle auto-refresh (shown in the top bar) |
| `a` / `x` | Approve / reject the selected stage (run view) |

Favorites are listed in their own section at the top of the list.

## Configuration

Everything is stored in `~/.lgtmfaster/config.json`: PATs, linked repositories, pipeline favorites, the last session and these settings (edit them with `:settings`):

| Setting | Default | |
|---|---|---|
| `HideOldOtherPRs` | `false` | Hide other people's PRs older than `HideOldOtherPRsDays` |
| `HideOldOtherPRsDays` | `30` | |
| `PipelineSortMode` | `favorites` | Last used pipeline sort |
| `PipelineAutoRefreshEnabled` | `false` | Auto-refresh the pipeline views |
| `PipelineAutoRefreshSeconds` | `15` | How often |
| `RecentBranchMinutes` | `60` | How far back `:pr` looks for pushed branches |

## Development

```bash
make test    # tests with the race detector and coverage
make lint    # golangci-lint
make fmt
```

```
cmd/lgtmfaster/      entry point
internal/
├── domain/          models and the Provider / Repository interfaces
├── provider/        Azure DevOps and GitHub implementations (+ shared diff and identifier parsing)
├── storage/         config.json persistence
├── git/             local repository operations
└── ui/              Bubble Tea app: command registry, layout, session
    ├── components/  top bar, command bar, status bar
    └── views/       lists, split panes, PR inspect, forms
```

Provider-specific logic stays behind the `domain.Provider` interface; pipelines are the optional `domain.PipelineProvider`. See `CLAUDE.md` for the architecture notes.
