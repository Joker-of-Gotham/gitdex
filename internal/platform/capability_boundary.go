package platform

import "strings"

type CapabilityBoundary struct {
	CapabilityID string   `json:"capability_id"`
	Mode         string   `json:"mode"` // partial_mutate | inspect_only | composed
	Reason       string   `json:"reason"`
	Supported    []string `json:"supported,omitempty"`
	Missing      []string `json:"missing,omitempty"`
}

var capabilityBoundaryCatalog = map[Platform]map[string]CapabilityBoundary{
	PlatformGitHub: {
		"ai_assistant_deployment": {
			CapabilityID: "ai_assistant_deployment",
			Mode:         "composed",
			Reason:       "AI assistant deployment spans multiple GitHub surfaces rather than a single REST resource.",
			Supported:    []string{"workflow orchestration", "deployment", "environments", "actions secrets and variables", "webhooks", "pages"},
			Missing:      []string{"single-surface CRUD executor"},
		},
		"release": {
			CapabilityID: "release",
			Mode:         "partial_mutate",
			Reason:       "Release assets are now uploadable and deletable, but restore still depends on recoverable source bytes or a downloadable prior asset.",
			Supported:    []string{"inspect releases", "create/update/delete release", "generate notes", "publish draft", "asset upload/list/delete", "conditional asset restore"},
			Missing:      []string{"guaranteed restore when no local/downloadable asset source exists"},
		},
		"pull_request": {
			CapabilityID: "pull_request",
			Mode:         "partial_mutate",
			Reason:       "Pull request lifecycle is API-backed, but merge rollback and some auto-merge transitions are not fully reversible.",
			Supported:    []string{"inspect pull requests", "create/update/close/reopen", "merge", "enable/disable auto-merge"},
			Missing:      []string{"automatic unmerge rollback"},
		},
		"pages": {
			CapabilityID: "pages",
			Mode:         "partial_mutate",
			Reason:       "GitHub-managed Pages config is API-backed, but custom domain DNS and certificate readiness still depend on external DNS state and limited GitHub health feedback.",
			Supported:    []string{"inspect config/build history/latest build/health", "create/update/delete pages config", "trigger/retrigger build", "external DNS validation"},
			Missing:      []string{"full registrar/DNS/certificate lifecycle automation"},
		},
		"dependency_graph": {
			CapabilityID: "dependency_graph",
			Mode:         "inspect_only",
			Reason:       "GitHub exposes SBOM/dependency graph export, but not repository-level dependency-graph mutation through REST.",
			Supported:    []string{"inspect sbom export"},
			Missing:      []string{"repository-level mutate/rollback API"},
		},
		"grouped_security_updates": {
			CapabilityID: "grouped_security_updates",
			Mode:         "partial_mutate",
			Reason:       "Detailed grouping policy is primarily expressed through dependabot.yml rather than a dedicated repository settings surface.",
			Supported:    []string{"repository-level security posture toggles", "dependabot_config executor"},
			Missing:      []string{"standalone grouped-updates CRUD surface"},
		},
		"dependabot_version_updates": {
			CapabilityID: "dependabot_version_updates",
			Mode:         "partial_mutate",
			Reason:       "Version-update cadence and ecosystem policy are primarily managed through dependabot.yml.",
			Supported:    []string{"repository-level posture toggle", "dependabot_config executor"},
			Missing:      []string{"standalone cadence/policy CRUD surface"},
		},
		"notifications": {
			CapabilityID: "notifications",
			Mode:         "partial_mutate",
			Reason:       "Repository/thread subscription is API-backed, but broader account-level notification preferences are outside repository admin REST flows.",
			Supported:    []string{"repo subscription", "thread subscription", "repo inbox", "global inbox", "participating inbox", "mark read"},
			Missing:      []string{"account-level notification preference CRUD"},
		},
		"email_notifications": {
			CapabilityID: "email_notifications",
			Mode:         "partial_mutate",
			Reason:       "GitHub exposes subscription state, not the full account email-routing preferences as repository mutations.",
			Supported:    []string{"repo/thread subscription inspection and mutation"},
			Missing:      []string{"account-level email notification preference CRUD"},
		},
		"packages": {
			CapabilityID: "packages",
			Mode:         "partial_mutate",
			Reason:       "Package cleanup is supported, but richer package administration remains registry- and package-type-specific.",
			Supported:    []string{"inspect packages/versions/latest version", "delete/restore package or version"},
			Missing:      []string{"package settings/visibility lifecycle across all registry types"},
		},
		"advanced_security": {
			CapabilityID: "advanced_security",
			Mode:         "partial_mutate",
			Reason:       "The executor covers public repository security surfaces, but GitHub does not expose every UI subsetting as CRUD REST.",
			Supported:    []string{"security_and_analysis posture", "summary", "configuration", "automated security fixes"},
			Missing:      []string{"all UI-only subsettings"},
		},
		"dependabot_posture": {
			CapabilityID: "dependabot_posture",
			Mode:         "partial_mutate",
			Reason:       "GitHub exposes automated security fixes and posture inspection, but detailed version-update policy still lives in dependabot.yml.",
			Supported:    []string{"inspect automated security fixes", "inspect repo posture summary", "enable/disable automated security fixes"},
			Missing:      []string{"full ecosystem cadence and grouping CRUD outside dependabot.yml"},
		},
		"secret_scanning_settings": {
			CapabilityID: "secret_scanning_settings",
			Mode:         "partial_mutate",
			Reason:       "Repository-level secret scanning settings are public API-backed, but downstream provider coverage and bypass handling remain separate surfaces.",
			Supported:    []string{"inspect secret scanning posture", "enable/disable secret scanning"},
			Missing:      []string{"provider-specific bypass governance parity with every UI pane"},
		},
		"secret_scanning_alerts": {
			CapabilityID: "secret_scanning_alerts",
			Mode:         "partial_mutate",
			Reason:       "Secret scanning alerts are triageable through REST, but the wider secret-protection experience spans push protection and organization policy.",
			Supported:    []string{"inspect alerts", "inspect locations", "resolve/reopen alerts"},
			Missing:      []string{"full org-wide protection policy parity"},
		},
		"code_scanning_tool_settings": {
			CapabilityID: "code_scanning_tool_settings",
			Mode:         "inspect_only",
			Reason:       "GitHub exposes code security configuration inspection, but not a standalone public CRUD surface for every tool-level setting.",
			Supported:    []string{"inspect code security configuration"},
			Missing:      []string{"standalone mutate/rollback API"},
		},
		"code_scanning_default_setup": {
			CapabilityID: "code_scanning_default_setup",
			Mode:         "partial_mutate",
			Reason:       "Default setup is API-backed, but analysis history cleanup and tool-specific workflows are only partially reversible.",
			Supported:    []string{"inspect default setup", "update default setup", "validate/rollback default setup"},
			Missing:      []string{"full parity with every code scanning workflow customization path"},
		},
		"codeql_setup": {
			CapabilityID: "codeql_setup",
			Mode:         "partial_mutate",
			Reason:       "CodeQL default setup is API-backed, but full workflow-level customization still extends beyond the default-setup endpoint.",
			Supported:    []string{"inspect CodeQL default setup", "update default setup", "inspect/delete analyses"},
			Missing:      []string{"full custom workflow CRUD parity"},
		},
		"security": {
			CapabilityID: "security",
			Mode:         "partial_mutate",
			Reason:       "This is an aggregate executor over public security_and_analysis surfaces, not a one-to-one mirror of every security screen.",
			Supported:    []string{"inspect summary", "apply aggregate security_and_analysis payload"},
			Missing:      []string{"non-public UI-only surfaces"},
		},
		"copilot_code_review": {
			CapabilityID: "copilot_code_review",
			Mode:         "partial_mutate",
			Reason:       "GitHub exposes a subset of Copilot org-admin surfaces via REST and many are org-scoped or plan-gated.",
			Supported:    []string{"inspect billing/seats/metrics/content exclusions", "update/delete content exclusions"},
			Missing:      []string{"all feature toggles across every Copilot UI pane"},
		},
		"copilot_coding_agent": {
			CapabilityID: "copilot_coding_agent",
			Mode:         "partial_mutate",
			Reason:       "Coding-agent administration is narrower than generic Copilot UI and remains plan- and org-scope dependent.",
			Supported:    []string{"inspect billing/seats/metrics/content exclusions", "update/delete content exclusions"},
			Missing:      []string{"full agent feature-admin parity with every UI pane"},
		},
		"copilot_seat_management": {
			CapabilityID: "copilot_seat_management",
			Mode:         "partial_mutate",
			Reason:       "Seat management is public API-backed, but broader Copilot org policy remains split across other endpoints and plan gating.",
			Supported:    []string{"inspect seats/billing", "add/remove users", "add/remove teams"},
			Missing:      []string{"broader org policy parity across all Copilot admin panes"},
		},
		"copilot_autofix": {
			CapabilityID: "copilot_autofix",
			Mode:         "inspect_only",
			Reason:       "Autofix suggestions are inspectable through code scanning surfaces, but not fully writable as independent resources.",
			Supported:    []string{"inspect autofix suggestions and commits"},
			Missing:      []string{"standalone mutate/rollback API"},
		},
		"check_runs_failure_threshold": {
			CapabilityID: "check_runs_failure_threshold",
			Mode:         "partial_mutate",
			Reason:       "GitHub models this through rulesets/required checks rather than a dedicated standalone endpoint.",
			Supported:    []string{"ruleset-backed inspect/mutate"},
			Missing:      []string{"standalone threshold CRUD surface"},
		},
		"actions": {
			CapabilityID: "actions",
			Mode:         "partial_mutate",
			Reason:       "Repository-level workflows, runs, and permissions are API-backed, but not every Actions UI operation is reversible.",
			Supported:    []string{"inspect workflows/runs/permissions", "update permissions", "enable/disable workflow", "dispatch", "rerun", "cancel"},
			Missing:      []string{"full rollback for dispatch and run control"},
		},
		"codespaces": {
			CapabilityID: "codespaces",
			Mode:         "partial_mutate",
			Reason:       "Codespace lifecycle is API-backed, but recreation after delete and some user-preference surfaces remain non-reversible or out of repo scope.",
			Supported:    []string{"inspect list/single/devcontainers", "create", "start", "stop", "delete"},
			Missing:      []string{"automatic recreate-on-rollback", "all account-level Codespaces preferences"},
		},
	},
	PlatformGitLab: {
		"merge_requests": {
			CapabilityID: "merge_requests",
			Mode:         "partial_mutate",
			Reason:       "GitLab merge request lifecycle is public API-backed, but merged-state reversal and some approval nuances are not automatically reversible.",
			Supported:    []string{"inspect merge requests", "create/update/close/reopen merge request"},
			Missing:      []string{"automatic unmerge rollback", "full approval-rule parity"},
		},
		"pipelines": {
			CapabilityID: "pipelines",
			Mode:         "partial_mutate",
			Reason:       "Pipelines and retry/cancel control are public API-backed, but run-control operations are not reversible.",
			Supported:    []string{"inspect pipelines", "trigger/retry/cancel pipeline"},
			Missing:      []string{"rollback for run-control actions"},
		},
		"environments": {
			CapabilityID: "environments",
			Mode:         "partial_mutate",
			Reason:       "Environment inspection and stop actions are API-backed, but full environment policy parity depends on broader project settings.",
			Supported:    []string{"inspect environments", "inspect deployments", "stop environment"},
			Missing:      []string{"full settings CRUD parity"},
		},
		"pages": {
			CapabilityID: "pages",
			Mode:         "partial_mutate",
			Reason:       "GitLab Pages is inspectable, but domain, certificate, and external DNS readiness still depends on external state.",
			Supported:    []string{"inspect Pages settings", "inspect deployment status"},
			Missing:      []string{"registrar and certificate lifecycle automation"},
		},
		"security": {
			CapabilityID: "security",
			Mode:         "inspect_only",
			Reason:       "GitLab exposes rich security inspection surfaces, but end-to-end admin mutation parity is fragmented across analyzers and project settings.",
			Supported:    []string{"inspect security dashboard posture"},
			Missing:      []string{"unified mutate/rollback surface"},
		},
	},
	PlatformBitbucket: {
		"pull_requests": {
			CapabilityID: "pull_requests",
			Mode:         "partial_mutate",
			Reason:       "Bitbucket pull request lifecycle is API-backed, but merge reversal is not automatically reversible.",
			Supported:    []string{"inspect pull requests", "create/update/decline/reopen pull request"},
			Missing:      []string{"automatic unmerge rollback"},
		},
		"pipelines": {
			CapabilityID: "pipelines",
			Mode:         "partial_mutate",
			Reason:       "Pipeline execution and rerun/cancel flows are public API-backed, but run-control operations are non-reversible.",
			Supported:    []string{"inspect pipelines", "trigger/rerun/stop pipeline"},
			Missing:      []string{"rollback for run-control actions"},
		},
		"deployments": {
			CapabilityID: "deployments",
			Mode:         "partial_mutate",
			Reason:       "Deployment visibility is API-backed, but full deployment environment lifecycle parity is broader than a single public surface.",
			Supported:    []string{"inspect deployments", "inspect deployment environments"},
			Missing:      []string{"full environment-policy CRUD parity"},
		},
		"branch_restrictions": {
			CapabilityID: "branch_restrictions",
			Mode:         "partial_mutate",
			Reason:       "Branch restrictions are API-backed, but some policy combinations still require manual review of repository settings.",
			Supported:    []string{"inspect branch restrictions", "create/update/delete restrictions"},
			Missing:      []string{"full policy-composition parity with UI"},
		},
		"webhooks": {
			CapabilityID: "webhooks",
			Mode:         "partial_mutate",
			Reason:       "Webhooks are public API-backed, but downstream delivery semantics and secret rotation remain operator-managed.",
			Supported:    []string{"inspect webhooks", "create/update/delete webhooks"},
			Missing:      []string{"delivery rollback across downstream systems"},
		},
		"repository_variables": {
			CapabilityID: "repository_variables",
			Mode:         "partial_mutate",
			Reason:       "Repository variables are public API-backed, but secure secret rotation and plaintext restore still require operator-provided previous values.",
			Supported:    []string{"inspect variables", "create/update/delete variables"},
			Missing:      []string{"automatic restore without previous plaintext"},
		},
	},
}

func CapabilityBoundaries(p Platform) []CapabilityBoundary {
	items := capabilityBoundaryCatalog[p]
	if len(items) == 0 {
		return nil
	}
	out := make([]CapabilityBoundary, 0, len(items))
	for _, capability := range CapabilityCatalog(p) {
		boundary, ok := items[capability.ID]
		if !ok {
			continue
		}
		out = append(out, cloneCapabilityBoundary(boundary))
	}
	return out
}

func CapabilityBoundaryFor(p Platform, capabilityID string) (CapabilityBoundary, bool) {
	items := capabilityBoundaryCatalog[p]
	if len(items) == 0 {
		return CapabilityBoundary{}, false
	}
	boundary, ok := items[strings.TrimSpace(capabilityID)]
	if !ok {
		return CapabilityBoundary{}, false
	}
	return cloneCapabilityBoundary(boundary), true
}

func RelevantCapabilityBoundaries(p Platform, ids []string) []CapabilityBoundary {
	if len(ids) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]CapabilityBoundary, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		boundary, ok := CapabilityBoundaryFor(p, id)
		if !ok {
			continue
		}
		out = append(out, boundary)
	}
	return out
}

func cloneCapabilityBoundary(in CapabilityBoundary) CapabilityBoundary {
	out := in
	out.Supported = append([]string(nil), in.Supported...)
	out.Missing = append([]string(nil), in.Missing...)
	return out
}
