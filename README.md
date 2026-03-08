<p align="center">
  <h1 align="center">logq</h1>
  <p align="center">
    <strong>Your logs, queryable. Instantly.</strong>
  </p>
  <p align="center">
    A fast, interactive terminal log explorer that treats your log files like a database.
  </p>
  <p align="center">
    <a href="#install">Install</a> &middot;
    <a href="#quick-start">Quick Start</a> &middot;
    <a href="#query-syntax">Query Syntax</a> &middot;
    <a href="docs/query-syntax.md">Full Reference</a>
  </p>
</p>

---

```
┌──────────────────────────────────────────────────────────────────────────────┐
│ 10:00:01  ERROR  [auth]  token expired  user=u_882        │  Timeline      │
│ 10:00:02  INFO   [api]   request ok  latency=45           │  10:00 ████ 12 │
│ 10:00:03  ERROR  [api]   connection refused  retries=3    │  10:01 ██████ 8│
│ 10:00:04  WARN   [db]    slow query  latency=1523         │  10:02 ████ 15 │
│ 10:00:05  INFO   [auth]  login ok  method=oauth           │  10:03 ██    4 │
├───────────────────────────────────────────────────────────┴────────────────┤
│ Filter: level:error AND latency>500                                        │
├────────────────────────────────────────────────────────────────────────────┤
│ 47 matches / 12,302 total  │  query: 0.2ms  │  server.log (3.2MB)         │
└────────────────────────────────────────────────────────────────────────────┘
```

## Why logq?

Debugging with logs today means chaining `grep | jq | less` or scrolling through a cloud UI. There's no fast, local, interactive way to explore structured logs the way you explore data in a spreadsheet.

**logq** changes that. Point it at a file (or pipe logs in) and get:

- **Instant filtering** &#8212; type a query, results update as you type
- **Time histogram** &#8212; see log volume and error spikes at a glance
- **Record detail** &#8212; press Enter to inspect any log line fully
- **Zero setup** &#8212; auto-detects JSON, logfmt, and plain text
- **Single binary** &#8212; no dependencies, no config files, just run it

## Install

```bash
# Go
go install github.com/riccardomerenda/logq@latest

# Or download a binary from GitHub Releases
# https://github.com/riccardomerenda/logq/releases
```

## Quick Start

```bash
# Explore a log file
logq server.log

# Pipe from anywhere
kubectl logs myapp | logq
docker logs mycontainer 2>&1 | logq

# Gzipped? No problem
logq server.log.gz
```

## Query Syntax

Type queries in the filter bar (`/`). Results update live.

| Pattern | Meaning | Example |
|---|---|---|
| `word` | Full-text search across all fields | `timeout` |
| `field:value` | Exact match on a field | `level:error` |
| `field>n` | Numeric comparison (`>`, `>=`, `<`, `<=`) | `latency>500` |
| `field~"regex"` | Regex match | `message~"timeout.*retry"` |
| `A AND B` | Both conditions must match | `level:error AND service:auth` |
| `A OR B` | Either condition matches | `level:error OR level:fatal` |
| `NOT A` | Negate a condition | `NOT service:healthcheck` |
| `(A OR B) AND C` | Group with parentheses | `(level:error OR level:fatal) AND service:api` |

Compound queries work naturally:

```
level:error AND latency>1000 AND NOT service:healthcheck
```

See the [full query reference](docs/query-syntax.md) for details.

## Keyboard Shortcuts

| Key | Action |
|---|---|
| `/` | Focus the filter bar |
| `j` / `k` or `Up` / `Down` | Scroll through logs |
| `PgUp` / `PgDn` | Page scroll |
| `Home` / `End` | Jump to start / end |
| `Enter` | Show full record detail |
| `Escape` | Clear filter / close detail overlay |
| `Tab` | Toggle focus between log view and histogram |
| `q` | Quit |

## Supported Log Formats

logq auto-detects the format of **each line** independently:

| Format | Example |
|---|---|
| **JSON Lines** | `{"level":"error","message":"timeout","latency":523}` |
| **logfmt** | `level=error msg="timeout" latency=523` |
| **Plain text** | `ERROR: connection timeout after 523ms` |

Timestamps are auto-parsed from RFC3339, ISO 8601, Unix epoch, syslog, nginx/Apache formats, and more. Log levels are normalized from dozens of variants (`WARNING`, `WARN`, `WRN`, `W` all become `warn`).

Mixed formats in the same file are handled gracefully.

## Architecture

```
  File / stdin / .gz
        │
        ▼
   ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌───────────┐
   │  Input   │───▶│ Parser  │───▶│  Index  │───▶│  Query    │
   │  Reader  │    │ JSON    │    │ Inverted│    │  Engine   │
   │          │    │ logfmt  │    │ Numeric │    │  Lexer    │
   │  gzip    │    │ plain   │    │ Time    │    │  Parser   │
   └─────────┘    └─────────┘    └─────────┘    │  Evaluator│
                                                 └─────┬─────┘
                                                       │
                                                       ▼
                                                 ┌───────────┐
                                                 │  TUI      │
                                                 │  Log View │
                                                 │  Histogram│
                                                 │  Query Bar│
                                                 │  Detail   │
                                                 └───────────┘
```

**Performance by design:**
- Field lookups are **O(1)** via inverted indexes
- Numeric range queries use **binary search** &#8212; O(log n)
- Time navigation uses **sorted indexes** &#8212; O(log n)
- Full-text search scans sequentially with early exit &#8212; fast enough for millions of lines

## Building From Source

```bash
git clone https://github.com/riccardomerenda/logq.git
cd logq
make build
./logq testdata/sample.jsonl     # or logq.exe on Windows
```

### Requirements

- Go 1.17+

### Development

```bash
make test     # run all 53 tests
make lint     # run linter (requires golangci-lint)
make run      # build and run with sample data
```

### Project Structure

```
logq/
├── main.go                     # CLI entry point
├── internal/
│   ├── input/reader.go         # File, stdin, gzip reading
│   ├── parser/                 # JSON, logfmt, plain text, timestamps
│   ├── index/                  # In-memory inverted + numeric + time indexes
│   ├── query/                  # Lexer, recursive descent parser, evaluator
│   └── ui/                     # Bubbletea TUI components
└── testdata/                   # Sample log files for testing
```

## License

[MIT](LICENSE)
