package api

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/your-org/gitdex/internal/audit"
	"github.com/your-org/gitdex/internal/orchestrator"
	"github.com/your-org/gitdex/internal/planning"
	"github.com/your-org/gitdex/internal/storage"
)

type submitResponseEnvelope struct {
	ID        string          `json:"id"`
	Intent    json.RawMessage `json:"intent"`
	Plan      json.RawMessage `json:"plan"`
	Task      json.RawMessage `json:"task"`
	Accepted  bool            `json:"accepted"`
	CreatedAt string          `json:"created_at"`
}

// PersistSubmitResponse writes successful API submit payloads to the configured storage.
// The router may still use MemoryAPIRouter for request handling; persistence is applied here when a provider exists.
func PersistSubmitResponse(provider storage.StorageProvider, requestPath string, resp *APIResponse) error {
	if provider == nil || resp == nil || resp.StatusCode != 201 || len(resp.Payload) == 0 {
		return nil
	}
	path := normalizePath(requestPath)
	var env submitResponseEnvelope
	if err := json.Unmarshal(resp.Payload, &env); err != nil {
		return fmt.Errorf("parse submit response: %w", err)
	}
	if env.ID == "" {
		return fmt.Errorf("submit response missing id")
	}

	switch path {
	case "/api/v1/intents":
		return persistIntent(provider, env)
	case "/api/v1/plans":
		return persistPlan(provider, env)
	case "/api/v1/tasks":
		return persistTask(provider, env)
	default:
		return nil
	}
}

func persistIntent(provider storage.StorageProvider, env submitResponseEnvelope) error {
	ledger := provider.AuditLedger()
	if ledger == nil {
		return fmt.Errorf("audit ledger is required to persist intents")
	}
	payload := string(env.Intent)
	if payload == "" {
		payload = "{}"
	}
	const maxSnap = 4096
	if len(payload) > maxSnap {
		payload = payload[:maxSnap] + "…"
	}
	return ledger.Append(&audit.AuditEntry{
		EntryID:      audit.GenerateEntryID(),
		EventType:    audit.EventPolicyEvaluated,
		Actor:        "api",
		Action:       "intent_submitted",
		Target:       env.ID,
		PolicyResult: payload,
		Timestamp:    time.Now().UTC(),
	})
}

func persistPlan(provider storage.StorageProvider, env submitResponseEnvelope) error {
	store := provider.PlanStore()
	if store == nil {
		return fmt.Errorf("plan store is required")
	}
	var plan planning.Plan
	if len(env.Plan) > 0 && string(env.Plan) != "null" {
		_ = json.Unmarshal(env.Plan, &plan)
	}
	if plan.PlanID == "" {
		plan.PlanID = env.ID
	}
	if plan.Status == "" {
		plan.Status = planning.PlanDraft
	}
	if plan.CreatedAt.IsZero() {
		if t, err := time.Parse(time.RFC3339, env.CreatedAt); err == nil {
			plan.CreatedAt = t.UTC()
		} else {
			plan.CreatedAt = time.Now().UTC()
		}
	}
	if plan.UpdatedAt.IsZero() {
		plan.UpdatedAt = plan.CreatedAt
	}
	if plan.Intent.RawInput == "" && len(env.Plan) > 0 {
		plan.Intent.RawInput = string(env.Plan)
	}
	if plan.Intent.Source == "" {
		plan.Intent.Source = "api"
	}
	return store.Save(&plan)
}

func persistTask(provider storage.StorageProvider, env submitResponseEnvelope) error {
	store := provider.TaskStore()
	if store == nil {
		return fmt.Errorf("task store is required")
	}
	var task orchestrator.Task
	if len(env.Task) > 0 && string(env.Task) != "null" {
		_ = json.Unmarshal(env.Task, &task)
	}
	if task.TaskID == "" {
		task.TaskID = env.ID
	}
	if task.Status == "" {
		task.Status = orchestrator.TaskQueued
	}
	if task.UpdatedAt.IsZero() {
		task.UpdatedAt = time.Now().UTC()
	}
	if task.CorrelationID == "" {
		task.CorrelationID = orchestrator.GenerateCorrelationID()
	}
	return store.SaveTask(&task)
}
