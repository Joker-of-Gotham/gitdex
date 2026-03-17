package observability

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	"github.com/Joker-of-Gotham/gitdex/internal/contract"
)

type traceContextKey struct{}

func WithTrace(ctx context.Context, trace contract.TraceMetadata) context.Context {
	return context.WithValue(ctx, traceContextKey{}, trace)
}

func TraceFromContext(ctx context.Context) (contract.TraceMetadata, bool) {
	if ctx == nil {
		return contract.TraceMetadata{}, false
	}
	v := ctx.Value(traceContextKey{})
	if v == nil {
		return contract.TraceMetadata{}, false
	}
	trace, ok := v.(contract.TraceMetadata)
	return trace, ok
}

func NewTraceID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "trace-unknown"
	}
	return "trace-" + hex.EncodeToString(b[:])
}

