package platform

import "strings"

type ValidationPolicy struct {
	Strategy       string   `json:"strategy,omitempty"`
	AllowNoOp      bool     `json:"allow_no_op,omitempty"`
	RequiresTarget bool     `json:"requires_target,omitempty"`
	ExternalChecks []string `json:"external_checks,omitempty"`
}

type RollbackPolicy struct {
	Kind                   RollbackKind `json:"kind,omitempty"`
	RequiresBeforeSnapshot bool         `json:"requires_before_snapshot,omitempty"`
	Note                   string       `json:"note,omitempty"`
}

type CompensationPolicy struct {
	Required         bool   `json:"required,omitempty"`
	Kind             string `json:"kind,omitempty"`
	OperatorRequired bool   `json:"operator_required,omitempty"`
	Note             string `json:"note,omitempty"`
}

type ExecutionPolicies struct {
	Validation   ValidationPolicy   `json:"validation,omitempty"`
	Rollback     RollbackPolicy     `json:"rollback,omitempty"`
	Compensation CompensationPolicy `json:"compensation,omitempty"`
}

func PoliciesFor(p Platform, capabilityID, flow, operation string) ExecutionPolicies {
	meta := ExecutionMetaFor(p, capabilityID, flow, operation)
	policies := ExecutionPolicies{
		Validation: ValidationPolicy{
			Strategy:       "subset_match",
			AllowNoOp:      true,
			RequiresTarget: strings.TrimSpace(flow) != "inspect",
		},
		Rollback: RollbackPolicy{
			Kind:                   meta.Rollback,
			RequiresBeforeSnapshot: meta.Rollback != RollbackNotSupported,
		},
	}
	switch strings.TrimSpace(capabilityID) {
	case "pages":
		policies.Validation.ExternalChecks = []string{"dns", "https_certificate", "health"}
	case "release":
		policies.Validation.ExternalChecks = []string{"published_state", "asset_recoverability"}
	case "dependabot_config":
		policies.Validation.ExternalChecks = []string{"schema", "deterministic_reencode", "no_op_diff"}
	case "packages":
		policies.Validation.ExternalChecks = []string{"registry_metadata"}
	}
	switch meta.Rollback {
	case RollbackCompensating:
		policies.Compensation = CompensationPolicy{
			Required:         true,
			Kind:             "compensating",
			OperatorRequired: true,
			Note:             strings.TrimSpace(meta.BoundaryReason),
		}
	case RollbackNotSupported:
		policies.Compensation = CompensationPolicy{
			Required:         true,
			Kind:             "manual_restore_required",
			OperatorRequired: true,
			Note:             firstPolicyNote(meta.BoundaryReason, "manual restore required"),
		}
	}
	if strings.EqualFold(strings.TrimSpace(flow), "inspect") || strings.EqualFold(strings.TrimSpace(flow), "validate") {
		policies.Rollback.Kind = RollbackNotSupported
		policies.Rollback.RequiresBeforeSnapshot = false
		policies.Rollback.Note = "inspect and validate flows do not mutate state"
	}
	if policies.Rollback.Note == "" {
		switch policies.Rollback.Kind {
		case RollbackReversible:
			policies.Rollback.Note = "previous state snapshot can be replayed automatically"
		case RollbackCompensating:
			policies.Rollback.Note = "rollback may require a compensating mutation instead of exact state replay"
		case RollbackNotSupported:
			policies.Rollback.Note = "manual restore required"
		}
	}
	return policies
}

func firstPolicyNote(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
