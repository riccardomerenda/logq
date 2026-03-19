package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/riccardomerenda/logq/internal/alias"
	"github.com/riccardomerenda/logq/internal/config"
	"github.com/riccardomerenda/logq/internal/pattern"
	"github.com/riccardomerenda/logq/internal/trace"
	"github.com/riccardomerenda/logq/internal/history"
	"github.com/riccardomerenda/logq/internal/index"
	"github.com/riccardomerenda/logq/internal/input"
	"github.com/riccardomerenda/logq/internal/output"
	"github.com/riccardomerenda/logq/internal/parser"
	"github.com/riccardomerenda/logq/internal/query"
	"github.com/riccardomerenda/logq/internal/ui"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Printf("logq %s\n", version)
		os.Exit(0)
	}

	if len(os.Args) > 1 && (os.Args[1] == "--help" || os.Args[1] == "-h") {
		printUsage()
		os.Exit(0)
	}

	// Parse arguments
	args := os.Args[1:]
	follow := false
	queryStr := ""
	outputPath := ""
	outputFmt := ""
	countOnly := false
	patternsMode := false
	themeName := ""
	groupByField := ""
	topN := 0
	var columns []string

	if len(args) > 0 && args[0] == "update" {
		selfUpdate()
		os.Exit(0)
	}
	if len(args) > 0 && args[0] == "init" {
		if _, err := os.Stat(".logq.toml"); err == nil {
			fmt.Fprintln(os.Stderr, ".logq.toml already exists in this directory")
			os.Exit(1)
		}
		if err := os.WriteFile(".logq.toml", []byte(config.ScaffoldTemplate()), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating .logq.toml: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Created .logq.toml — edit it to configure logq for this project")
		os.Exit(0)
	}

	// Parse flags
	var fileArgs []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-f":
			follow = true
		case "-q":
			if i+1 < len(args) {
				i++
				queryStr = args[i]
			}
		case "-o":
			if i+1 < len(args) {
				i++
				outputPath = args[i]
			}
		case "--format":
			if i+1 < len(args) {
				i++
				outputFmt = args[i]
			}
		case "--count":
			countOnly = true
		case "--patterns":
			patternsMode = true
		case "--theme":
			if i+1 < len(args) {
				i++
				themeName = args[i]
			}
		case "--group-by":
			if i+1 < len(args) {
				i++
				groupByField = args[i]
			}
		case "--top":
			if i+1 < len(args) {
				i++
				n, err := strconv.Atoi(args[i])
				if err != nil {
					fmt.Fprintf(os.Stderr, "Invalid --top value: %s\n", args[i])
					os.Exit(1)
				}
				topN = n
			}
		case "--columns":
			if i+1 < len(args) {
				i++
				columns = strings.Split(args[i], ",")
			}
		default:
			fileArgs = append(fileArgs, args[i])
		}
	}

	// Load config file (.logq.toml)
	cfgPath, _ := config.FindConfig()
	cfg, cfgErr := config.Load(cfgPath)
	if cfgErr != nil {
		fmt.Fprintf(os.Stderr, "Warning: %v\n", cfgErr)
		cfg = &config.Config{}
	}
	// Merge: CLI flags win over config
	if themeName == "" {
		themeName = cfg.Theme
	}
	if len(columns) == 0 && len(cfg.Columns) > 0 {
		columns = cfg.Columns
	}
	// Build alias registry
	aliasReg := alias.NewRegistry(cfg.Aliases)

	// Resolve trace ID fields
	traceFields := cfg.Trace.IDFields
	if len(traceFields) == 0 {
		traceFields = trace.DefaultIDFields
	}

	var records []parser.Record
	var filename string
	var fileSize string
	var followOffset int64
	multiFile := len(fileArgs) > 1

	if len(fileArgs) > 0 {
		for _, path := range fileArgs {
			recs, err := readFile(path, multiFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", path, err)
				os.Exit(1)
			}
			records = append(records, recs...)
		}

		if multiFile {
			filename = fmt.Sprintf("%d files", len(fileArgs))
			// Sort by timestamp for merged timeline
			sort.SliceStable(records, func(i, j int) bool {
				if records[i].Timestamp.IsZero() || records[j].Timestamp.IsZero() {
					return false // keep original order for records without timestamps
				}
				return records[i].Timestamp.Before(records[j].Timestamp)
			})
			// Compute total size
			var totalSize int64
			for _, path := range fileArgs {
				if info, e := os.Stat(path); e == nil {
					totalSize += info.Size()
				}
			}
			fileSize = formatSize(totalSize)
		} else {
			filename = fileArgs[0]
			if info, e := os.Stat(fileArgs[0]); e == nil {
				fileSize = formatSize(info.Size())
				followOffset = info.Size()
			}
		}
	} else if input.IsStdinPipe() {
		recs, err := readStdin()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
			os.Exit(1)
		}
		records = recs
		filename = "stdin"
		follow = false
	} else {
		printUsage()
		os.Exit(1)
	}

	if len(records) == 0 {
		fmt.Fprintln(os.Stderr, "No log lines found in input.")
		os.Exit(1)
	}

	// Build index
	idx := index.Build(records)

	// Apply theme
	switch themeName {
	case "light":
		ui.ApplyTheme(ui.LightTheme)
	case "dark":
		ui.ApplyTheme(ui.DarkTheme)
	case "auto", "":
		ui.ApplyTheme(ui.DetectTheme())
	default:
		fmt.Fprintf(os.Stderr, "Unknown theme %q (valid: auto, dark, light)\n", themeName)
		os.Exit(1)
	}

	// Batch mode: -q flag provided, --group-by, --count, or --patterns
	if queryStr != "" || countOnly || groupByField != "" || patternsMode {
		// Expand aliases in batch query
		if queryStr != "" {
			expanded, err := aliasReg.Expand(queryStr)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Alias error: %v\n", err)
				os.Exit(1)
			}
			queryStr = expanded
		}
		runBatch(idx, queryStr, outputPath, outputFmt, countOnly, patternsMode, groupByField, topN, columns)
		return
	}

	// Follow mode only works with a single file
	if follow && multiFile {
		fmt.Fprintln(os.Stderr, "Warning: follow mode (-f) only works with a single file, ignoring -f")
		follow = false
	}

	// Load persistent history
	histPath := history.HistoryPath()
	histEntries, _ := history.Load(histPath)

	// Start TUI
	model := ui.NewModel(idx, filename, fileSize)
	model.SetHistory(histEntries, histPath)
	model.SetAliases(aliasReg)
	model.SetTraceFields(traceFields)
	if len(columns) > 0 {
		model.SetColumns(columns)
	}
	if len(cfg.Views) > 0 {
		names := make([]string, 0, len(cfg.Views))
		for name := range cfg.Views {
			names = append(names, name)
		}
		sort.Strings(names)
		views := make([]ui.SavedView, len(names))
		for i, name := range names {
			v := cfg.Views[name]
			views[i] = ui.SavedView{Name: name, Query: v.Query, Columns: v.Columns}
		}
		model.SetViews(views)
	}
	if follow && len(fileArgs) == 1 {
		fr := input.NewFollowReader(fileArgs[0], followOffset)
		model.SetFollowReader(fr)
	}
	p := tea.NewProgram(model, tea.WithAltScreen())

	if err := p.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Deduplicate and cap history on exit
	if entries, loadErr := history.Load(histPath); loadErr == nil && len(entries) > 0 {
		_ = history.Save(histPath, entries)
	}
}

// readFile reads and parses a single file. If multiFile is true, adds source field.
func readFile(path string, multiFile bool) ([]parser.Record, error) {
	reader, err := input.NewFileReader(path)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	lines, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	entries := input.GroupLines(lines)
	records := make([]parser.Record, 0, len(entries))
	source := filepath.Base(path)
	for _, entry := range entries {
		r := parser.Parse(entry.Text, entry.LineNumber)
		if multiFile {
			r.Fields["source"] = source
		}
		records = append(records, r)
	}
	return records, nil
}

// readStdin reads and parses from stdin.
func readStdin() ([]parser.Record, error) {
	reader := input.NewStdinReader()
	lines, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	entries := input.GroupLines(lines)
	records := make([]parser.Record, 0, len(entries))
	for _, entry := range entries {
		records = append(records, parser.Parse(entry.Text, entry.LineNumber))
	}
	return records, nil
}

func runBatch(idx *index.Index, queryStr, outputPath, outputFmt string, countOnly, patternsMode bool, groupByField string, topN int, columns []string) {
	// Parse and evaluate query
	var results []int
	if queryStr == "" {
		results = idx.AllIDs()
	} else {
		node, err := query.ParseQuery(queryStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Query error: %v\n", err)
			os.Exit(1)
		}
		results = query.Evaluate(node, idx)
	}

	// Count-only mode
	if countOnly {
		fmt.Println(len(results))
		return
	}

	// Determine output format
	format, err := output.ParseFormat(outputFmt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	// Determine output writer
	var w *os.File
	if outputPath != "" {
		w, err = os.Create(outputPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
			os.Exit(1)
		}
		defer w.Close()
	} else {
		w = os.Stdout
	}

	// Pattern clustering mode
	if patternsMode {
		clusters := pattern.Clusterize(idx.Records, results)
		if topN > 0 && topN < len(clusters) {
			clusters = clusters[:topN]
		}
		patternResults := make([]output.PatternResult, len(clusters))
		for i, c := range clusters {
			patternResults[i] = output.PatternResult{Template: c.Template, Count: c.Count}
		}
		if err := output.WritePatterns(w, patternResults, format); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Aggregation mode
	if groupByField != "" {
		groups := output.GroupBy(idx.Records, results, groupByField)
		groups = output.TopN(groups, topN)
		if err := output.WriteGroups(w, groups, format); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Regular output (with optional column filtering)
	if len(columns) > 0 {
		err = output.WriteWithColumns(w, idx.Records, results, format, columns)
	} else {
		err = output.Write(w, idx.Records, results, format)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
		os.Exit(1)
	}

	if outputPath != "" {
		fmt.Fprintf(os.Stderr, "%d records written to %s\n", len(results), outputPath)
	}
}

func selfUpdate() {
	fmt.Printf("logq %s — checking for updates...\n", version)

	goPath, err := exec.LookPath("go")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: Go is not installed. Install Go from https://go.dev or download a binary from:")
		fmt.Fprintln(os.Stderr, "  https://github.com/riccardomerenda/logq/releases")
		os.Exit(1)
	}

	cmd := exec.Command(goPath, "install", "github.com/riccardomerenda/logq@latest")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Println("Running: go install github.com/riccardomerenda/logq@latest")
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Update failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Updated successfully. Run 'logq --version' to check the new version.")
}

func printUsage() {
	fmt.Printf(`logq %s — Fast, interactive log explorer for the terminal

Usage:
  logq <file>              Open a log file
  logq <file1> <file2>     Open multiple files (merged by timestamp)
  logq <file.gz>           Open a gzipped log file
  logq -f <file>           Follow a growing file (like tail -f)
  <cmd> | logq             Read from stdin pipe
  logq init                Create a .logq.toml config file in the current directory
  logq update              Update to the latest version

Options:
  -f                   Follow mode — watch for new lines appended to the file
  -q <query>           Run query in batch mode (no TUI), output to stdout
  -o <file>            Write results to file (use with -q)
  --format <fmt>       Output format: raw (default), json, csv
  --count              Print match count only (use with -q)
  --theme <name>       Color theme: auto (default), dark, light
  --group-by <field>   Group results by field and show counts (batch mode)
  --top <n>            Show only top N groups (use with --group-by)
  --columns <fields>   Comma-separated list of fields to display as columns
  --patterns           Show log pattern clusters (batch mode)
  -h, --help           Show this help
  -v, --version        Show version

Keyboard:
  /          Focus filter bar
  j/k, Up/Dn Scroll logs
  Enter      Show record detail / toggle fold (tree view)
  d          Copy dot-path (in JSON tree view)
  Left/Right Collapse/expand node (in JSON tree view)
  t          Follow trace ID (in detail view)
  T          Clear trace filter
  p          Toggle pattern clustering view
  m          Toggle bookmark on current record
  '          Jump to next bookmark
  B          Filter to bookmarked records only
  1-9        Switch to saved view (from .logq.toml)
  0          Clear saved view
  s          Save filtered results to file
  Tab        Toggle histogram focus
  Esc        Clear filter / close detail
  q          Quit

Query examples:
  level:error                     Exact field match
  latency>500                     Numeric comparison
  message~"timeout.*"             Regex match
  level:error AND service:auth    Compound query
  NOT service:healthcheck         Negation
  last:5m                         Last 5 minutes
  @err                            Alias for level:error OR level:fatal
  @slow AND service:api           Aliases work in compound queries
  source:app.log AND level:error  Filter by source file (multi-file)
  trace_id:abc-123               Follow a trace across files

Built-in aliases:
  @err     level:error OR level:fatal
  @warn    level:warn OR level:warning
  @slow    latency>1000

Config:
  Place a .logq.toml in your project root to set defaults (theme, columns,
  custom aliases). Run "logq init" to create a starter config file.
`, version)
}

func formatSize(bytes int64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
	)
	switch {
	case bytes >= gb:
		return fmt.Sprintf("%.1fGB", float64(bytes)/float64(gb))
	case bytes >= mb:
		return fmt.Sprintf("%.1fMB", float64(bytes)/float64(mb))
	case bytes >= kb:
		return fmt.Sprintf("%.1fKB", float64(bytes)/float64(kb))
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}
