package platform

import (
	"context"
	"encoding/json"
)

type AdminInspectRequest struct {
	ResourceID string            `json:"resource_id,omitempty"`
	Scope      map[string]string `json:"scope,omitempty"`
	Query      map[string]string `json:"query,omitempty"`
}

type AdminMutationRequest struct {
	Operation       string            `json:"operation"`
	ResourceID      string            `json:"resource_id,omitempty"`
	Scope           map[string]string `json:"scope,omitempty"`
	Payload         json.RawMessage   `json:"payload,omitempty"`
	RollbackPayload json.RawMessage   `json:"rollback_payload,omitempty"`
}

type AdminSnapshot struct {
	CapabilityID string            `json:"capability_id"`
	ResourceID   string            `json:"resource_id,omitempty"`
	State        json.RawMessage   `json:"state,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	ExecMeta     ExecutionMeta     `json:"exec_meta,omitempty"`
}

type AdminMutationResult struct {
	CapabilityID string            `json:"capability_id"`
	Operation    string            `json:"operation"`
	ResourceID   string            `json:"resource_id,omitempty"`
	Before       *AdminSnapshot    `json:"before,omitempty"`
	After        *AdminSnapshot    `json:"after,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	ExecMeta     ExecutionMeta     `json:"exec_meta,omitempty"`
	LedgerID     string            `json:"ledger_id,omitempty"`
}

type AdminValidationRequest struct {
	ResourceID string               `json:"resource_id,omitempty"`
	Scope      map[string]string    `json:"scope,omitempty"`
	Payload    json.RawMessage      `json:"payload,omitempty"`
	Mutation   *AdminMutationResult `json:"mutation,omitempty"`
}

type AdminValidationResult struct {
	OK         bool              `json:"ok"`
	Summary    string            `json:"summary,omitempty"`
	ResourceID string            `json:"resource_id,omitempty"`
	Snapshot   *AdminSnapshot    `json:"snapshot,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	ExecMeta   ExecutionMeta     `json:"exec_meta,omitempty"`
}

type AdminRollbackRequest struct {
	Scope    map[string]string    `json:"scope,omitempty"`
	Mutation *AdminMutationResult `json:"mutation,omitempty"`
	Payload  json.RawMessage      `json:"payload,omitempty"`
}

type AdminRollbackResult struct {
	OK           bool                `json:"ok"`
	Summary      string              `json:"summary,omitempty"`
	Snapshot     *AdminSnapshot      `json:"snapshot,omitempty"`
	Metadata     map[string]string   `json:"metadata,omitempty"`
	ExecMeta     ExecutionMeta       `json:"exec_meta,omitempty"`
	Compensation *CompensationAction `json:"compensation,omitempty"`
}

type AdminExecutor interface {
	CapabilityID() string
	Inspect(ctx context.Context, req AdminInspectRequest) (*AdminSnapshot, error)
	Mutate(ctx context.Context, req AdminMutationRequest) (*AdminMutationResult, error)
	Validate(ctx context.Context, req AdminValidationRequest) (*AdminValidationResult, error)
	Rollback(ctx context.Context, req AdminRollbackRequest) (*AdminRollbackResult, error)
}

type stubAdminExecutor struct {
	capabilityID string
}

func NewStubAdminExecutor(capabilityID string) AdminExecutor {
	return stubAdminExecutor{capabilityID: capabilityID}
}

func NewStubAdminExecutors(capabilityIDs []string) map[string]AdminExecutor {
	if len(capabilityIDs) == 0 {
		return nil
	}
	out := make(map[string]AdminExecutor, len(capabilityIDs))
	for _, capabilityID := range capabilityIDs {
		if capabilityID == "" {
			continue
		}
		out[capabilityID] = NewStubAdminExecutor(capabilityID)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func (s stubAdminExecutor) CapabilityID() string { return s.capabilityID }

func (s stubAdminExecutor) Inspect(context.Context, AdminInspectRequest) (*AdminSnapshot, error) {
	return nil, nil
}

func (s stubAdminExecutor) Mutate(context.Context, AdminMutationRequest) (*AdminMutationResult, error) {
	return nil, nil
}

func (s stubAdminExecutor) Validate(context.Context, AdminValidationRequest) (*AdminValidationResult, error) {
	return nil, nil
}

func (s stubAdminExecutor) Rollback(context.Context, AdminRollbackRequest) (*AdminRollbackResult, error) {
	return nil, nil
}
