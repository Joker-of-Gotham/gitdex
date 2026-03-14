package platform

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type MutationLedgerEntry struct {
	ID                 string             `json:"id"`
	At                 time.Time          `json:"at"`
	Platform           string             `json:"platform"`
	CapabilityID       string             `json:"capability_id"`
	Flow               string             `json:"flow"`
	Operation          string             `json:"operation,omitempty"`
	ResourceID         string             `json:"resource_id,omitempty"`
	RequestRevision    int                `json:"request_revision,omitempty"`
	ExecMeta           ExecutionMeta      `json:"exec_meta,omitempty"`
	Request            json.RawMessage    `json:"request,omitempty"`
	Before             json.RawMessage    `json:"before,omitempty"`
	After              json.RawMessage    `json:"after,omitempty"`
	Validate           json.RawMessage    `json:"validate,omitempty"`
	Rollback           json.RawMessage    `json:"rollback,omitempty"`
	Metadata           map[string]string  `json:"metadata,omitempty"`
	Summary            string             `json:"summary,omitempty"`
	Failure            FailureTaxonomy    `json:"failure,omitempty"`
	WorkflowStepID     string             `json:"workflow_step_id,omitempty"`
	DiagnosticDecision DiagnosticDecision `json:"diagnostic_decision,omitempty"`
	Diagnostics        []DiagnosticItem   `json:"diagnostics,omitempty"`
}

func NewLedgerID(capabilityID, flow, operation, resourceID string, at time.Time) string {
	return fmt.Sprintf(
		"%s-%s-%s-%s-%d",
		slugValue(capabilityID),
		slugValue(flow),
		slugValue(operation),
		slugValue(resourceID),
		at.UnixNano(),
	)
}

func slugValue(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	if v == "" {
		return "na"
	}
	replacer := strings.NewReplacer(" ", "-", "/", "-", "\\", "-", ":", "-", ".", "-", "@", "-", "#", "-")
	return replacer.Replace(v)
}
