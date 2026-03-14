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

## Planned

### Medium Priority

#### 🔖 Bookmarks
Mark interesting records with `m`, navigate between them with `'`, filter to bookmarks-only with `B`. Useful for long debugging sessions.

#### 🔍 JSON Drill-Down
Collapsible nested objects in the detail view. Expand/collapse with Enter, copy dot-paths (`request.headers.host`).

#### ⚡ Query Aliases
Built-in and user-defined shortcuts:
```
@err    → level:error OR level:fatal
@slow   → latency>1000
```

### Future Ideas

| Idea | Description |
|------|-------------|
| Cloud streaming | Direct integration with `kubectl logs`, CloudWatch, etc. |
| Saved views | Per-project presets: query + columns + theme in `.logq.toml` |
| Web playground | Browser-based demo where users can paste logs and try logq |
| Plugin system | Custom parsers for proprietary log formats |

## Design Principles

These guide every decision:

1. **Zero config** — logq works out of the box, always
2. **Single binary** — no runtime dependencies, no setup
3. **Speed first** — O(1) field lookups, O(log n) range queries, binary search everywhere
4. **Terminal-native** — no Electron, no browser, no cloud account required
