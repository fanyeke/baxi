# adapter: Channel Adapters

**Branch:** main

## OVERVIEW
Strategy pattern implementations for dispatching actions to external channels. 10 files.

## WHERE TO LOOK

| Task | File | Notes |
|------|------|-------|
| Feishu notifications | `feishu.go` | Sends alert messages via Feishu API |
| GitHub issues | `github.go` | Creates GitHub issues with labels |
| CLI output | `cli.go` | Writes to local log files |
| Manual review | `manual.go` | Creates review tasks for human approval |
| Channel mapping | `domain.go` | Maps action types to channels |

## KEY PATTERNS

- **Strategy pattern**: All adapters implement `ActionExecutor` interface
- **Dry-run support**: Every adapter checks dryRun flag before external calls
- **Channel independence**: Each adapter can be tested in isolation with mocks
- **Test coverage**: 57 tests across adapter test files (near-complete coverage)

## ANTI-PATTERNS

- Feishu client.go has inline token caching with expiry — should use a proper token manager
- BuildLabels in github.go does dedup but test expected 2 labels (now fixed to expect 1)
