# ShipItTTY Architecture

## Package Structure

```
shipittty/
├── cmd/
│   └── shipittty/
│       └── main.go                 # Entry point
├── internal/
│   ├── domain/
│   │   ├── models.go              # Core domain models (PR, Comment, Diff, etc.)
│   │   ├── provider.go            # Provider interface
│   │   └── repository.go          # Repository interface for storage
│   ├── provider/
│   │   ├── github/
│   │   │   ├── client.go          # GitHub API client
│   │   │   └── provider.go        # GitHub provider implementation
│   │   └── azuredevops/
│   │       ├── client.go          # Azure DevOps API client
│   │       └── provider.go        # Azure DevOps provider implementation
│   ├── storage/
│   │   ├── local.go               # Local file-based storage for PATs
│   │   └── models.go              # Storage models
│   ├── ui/
│   │   ├── app.go                 # Main application state/model
│   │   ├── commands.go            # Vim-style command parsing
│   │   ├── styles.go              # Lipgloss styles
│   │   ├── components/
│   │   │   ├── topbar.go          # k9s-style top bar
│   │   │   ├── statusbar.go       # Bottom status bar
│   │   │   └── commandbar.go      # Command input bar
│   │   └── views/
│   │       ├── pats.go            # PAT management view
│   │       ├── prlist.go          # PR list view
│   │       ├── prinspect.go       # PR inspection/detail view
│   │       └── diff.go            # Diff viewer component
│   └── service/
│       └── pr.go                  # PR service orchestrating providers
├── go.mod
└── go.sum
```

## Core Interfaces

### Provider Interface
Abstracts GitHub and Azure DevOps behind a common interface for:
- Listing pull requests
- Fetching PR details and diffs
- Adding comments
- Submitting reviews (approve, request changes, comment)

### Repository Interface
Abstracts storage operations for:
- Storing and retrieving PATs
- Managing provider configurations

## State Management

### Bubble Tea Model
- **App State**: Current view (PAT list, PR list, PR inspect)
- **Active PAT**: Currently selected PAT for API calls
- **PR Cache**: Cached PR data to avoid repeated API calls
- **Navigation Stack**: For returning to previous views

### Views
1. **PAT Management**: List/add/select PATs
2. **PR List**: Categorized list of PRs (authored, assigned, accessible)
3. **PR Inspect**: Detailed view with diffs and comments

## Command System

Vim-style commands:
- `:pats` or `:p` - Manage PATs
- `:pr` - List pull requests
- `:q` - Quit
- ESC - Return to previous view
- Enter - Select/drill down
- j/k - Navigate up/down
- gg/G - Jump to top/bottom

## Data Flow

```
User Input → Command Parser → App Update → Provider API → UI Render
                                    ↓
                              Local Storage
```

## Design Principles

1. **Clean Architecture**: UI → Service → Provider → API
2. **Interface Segregation**: Providers implement common interface
3. **Dependency Injection**: Pass dependencies through constructors
4. **Immutability**: Bubble Tea model updates return new state
5. **Single Responsibility**: Each package has one clear purpose
