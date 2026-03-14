package platform

import (
	"encoding/json"
	"regexp"
	"strings"
	"time"

	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
)

type DiagnosticSeverity string

const (
	DiagnosticInfo    DiagnosticSeverity = "info"
	DiagnosticWarning DiagnosticSeverity = "warning"
	DiagnosticBlock   DiagnosticSeverity = "blocking"
)

type DiagnosticDecision string

const (
	DiagnosticAllow      DiagnosticDecision = "allow"
	DiagnosticAutoRepair DiagnosticDecision = "auto_repair_once"
	DiagnosticBlocked    DiagnosticDecision = "block"
)

type DiagnosticItem struct {
	Code     string             `json:"code,omitempty"`
	Severity DiagnosticSeverity `json:"severity,omitempty"`
	Summary  string             `json:"summary,omitempty"`
	Detail   string             `json:"detail,omitempty"`
}

type DiagnosticSet struct {
	GeneratedAt time.Time           `json:"generated_at"`
	Platform    string              `json:"platform,omitempty"`
	Capability  string              `json:"capability,omitempty"`
	Flow        string              `json:"flow,omitempty"`
	Operation   string              `json:"operation,omitempty"`
	Decision    DiagnosticDecision  `json:"decision,omitempty"`
	Boundary    *CapabilityBoundary `json:"boundary,omitempty"`
	Policies    ExecutionPolicies   `json:"policies,omitempty"`
	Items       []DiagnosticItem    `json:"items,omitempty"`
}

func DiagnosePlatformOperation(p Platform, state *status.GitState, op *git.PlatformExecInfo) (DiagnosticSet, *git.PlatformExecInfo) {
	set := DiagnosticSet{
		GeneratedAt: time.Now(),
		Platform:    p.String(),
	}
	if op == nil {
		set.Decision = DiagnosticBlocked
		set.Items = append(set.Items, DiagnosticItem{
			Code:     "platform_op_missing",
			Severity: DiagnosticBlock,
			Summary:  "platform operation metadata is required",
		})
		return set, nil
	}
	set.Capability = strings.TrimSpace(op.CapabilityID)
	set.Flow = strings.TrimSpace(op.Flow)
	set.Operation = strings.TrimSpace(op.Operation)
	set.Policies = PoliciesFor(p, op.CapabilityID, op.Flow, op.Operation)
	if boundary, ok := CapabilityBoundaryFor(p, op.CapabilityID); ok {
		set.Boundary = &boundary
		if (strings.EqualFold(op.Flow, "mutate") || strings.EqualFold(op.Flow, "rollback")) && strings.EqualFold(boundary.Mode, string(CoverageInspectOnly)) {
			set.Items = append(set.Items, DiagnosticItem{
				Code:     "boundary_inspect_only",
				Severity: DiagnosticBlock,
				Summary:  "surface is inspect-only on the current platform",
				Detail:   boundary.Reason,
			})
		}
	}
	if state == nil || strings.TrimSpace(PreferredRemoteURL(state.RemoteInfos)) == "" {
		set.Items = append(set.Items, DiagnosticItem{
			Code:     "repo_remote_missing",
			Severity: DiagnosticWarning,
			Summary:  "repository remote URL is required for platform execution",
		})
	}
	repaired := cloneDiagnosticOp(op)
	if next, changed := repairPlatformTokens(state, repaired); changed {
		repaired = next
		set.Items = append(set.Items, DiagnosticItem{
			Code:     "tokens_auto_repaired",
			Severity: DiagnosticInfo,
			Summary:  "placeholder tokens were resolved from repository state",
		})
	}
	if placeholders := unresolvedDiagnosticPlaceholders(repaired); len(placeholders) > 0 {
		set.Items = append(set.Items, DiagnosticItem{
			Code:     "placeholders_unresolved",
			Severity: DiagnosticBlock,
			Summary:  "unresolved placeholders remain in platform request",
			Detail:   strings.Join(placeholders, ", "),
		})
	}
	if set.Policies.Compensation.Required && set.Policies.Rollback.Kind != RollbackReversible {
		set.Items = append(set.Items, DiagnosticItem{
			Code:     "compensation_required",
			Severity: DiagnosticWarning,
			Summary:  "operator-visible compensation path is required",
			Detail:   set.Policies.Compensation.Note,
		})
	}
	set.Decision = DiagnosticAllow
	for _, item := range set.Items {
		if item.Severity == DiagnosticBlock {
			set.Decision = DiagnosticBlocked
			return set, repaired
		}
	}
	for _, item := range set.Items {
		if item.Code == "tokens_auto_repaired" {
			set.Decision = DiagnosticAutoRepair
			break
		}
	}
	return set, repaired
}

var diagnosticPlaceholderRE = regexp.MustCompile(`<[^>]+>`)

func unresolvedDiagnosticPlaceholders(op *git.PlatformExecInfo) []string {
	if op == nil {
		return nil
	}
	values := []string{op.ResourceID, string(op.Payload), string(op.ValidatePayload), string(op.RollbackPayload)}
	for _, item := range op.Scope {
		values = append(values, item)
	}
	for _, item := range op.Query {
		values = append(values, item)
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, 4)
	for _, value := range values {
		matches := diagnosticPlaceholderRE.FindAllString(value, -1)
		for _, match := range matches {
			match = strings.TrimSpace(match)
			if match == "" {
				continue
			}
			if _, ok := seen[match]; ok {
				continue
			}
			seen[match] = struct{}{}
			out = append(out, match)
		}
	}
	return out
}

func repairPlatformTokens(state *status.GitState, op *git.PlatformExecInfo) (*git.PlatformExecInfo, bool) {
	if state == nil || op == nil {
		return op, false
	}
	replacements := map[string]string{
		"<current_branch>": strings.TrimSpace(state.LocalBranch.Name),
		"<default_branch>": strings.TrimSpace(firstNonEmptyRemote(state.RepoConfig.DefaultBranch, state.LocalBranch.Name)),
	}
	if remoteURL := strings.TrimSpace(PreferredRemoteURL(state.RemoteInfos)); remoteURL != "" && DetectPlatform(remoteURL) == PlatformGitHub {
		if owner, repo, err := GitHubOwnerRepoFromRemote(remoteURL); err == nil {
			replacements["<repo_owner>"] = strings.TrimSpace(owner)
			replacements["<repo_name>"] = strings.TrimSpace(repo)
		}
	}
	changed := false
	replaceText := func(in string) string {
		out := in
		for token, value := range replacements {
			if strings.TrimSpace(value) == "" {
				continue
			}
			next := strings.ReplaceAll(out, token, value)
			if next != out {
				changed = true
				out = next
			}
		}
		return out
	}
	op.ResourceID = replaceText(op.ResourceID)
	op.Scope = replaceDiagnosticMap(op.Scope, replaceText)
	op.Query = replaceDiagnosticMap(op.Query, replaceText)
	op.Payload = replaceDiagnosticJSON(op.Payload, replaceText)
	op.ValidatePayload = replaceDiagnosticJSON(op.ValidatePayload, replaceText)
	op.RollbackPayload = replaceDiagnosticJSON(op.RollbackPayload, replaceText)
	return op, changed
}

func replaceDiagnosticMap(in map[string]string, replace func(string) string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = replace(value)
	}
	return out
}

func replaceDiagnosticJSON(raw json.RawMessage, replace func(string) string) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	return json.RawMessage(replace(string(raw)))
}

func cloneDiagnosticOp(op *git.PlatformExecInfo) *git.PlatformExecInfo {
	if op == nil {
		return nil
	}
	return &git.PlatformExecInfo{
		CapabilityID:    strings.TrimSpace(op.CapabilityID),
		Flow:            strings.TrimSpace(op.Flow),
		Operation:       strings.TrimSpace(op.Operation),
		ResourceID:      strings.TrimSpace(op.ResourceID),
		Scope:           cloneDiagnosticMap(op.Scope),
		Query:           cloneDiagnosticMap(op.Query),
		Payload:         append(json.RawMessage(nil), op.Payload...),
		ValidatePayload: append(json.RawMessage(nil), op.ValidatePayload...),
		RollbackPayload: append(json.RawMessage(nil), op.RollbackPayload...),
	}
}

func cloneDiagnosticMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func firstNonEmptyRemote(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
