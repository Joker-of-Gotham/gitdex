package observability

import (
	"context"
	"testing"
	"time"

	"github.com/Joker-of-Gotham/gitdex/internal/contract"
)

func TestTraceContextRoundTrip(t *testing.T) {
	trace := contract.TraceMetadata{TraceID: "trace-test", RoundID: "r1"}
	ctx := WithTrace(context.Background(), trace)
	got, ok := TraceFromContext(ctx)
	if !ok {
		t.Fatal("expected trace in context")
	}
	if got.TraceID != trace.TraceID {
		t.Fatalf("trace mismatch: got %q want %q", got.TraceID, trace.TraceID)
	}
}

func TestMetricsRecording(t *testing.T) {
	RecordLLMCall(10*time.Millisecond, true)
	RecordCommand(true)
	RecordCommand(false)
	RecordReplanAttempt()
	SetProviderAvailability(true)

	snap := SnapshotMetrics()
	if snap.LLMCallsTotal == 0 {
		t.Fatal("expected llm_calls_total > 0")
	}
	if snap.CommandsTotal < 2 {
		t.Fatal("expected commands_total >= 2")
	}
	if snap.ProviderAvailable != 1 {
		t.Fatal("expected provider_available=1")
	}
}

