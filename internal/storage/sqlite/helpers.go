package sqlite

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"time"
)

// ctx returns a background context for store operations that don't receive context.
func ctx() context.Context {
	return context.Background()
}

// orNilSlice returns nil for empty slices so JSON marshals to null.
func orNilSlice[S ~[]E, E any](s S) S {
	if len(s) == 0 {
		return nil
	}
	return s
}

// orNilMap returns nil for empty maps.
func orNilMap[M ~map[K]V, K comparable, V any](m M) M {
	if len(m) == 0 {
		return nil
	}
	return m
}

func generateShortID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}

// formatTime formats time for SQLite TEXT storage (ISO 8601).
func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

// parseTime parses ISO 8601 text from SQLite into time.Time.
func parseTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339, s)
}

// scanTime scans a nullable TEXT column into *time.Time.
func scanTime(s sql.NullString) (*time.Time, error) {
	if !s.Valid || s.String == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, s.String)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// boolToInt converts bool to SQLite INTEGER (0/1).
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// intToBool converts SQLite INTEGER to bool.
func intToBool(i int) bool {
	return i != 0
}
