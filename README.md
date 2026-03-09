<p align="center">
  <h1 align="center">logq</h1>
  <p align="center">
    <strong>Your logs, queryable. Instantly.</strong>
  </p>
  <p align="center">
    A fast, interactive terminal log explorer that treats your log files like a database.
  </p>
  <p align="center">
    <a href="https://github.com/riccardomerenda/logq/actions/workflows/ci.yml"><img src="https://github.com/riccardomerenda/logq/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
    <a href="https://github.com/riccardomerenda/logq/releases/latest"><img src="https://img.shields.io/github/v/release/riccardomerenda/logq?label=release&color=50FA7B" alt="Release"></a>
    <a href="https://goreportcard.com/report/github.com/riccardomerenda/logq"><img src="https://goreportcard.com/badge/github.com/riccardomerenda/logq" alt="Go Report Card"></a>
    <a href="https://github.com/riccardomerenda/logq/blob/main/LICENSE"><img src="https://img.shields.io/github/license/riccardomerenda/logq?color=BD93F9" alt="License"></a>
    <a href="https://pkg.go.dev/github.com/riccardomerenda/logq"><img src="https://pkg.go.dev/badge/github.com/riccardomerenda/logq.svg" alt="Go Reference"></a>
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
+-----------------------------------------------------------+------------------+
| 10:00:01  ERROR  [auth]  token expired  user=u_882        |  Timeline        |
| 10:00:02  INFO   [api]   request ok  latency=45           |  10:00 ####  12  |
| 10:00:03  ERROR  [api]   connection refused  retries=3    |  10:01 ######  8 |
| 10:00:04  WARN   [db]    slow query  latency=1523         |  10:02 ####  15  |
| 10:00:05  INFO   [auth]  login ok  method=oauth           |  10:03 ##     4  |
+-----------------------------------------------------------+------------------+
| Filter: level:error AND latency>500                                          |
+------------------------------------------------------------------------------+
| 47 matches / 12,302 total  |  query: 0.2ms  |  server.log (3.2MB)           |
+------------------------------------------------------------------------------+
```

## Why logq?

Debugging with logs today means chaining `grep | jq | less` or scrolling through a cloud UI. There's no fast, local, interactive way to explore structured logs the way you explore data in a spreadsheet.

**logq** changes that. Point it at a file (or pipe logs in) and get:

- **Instant filtering** &#8212; type a query, results update as you type
- **Follow mode** &#8212; `logq -f` tails growing files with live updates (like `tail -f`, but queryable)
- **Time histogram** &#8212; see log volume and error spikes at a glance
- **Record detail** &#8212; press Enter to inspect any log line, `c` to copy to clipboard
- **Export & batch mode** &#8212; `logq -q "level:error" -o errors.jsonl` for scripting, or press `s` to save from the TUI
- **Query history** &#8212; Up/Down arrows in the filter bar to recall previous queries
- **Multi-line grouping** &#8212; stack traces and multi-line exceptions are grouped into single entries automatically
- **Zero setup** &#8212; auto-detects JSON, logfmt, and plain text
- **Single binary** &#8212; no dependencies, no config files, just run it

## Install

```bash
# Go
go install github.com/riccardomerenda/logq@latest

# Or download a binary from GitHub Releases
# https://github.com/riccardomerenda/logq/releases
```

### Updating

```bash
# If installed via go install
logq update

# Or manually
go install github.com/riccardomerenda/logq@latest
```

## Quick Start

```bash
# Explore a log file
logq server.log

# Follow a growing file (like tail -f, but interactive)
logq -f /var/log/app.log

# Pipe from anywhere
kubectl logs myapp | logq
docker logs mycontainer 2>&1 | logq

# Gzipped? No problem
logq server.log.gz
```

## Batch Mode & Export

Run queries without the TUI for scripting and pipelines:

```bash
# Filter and print to stdout
logq server.log -q "level:error"

# Save to file
logq server.log -q "level:error AND service:auth" -o errors.jsonl

# Output as JSON (re-serialized fields) or CSV
logq server.log -q "latency>1000" --format json
logq server.log -q "latency>1000" --format csv

# Count matches only
logq server.log -q "level:error" --count
```

In the TUI, press `s` to save the current filtered results to a file.

## Query Syntax

Type queries in the filter bar (`/`). Results update live.

| Pattern | Meaning | Example |
|---|---|---|
| `word` | Full-text search across all fields | `timeout` |
| `field:value` | Exact match on a field | `level:error` |
| `field>n` | Numeric comparison (`>`, `>=`, `<`, `<=`) | `latency>500` |
| `field~"regex"` | Regex match | `message~"timeout.*retry"` |
| `timestamp>"time"` | Time range (absolute) | `timestamp>"2026-03-08T10:00:00Z"` |
| `last:duration` | Time range (relative to now) | `last:5m`, `last:1h`, `last:2d` |
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
| `Up` / `Down` (in filter bar) | Browse query history |
| `PgUp` / `PgDn` | Page scroll |
| `Home` / `End` | Jump to start / end |
| `Enter` | Show full record detail |
| `c` | Copy raw record to clipboard (in detail view) |
| `s` | Save filtered results to file |
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

Timestamps are auto-parsed from RFC3339, ISO 8601, Unix epoch, syslog, nginx/Apache formats, time-only (`HH:MM:SS`), and more. Log levels are normalized from dozens of variants (`WARNING`, `WARN`, `WRN`, `W` all become `warn`).

Mixed formats in the same file are handled gracefully.

### Multi-Line Log Entries

logq automatically groups multi-line log entries like stack traces, exception dumps, and multi-line error messages into single records. The grouping strategy is auto-detected:

- **Timestamp-anchored** &#8212; entries start with a timestamp; continuation lines (indented stack traces, JSON payloads, etc.) are grouped with the preceding entry
- **Structured** &#8212; entries start with `{` (JSON) or `key=value` (logfmt); everything else is a continuation
- **Single-line** &#8212; for files where every line is its own entry (standard JSON Lines, logfmt), no grouping overhead is added

This works out of the box for .NET exceptions, Java stack traces, Python tracebacks, and any log format where entries start with a timestamp.

**Example:** a 1300-line .NET exception log with embedded Elasticsearch JSON errors is automatically grouped into 15 logical entries, each with its full stack trace accessible via the detail view (Enter).

### Plain Text Timestamp & Level Detection

For unstructured plain text logs, logq extracts:

- **Timestamps** from the start of lines: `12:43:10 ...`, `2026-03-08 10:00:01 ...`, `Mar  8 10:00:01 ...`, etc.
- **Log levels** from keywords near the start: `ERROR`, `WARN`, `INFO`, `DEBUG`, `FATAL`, `CRITICAL`, `PANIC`

## Architecture

```
  File / stdin / .gz
        |
        v
  +---------+    +----------+    +---------+    +---------+    +-----------+
  |  Input  |--->|Multiline |--->| Parser  |--->|  Index  |--->|  Query    |
  |  Reader |    | Grouper  |    | JSON    |    | Inverted|    |  Engine   |
  |         |    |          |    | logfmt  |    | Numeric |    |  Lexer    |
  |  gzip   |    | auto-    |    | plain   |    | Time    |    |  Parser   |
  +---------+    | detect   |    +---------+    +---------+    | Evaluator |
                 +----------+                                  +-----+-----+
                                                                     |
                                                               +-----+-----+
                                                               |           |
                                                               v           v
                                                          +---------+ +---------+
                                                          |   TUI   | |  Batch  |
                                                          |Log View | | Export  |
                                                          |Histogram| |raw/json |
                                                          |QueryBar | |  csv    |
                                                          | Detail  | | stdout  |
                                                          +---------+ +---------+
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
make test     # run all tests
make lint     # run linter (requires golangci-lint)
make run      # build and run with sample data
```

### Benchmarks

```bash
go test ./benchmarks/ -bench=. -benchmem
```

### Project Structure

```
logq/
├── main.go                     # CLI entry point
├── internal/
│   ├── input/
│   │   ├── reader.go           # File, stdin, gzip reading
│   │   ├── multiline.go        # Multi-line entry grouping
│   │   └── follow.go           # File tailing for follow mode (-f)
│   ├── parser/                 # JSON, logfmt, plain text, timestamps
│   ├── index/                  # In-memory inverted + numeric + time indexes
│   ├── query/                  # Lexer, recursive descent parser, evaluator
│   ├── output/                 # Export writers (raw, JSON, CSV)
│   └── ui/                     # Bubbletea TUI components
├── benchmarks/                 # Performance benchmarks
└── testdata/                   # Sample log files for testing
```

## License

[MIT](LICENSE)
