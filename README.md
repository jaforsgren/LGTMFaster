# LGTMFaster

Fast, keyboard-driven TUI for reviewing pull requests from GitHub and Azure DevOps.

## Features

- **Multi-Provider Support**: GitHub and Azure DevOps (GitHub fully implemented)
- **Personal Access Token Management**: Store and switch between multiple PATs
- **Smart PR Categorization**: Automatically categorizes PRs as authored, assigned, or other
- **Keyboard-First Navigation**: Vim-style commands and keyboard shortcuts
- **PR Inspection**: View diffs, comments, and PR metadata
- **Review Actions**: Approve, request changes, or comment on PRs
- **k9s-Style Top Bar**: Real-time stats showing PR count and repo count

1. Launch the application
2. Press `:` to open the command bar
3. Type `pats` or `p` to manage Personal Access Tokens
4. Press `a` to add a new PAT
5. Fill in:
   - Name: A friendly name for this PAT
   - Token: Your GitHub Personal Access Token
   - Provider: `github` or `azuredevops`
   - Username: Your GitHub username
6. Press `Enter` to save
7. Select the PAT and press `Enter` to activate it

**Note**: You can edit existing PATs by selecting them and pressing `e`

## Commands

**Vim-style Commands** (press `:` to activate):

- `:pats` or `:p` - Manage Personal Access Tokens
- `:pr` - List pull requests
- `:logs` - View session logs (scrollable, color-coded)
- `:q` - Quit

**PAT Management View**:

- `a` - Add new PAT
- `e` - Edit selected PAT
- `d` - Delete selected PAT
- `Enter` - Activate selected PAT

**PR List View**:

- `r` - Refresh PR list
- `Enter` - Inspect selected PR

**PAT Management View**:

- `a` - Add new PAT
- `e` - Edit selected PAT
- `d` - Delete selected PAT
- `Enter` - Activate selected PAT

**PR List View**:

- `r` - Refresh PR list
- `Enter` - Inspect selected PR

**PAT Management View**:

- `a` - Add new PAT
- `e` - Edit selected PAT
- `d` - Delete selected PAT
- `Enter` - Activate selected PAT

**PR List View**:

- `r` - Refresh PR list
- `Enter` - Inspect selected PR

**Navigation**:

- `j/k` or arrow keys - Navigate up/down in lists
- `Enter` - Select item or drill down
- `Esc` or `q` - Go back to previous view
- `/` - Filter/search (in PR list)

**PAT Management View**:

- `a` - Add new PAT
- `e` - Edit selected PAT
- `d` - Delete selected PAT
- `Enter` - Activate selected PAT

**PR List View**:

- `r` - Refresh PR list
- `Enter` - Inspect selected PR

**PR Inspection View**:

- `n/p` - Next/Previous file in diff
- `c` - Toggle comments visibility
- `a` - Approve PR
- `r` - Request changes
- `Enter` - Add comment

**Legend**:

- ✎ - Authored by you
- → - Assigned to you
- ○ - Other PRs you have access to

## Configuration

Configuration is stored in `~/.lgtmfaster/config.json`

## Project Structure

```
lgtmfaster/
├── cmd/lgtmfaster/          # Application entry point
├── internal/
│   ├── domain/              # Core domain models and interfaces
│   ├── provider/            # GitHub and Azure DevOps implementations
│   ├── storage/             # Local PAT storage
│   └── ui/                  # Bubble Tea TUI components
│       ├── components/      # Reusable UI components
│       └── views/           # Application views
```

## Architecture

The application follows clean architecture principles:

- **Domain Layer**: Defines core models and provider interfaces
- **Provider Layer**: Implements GitHub/Azure DevOps API clients
- **Storage Layer**: Handles local persistence of PATs
- **UI Layer**: Bubble Tea components and views

All provider-specific logic is abstracted behind the `Provider` interface, making it easy to add new providers.
