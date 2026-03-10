package oplog

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLog_AddRespectsCapacity(t *testing.T) {
	l := New(2)
	l.Add(Entry{Timestamp: time.Unix(1, 0), Summary: "first"})
	l.Add(Entry{Timestamp: time.Unix(2, 0), Summary: "second"})
	l.Add(Entry{Timestamp: time.Unix(3, 0), Summary: "third"})

	got := l.Entries()
	if assert.Len(t, got, 2) {
		assert.Equal(t, "second", got[0].Summary)
		assert.Equal(t, "third", got[1].Summary)
	}
}

func TestLog_ViewWithOffset(t *testing.T) {
	l := New(10)
	l.Add(Entry{Timestamp: time.Unix(1, 0), Summary: "one"})
	l.Add(Entry{Timestamp: time.Unix(2, 0), Summary: "two"})
	l.Add(Entry{Timestamp: time.Unix(3, 0), Summary: "three"})

	latest := l.ViewWithOffset(80, 2, 0)
	assert.Contains(t, latest, "two")
	assert.Contains(t, latest, "three")
	assert.NotContains(t, latest, "one")

	older := l.ViewWithOffset(80, 2, 1)
	assert.Contains(t, older, "one")
	assert.Contains(t, older, "two")
	assert.False(t, strings.Contains(older, "three"))
}
