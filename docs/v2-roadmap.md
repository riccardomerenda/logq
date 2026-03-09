# logq v2 Roadmap — Implementation Plan

## Current State (v1 — Complete)

All original 6 phases are shipped. logq is a working, interactive terminal log explorer with:

| Feature | Status |
|---|---|
| Multi-format parsing (JSON, logfmt, plain text) | Done |
| In-memory indexing (inverted, numeric, time) | Done |
| Query language (field match, numeric, regex, boolean ops, parentheses) | Done |
| Interactive TUI (log view, histogram, query bar, status bar) | Done |
| Input (file, stdin, gzip) | Done |
| Follow mode (`-f` for files) | Done |
| Multi-line grouping (stack traces, exceptions) | Done |
| Record detail view + copy to clipboard | Done |
| Histogram with bucket selection (Tab + Enter to jump) | Done |
| Self-update (`logq update`) | Done |
| CI/CD (GitHub Actions) | Done |
| GoReleaser (cross-platform binaries) | Done |
| README with badges, docs, query reference | Done |

---

## v2 Features

### Phase 7: Time Range Queries
**Priority:** High — Infrastructure exists, just needs query syntax exposure
**Effort:** Medium
**Status:** [ ] Not started

The index already has `TimeRange(start, end)` with binary search. This phase exposes it to users via the query language.

#### 7.1 — Time range operators in query syntax
**Status:** [ ] Not started

Add support for these query patterns:
```
timestamp>"2026-03-08T10:00:00"        # after a specific time
timestamp<"2026-03-08T11:00:00"        # before a specific time
timestamp>"10:00" AND timestamp<"11:00" # time-only (relative to file's date)
```

**Files to modify:**
- `internal/query/lexer.go` — recognize timestamp-like values after `>`, `<`, `>=`, `<=` operators on the `timestamp` field
- `internal/query/evaluator.go` — route timestamp comparisons to `index.TimeRange()` instead of numeric index
- `internal/index/index.go` — add `TimeBefore(t)`, `TimeAfter(t)` methods returning `[]int`

#### 7.2 — Relative time shorthand
**Status:** [ ] Not started

Add convenience syntax for common time filters:
```
last:5m          # last 5 minutes
last:1h          # last 1 hour
last:30s         # last 30 seconds
last:2d          # last 2 days
```

**Files to modify:**
- `internal/query/lexer.go` — tokenize `last:` as a special field with duration value
- `internal/query/parser.go` — parse `last:Xunit` into a time range AST node
- `internal/query/evaluator.go` — resolve relative time against `time.Now()` and use time index

#### 7.3 — Tests
**Status:** [ ] Not started

- Parse `timestamp>"2026-03-08T10:00:00"` → correct AST with time comparison
- Parse `last:5m` → correct relative time node
- Evaluate time range queries against index → correct record subset
- Edge cases: `last:0s`, `last:999d`, invalid formats → meaningful errors
- Combine with other queries: `level:error AND last:1h`

#### 7.4 — Update query-syntax.md documentation
**Status:** [ ] Not started

Add "Time Range Queries" section to `docs/query-syntax.md` with examples.

---

### Phase 8: Export Filtered Results
**Priority:** High — Users need to share/save findings
**Effort:** Low
**Status:** [x] Complete

#### 8.1 — CLI export mode (non-interactive)
**Status:** [x] Complete

Add flags for headless/batch export:
```bash
logq server.log -q "level:error" -o errors.jsonl    # write matches to file
logq server.log -q "level:error" --format json       # output format (json, raw, csv)
logq server.log -q "latency>1000" --count            # just print match count
```

**Files to modify:**
- `main.go` — add `-q`, `-o`, `--format`, `--count` flags
- `main.go` — when `-q` is set, skip TUI: parse → index → query → write output → exit
- New file: `internal/output/writer.go` — format and write matched records

Supported output formats:
- `json` — one JSON object per line (re-serialize from parsed fields)
- `raw` — original log lines as-is
- `csv` — header row + values (field order from first record)

#### 8.2 — TUI export keybinding
**Status:** [x] Complete

Add `s` key to save current filtered results to a file from within the TUI.

**Files to modify:**
- `internal/ui/keys.go` — add Save keybinding
- `internal/ui/app.go` — handle save: prompt for filename (or auto-generate `logq-export-TIMESTAMP.jsonl`), write filtered records
- `internal/ui/statusbar.go` — show "Saved N records to file" confirmation

#### 8.3 — Tests
**Status:** [x] Complete

- CLI mode: `-q "level:error" -o out.jsonl` produces correct file
- JSON/raw/CSV output formats produce valid output
- `--count` prints number only
- Empty result set produces empty file (not error)
- TUI save writes correct records

---

### Phase 9: Multiple File Support
**Priority:** High — Very common real-world need
**Effort:** Medium
**Status:** [ ] Not started

#### 9.1 — Accept multiple file arguments
**Status:** [ ] Not started

```bash
logq app.log db.log auth.log        # multiple explicit files
logq /var/log/app/*.log             # glob expansion (shell does this)
logq app.log.1.gz app.log.2.gz     # mixed gzip and plain
```

**Files to modify:**
- `main.go` — accept `args[0:]` instead of `args[0]`, iterate and read all
- `internal/input/reader.go` — add `NewMultiFileReader(paths []string)` that concatenates readers
- `internal/parser/parser.go` — add `Source` field to `Record` (filename origin)

#### 9.2 — Merge and sort by timestamp
**Status:** [ ] Not started

Records from multiple files should be interleaved by timestamp for a unified timeline.

**Files to modify:**
- `internal/index/index.go` — after building, sort records by timestamp if multi-file
- `internal/ui/logview.go` — show source file indicator: `[app.log]` prefix or color-coded

#### 9.3 — Source file as queryable field
**Status:** [ ] Not started

```
source:app.log AND level:error       # errors from specific file
source~"auth.*" AND latency>500      # from auth-related files
```

**Files to modify:**
- `internal/parser/parser.go` — populate `source` field in Record.Fields
- Index automatically picks it up (no index changes needed)

#### 9.4 — Tests
**Status:** [ ] Not started

- Two files with overlapping timestamps → merged in correct order
- `source:filename` query works
- Mixed gzip + plain files work together
- File with no timestamps → appended after timestamped records
- Single file still works as before (no regression)

---

### Phase 10: Query History
**Priority:** Medium — Low effort, high quality-of-life
**Effort:** Low
**Status:** [x] Complete (in-session), [ ] Persistent history not yet

#### 10.1 — In-session query history
**Status:** [x] Complete

Up/Down arrows in the query bar recall previous queries (when query bar is focused).
Draft text is preserved when entering history mode and restored when navigating past the end.

**Files modified:**
- `internal/ui/querybar.go` — added `history []string`, `historyIdx int`, `draft string`; `PushHistory()`, `HistoryUp()`, `HistoryDown()`
- `internal/ui/app.go` — pushes to history on Enter, handles Up/Down keys in query bar focus

#### 10.2 — Persistent history (optional)
**Status:** [ ] Not started

Save query history to `~/.local/share/logq/history` (XDG-compliant). Load on startup. Cap at 100 entries.

**Files to create:**
- `internal/config/history.go` — load/save history file, dedup, cap size

#### 10.3 — Tests
**Status:** [x] Complete

- Type query → execute → Up arrow → previous query shown
- Multiple queries → navigate full history
- Down arrow past end → restores draft
- Dedup consecutive entries
- Empty string ignored
- Cap at 100 entries
- Blur resets history index

---

### Phase 11: Field Auto-Complete
**Priority:** Medium — Helps discoverability on unfamiliar logs
**Effort:** Medium
**Status:** [ ] Not started

#### 11.1 — Tab completion for field names
**Status:** [ ] Not started

When the cursor is at a word boundary in the query bar, pressing Tab shows/cycles through matching field names from the index.

**Files to modify:**
- `internal/ui/querybar.go` — extract current word prefix, match against index field names, cycle through completions on Tab
- `internal/ui/app.go` — pass field names from index to querybar on init and when index updates (follow mode)

#### 11.2 — Completion popup (stretch)
**Status:** [ ] Not started

Show a small dropdown of matching field names below the query bar.

**Files to modify:**
- `internal/ui/querybar.go` — render suggestion list overlay
- `internal/ui/app.go` — layout the overlay above the status bar

#### 11.3 — Value suggestions for known fields
**Status:** [ ] Not started

For fields with low cardinality (e.g., `level`, `service`), suggest values after typing `field:`.

**Files to modify:**
- `internal/index/index.go` — add `FieldValues(field string) []string` method (return unique values, cap at 20)
- `internal/ui/querybar.go` — detect `field:` pattern, show value completions

#### 11.4 — Tests
**Status:** [ ] Not started

- Tab on `lev` → completes to `level`
- Tab on `level:` → shows `error`, `info`, `warn`, etc.
- Tab with no prefix → shows all field names
- Fields from index update when new records arrive (follow mode)

---

### Phase 12: Regex & Match Highlighting
**Priority:** Medium — Visual feedback for what matched
**Effort:** Medium
**Status:** [ ] Not started

#### 12.1 — Highlight matching text in log view
**Status:** [ ] Not started

When a query is active, highlight the matching portions of each log line (bold, underline, or background color).

**Files to modify:**
- `internal/ui/logview.go` — in `formatLogLine()`, find matching substrings and wrap with highlight style
- `internal/ui/theme.go` — add `StyleMatchHighlight`

Matching logic:
- Full-text search: highlight the search term in all fields
- `field:value`: highlight the value in that field
- `field~"regex"`: highlight the regex match
- For AND/OR/NOT compounds: highlight all positive match terms

#### 12.2 — Highlight in detail view
**Status:** [ ] Not started

Apply the same highlighting in the record detail overlay.

**Files to modify:**
- `internal/ui/detail.go` — highlight matching field values

#### 12.3 — Tests
**Status:** [ ] Not started

- Search `error` → "error" substring highlighted in rendered output
- Regex match → matched portion highlighted
- Field match → only matching field highlighted, not others
- No query → no highlighting (clean render)

---

### Phase 13: Color Themes
**Priority:** Medium — Light terminal users struggle with dark-only themes
**Effort:** Low
**Status:** [ ] Not started

#### 13.1 — Built-in theme selection
**Status:** [ ] Not started

```bash
logq server.log --theme light
logq server.log --theme dracula      # current default
logq server.log --theme monokai
```

**Files to modify:**
- `internal/ui/theme.go` — refactor hardcoded colors into `Theme` struct; define 3-4 built-in themes (dracula, light, monokai, nord)
- `main.go` — add `--theme` flag, pass to UI init
- `internal/ui/app.go` — accept theme in model init

#### 13.2 — Auto-detect terminal background
**Status:** [ ] Not started

Use `lipgloss.HasDarkBackground()` to default to light/dark theme automatically.

**Files to modify:**
- `internal/ui/theme.go` — check terminal background on init, pick default theme

#### 13.3 — Tests
**Status:** [ ] Not started

- Each theme produces valid lipgloss styles (no panics)
- `--theme invalid` shows error with available themes
- Default theme auto-selection doesn't crash on any terminal

---

### Phase 14: Bookmarks
**Priority:** Low — Nice to have for long debugging sessions
**Effort:** Low
**Status:** [ ] Not started

#### 14.1 — Toggle bookmark on current record
**Status:** [ ] Not started

Press `m` to bookmark/unbookmark the selected log line. Bookmarked lines get a visual marker (e.g., `*` or colored dot).

**Files to modify:**
- `internal/ui/app.go` — add `bookmarks map[int]bool` to model; handle `m` key
- `internal/ui/logview.go` — render bookmark indicator
- `internal/ui/keys.go` — add bookmark keybinding

#### 14.2 — Navigate between bookmarks
**Status:** [ ] Not started

Press `'` (single quote) to jump to the next bookmark. `Shift+'` for previous.

**Files to modify:**
- `internal/ui/app.go` — find next/prev bookmarked record index, scroll to it

#### 14.3 — Filter to bookmarked records only
**Status:** [ ] Not started

Press `B` to toggle showing only bookmarked records.

**Files to modify:**
- `internal/ui/app.go` — apply bookmark filter on top of query filter

#### 14.4 — Tests
**Status:** [ ] Not started

- Bookmark a record → marker visible
- Navigate to next bookmark → correct record
- Filter bookmarks only → only bookmarked records shown
- Bookmark + query filter → intersection works correctly

---

### Phase 15: JSON Drill-Down in Detail View
**Priority:** Low — Useful for deeply nested JSON logs
**Effort:** Medium
**Status:** [ ] Not started

#### 15.1 — Collapsible nested fields
**Status:** [ ] Not started

In detail view, nested JSON objects can be expanded/collapsed with Enter or arrow keys.

```
  timestamp   2026-03-08T10:00:04Z
  level       error
▸ request     {3 fields}              ← collapsed
  message     connection refused
```

Pressing Enter on `request`:
```
▾ request
    method    GET
    path      /users
    headers   {2 fields}              ← still collapsed
  message     connection refused
```

**Files to modify:**
- `internal/ui/detail.go` — rewrite from flat list to tree model with expand/collapse state
- `internal/parser/json.go` — preserve nested structure (currently flattened); add `RawJSON` or structured field to Record

#### 15.2 — Copy nested path
**Status:** [ ] Not started

Press `c` on a nested field to copy its dot-path and value (e.g., `request.headers.host: "api.example.com"`).

**Files to modify:**
- `internal/ui/detail.go` — track selected field path, copy formatted

#### 15.3 — Tests
**Status:** [ ] Not started

- Nested JSON → collapsible tree in detail view
- Expand/collapse toggles correctly
- Deeply nested (5+ levels) doesn't break layout
- Non-JSON records → flat display (no regression)

---

### Phase 16: Config File
**Priority:** Low — "Zero config" is a feature, but power users want defaults
**Effort:** Low
**Status:** [ ] Not started

#### 16.1 — Optional config file
**Status:** [ ] Not started

Support `~/.config/logq/config.toml` (XDG-compliant). Entirely optional — logq works without it.

```toml
# ~/.config/logq/config.toml
theme = "light"
default_query = "NOT level:debug"

[aliases]
err = "level:error OR level:fatal"
slow = "latency>1000"
```

**Files to create:**
- `internal/config/config.go` — load config, merge with CLI flags (flags win)

**Files to modify:**
- `main.go` — load config before parsing flags, apply defaults

#### 16.2 — Query aliases
**Status:** [ ] Not started

```
@err                    # expands to level:error OR level:fatal
@slow AND service:api   # expands inline
```

**Files to modify:**
- `internal/query/lexer.go` — recognize `@alias` tokens
- `internal/query/parser.go` — expand aliases before parsing
- `internal/config/config.go` — load aliases from config

#### 16.3 — Tests
**Status:** [ ] Not started

- Config loads correctly
- Missing config file → no error, defaults apply
- CLI flags override config values
- Query aliases expand correctly
- Circular alias → error, not infinite loop

---

### Phase 17: Homebrew Tap
**Priority:** Low — Distribution convenience
**Effort:** Low
**Status:** [ ] Not started

#### 17.1 — Create Homebrew tap repository
**Status:** [ ] Not started

Create `homebrew-tap` repo with formula:
```ruby
class Logq < Formula
  desc "Fast, interactive terminal log explorer"
  homepage "https://github.com/riccardomerenda/logq"
  url "https://github.com/riccardomerenda/logq/releases/download/v#{version}/logq_#{version}_darwin_amd64.tar.gz"
  # ...
end
```

#### 17.2 — GoReleaser Homebrew integration
**Status:** [ ] Not started

Add to `.goreleaser.yml`:
```yaml
brews:
  - repository:
      owner: riccardomerenda
      name: homebrew-tap
    homepage: "https://github.com/riccardomerenda/logq"
    description: "Fast, interactive terminal log explorer"
```

**Files to modify:**
- `.goreleaser.yml` — add `brews` section

#### 17.3 — Install instructions
**Status:** [ ] Not started

Update README:
```bash
brew install riccardomerenda/tap/logq
```

---

### Phase 18: Demo GIF
**Priority:** Low — Important for GitHub discoverability
**Effort:** Low
**Status:** [ ] Not started

#### 18.1 — Create VHS tape file
**Status:** [ ] Not started

Use [vhs](https://github.com/charmbracelet/vhs) to record a terminal demo.

**Files to create:**
- `demo.tape` — VHS script showing: open file → scroll → type query → filter → inspect detail → quit

```
Output demo.gif
Set Width 1200
Set Height 600
Set FontSize 14
Set Theme "Dracula"

Type "logq testdata/sample.jsonl"
Enter
Sleep 2s
Type "/"
Sleep 500ms
Type "level:error"
Sleep 2s
Type " AND latency>100"
Sleep 3s
Escape
Sleep 1s
Down Down Down
Enter
Sleep 3s
Escape
Type "q"
```

#### 18.2 — Add to README
**Status:** [ ] Not started

Replace ASCII art with GIF:
```markdown
![logq demo](demo.gif)
```

---

## Implementation Priority Matrix

```
                        Low Effort ──────────────── High Effort
                        │                                    │
  High    ┌─────────────┼────────────────────────────────────┤
  Value   │  Phase 10   │  Phase 7    Phase 9                │
          │  (history)  │  (time)     (multi-file)           │
          │  Phase 8    │                                    │
          │  (export)   │                                    │
          ├─────────────┼────────────────────────────────────┤
  Medium  │  Phase 13   │  Phase 11   Phase 12               │
  Value   │  (themes)   │  (complete) (highlight)            │
          │  Phase 18   │                                    │
          │  (demo gif) │                                    │
          ├─────────────┼────────────────────────────────────┤
  Low     │  Phase 14   │  Phase 15   Phase 16               │
  Value   │  (bookmark) │  (drill)    (config)               │
          │  Phase 17   │                                    │
          │  (homebrew) │                                    │
          └─────────────┴────────────────────────────────────┘
```

## Suggested Implementation Order

| Order | Phase | Feature | Why |
|---|---|---|---|
| 1 | 8 | Export filtered results | Low effort, immediately useful, enables scripting |
| 2 | 10 | Query history | Very low effort, big UX win |
| 3 | 7 | Time range queries | Infrastructure already exists, high impact |
| 4 | 9 | Multiple file support | Common real-world need |
| 5 | 13 | Color themes | Quick win for light terminal users |
| 6 | 12 | Regex highlighting | Visual feedback improves query confidence |
| 7 | 11 | Field auto-complete | Discoverability for new users |
| 8 | 14 | Bookmarks | Nice for long debugging sessions |
| 9 | 18 | Demo GIF | GitHub discoverability |
| 10 | 17 | Homebrew tap | Distribution convenience |
| 11 | 15 | JSON drill-down | Useful but niche (deeply nested logs) |
| 12 | 16 | Config file | Power user feature, keep zero-config default |

---

## Status Legend

- `[x]` Complete
- `[~]` Partial / in progress
- `[ ]` Not started
