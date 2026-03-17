package oplog

import "time"

const DefaultMaxEntries = 100

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

// Add appends an entry, evicting the oldest if at capacity.
func (l *Log) Add(e Entry) {
	e = e.Normalized(time.Now())
	l.entries = append(l.entries, e)
	if len(l.entries) > l.maxEntries {
		l.entries = l.entries[len(l.entries)-l.maxEntries:]
	}
}

// Entries returns a copy of all entries.
func (l *Log) Entries() []Entry {
	out := make([]Entry, len(l.entries))
	copy(out, l.entries)
	return out
}

// Len returns the number of entries.
func (l *Log) Len() int {
	return len(l.entries)
}
