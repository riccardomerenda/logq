# logq Query Syntax Reference

logq uses a simple, powerful query language for filtering logs. Type queries in the filter bar (`/`) and results update live as you type. Queries also work in batch mode with the `-q` flag (see [Batch Mode](#batch-mode) below).

## Full-Text Search

Just type a word to search across all fields:

```
error                           # any field containing "error"
timeout                         # any field containing "timeout"
connection                      # any field containing "connection"
```

Full-text search is case-insensitive.

## Field Match

Use `field:value` for exact matches on a specific field:

```
level:error                     # level equals "error"
service:auth                    # service equals "auth"
method:GET                      # HTTP method is GET
request_id:abc123               # find a specific request
```

Use quotes for values containing spaces:

```
message:"connection refused"
query:"SELECT * FROM orders"
```

## Numeric Comparisons

Use `>`, `>=`, `<`, `<=` for numeric fields:

```
latency>500                     # latency greater than 500ms
latency>=1000                   # latency at least 1 second
rows<100                        # fewer than 100 rows
retries<=3                      # at most 3 retries
```

Numeric fields are auto-detected during indexing &#8212; no type declarations needed.

## Regex Match

Use `field~"pattern"` for regular expression matching:

```
message~"timeout.*retry"        # message matches regex
path~"/users/[0-9]+"            # path matches user ID pattern
service~"auth|gateway"          # service is auth or gateway
```

Regex uses Go's `regexp` syntax (RE2).

## Boolean Operators

Combine conditions with `AND`, `OR`, `NOT`:

```
level:error AND service:auth
level:error OR level:fatal
NOT service:healthcheck
```

Build complex queries by chaining operators:

```
level:error AND latency>1000 AND NOT service:healthcheck
```

## Grouping with Parentheses

Control evaluation order with parentheses:

```
(level:error OR level:fatal) AND service:api
(service:auth OR service:gateway) AND latency>500
```

Without parentheses, `AND` binds tighter than `OR`:

```
# These are equivalent:
level:error OR level:fatal AND service:api
level:error OR (level:fatal AND service:api)
```

## Time Range Queries

Filter by timestamp using comparison operators on the `timestamp` field:

```
timestamp>"2026-03-08T10:00:00Z"                    # after a specific time
timestamp<"2026-03-08T11:00:00Z"                    # before a specific time
timestamp>="2026-03-08T10:00:00" AND timestamp<"2026-03-08T10:05:00"   # time window
```

All common timestamp formats are supported: RFC3339, ISO 8601, `YYYY-MM-DD HH:MM:SS`, and more.

Field aliases `ts`, `time`, `@timestamp`, `datetime`, and `t` also work:

```
ts>"2026-03-08T10:00:00Z"
@timestamp<="2026-03-08T12:00:00Z"
```

### Relative Time

Use `last:` for quick time-based filtering relative to now:

```
last:5m                         # last 5 minutes
last:1h                         # last 1 hour
last:30s                        # last 30 seconds
last:2d                         # last 2 days
```

Combine with other conditions:

```
level:error AND last:1h
service:api AND last:30m AND latency>500
```

Supported units: `s` (seconds), `m` (minutes), `h` (hours), `d` (days).

## Operator Precedence

From highest to lowest:

1. `NOT`
2. `AND`
3. `OR`

Use parentheses to override.

## Empty Query

An empty filter bar matches all records &#8212; press `Escape` to clear.

## Examples

### Structured logs (JSON / logfmt)

```
# Find all errors in the auth service
level:error AND service:auth

# Find slow database queries
service:db AND latency>1000

# Find errors excluding health checks
level:error AND NOT message:"health check"

# Find anything related to a specific user
u_882

# Find timeout-related errors with high latency
message~"timeout" AND latency>500

# Find critical issues across payment services
(level:error OR level:fatal) AND service~"payment.*"
```

### Multi-file queries

When opening multiple files, each record gets a `source` field with the originating filename:

```
# Errors from a specific file
source:app.log AND level:error

# Records from auth-related files (regex)
source~"auth.*"

# Combine source with other conditions
source:db.log AND latency>500
```

### Plain text / multi-line logs

For unstructured logs (stack traces, application output, etc.), full-text search is the most useful:

```
# Find entries containing an exception type
NullPointerException

# Search for a specific error message in .NET stack traces
too_many_clauses

# Find entries mentioning a PID
73902

# Combine full-text with level detection
level:error AND connection refused

# Find entries mentioning specific classes
DocumentManager
```

## Batch Mode

All query syntax works in batch (non-interactive) mode using the `-q` flag:

```bash
# Print matching lines to stdout
logq server.log -q "level:error AND service:auth"

# Save to a file
logq server.log -q "level:error" -o errors.jsonl

# Output as JSON (re-serialized from parsed fields)
logq server.log -q "latency>1000" --format json

# Output as CSV (with header row)
logq server.log -q "latency>1000" --format csv

# Count matches only
logq server.log -q "level:error" --count
```

Batch mode skips the TUI and writes directly to stdout (or a file with `-o`). This is useful for scripting, pipelines, and automated log processing.

### Output formats

| Format | Description |
|---|---|
| `raw` (default) | Original log lines as-is |
| `json` | One JSON object per line (re-serialized from parsed fields) |
| `csv` | Header row + values, with all fields across matched records |

## Query History

In the TUI, the filter bar (`/`) supports query history:

- **Up arrow** &#8212; recall the previous query
- **Down arrow** &#8212; go to the next query, or back to what you were typing
- History is kept for the current session (up to 100 entries)
- Consecutive duplicate queries are deduplicated
