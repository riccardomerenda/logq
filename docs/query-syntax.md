# logq Query Syntax

## Basics

Type a query in the filter bar to search your logs. Results update as you type.

## Full-Text Search

Just type a word to search all fields:

```
error                           # any field containing "error"
timeout                         # any field containing "timeout"
```

## Field Match

Use `field:value` for exact matches on a specific field:

```
level:error                     # level equals "error"
service:auth                    # service equals "auth"
method:GET                      # method equals "GET"
```

## Numeric Comparisons

Use `>`, `>=`, `<`, `<=` for numeric fields:

```
latency>500                     # latency greater than 500
latency>=1000                   # latency greater than or equal to 1000
rows<100                        # rows less than 100
```

## Regex Match

Use `field~"pattern"` for regex matching:

```
message~"timeout.*retry"        # message matches the regex
path~"/users/.*"                # path matches the regex
```

## Boolean Operators

Combine conditions with `AND`, `OR`, `NOT`:

```
level:error AND service:auth
level:error OR level:fatal
NOT service:healthcheck
level:error AND latency>1000 AND NOT service:healthcheck
```

## Parentheses

Group expressions with parentheses:

```
(level:error OR level:fatal) AND service:api
```

## Quoted Values

Use quotes for values containing spaces:

```
message:"connection refused"
query:"SELECT * FROM orders"
```
