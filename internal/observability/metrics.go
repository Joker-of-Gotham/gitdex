package observability

import (
	"sync/atomic"
	"time"
)

type MetricsSnapshot struct {
	LLMCallsTotal      uint64 `json:"llm_calls_total"`
	LLMCallsFailed     uint64 `json:"llm_calls_failed"`
	LLMLatencyMsTotal  uint64 `json:"llm_latency_ms_total"`
	CommandsTotal      uint64 `json:"commands_total"`
	CommandsSucceeded  uint64 `json:"commands_succeeded"`
	CommandsFailed     uint64 `json:"commands_failed"`
	ReplanAttempts     uint64 `json:"replan_attempts"`
	ProviderAvailable  uint64 `json:"provider_available"`
}

var metrics struct {
	llmCallsTotal     atomic.Uint64
	llmCallsFailed    atomic.Uint64
	llmLatencyMsTotal atomic.Uint64
	commandsTotal     atomic.Uint64
	commandsSucceeded atomic.Uint64
	commandsFailed    atomic.Uint64
	replanAttempts    atomic.Uint64
	providerAvailable atomic.Uint64
}

func RecordLLMCall(duration time.Duration, success bool) {
	metrics.llmCallsTotal.Add(1)
	if !success {
		metrics.llmCallsFailed.Add(1)
	}
	metrics.llmLatencyMsTotal.Add(uint64(duration.Milliseconds()))
}

func RecordCommand(success bool) {
	metrics.commandsTotal.Add(1)
	if success {
		metrics.commandsSucceeded.Add(1)
	} else {
		metrics.commandsFailed.Add(1)
	}
}

func RecordReplanAttempt() {
	metrics.replanAttempts.Add(1)
}

func SetProviderAvailability(available bool) {
	if available {
		metrics.providerAvailable.Store(1)
	} else {
		metrics.providerAvailable.Store(0)
	}
}

func SnapshotMetrics() MetricsSnapshot {
	return MetricsSnapshot{
		LLMCallsTotal:      metrics.llmCallsTotal.Load(),
		LLMCallsFailed:     metrics.llmCallsFailed.Load(),
		LLMLatencyMsTotal:  metrics.llmLatencyMsTotal.Load(),
		CommandsTotal:      metrics.commandsTotal.Load(),
		CommandsSucceeded:  metrics.commandsSucceeded.Load(),
		CommandsFailed:     metrics.commandsFailed.Load(),
		ReplanAttempts:     metrics.replanAttempts.Load(),
		ProviderAvailable:  metrics.providerAvailable.Load(),
	}
}

