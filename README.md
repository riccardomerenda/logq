# logq

Fast, interactive log explorer for the terminal.

<!-- TODO: Add demo GIF here -->
<!-- ![logq demo](docs/demo.gif) -->

**logq** treats your log files like a queryable database. Point it at a file (or pipe logs in), and get an interactive TUI with filterable log lines, a time histogram, and a powerful query bar — all from your terminal, with zero setup.

## Install

```bash
# Go
go install github.com/riccardomerenda/logq@latest

# Binary — download from GitHub Releases
# https://github.com/riccardomerenda/logq/releases
```

## Quick Start

```bash
# Explore a log file
logq server.log

# Pipe from any command
kubectl logs myapp | logq
docker logs mycontainer 2>&1 | logq
cat server.log | logq

# Follow a growing file
logq -f /var/log/app.log

# Open a compressed file
logq server.log.gz
```

## Query Syntax

Type queries in the filter bar. Results update as you type.

```
error                              # full-text: any field contains "error"
level:error                        # exact: level field equals "error"
level:error AND service:auth       # compound filter
latency>500                        # numeric comparison
message~"timeout.*retry"           # regex match
NOT service:healthcheck            # negation
level:error AND latency>1000 AND NOT service:healthcheck
```

See [docs/query-syntax.md](docs/query-syntax.md) for the full reference.

## Keyboard Shortcuts

| Key | Action |
|---|---|
| `j` / `k` or `Up` / `Down` | Scroll log view |
| `PgUp` / `PgDn` | Page scroll |
| `Home` / `End` | Jump to start / end |
| `/` | Focus query bar |
| `Enter` | Execute query / Show record detail |
| `Escape` | Clear query / Close detail |
| `Tab` | Toggle focus between log view and histogram |
| `q` | Quit |

## Supported Formats

logq auto-detects the format of each line:

- **JSON Lines (JSONL)** — one JSON object per line
- **logfmt** — `key=value` pairs per line
- **Plain text** — unstructured lines (treated as a message field)

Mixed formats within the same file are handled gracefully.

## How It Works

```
┌──────────────────────────────────────────────────────────────┐
│  Log Table View (scrollable)       │  Histogram (time-based) │
│                                    │                         │
│  10:00:01 ERR [auth] token expired │  10:00 ████████  234    │
│  10:00:01 INF [api]  request ok    │  10:01 ██████     89    │
│  10:00:02 ERR [api]  conn refused  │  10:02 ██████████ 445   │
│  10:00:03 WRN [db]   slow query    │  10:03 ████        67   │
│                                    │  10:04 ████████   198   │
├────────────────────────────────────┴─────────────────────────┤
│  Query: level:error AND latency>500                          │
├──────────────────────────────────────────────────────────────┤
│  847 matches / 124,302 total  │  0.4ms  │  server.log 3.2MB │
└──────────────────────────────────────────────────────────────┘
```

Logs are parsed into structured records, indexed in memory with inverted indexes for O(1) exact-match lookups and O(log n) range queries. Even with millions of log lines, filtering feels instant.

## Building From Source

```bash
git clone https://github.com/riccardomerenda/logq.git
cd logq
make build
./logq testdata/sample.jsonl
```

### Requirements

- Go 1.22+

### Development

```bash
make test     # run tests
make lint     # run linter (requires golangci-lint)
make run      # build and run with sample data
```

## License

[MIT](LICENSE)
