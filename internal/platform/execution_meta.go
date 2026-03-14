package platform

import "strings"

type CoverageMode string

const (
	CoverageFull        CoverageMode = "full"
	CoveragePartial     CoverageMode = "partial_mutate"
	CoverageInspectOnly CoverageMode = "inspect_only"
	CoverageComposed    CoverageMode = "composed"
)

type AdapterKind string

const (
	AdapterAPI     AdapterKind = "api"
	AdapterCLI     AdapterKind = "gh"
	AdapterBrowser AdapterKind = "browser"
)

type RollbackKind string

const (
	RollbackNotSupported RollbackKind = "not_supported"
	RollbackReversible   RollbackKind = "reversible"
	RollbackCompensating RollbackKind = "compensating"
)

type FailureTaxonomy string

const (
	FailureValidation  FailureTaxonomy = "validation_failure"
	FailureExecutor    FailureTaxonomy = "executor_failure"
	FailureAdapter     FailureTaxonomy = "adapter_failure"
	FailureBoundary    FailureTaxonomy = "boundary_violation"
	FailureReversible  FailureTaxonomy = "non_reversible"
	FailureRateLimited FailureTaxonomy = "rate_limited"
	FailureAuthMissing FailureTaxonomy = "auth_missing"
)

type ExecutionMeta struct {
	Coverage         CoverageMode `json:"coverage,omitempty"`
	Adapter          AdapterKind  `json:"adapter,omitempty"`
	Rollback         RollbackKind `json:"rollback,omitempty"`
	SchedulerSafe    bool         `json:"scheduler_safe,omitempty"`
	ApprovalRequired bool         `json:"approval_required,omitempty"`
	BoundaryReason   string       `json:"boundary_reason,omitempty"`
}

type CompensationAction struct {
	Kind        string            `json:"kind,omitempty"`
	Summary     string            `json:"summary,omitempty"`
	Scope       map[string]string `json:"scope,omitempty"`
	Payload     map[string]string `json:"payload,omitempty"`
	OperatorRef string            `json:"operator_ref,omitempty"`
	LedgerChain []string          `json:"ledger_chain,omitempty"`
}

type executionPolicy struct {
	Adapter          AdapterKind
	Rollback         RollbackKind
	SchedulerSafe    bool
	ApprovalRequired bool
}

var platformExecutionPolicies = map[Platform]map[string]executionPolicy{
	PlatformGitHub: {
		"release":                     {Adapter: AdapterAPI, Rollback: RollbackCompensating, SchedulerSafe: false, ApprovalRequired: true},
		"pull_request":                {Adapter: AdapterAPI, Rollback: RollbackCompensating, SchedulerSafe: false, ApprovalRequired: true},
		"pr_review":                   {Adapter: AdapterAPI, Rollback: RollbackReversible, SchedulerSafe: false, ApprovalRequired: true},
		"pages":                       {Adapter: AdapterAPI, Rollback: RollbackCompensating, SchedulerSafe: false, ApprovalRequired: true},
		"deployment":                  {Adapter: AdapterAPI, Rollback: RollbackCompensating, SchedulerSafe: false, ApprovalRequired: true},
		"webhooks":                    {Adapter: AdapterAPI, Rollback: RollbackReversible, SchedulerSafe: false, ApprovalRequired: true},
		"rulesets":                    {Adapter: AdapterAPI, Rollback: RollbackReversible, SchedulerSafe: false, ApprovalRequired: true},
		"branch_rulesets":             {Adapter: AdapterAPI, Rollback: RollbackReversible, SchedulerSafe: false, ApprovalRequired: true},
		"actions":                     {Adapter: AdapterAPI, Rollback: RollbackCompensating, SchedulerSafe: false, ApprovalRequired: true},
		"actions_secrets_variables":   {Adapter: AdapterAPI, Rollback: RollbackCompensating, SchedulerSafe: false, ApprovalRequired: true},
		"codespaces":                  {Adapter: AdapterAPI, Rollback: RollbackCompensating, SchedulerSafe: false, ApprovalRequired: true},
		"codespaces_secrets":          {Adapter: AdapterAPI, Rollback: RollbackCompensating, SchedulerSafe: false, ApprovalRequired: true},
		"dependabot_secrets":          {Adapter: AdapterAPI, Rollback: RollbackCompensating, SchedulerSafe: false, ApprovalRequired: true},
		"dependabot_config":           {Adapter: AdapterAPI, Rollback: RollbackReversible, SchedulerSafe: false, ApprovalRequired: true},
		"notifications":               {Adapter: AdapterAPI, Rollback: RollbackCompensating, SchedulerSafe: false, ApprovalRequired: true},
		"email_notifications":         {Adapter: AdapterAPI, Rollback: RollbackCompensating, SchedulerSafe: false, ApprovalRequired: true},
		"packages":                    {Adapter: AdapterAPI, Rollback: RollbackCompensating, SchedulerSafe: false, ApprovalRequired: true},
		"advanced_security":           {Adapter: AdapterAPI, Rollback: RollbackCompensating, SchedulerSafe: false, ApprovalRequired: true},
		"dependabot_posture":          {Adapter: AdapterAPI, Rollback: RollbackCompensating, SchedulerSafe: false, ApprovalRequired: true},
		"secret_scanning_settings":    {Adapter: AdapterAPI, Rollback: RollbackCompensating, SchedulerSafe: false, ApprovalRequired: true},
		"secret_scanning_alerts":      {Adapter: AdapterAPI, Rollback: RollbackCompensating, SchedulerSafe: false, ApprovalRequired: true},
		"code_scanning_default_setup": {Adapter: AdapterAPI, Rollback: RollbackCompensating, SchedulerSafe: false, ApprovalRequired: true},
		"codeql_setup":                {Adapter: AdapterAPI, Rollback: RollbackCompensating, SchedulerSafe: false, ApprovalRequired: true},
		"security":                    {Adapter: AdapterAPI, Rollback: RollbackCompensating, SchedulerSafe: false, ApprovalRequired: true},
		"copilot_code_review":         {Adapter: AdapterAPI, Rollback: RollbackCompensating, SchedulerSafe: false, ApprovalRequired: true},
		"copilot_coding_agent":        {Adapter: AdapterAPI, Rollback: RollbackCompensating, SchedulerSafe: false, ApprovalRequired: true},
		"copilot_seat_management":     {Adapter: AdapterAPI, Rollback: RollbackCompensating, SchedulerSafe: false, ApprovalRequired: true},
	},
	PlatformGitLab: {
		"merge_requests": {Adapter: AdapterAPI, Rollback: RollbackCompensating, SchedulerSafe: false, ApprovalRequired: true},
		"pipelines":      {Adapter: AdapterAPI, Rollback: RollbackCompensating, SchedulerSafe: false, ApprovalRequired: true},
		"environments":   {Adapter: AdapterAPI, Rollback: RollbackCompensating, SchedulerSafe: false, ApprovalRequired: true},
		"pages":          {Adapter: AdapterAPI, Rollback: RollbackCompensating, SchedulerSafe: false, ApprovalRequired: true},
		"security":       {Adapter: AdapterAPI, Rollback: RollbackNotSupported, SchedulerSafe: true, ApprovalRequired: false},
	},
	PlatformBitbucket: {
		"pull_requests":         {Adapter: AdapterAPI, Rollback: RollbackCompensating, SchedulerSafe: false, ApprovalRequired: true},
		"pipelines":             {Adapter: AdapterAPI, Rollback: RollbackCompensating, SchedulerSafe: false, ApprovalRequired: true},
		"deployments":           {Adapter: AdapterAPI, Rollback: RollbackCompensating, SchedulerSafe: false, ApprovalRequired: true},
		"branch_restrictions":   {Adapter: AdapterAPI, Rollback: RollbackReversible, SchedulerSafe: false, ApprovalRequired: true},
		"webhooks":              {Adapter: AdapterAPI, Rollback: RollbackReversible, SchedulerSafe: false, ApprovalRequired: true},
		"repository_variables":  {Adapter: AdapterAPI, Rollback: RollbackCompensating, SchedulerSafe: false, ApprovalRequired: true},
	},
}

func ExecutionMetaFor(p Platform, capabilityID, flow, operation string) ExecutionMeta {
	capabilityID = strings.TrimSpace(capabilityID)
	flow = strings.ToLower(strings.TrimSpace(flow))
	operation = strings.ToLower(strings.TrimSpace(operation))

	meta := ExecutionMeta{
		Coverage:         CoverageFull,
		Adapter:          AdapterAPI,
		Rollback:         RollbackReversible,
		SchedulerSafe:    flow == "inspect" || flow == "validate",
		ApprovalRequired: flow == "mutate" || flow == "rollback",
	}

	if boundary, ok := CapabilityBoundaryFor(p, capabilityID); ok {
		meta.Coverage = CoverageMode(boundary.Mode)
		meta.BoundaryReason = boundary.Reason
		switch meta.Coverage {
		case CoverageInspectOnly:
			meta.Rollback = RollbackNotSupported
			meta.SchedulerSafe = flow != "mutate" && flow != "rollback"
			meta.ApprovalRequired = false
		case CoverageComposed, CoveragePartial:
			if flow == "mutate" || flow == "rollback" {
				meta.Rollback = RollbackCompensating
			}
		}
	}

	if policies := platformExecutionPolicies[p]; len(policies) > 0 {
		if policy, ok := policies[capabilityID]; ok {
			meta.Adapter = policy.Adapter
			if flow == "mutate" || flow == "rollback" {
				meta.Rollback = policy.Rollback
				meta.SchedulerSafe = policy.SchedulerSafe
				meta.ApprovalRequired = policy.ApprovalRequired
			}
		}
	}

	switch flow {
	case "inspect", "validate":
		meta.Rollback = RollbackNotSupported
		meta.SchedulerSafe = true
		meta.ApprovalRequired = false
	case "rollback":
		meta.SchedulerSafe = false
		meta.ApprovalRequired = true
	}

	if capabilityID == "actions" {
		switch operation {
		case "dispatch", "rerun", "cancel_run":
			meta.Rollback = RollbackCompensating
			meta.SchedulerSafe = false
			meta.ApprovalRequired = true
		}
	}

	return meta
}
