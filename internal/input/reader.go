package input

import (
	"bufio"
	"compress/gzip"
	"io"
	"os"
)

// Reader reads log lines from a source.
type Reader struct {
	scanner *bufio.Scanner
	closer  io.Closer
}

// NewFileReader opens a file for reading. Detects gzip by magic bytes.
func NewFileReader(path string) (*Reader, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	// Check for gzip magic bytes
	buf := make([]byte, 2)
	n, _ := f.Read(buf)
	// Seek back to start
	if _, err := f.Seek(0, 0); err != nil {
		f.Close()
		return nil, err
	}

	var r io.Reader = f
	var closer io.Closer = f

	if n == 2 && buf[0] == 0x1f && buf[1] == 0x8b {
		gz, err := gzip.NewReader(f)
		if err != nil {
			f.Close()
			return nil, err
		}
		r = gz
		closer = &multiCloser{closers: []io.Closer{gz, f}}
	}

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024) // up to 10MB per line

	return &Reader{scanner: scanner, closer: closer}, nil
}

// NewStdinReader reads from stdin.
func NewStdinReader() *Reader {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)
	return &Reader{scanner: scanner}
}

// ReadAll reads all lines and returns them.
func (r *Reader) ReadAll() ([]string, error) {
	var lines []string
	for r.scanner.Scan() {
		lines = append(lines, r.scanner.Text())
	}
	return lines, r.scanner.Err()
}

// Close releases resources.
func (r *Reader) Close() error {
	if r.closer != nil {
		return r.closer.Close()
	}
	return nil
}

// IsStdinPipe returns true if stdin is a pipe (not a terminal).
func IsStdinPipe() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) == 0
}

type multiCloser struct {
	closers []io.Closer
}

func (mc *multiCloser) Close() error {
	var firstErr error
	for _, c := range mc.closers {
		if err := c.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
