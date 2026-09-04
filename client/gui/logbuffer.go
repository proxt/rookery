package main

import (
	"strings"
	"sync"
)

// logRingSize caps how many formatted log lines are kept for the GUI's Logs
// view — enough for a useful troubleshooting window without holding an
// unbounded amount of text in memory for a long-running background app.
const logRingSize = 500

// logRing is an io.Writer that keeps only the last logRingSize lines
// written to it, safe for concurrent use — slog writes to it from every
// goroutine in the process. It's the backing store for App.GetLogs.
type logRing struct {
	mu    sync.Mutex
	lines []string
}

func newLogRing() *logRing {
	return &logRing{}
}

func (r *logRing) Write(p []byte) (int, error) {
	line := strings.TrimRight(string(p), "\n")
	if line != "" {
		r.mu.Lock()
		r.lines = append(r.lines, line)
		if len(r.lines) > logRingSize {
			r.lines = r.lines[len(r.lines)-logRingSize:]
		}
		r.mu.Unlock()
	}
	return len(p), nil
}

// Lines returns a snapshot of the buffered lines, oldest first.
func (r *logRing) Lines() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, len(r.lines))
	copy(out, r.lines)
	return out
}
