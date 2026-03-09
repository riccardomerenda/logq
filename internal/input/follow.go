package input

import (
	"bytes"
	"io"
	"os"
	"strings"
)

// FollowReader tails a file for new content appended after a given offset.
type FollowReader struct {
	path   string
	offset int64
}

// NewFollowReader creates a reader that watches for new content in a file
// starting from the given byte offset.
func NewFollowReader(path string, offset int64) *FollowReader {
	return &FollowReader{path: path, offset: offset}
}

// ReadNew reads any new complete lines appended since the last read.
// Returns nil if no new data. Partial lines (no trailing newline) are
// left for the next call.
func (fr *FollowReader) ReadNew() ([]string, error) {
	f, err := os.Open(fr.path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}

	if fi.Size() <= fr.offset {
		return nil, nil
	}

	if _, err := f.Seek(fr.offset, io.SeekStart); err != nil {
		return nil, err
	}

	data := make([]byte, fi.Size()-fr.offset)
	n, err := io.ReadFull(f, data)
	if err != nil && err != io.ErrUnexpectedEOF {
		return nil, err
	}
	data = data[:n]

	// Only consume up to the last complete line
	lastNewline := bytes.LastIndexByte(data, '\n')
	if lastNewline < 0 {
		return nil, nil // no complete line yet
	}

	fr.offset += int64(lastNewline + 1)

	content := string(data[:lastNewline])
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, "\r")
	}

	return lines, nil
}
