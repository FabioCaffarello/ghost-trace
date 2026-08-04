package web

import (
	"encoding/json"
	"os"
	"sync"
)

// FileCaptureSink appends labelled human sessions to a JSONL file —
// the capture instrument for the M2 study, isolated here so the
// handler owns the decision to record and this adapter owns the I/O.
type FileCaptureSink struct {
	path string

	// mu serializes appends. Volunteers overlap in practice — a link
	// goes out to a group chat and several people open it at once —
	// and interleaved writes would corrupt the JSONL.
	mu sync.Mutex
}

// NewFileCaptureSink records rows at path, creating it on first append.
func NewFileCaptureSink(path string) *FileCaptureSink {
	return &FileCaptureSink{path: path}
}

// Append writes one row. The error is returned rather than logged so
// the caller decides what a lost row means (the demo logs and
// swallows: a volunteer's five minutes must not be wasted on a full
// disk).
func (s *FileCaptureSink) Append(row CaptureRow) error {
	line, err := json.Marshal(row)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := os.OpenFile(s.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	_, err = f.Write(append(line, '\n'))
	return err
}

// NoCapture discards rows — the null object that replaces the old
// "empty path disables capture" convention, so the handler never
// branches on whether a study is running.
type NoCapture struct{}

// Append drops the row.
func (NoCapture) Append(CaptureRow) error { return nil }
