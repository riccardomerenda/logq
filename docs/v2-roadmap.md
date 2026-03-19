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
| v1.0.0 | Pattern clustering & bookmarks | `p` to cluster similar messages by template, drill into clusters, `--patterns` batch mode; `m`/`'`/`B` bookmarks |
| v1.1.0 | JSON drill-down & saved views | Collapsible JSON tree in detail view with fold/expand, dot-path copy; `[views]` in `.logq.toml` with `1`-`9` key switching |

## Future Ideas

| Idea | Description |
|------|-------------|
| Cloud streaming | Direct integration with `kubectl logs`, CloudWatch, GCP Logging |
| Web playground | Browser-based demo where users can paste logs and try logq |
| Plugin system | Custom parsers for proprietary log formats (via Wasm or Go plugins) |
| Live alerts | `logq watch -q "@err" --alert "slack://..."` — trigger on pattern |

## Design Principles

These guide every decision:

1. **Zero config** — logq works out of the box, always. Config files add power, never requirements.
2. **Single binary** — no runtime dependencies, no setup
3. **Speed first** — O(1) field lookups, O(log n) range queries, binary search everywhere
4. **Terminal-native** — no Electron, no browser, no cloud account required
5. **Local-first** — your logs never leave your machine
