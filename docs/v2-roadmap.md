# logq Roadmap

## Shipped

| Version | Feature | Description |
|---------|---------|-------------|
| v0.1.0 | Core engine | Multi-format parsing (JSON, logfmt, plain text), in-memory indexing, query language |
| v0.2.0 | Interactive TUI | Log view, histogram, query bar, status bar, record detail |
| v0.3.0 | Input flexibility | File, stdin, gzip support, follow mode (`-f`), multi-line grouping |
| v0.4.0 | Time queries & export | `timestamp>"..."`, `last:5m`, batch mode (`-q`, `-o`, `--format`), query history |
| v0.5.0 | Multi-file support | Merged timeline from multiple files, `source:filename` queries |
| v0.6.0 | Search UX | Match highlighting, field auto-complete with ghost text, demo GIF |
| v0.6.1 | Code quality | Go 1.22, fix Bubbletea anti-patterns, remove dead code |
| v0.7.0 | Features | Persistent history, color themes, aggregations, column mode, Homebrew & Scoop |
| v0.8.0 | Config & aliases | `.logq.toml` config file with auto-discovery, query aliases (`@err`, `@warn`, `@slow`), custom aliases, `logq init` |
| v0.9.0 | Trace following | `t` in detail view to follow trace/request/correlation IDs across files, `T` to clear, ID pick menu, configurable `[trace]` in `.logq.toml` |

## Planned

### v1.0.0 — Pattern Clustering & Polish

**Theme:** From tool to platform — intelligent log analysis.

#### 🧠 Log Pattern Clustering
Automatically group similar log lines by extracting message templates and collapsing variable parts.

**Example:**
```
# These three lines:
Connection timeout to 10.0.1.5:5432 after 3000ms
Connection timeout to 10.0.2.8:5432 after 5200ms
Connection timeout to 10.0.1.12:5432 after 4100ms

# Become one cluster:
Connection timeout to <ip>:<port> after <ms>ms  (3 occurrences)
```

**Scope:**
- Template extraction: replace IPs, UUIDs, numbers, paths, timestamps with `<placeholder>`
- Cluster view mode (toggle with `p` for patterns): shows unique templates ranked by count
- Drill into a cluster to see all individual entries
- Batch mode: `logq server.log --patterns` to list top templates
- Combine with queries: `logq server.log -q "level:error" --patterns` to cluster only errors

#### 🔖 Bookmarks
Mark interesting records during exploration and navigate between them.

- `m` — toggle bookmark on current record
- `'` (quote) — jump to next bookmark
- `B` — filter to bookmarked records only
- Bookmarks persist for the session (not across restarts)

#### 🎯 v1.0 Polish
- Performance validation with multi-GB files
- Shell completions (bash, zsh, fish)
- Man page generation
- `logq --explain "query"` to show how a query will be evaluated
- Error messages with suggestions ("did you mean `level:error`?")

---

### Future Ideas

| Idea | Description |
|------|-------------|
| Cloud streaming | Direct integration with `kubectl logs`, CloudWatch, GCP Logging |
| Saved views | Named views combining query + columns + theme, switchable with `1`-`9` |
| Web playground | Browser-based demo where users can paste logs and try logq |
| Plugin system | Custom parsers for proprietary log formats (via Wasm or Go plugins) |
| JSON drill-down | Collapsible nested objects in detail view, copy dot-paths |
| Live alerts | `logq watch -q "@err" --alert "slack://..."` — trigger on pattern |

## Design Principles

These guide every decision:

1. **Zero config** — logq works out of the box, always. Config files add power, never requirements.
2. **Single binary** — no runtime dependencies, no setup
3. **Speed first** — O(1) field lookups, O(log n) range queries, binary search everywhere
4. **Terminal-native** — no Electron, no browser, no cloud account required
5. **Local-first** — your logs never leave your machine
