package ui

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/riccardomerenda/logq/internal/alias"
	"github.com/riccardomerenda/logq/internal/history"
	"github.com/riccardomerenda/logq/internal/index"
	"github.com/riccardomerenda/logq/internal/input"
	"github.com/riccardomerenda/logq/internal/output"
	"github.com/riccardomerenda/logq/internal/parser"
	"github.com/riccardomerenda/logq/internal/pattern"
	"github.com/riccardomerenda/logq/internal/query"
	"github.com/riccardomerenda/logq/internal/trace"
)

// followTickMsg triggers a check for new file content.
type followTickMsg time.Time

// newRecordsMsg carries newly parsed records from follow mode.
type newRecordsMsg struct {
	records []parser.Record
}

// clearFlashMsg signals that the status bar flash message should be cleared.
type clearFlashMsg struct{}

// clearCopyMsg signals that the detail view copy message should be cleared.
type clearCopyMsg struct{}

// Focus tracks which panel has focus.
type Focus int

const (
	FocusLogView Focus = iota
	FocusQueryBar
	FocusHistogram
)

// Model is the main bubbletea model.
type Model struct {
	// Data
	index      *index.Index
	results    []int
	queryStr   string
	queryError string
	queryTime  time.Duration

	// UI components
	logView   LogView
	histogram Histogram
	queryBar  QueryBar
	statusBar StatusBar
	detail    DetailView
	keys      KeyMap

	// UI state
	width      int
	height     int
	focus      Focus
	showDetail bool
	filename   string
	fileSize   string

	// Follow mode
	followReader *input.FollowReader
	following    bool

	// Persistent history
	historyPath string

	// Column mode
	columns []string

	// Aliases
	aliases *alias.Registry

	// Trace following
	traceFields    []string // ID field names to detect
	traceActive    bool
	traceOriginIdx int    // record index that started the trace
	prevQuery      string // query before trace was activated

	// Pattern clustering
	patternView     PatternView
	patternMode     bool
	preDrillResults []int // results before drilling into a cluster

	// Bookmarks
	bookmarks          map[int]bool
	bookmarkFilter     bool
	preBookmarkResults []int
}

// NewModel creates a new app model.
func NewModel(idx *index.Index, filename, fileSize string) Model {
	m := Model{
		index:       idx,
		results:     idx.AllIDs(),
		logView:     NewLogView(),
		histogram:   NewHistogram(),
		queryBar:    NewQueryBar(),
		statusBar:   NewStatusBar(),
		detail:      NewDetailView(),
		patternView: NewPatternView(),
		keys:        DefaultKeyMap(),
		filename:    filename,
		fileSize:    fileSize,
		bookmarks:   make(map[int]bool),
	}
	m.logView.SetBookmarks(m.bookmarks)

	m.logView.SetResults(idx.Records, m.results)
	m.updateHistogram()
	m.updateStatusBar()

	return m
}

// SetFollowReader enables follow mode for tailing a file.
func (m *Model) SetFollowReader(fr *input.FollowReader) {
	m.followReader = fr
	m.following = true
}

// SetHistory loads persistent history entries and sets the history file path.
func (m *Model) SetHistory(entries []string, path string) {
	m.historyPath = path
	m.queryBar.SetHistory(entries)
}

// SetColumns sets the column names for column mode display.
func (m *Model) SetColumns(cols []string) {
	m.columns = cols
	m.logView.SetColumns(cols)
}

// SetAliases sets the alias registry for query expansion and autocomplete.
func (m *Model) SetAliases(reg *alias.Registry) {
	m.aliases = reg
	m.queryBar.SetAliases(reg)
}

// SetTraceFields sets the list of field names to use for trace ID detection.
func (m *Model) SetTraceFields(fields []string) {
	m.traceFields = fields
}


// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	if m.following {
		return m.followTick()
	}
	return nil
}

func (m Model) followTick() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return followTickMsg(t)
	})
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateLayout()
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case followTickMsg:
		return m.handleFollowTick()

	case newRecordsMsg:
		m.index.AddRecords(msg.records)
		m.executeQuery()
		return m, m.followTick()

	case clearFlashMsg:
		m.statusBar.flashMsg = ""
		return m, nil

	case clearCopyMsg:
		m.detail.copyMsg = ""
		return m, nil
	}

	// Pass through to query bar if focused
	if m.focus == FocusQueryBar {
		ti := m.queryBar.TextInput()
		newTi, cmd := ti.Update(msg)
		*ti = newTi

		// Re-execute query if text changed
		if m.queryBar.Value() != m.queryStr {
			m.executeQuery()
		}

		return m, cmd
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Detail overlay takes priority
	if m.showDetail {
		// Pick mode for trace ID selection
		if m.detail.pickMode {
			switch {
			case key.Matches(msg, m.keys.Up):
				m.detail.PickUp()
			case key.Matches(msg, m.keys.Down):
				m.detail.PickDown()
			case key.Matches(msg, m.keys.Enter):
				selected := m.detail.PickSelected()
				m.detail.ExitPickMode()
				m.applyTrace(selected)
			case key.Matches(msg, m.keys.Escape):
				m.detail.ExitPickMode()
			}
			return m, nil
		}

		if key.Matches(msg, m.keys.Escape) {
			m.showDetail = false
		}
		if key.Matches(msg, m.keys.Copy) {
			if m.detail.record != nil {
				if copyToClipboard(m.detail.record.Raw) == nil {
					m.detail.copyMsg = "Copied!"
				} else {
					m.detail.copyMsg = "Copy failed"
				}
				return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg { return clearCopyMsg{} })
			}
		}
		if key.Matches(msg, m.keys.Trace) {
			if m.detail.record != nil {
				ids := trace.DetectIDFields(*m.detail.record, m.traceFields)
				if len(ids) == 0 {
					m.detail.copyMsg = "No trace IDs detected"
					return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg { return clearCopyMsg{} })
				}
				if len(ids) == 1 {
					m.applyTrace(ids[0])
				} else {
					m.detail.EnterPickMode(ids)
				}
			}
		}
		return m, nil
	}

	// Query bar focused
	if m.focus == FocusQueryBar {
		switch {
		case key.Matches(msg, m.keys.Escape):
			m.queryBar.Blur()
			m.focus = FocusLogView
			return m, nil
		case key.Matches(msg, m.keys.Enter):
			q := m.queryBar.Value()
			m.queryBar.PushHistory(q)
			if m.historyPath != "" && q != "" {
				_ = history.Append(m.historyPath, q)
			}
			m.executeQuery()
			m.queryBar.Blur()
			m.focus = FocusLogView
			return m, nil
		case key.Matches(msg, m.keys.Tab):
			if m.queryBar.AcceptCompletion() {
				m.queryBar.UpdateCompletions(m.index)
				if m.queryBar.Value() != m.queryStr {
					m.executeQuery()
				}
			}
			return m, nil
		case key.Matches(msg, m.keys.Up):
			if m.queryBar.HistoryUp() {
				m.queryBar.completer.Reset()
				if m.queryBar.Value() != m.queryStr {
					m.executeQuery()
				}
				return m, nil
			}
			return m, nil
		case key.Matches(msg, m.keys.Down):
			if m.queryBar.HistoryDown() {
				m.queryBar.completer.Reset()
				if m.queryBar.Value() != m.queryStr {
					m.executeQuery()
				}
				return m, nil
			}
			return m, nil
		default:
			ti := m.queryBar.TextInput()
			newTi, cmd := ti.Update(msg)
			*ti = newTi

			// Live filtering
			if m.queryBar.Value() != m.queryStr {
				m.executeQuery()
			}

			// Update completions after text change
			m.queryBar.UpdateCompletions(m.index)

			return m, cmd
		}
	}

	// Log view or histogram focused
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit
	case key.Matches(msg, m.keys.Search):
		if m.patternMode {
			return m, nil // no search in pattern mode
		}
		m.focus = FocusQueryBar
		m.queryBar.Focus()
		return m, nil
	case key.Matches(msg, m.keys.Tab):
		if m.patternMode {
			return m, nil
		}
		if m.focus == FocusLogView {
			m.focus = FocusHistogram
			m.histogram.SetFocused(true)
		} else {
			m.focus = FocusLogView
			m.histogram.SetFocused(false)
		}
		return m, nil
	case key.Matches(msg, m.keys.Enter):
		if m.patternMode {
			// Drill into selected cluster
			if c := m.patternView.SelectedCluster(); c != nil {
				m.preDrillResults = m.results
				m.results = c.RecordIDs
				m.logView.SetResults(m.index.Records, m.results)
				m.patternMode = false
				m.updateHistogram()
				m.updateStatusBar()
			}
			return m, nil
		}
		if m.focus == FocusLogView {
			idx := m.logView.SelectedRecordIndex()
			if idx >= 0 && idx < len(m.index.Records) {
				r := m.index.Records[idx]
				m.detail.SetRecord(&r)
				m.showDetail = true
			}
		} else if m.focus == FocusHistogram {
			// Jump log view to the selected time bucket
			if b := m.histogram.SelectedBucket(); b != nil {
				ids := m.index.TimeRange(b.Start, b.End)
				if len(ids) > 0 {
					// Find this record in results
					for i, rid := range m.results {
						if rid == ids[0] {
							m.logView.cursor = i
							m.logView.ensureVisible()
							break
						}
					}
				}
				m.focus = FocusLogView
				m.histogram.SetFocused(false)
			}
		}
		return m, nil
	case key.Matches(msg, m.keys.Up):
		if m.patternMode {
			m.patternView.ScrollUp(1)
			return m, nil
		}
		if m.focus == FocusHistogram {
			m.histogram.ScrollUp()
		} else {
			m.logView.ScrollUp(1)
		}
		return m, nil
	case key.Matches(msg, m.keys.Down):
		if m.patternMode {
			m.patternView.ScrollDown(1)
			return m, nil
		}
		if m.focus == FocusHistogram {
			m.histogram.ScrollDown()
		} else {
			m.logView.ScrollDown(1)
		}
		return m, nil
	case key.Matches(msg, m.keys.PageUp):
		if m.patternMode {
			m.patternView.ScrollUp(m.patternView.height)
		} else {
			m.logView.ScrollUp(m.logView.height)
		}
		return m, nil
	case key.Matches(msg, m.keys.PageDown):
		if m.patternMode {
			m.patternView.ScrollDown(m.patternView.height)
		} else {
			m.logView.ScrollDown(m.logView.height)
		}
		return m, nil
	case key.Matches(msg, m.keys.Home):
		if m.patternMode {
			m.patternView.GoToStart()
		} else {
			m.logView.GoToStart()
		}
		return m, nil
	case key.Matches(msg, m.keys.End):
		if m.patternMode {
			m.patternView.GoToEnd()
		} else {
			m.logView.GoToEnd()
		}
		return m, nil
	case key.Matches(msg, m.keys.Escape):
		if m.patternMode {
			if m.preDrillResults != nil {
				// Restore pre-drill results and re-enter pattern mode
				m.results = m.preDrillResults
				m.preDrillResults = nil
				m.logView.SetResults(m.index.Records, m.results)
				m.updateHistogram()
				clusters := pattern.Clusterize(m.index.Records, m.results)
				m.patternView.SetClusters(clusters, len(m.results))
				m.patternMode = true
			} else {
				m.patternMode = false
			}
			m.updateStatusBar()
			return m, nil
		}
		if m.bookmarkFilter {
			// Restore pre-bookmark results
			m.results = m.preBookmarkResults
			m.preBookmarkResults = nil
			m.bookmarkFilter = false
			m.logView.SetResults(m.index.Records, m.results)
			m.updateHistogram()
			m.updateStatusBar()
			return m, nil
		}
		// Clear the query
		m.queryBar.SetValue("")
		m.executeQuery()
		return m, nil
	case key.Matches(msg, m.keys.Pattern):
		if m.patternMode {
			m.patternMode = false
		} else {
			clusters := pattern.Clusterize(m.index.Records, m.results)
			m.patternView.SetClusters(clusters, len(m.results))
			m.patternMode = true
			m.preDrillResults = nil
		}
		m.updateStatusBar()
		return m, nil
	case key.Matches(msg, m.keys.BookmarkToggle):
		if m.patternMode {
			return m, nil
		}
		recIdx := m.logView.SelectedRecordIndex()
		if recIdx >= 0 {
			if m.bookmarks[recIdx] {
				delete(m.bookmarks, recIdx)
			} else {
				m.bookmarks[recIdx] = true
			}
			m.updateStatusBar()
		}
		return m, nil
	case key.Matches(msg, m.keys.BookmarkNext):
		if m.patternMode || len(m.bookmarks) == 0 {
			return m, nil
		}
		current := m.logView.cursor
		// Search forward from current+1, then wrap
		for i := current + 1; i < len(m.results); i++ {
			if m.bookmarks[m.results[i]] {
				m.logView.cursor = i
				m.logView.ensureVisible()
				return m, nil
			}
		}
		for i := 0; i <= current; i++ {
			if m.bookmarks[m.results[i]] {
				m.logView.cursor = i
				m.logView.ensureVisible()
				return m, nil
			}
		}
		return m, nil
	case key.Matches(msg, m.keys.BookmarkFilter):
		if m.patternMode {
			return m, nil
		}
		if m.bookmarkFilter {
			m.results = m.preBookmarkResults
			m.preBookmarkResults = nil
			m.bookmarkFilter = false
		} else if len(m.bookmarks) > 0 {
			m.preBookmarkResults = m.results
			filtered := make([]int, 0)
			for _, id := range m.results {
				if m.bookmarks[id] {
					filtered = append(filtered, id)
				}
			}
			m.results = filtered
			m.bookmarkFilter = true
		}
		m.logView.SetResults(m.index.Records, m.results)
		m.updateHistogram()
		m.updateStatusBar()
		return m, nil
	case key.Matches(msg, m.keys.Save):
		path := m.saveResults()
		if path != "" {
			m.statusBar.flashMsg = fmt.Sprintf("Saved %d records to %s", len(m.results), path)
			return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg { return clearFlashMsg{} })
		}
		return m, nil
	case key.Matches(msg, m.keys.TraceClear):
		if m.traceActive {
			m.traceActive = false
			m.queryBar.SetValue(m.prevQuery)
			m.logView.SetTraceOrigin(-1)
			m.executeQuery()
			m.statusBar.flashMsg = "Trace filter cleared"
			return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg { return clearFlashMsg{} })
		}
		return m, nil
	}

	return m, nil
}

func (m *Model) executeQuery() {
	m.queryStr = m.queryBar.Value()

	// Expand aliases before parsing
	queryToEval := m.queryStr
	if m.aliases != nil && strings.Contains(queryToEval, "@") {
		expanded, err := m.aliases.Expand(queryToEval)
		if err != nil {
			if m.focus == FocusQueryBar {
				m.queryBar.SetError("")
				return
			}
			m.queryError = err.Error()
			m.queryBar.SetError(err.Error())
			return
		}
		queryToEval = expanded
	}

	start := time.Now()
	if queryToEval == "" {
		m.results = m.index.AllIDs()
		m.queryError = ""
		m.queryBar.SetError("")
		m.logView.SetHighlights(nil)
		m.detail.SetHighlights(nil)
	} else {
		node, err := query.ParseQuery(queryToEval)
		if err != nil {
			// While typing, silently keep previous results
			if m.focus == FocusQueryBar {
				m.queryBar.SetError("")
				return
			}
			// After Enter (blurred), show the error
			m.queryError = err.Error()
			m.queryBar.SetError(err.Error())
			return
		}
		m.results = query.Evaluate(node, m.index)
		m.queryError = ""
		m.queryBar.SetError("")

		highlights := ExtractHighlightTerms(node)
		m.logView.SetHighlights(highlights)
		m.detail.SetHighlights(highlights)
	}
	m.queryTime = time.Since(start)

	m.logView.SetResults(m.index.Records, m.results)
	m.updateHistogram()
	m.updateStatusBar()
}

func (m *Model) updateHistogram() {
	bucketCount := m.histogram.height - 2
	if bucketCount < 5 {
		bucketCount = 10
	}
	buckets := m.index.Histogram(bucketCount, m.results)
	m.histogram.SetBuckets(buckets)
}

func (m *Model) updateStatusBar() {
	m.statusBar.Update(len(m.results), m.index.TotalCount, m.queryTime, m.filename, m.fileSize)
	m.statusBar.following = m.following
	m.statusBar.traceActive = m.traceActive
	m.statusBar.patternMode = m.patternMode
	m.statusBar.bookmarkCount = len(m.bookmarks)
	m.statusBar.bookmarkFilter = m.bookmarkFilter
}

// applyTrace activates trace following for the given ID field.
func (m *Model) applyTrace(id trace.IDField) {
	// Save current query for restoration with T
	m.prevQuery = m.queryBar.Value()
	m.traceActive = true
	m.traceOriginIdx = m.logView.SelectedRecordIndex()
	m.logView.SetTraceOrigin(m.traceOriginIdx)

	// Set the trace query and execute
	q := trace.BuildQuery(id.Name, id.Value)
	m.queryBar.SetValue(q)
	m.executeQuery()

	// Close detail view
	m.showDetail = false
}

func (m *Model) updateLayout() {
	histWidth := m.width / 4
	if histWidth < 25 {
		histWidth = 25
	}
	if histWidth > 40 {
		histWidth = 40
	}
	logWidth := m.width - histWidth - 2 // 2 for separator

	contentHeight := m.height - 4 // query bar + status bar + borders

	m.logView.SetSize(logWidth, contentHeight)
	m.patternView.SetSize(logWidth, contentHeight)
	m.histogram.SetSize(histWidth, contentHeight)
	m.queryBar.SetWidth(m.width)
	m.statusBar.SetSize(m.width)
	m.detail.SetSize(m.width, m.height)

	m.updateHistogram()
}

func (m Model) handleFollowTick() (tea.Model, tea.Cmd) {
	if m.followReader == nil {
		return m, nil
	}

	lines, err := m.followReader.ReadNew()
	if err != nil || len(lines) == 0 {
		return m, m.followTick()
	}

	entries := input.GroupLines(lines)
	records := make([]parser.Record, 0, len(entries))
	for _, e := range entries {
		records = append(records, parser.Parse(e.Text, e.LineNumber))
	}

	if len(records) == 0 {
		return m, m.followTick()
	}

	return m, func() tea.Msg {
		return newRecordsMsg{records: records}
	}
}

func (m *Model) saveResults() string {
	if len(m.results) == 0 {
		return ""
	}
	filename := fmt.Sprintf("logq-export-%s.jsonl", time.Now().Format("20060102-150405"))
	f, err := os.Create(filename)
	if err != nil {
		return ""
	}
	defer f.Close()
	if err := output.Write(f, m.index.Records, m.results, output.FormatRaw); err != nil {
		return ""
	}
	return filename
}

func copyToClipboard(text string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("clip")
	case "darwin":
		cmd = exec.Command("pbcopy")
	default:
		cmd = exec.Command("xclip", "-selection", "clipboard")
	}
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

// View implements tea.Model.
func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	// Detail overlay
	if m.showDetail {
		return m.detail.View()
	}

	// Main layout: log/pattern view (left) | histogram (right)
	// Manual horizontal join to ensure styled separators on every line
	var logContent string
	if m.patternMode {
		logContent = m.patternView.View()
	} else {
		logContent = m.logView.View()
	}
	histContent := m.histogram.View()
	logLines := strings.Split(logContent, "\n")
	histLines := strings.Split(histContent, "\n")
	sep := StyleBase.Render("  ")

	lineCount := len(logLines)
	if len(histLines) > lineCount {
		lineCount = len(histLines)
	}
	mainLines := make([]string, lineCount)
	for i := 0; i < lineCount; i++ {
		ll := ""
		if i < len(logLines) {
			ll = logLines[i]
		}
		hl := ""
		if i < len(histLines) {
			hl = histLines[i]
		}
		mainLines[i] = ll + sep + hl
	}

	// Stack: main panel, query bar, status bar
	content := strings.Join(mainLines, "\n") + "\n" +
		m.queryBar.View() + "\n" +
		m.statusBar.View()
	return RenderAppBackground(content, m.width, m.height)
}
