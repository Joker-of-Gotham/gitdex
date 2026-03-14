package oplog

import (
	"fmt"
	"strings"
	"time"

	"github.com/mattn/go-runewidth"
)

const (
	// DefaultMaxEntries keeps the timeline bounded in memory.
	DefaultMaxEntries = 100
)

// Log stores operation timeline entries in insertion order.
type Log struct {
	maxEntries int
	entries    []Entry
}

// New creates a bounded operation log.
func New(maxEntries int) *Log {
	if maxEntries <= 0 {
		maxEntries = DefaultMaxEntries
	}
	return &Log{maxEntries: maxEntries}
}

// Add appends an entry and evicts oldest items if needed.
func (l *Log) Add(entry Entry) {
	if l == nil {
		return
	}
	e := entry.Normalized(time.Now())
	if e.Summary == "" && e.Detail == "" {
		return
	}
	l.entries = append(l.entries, e)
	if len(l.entries) <= l.maxEntries {
		return
	}
	overflow := len(l.entries) - l.maxEntries
	if overflow <= 0 {
		return
	}
	l.entries = append([]Entry(nil), l.entries[overflow:]...)
}

// Entries returns a safe copy of all entries.
func (l *Log) Entries() []Entry {
	if l == nil || len(l.entries) == 0 {
		return nil
	}
	out := make([]Entry, len(l.entries))
	copy(out, l.entries)
	return out
}

// Latest returns the newest n entries.
func (l *Log) Latest(n int) []Entry {
	if l == nil || len(l.entries) == 0 || n <= 0 {
		return nil
	}
	if n > len(l.entries) {
		n = len(l.entries)
	}
	start := len(l.entries) - n
	out := make([]Entry, n)
	copy(out, l.entries[start:])
	return out
}

// View renders up to `height` lines, defaulting to the latest entries.
func (l *Log) View(width, height int) string {
	return l.ViewWithOffset(width, height, 0)
}

// ViewWithOffset renders timeline lines with upward scroll offset.
func (l *Log) ViewWithOffset(width, height, offset int) string {
	if height <= 0 {
		return ""
	}
	lines := l.lines(width)
	if len(lines) == 0 {
		lines = []string{"(no operations yet)"}
	}
	if offset < 0 {
		offset = 0
	}
	if offset > len(lines)-1 {
		offset = len(lines) - 1
	}
	end := len(lines) - offset
	if end < 0 {
		end = 0
	}
	start := end - height
	if start < 0 {
		start = 0
	}
	visible := lines[start:end]
	if len(visible) == 0 {
		visible = []string{"(no operations yet)"}
	}
	return strings.Join(wrapLines(visible, width), "\n")
}

func (l *Log) lines(width int) []string {
	if l == nil || len(l.entries) == 0 {
		return nil
	}
	var lines []string
	for _, e := range l.entries {
		ts := e.Timestamp.Format("15:04:05")
		summary := strings.TrimSpace(e.Summary)
		if summary == "" {
			summary = strings.TrimSpace(e.Detail)
		}
		if summary == "" {
			summary = "(empty event)"
		}
		lines = append(lines, fmt.Sprintf("%s %s %s", ts, e.Icon(), summary))
		if e.Detail != "" && e.Detail != summary {
			lines = append(lines, fmt.Sprintf("         %s", strings.TrimSpace(e.Detail)))
		}
	}
	return wrapLines(lines, width)
}

func wrapLines(lines []string, width int) []string {
	if width <= 0 {
		return lines
	}
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			out = append(out, "")
			continue
		}
		wrapped := runewidth.Wrap(line, width)
		out = append(out, strings.Split(wrapped, "\n")...)
	}
	return out
}
