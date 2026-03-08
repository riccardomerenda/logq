package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/riccardomerenda/logq/internal/index"
	"github.com/riccardomerenda/logq/internal/input"
	"github.com/riccardomerenda/logq/internal/parser"
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

	var reader *input.Reader
	var filename string
	var fileSize string
	var err error

	if len(os.Args) > 1 {
		path := os.Args[1]
		reader, err = input.NewFileReader(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening file: %v\n", err)
			os.Exit(1)
		}
		defer reader.Close()
		filename = path

		if info, e := os.Stat(path); e == nil {
			fileSize = formatSize(info.Size())
		}
	} else if input.IsStdinPipe() {
		reader = input.NewStdinReader()
		filename = "stdin"
	} else {
		printUsage()
		os.Exit(1)
	}

	// Read all lines
	lines, err := reader.ReadAll()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		os.Exit(1)
	}

	if len(lines) == 0 {
		fmt.Fprintln(os.Stderr, "No log lines found in input.")
		os.Exit(1)
	}

	// Parse records
	records := make([]parser.Record, 0, len(lines))
	for i, line := range lines {
		if line == "" {
			continue
		}
		records = append(records, parser.Parse(line, i+1))
	}

	// Build index
	idx := index.Build(records)

	// Start TUI
	model := ui.NewModel(idx, filename, fileSize)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if err := p.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Printf(`logq %s — Fast, interactive log explorer for the terminal

Usage:
  logq <file>          Open a log file
  logq <file.gz>       Open a gzipped log file
  <cmd> | logq         Read from stdin pipe

Options:
  -h, --help           Show this help
  -v, --version        Show version

Keyboard:
  /          Focus filter bar
  j/k, ↑/↓  Scroll logs
  Enter      Show record detail
  Tab        Toggle histogram focus
  Esc        Clear filter / close detail
  q          Quit

Query examples:
  level:error                     Exact field match
  latency>500                     Numeric comparison
  message~"timeout.*"             Regex match
  level:error AND service:auth    Compound query
  NOT service:healthcheck         Negation
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
