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

## Match Highlighting

When a query is active, matching text is highlighted in yellow in both the log view and the detail overlay. This helps you instantly see *why* each record matched.

- **Full-text search** &#8212; the search term is highlighted across all visible fields (message, service, extra fields)
- **Field match** (`field:value`) &#8212; the value is highlighted only within the matching field
- **Regex match** (`field~"pattern"`) &#8212; the regex match is highlighted in the target field
- **AND / OR** &#8212; all positive terms are highlighted simultaneously
- **NOT** &#8212; negated terms are not highlighted (they represent exclusions)
- **Numeric / time comparisons** &#8212; no text highlighting (these filter by value, not by visible text)

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

## Field Auto-Complete

The filter bar supports inline auto-completion for field names and values:

- **Ghost text** &#8212; as you type, a dimmed suggestion appears after the cursor previewing the completion
- **Tab** &#8212; press Tab to accept the suggestion
- **Field names** &#8212; type a few characters of a field name (e.g., `lev`) and Tab to complete to `level:`; a colon is appended automatically
- **Field values** &#8212; after typing `field:` (e.g., `level:`), suggestions show known values for that field; type a prefix to narrow down
- **Keywords** &#8212; `AND`, `OR`, `NOT`, and `last` are also completable
- **Low-cardinality only** &#8212; value suggestions are shown only for fields with 50 or fewer unique values (e.g., `level`, `service`, `method`), not for high-cardinality fields like `request_id`

## Query Aliases

Aliases are shortcuts that expand to longer queries. Type `@name` anywhere in a query.

### Built-in Aliases

| Alias | Expands to |
|-------|-----------|
| `@err` | `level:error OR level:fatal` |
| `@warn` | `level:warn OR level:warning` |
| `@slow` | `latency>1000` |

### Using Aliases

```
@err                            # all errors and fatals
@err AND service:auth           # errors in auth service
@slow OR @err                   # slow requests or errors
(@err) AND last:5m              # recent errors
```

Aliases are expanded before the query is parsed, wrapped in parentheses for correct precedence. `@err AND service:auth` becomes `(level:error OR level:fatal) AND service:auth`.

### Custom Aliases

Define custom aliases in your `.logq.toml` config file:

```toml
[aliases]
noisy = "NOT service:healthcheck AND NOT service:ping"
auth  = "service:auth OR service:gateway"

# Rich alias with column override
[aliases.oncall]
query = "level:error AND last:15m"
columns = ["timestamp", "service", "message"]
```

Custom aliases override built-in aliases if they share the same name. Aliases can reference other aliases (e.g., `@oncall` can use `@err` internally).

### Alias Autocomplete

In the TUI, typing `@` triggers autocomplete for alias names. Press Tab to accept.

## Config File

logq looks for a `.logq.toml` file in the current directory and walks up to the filesystem root. Run `logq init` to create a starter config.

```toml
# .logq.toml
theme = "dark"
columns = ["timestamp", "level", "service", "message"]

[aliases]
err   = "level:error OR level:fatal"
noisy = "NOT service:healthcheck"
```

Settings in `.logq.toml`:
- **theme** &#8212; `"auto"`, `"dark"`, or `"light"`
- **columns** &#8212; default columns for TUI and batch mode
- **[aliases]** &#8212; custom query aliases (see [Query Aliases](#query-aliases))
- **[trace]** &#8212; customize trace ID field detection (see [Trace Following](#trace-following))

CLI flags always override config file settings.

## Trace Following

Press `t` in the detail view (Enter on any record) to follow its trace/request/correlation ID across all loaded files.

### How It Works

1. Open a record in the detail view (Enter)
2. Press `t` &#8212; logq auto-detects ID-like fields
3. If multiple ID fields exist, a pick menu lets you choose
4. The query is set to `field:value` and the log view shows the full request lifecycle
5. The originating record is marked with `>` in the gutter
6. Press `T` to clear the trace filter and restore your previous query

### ID Detection

logq detects trace IDs using two heuristics:

1. **Field name matching** &#8212; fields named `trace_id`, `request_id`, `correlation_id`, `span_id`, `x_request_id` (case-insensitive, supports camelCase, hyphens, dots)
2. **Value format matching** &#8212; fields whose values look like UUIDs (`xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx`) or long hex strings (16+ hex chars)

### Custom ID Fields

Configure which fields are treated as trace IDs in `.logq.toml`:

```toml
[trace]
id_fields = ["trace_id", "request_id", "correlation_id", "x_request_id"]
```

### Batch Mode

Trace following in batch mode uses standard field queries:

```bash
# Follow a specific trace across multiple files
logq api.log worker.log db.log -q "trace_id:550e8400-e29b-41d4-a716-446655440000"
```

## Query History

In the TUI, the filter bar (`/`) supports query history:

- **Up arrow** &#8212; recall the previous query
- **Down arrow** &#8212; go to the next query, or back to what you were typing
- History is kept for the current session (up to 100 entries)
- Consecutive duplicate queries are deduplicated
