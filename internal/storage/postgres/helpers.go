package postgres

import (
	"context"
	"crypto/rand"
	"fmt"
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
