# Configuration

Everything is stored in `~/.lgtmfaster/config.json`: PATs, linked repositories, pipeline favorites, the last session and these settings (edit them with `:settings`):

| Setting | Default | |
|---|---|---|
| `HideOldOtherPRs` | `false` | Hide other people's PRs older than `HideOldOtherPRsDays` |
| `HideOldOtherPRsDays` | `30` | |
| `PipelineSortMode` | `favorites` | Last used pipeline sort |
| `PipelineAutoRefreshEnabled` | `false` | Auto-refresh the pipeline views |
| `PipelineAutoRefreshSeconds` | `15` | How often |
| `RecentBranchMinutes` | `60` | How far back `:pr` looks for pushed branches |
