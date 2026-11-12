# Commands and keys

## Commands

`:` opens the command bar; `tab` completes command names (and branches after `:pr `).

| Command | Aliases | |
|---|---|---|
| `:pats` | `:p`, `:pat` | Manage PATs |
| `:pr` | `:prs`, `:pulls` | List pull requests |
| `:pr <repository>:<branch>` | | Create a pull request from a branch |
| `:dashboard` | `:dash`, `:d` | Dashboard |
| `:pipelines` | `:pl`, `:pipes` | Pipelines (Azure DevOps) |
| `:session` | `:resume` | Go back to where you were when you last quit |
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

### Dashboard

The app starts on the dashboard; `:dashboard` (or `D` in the PR list) brings you back. It shows four panes:

| | |
|---|---|
| **Waiting for your review**: assigned to you, not reviewed by you yet, not draft | **Failing checks**: your PRs whose latest builds or checks failed (the failing ones are named) |
| **Created by you**: all your open PRs, with their check status | **Ready to merge**: your PRs that are approved, mergeable, not draft, with passing or no checks |

Each pane has its own cursor: `tab` / `shift+tab` move to the next / previous pane, `→` / `←` to the pane beside, `j`/`k` within the focused (yellow) pane. Checks are fetched for your own PRs when the dashboard opens (Azure DevOps: the latest build of each pipeline on the PR's merge ref). `enter` inspects a PR and going back returns to the dashboard; `r` reloads.

### Pull request list

The details pane (`tab`) shows the PR's info, including how many comment threads are resolved (`💬 1/3 resolved`, also in the top bar while inspecting), and its description. Rows are marked `✎` yours, `→` assigned to you, `○` other; `✓`/`✗`/`◯` is the review state.

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
