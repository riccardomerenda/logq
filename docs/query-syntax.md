# logq Query Syntax Reference

logq uses a simple, powerful query language for filtering logs. Type queries in the filter bar (`/`) and results update live as you type.

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

## Operator Precedence

From highest to lowest:

1. `NOT`
2. `AND`
3. `OR`

Use parentheses to override.

## Empty Query

An empty filter bar matches all records &#8212; press `Escape` to clear.

## Examples

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
