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
	"github.com/riccardomerenda/logq/internal/history"
	"github.com/riccardomerenda/logq/internal/index"
	"github.com/riccardomerenda/logq/internal/input"
	"github.com/riccardomerenda/logq/internal/output"
	"github.com/riccardomerenda/logq/internal/parser"
	"github.com/riccardomerenda/logq/internal/query"
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
}

// NewModel creates a new app model.
func NewModel(idx *index.Index, filename, fileSize string) Model {
	m := Model{
		index:     idx,
		results:   idx.AllIDs(),
		logView:   NewLogView(),
		histogram: NewHistogram(),
		queryBar:  NewQueryBar(),
		statusBar: NewStatusBar(),
		detail:    NewDetailView(),
		keys:      DefaultKeyMap(),
		filename:  filename,
		fileSize:  fileSize,
	}

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
		m.focus = FocusQueryBar
		m.queryBar.Focus()
		return m, nil
	case key.Matches(msg, m.keys.Tab):
		if m.focus == FocusLogView {
			m.focus = FocusHistogram
			m.histogram.SetFocused(true)
		} else {
			m.focus = FocusLogView
			m.histogram.SetFocused(false)
		}
		return m, nil
	case key.Matches(msg, m.keys.Enter):
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
		if m.focus == FocusHistogram {
			m.histogram.ScrollUp()
		} else {
			m.logView.ScrollUp(1)
		}
		return m, nil
	case key.Matches(msg, m.keys.Down):
		if m.focus == FocusHistogram {
			m.histogram.ScrollDown()
		} else {
			m.logView.ScrollDown(1)
		}
		return m, nil
	case key.Matches(msg, m.keys.PageUp):
		m.logView.ScrollUp(m.logView.height)
		return m, nil
	case key.Matches(msg, m.keys.PageDown):
		m.logView.ScrollDown(m.logView.height)
		return m, nil
	case key.Matches(msg, m.keys.Home):
		m.logView.GoToStart()
		return m, nil
	case key.Matches(msg, m.keys.End):
		m.logView.GoToEnd()
		return m, nil
	case key.Matches(msg, m.keys.Escape):
		// Clear the query
		m.queryBar.SetValue("")
		m.executeQuery()
		return m, nil
	case key.Matches(msg, m.keys.Save):
		path := m.saveResults()
		if path != "" {
			m.statusBar.flashMsg = fmt.Sprintf("Saved %d records to %s", len(m.results), path)
			return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg { return clearFlashMsg{} })
		}
		return m, nil
	}

	return m, nil
}

func (m *Model) executeQuery() {
	m.queryStr = m.queryBar.Value()

	start := time.Now()
	if m.queryStr == "" {
		m.results = m.index.AllIDs()
		m.queryError = ""
		m.queryBar.SetError("")
		m.logView.SetHighlights(nil)
		m.detail.SetHighlights(nil)
	} else {
		node, err := query.ParseQuery(m.queryStr)
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

	// Main layout: log view (left) | histogram (right)
	// Manual horizontal join to ensure styled separators on every line
	logContent := m.logView.View()
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
