package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Joker-of-Gotham/gitdex/internal/config"
	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
	"github.com/Joker-of-Gotham/gitdex/internal/platform"
	platformruntime "github.com/Joker-of-Gotham/gitdex/internal/platform/runtime"
)

type adminBundleResolver func(*status.GitState, config.PlatformConfig, config.AdapterConfig) (*platformruntime.Bundle, error)

type platformActionState struct {
	CapabilityID    string
	Scope           map[string]string
	Mutation        *platform.AdminMutationResult
	ValidatePayload json.RawMessage
	RollbackPayload json.RawMessage
	ExecMeta        platform.ExecutionMeta
	LedgerID        string
	RequestRevision int
	LedgerChain     []string
}

type platformExecRequest struct {
	Op       *git.PlatformExecInfo
	Mutation *platform.AdminMutationResult
	Revision int
}

type platformExecResultMsg struct {
	Platform    platform.Platform
	Request     platformExecRequest
	Diagnostics platform.DiagnosticSet
	Inspect     *platform.AdminSnapshot
	Mutation    *platform.AdminMutationResult
	Validation  *platform.AdminValidationResult
	Rollback    *platform.AdminRollbackResult
	Err         error
}

func (m Model) executePlatformSuggestion(s git.Suggestion) tea.Cmd {
	return m.executePlatformRequest(platformExecRequest{Op: clonePlatformExecInfo(s.PlatformOp)})
}

func (m Model) executePlatformRequest(req platformExecRequest) tea.Cmd {
	timeout := m.platformExecutionTimeout(req.Op)
	return func() tea.Msg {
		if req.Op == nil {
			return platformExecResultMsg{Request: req, Err: fmt.Errorf("platform suggestion is missing operation metadata")}
		}
		platformID := platform.PlatformUnknown
		if m.gitState != nil {
			platformID = platform.DetectPlatform(platform.PreferredRemoteURL(m.gitState.RemoteInfos))
		}
		diagnostics, repaired := platform.DiagnosePlatformOperation(platformID, m.gitState, req.Op)
		if repaired != nil {
			req.Op = repaired
		}
		if diagnostics.Decision == platform.DiagnosticBlocked {
			return platformExecResultMsg{
				Platform:    platformID,
				Request:     req,
				Diagnostics: diagnostics,
				Err:         fmt.Errorf("diagnostic blocked execution: %s", summarizeDiagnostics(diagnostics)),
			}
		}
		resolve := m.resolveAdminBundle
		if resolve == nil {
			resolve = platformruntime.ResolveAdminBundle
		}
		bundle, err := resolve(m.gitState, m.platformCfg, m.adapterCfg)
		if err != nil {
			return platformExecResultMsg{Platform: platformID, Request: req, Diagnostics: diagnostics, Err: err}
		}
		exec := bundle.Executors[strings.TrimSpace(req.Op.CapabilityID)]
		if exec == nil {
			return platformExecResultMsg{
				Platform:    bundle.Platform,
				Request:     req,
				Diagnostics: diagnostics,
				Err:         fmt.Errorf("platform executor %q is unavailable on %s", req.Op.CapabilityID, bundle.Platform.String()),
			}
		}
		adapterExec := bundle.ExecutorAdapter
		if adapterExec == nil {
			adapterExec = platform.NewDirectAdapterExecutor(bundle.Adapter)
		}
		if !adapterExec.CanHandle(strings.TrimSpace(req.Op.CapabilityID)) {
			return platformExecResultMsg{
				Platform:    bundle.Platform,
				Request:     req,
				Diagnostics: diagnostics,
				Err:         fmt.Errorf("%s adapter cannot handle capability %q", adapterExec.Kind(), req.Op.CapabilityID),
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		switch strings.ToLower(strings.TrimSpace(req.Op.Flow)) {
		case "inspect":
			snap, err := adapterExec.Inspect(ctx, exec, platform.AdminInspectRequest{
				ResourceID: strings.TrimSpace(req.Op.ResourceID),
				Scope:      cloneStringMap(req.Op.Scope),
				Query:      cloneStringMap(req.Op.Query),
			})
			decoratePlatformInspect(bundle.Platform, bundle.Adapter, req.Op, snap)
			return platformExecResultMsg{Platform: bundle.Platform, Request: req, Diagnostics: diagnostics, Inspect: snap, Err: err}
		case "mutate":
			result, err := adapterExec.Mutate(ctx, exec, platform.AdminMutationRequest{
				Operation:       strings.TrimSpace(req.Op.Operation),
				ResourceID:      strings.TrimSpace(req.Op.ResourceID),
				Scope:           cloneStringMap(req.Op.Scope),
				Payload:         cloneRaw(req.Op.Payload),
				RollbackPayload: cloneRaw(req.Op.RollbackPayload),
			})
			decoratePlatformMutation(bundle.Platform, bundle.Adapter, req.Op, result)
			return platformExecResultMsg{Platform: bundle.Platform, Request: req, Diagnostics: diagnostics, Mutation: result, Err: err}
		case "validate":
			result, err := adapterExec.Validate(ctx, exec, platform.AdminValidationRequest{
				ResourceID: strings.TrimSpace(req.Op.ResourceID),
				Scope:      cloneStringMap(req.Op.Scope),
				Payload:    cloneRaw(req.Op.ValidatePayload),
				Mutation:   cloneMutation(req.Mutation),
			})
			decoratePlatformValidation(bundle.Platform, bundle.Adapter, req.Op, result)
			return platformExecResultMsg{Platform: bundle.Platform, Request: req, Diagnostics: diagnostics, Validation: result, Err: err}
		case "rollback":
			result, err := adapterExec.RollbackOrCompensate(ctx, exec, platform.AdminRollbackRequest{
				Scope:    cloneStringMap(req.Op.Scope),
				Mutation: cloneMutation(req.Mutation),
				Payload:  cloneRaw(req.Op.RollbackPayload),
			})
			decoratePlatformRollback(bundle.Platform, bundle.Adapter, req.Op, result)
			return platformExecResultMsg{Platform: bundle.Platform, Request: req, Diagnostics: diagnostics, Rollback: result, Err: err}
		default:
			return platformExecResultMsg{Platform: bundle.Platform, Request: req, Diagnostics: diagnostics, Err: fmt.Errorf("unsupported platform flow %q", req.Op.Flow)}
		}
	}
}

func (m Model) platformExecutionTimeout(op *git.PlatformExecInfo) time.Duration {
	if step := m.findWorkflowFlowStep(op); step != nil && step.Policy.TimeoutSecs > 0 {
		return time.Duration(step.Policy.TimeoutSecs) * time.Second
	}
	if op != nil {
		switch strings.ToLower(strings.TrimSpace(op.Flow)) {
		case "mutate", "rollback":
			return 45 * time.Second
		}
	}
	return 25 * time.Second
}

func (m Model) lastPlatformRequest(flow string) (platformExecRequest, bool) {
	if m.lastPlatform == nil || m.lastPlatform.Mutation == nil {
		return platformExecRequest{}, false
	}
	return platformExecRequest{
		Op: &git.PlatformExecInfo{
			CapabilityID:    m.lastPlatform.CapabilityID,
			Flow:            flow,
			Operation:       strings.TrimSpace(m.lastPlatform.Mutation.Operation),
			ResourceID:      strings.TrimSpace(m.lastPlatform.Mutation.ResourceID),
			Scope:           cloneStringMap(m.lastPlatform.Scope),
			ValidatePayload: cloneRaw(m.lastPlatform.ValidatePayload),
			RollbackPayload: cloneRaw(m.lastPlatform.RollbackPayload),
		},
		Mutation: cloneMutation(m.lastPlatform.Mutation),
	}, true
}

func applyPlatformInputs(op *git.PlatformExecInfo, fields []git.InputField, values []string) *git.PlatformExecInfo {
	next := clonePlatformExecInfo(op)
	if next == nil {
		return nil
	}
	for i, field := range fields {
		if i >= len(values) {
			continue
		}
		value := strings.TrimSpace(values[i])
		for _, token := range placeholderTokens(field.Key) {
			next.ResourceID = strings.ReplaceAll(next.ResourceID, token, value)
			next.Scope = replaceMapTokens(next.Scope, token, value)
			next.Query = replaceMapTokens(next.Query, token, value)
			next.Payload = replaceJSONTokens(next.Payload, token, value)
			next.ValidatePayload = replaceJSONTokens(next.ValidatePayload, token, value)
			next.RollbackPayload = replaceJSONTokens(next.RollbackPayload, token, value)
		}
	}
	return next
}

func placeholderTokens(key string) []string {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil
	}
	if strings.Contains(key, "<") {
		return []string{key}
	}
	return []string{"<" + key + ">", key}
}

func replaceMapTokens(in map[string]string, token, value string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, item := range in {
		out[key] = strings.ReplaceAll(item, token, value)
	}
	return out
}

func replaceJSONTokens(raw json.RawMessage, token, value string) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	escaped, _ := json.Marshal(value)
	payload := strings.ReplaceAll(string(raw), token, strings.Trim(string(escaped), `"`))
	return json.RawMessage(payload)
}

func clonePlatformExecInfo(op *git.PlatformExecInfo) *git.PlatformExecInfo {
	if op == nil {
		return nil
	}
	return &git.PlatformExecInfo{
		CapabilityID:    strings.TrimSpace(op.CapabilityID),
		Flow:            strings.TrimSpace(op.Flow),
		Operation:       strings.TrimSpace(op.Operation),
		ResourceID:      strings.TrimSpace(op.ResourceID),
		Scope:           cloneStringMap(op.Scope),
		Query:           cloneStringMap(op.Query),
		Payload:         cloneRaw(op.Payload),
		ValidatePayload: cloneRaw(op.ValidatePayload),
		RollbackPayload: cloneRaw(op.RollbackPayload),
	}
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func cloneRaw(in json.RawMessage) json.RawMessage {
	if len(in) == 0 {
		return nil
	}
	return append(json.RawMessage(nil), in...)
}

func cloneSnapshot(in *platform.AdminSnapshot) *platform.AdminSnapshot {
	if in == nil {
		return nil
	}
	return &platform.AdminSnapshot{
		CapabilityID: in.CapabilityID,
		ResourceID:   in.ResourceID,
		State:        cloneRaw(in.State),
		Metadata:     cloneStringMap(in.Metadata),
		ExecMeta:     in.ExecMeta,
	}
}

func cloneMutation(in *platform.AdminMutationResult) *platform.AdminMutationResult {
	if in == nil {
		return nil
	}
	metadata := make(map[string]string, len(in.Metadata))
	for key, value := range in.Metadata {
		metadata[key] = value
	}
	return &platform.AdminMutationResult{
		CapabilityID: in.CapabilityID,
		Operation:    in.Operation,
		ResourceID:   in.ResourceID,
		Before:       cloneSnapshot(in.Before),
		After:        cloneSnapshot(in.After),
		Metadata:     metadata,
		ExecMeta:     in.ExecMeta,
		LedgerID:     in.LedgerID,
	}
}

func platformActionTitle(op *git.PlatformExecInfo) string {
	if op == nil {
		return "platform action"
	}
	parts := []string{strings.TrimSpace(op.CapabilityID), strings.TrimSpace(op.Flow)}
	if strings.TrimSpace(op.Operation) != "" {
		parts = append(parts, strings.TrimSpace(op.Operation))
	}
	if strings.TrimSpace(op.ResourceID) != "" {
		parts = append(parts, strings.TrimSpace(op.ResourceID))
	}
	return strings.Join(parts, " / ")
}

func platformTraceFromResult(msg platformExecResultMsg) commandTrace {
	meta := execMetaFromPlatformResult(msg)
	trace := commandTrace{
		Title:              platformActionTitle(msg.Request.Op),
		At:                 time.Now(),
		ResultKind:         resultKindPlatformAdmin,
		PlatformCapability: strings.TrimSpace(msg.Request.Op.CapabilityID),
		PlatformFlow:       strings.TrimSpace(msg.Request.Op.Flow),
		PlatformOperation:  strings.TrimSpace(msg.Request.Op.Operation),
		PlatformResourceID: strings.TrimSpace(msg.Request.Op.ResourceID),
		PlatformAdapter:    string(meta.Adapter),
		PlatformRollback:   string(meta.Rollback),
	}
	switch {
	case msg.Inspect != nil:
		trace.Status = "platform inspect success"
		trace.Output = "inspect completed"
		trace.PlatformInspect = cloneRaw(msg.Inspect.State)
		trace.PlatformResourceID = strings.TrimSpace(firstNonEmpty(msg.Inspect.ResourceID, trace.PlatformResourceID))
		trace.PlatformBoundary = msg.Inspect.ExecMeta.BoundaryReason
	case msg.Mutation != nil:
		trace.Status = "platform mutate success"
		trace.Output = fmt.Sprintf("%s completed", firstNonEmpty(msg.Mutation.Operation, "mutation"))
		if msg.Mutation.Before != nil {
			trace.PlatformBefore = cloneRaw(msg.Mutation.Before.State)
		}
		if msg.Mutation.After != nil {
			trace.PlatformAfter = cloneRaw(msg.Mutation.After.State)
		}
		trace.PlatformResourceID = strings.TrimSpace(firstNonEmpty(msg.Mutation.ResourceID, trace.PlatformResourceID))
		trace.PlatformLedgerID = strings.TrimSpace(msg.Mutation.LedgerID)
		trace.PlatformBoundary = msg.Mutation.ExecMeta.BoundaryReason
		trace.PlatformApproval = msg.Mutation.ExecMeta.ApprovalRequired
	case msg.Validation != nil:
		trace.Status = "platform validate success"
		if !msg.Validation.OK {
			trace.Status = "platform validate failed"
		}
		trace.Output = strings.TrimSpace(firstNonEmpty(msg.Validation.Summary, "validation completed"))
		if msg.Validation.Snapshot != nil {
			trace.PlatformSnapshot = cloneRaw(msg.Validation.Snapshot.State)
			trace.PlatformResourceID = strings.TrimSpace(firstNonEmpty(msg.Validation.ResourceID, msg.Validation.Snapshot.ResourceID, trace.PlatformResourceID))
		}
		trace.PlatformBoundary = msg.Validation.ExecMeta.BoundaryReason
	case msg.Rollback != nil:
		trace.Status = "platform rollback success"
		if !msg.Rollback.OK {
			trace.Status = "platform rollback failed"
		}
		trace.Output = strings.TrimSpace(firstNonEmpty(msg.Rollback.Summary, "rollback completed"))
		if msg.Rollback.Snapshot != nil {
			trace.PlatformSnapshot = cloneRaw(msg.Rollback.Snapshot.State)
			trace.PlatformResourceID = strings.TrimSpace(firstNonEmpty(msg.Rollback.Snapshot.ResourceID, trace.PlatformResourceID))
		}
		trace.PlatformBoundary = msg.Rollback.ExecMeta.BoundaryReason
		if msg.Rollback.Compensation != nil {
			trace.PlatformCompensation = strings.TrimSpace(firstNonEmpty(msg.Rollback.Compensation.Summary, msg.Rollback.Compensation.Kind))
		}
	default:
		trace.Status = "platform success"
		trace.Output = "completed"
	}
	return trace
}

func decoratePlatformInspect(platformID platform.Platform, adapter platform.AdapterKind, op *git.PlatformExecInfo, snap *platform.AdminSnapshot) {
	if snap == nil || op == nil {
		return
	}
	snap.ExecMeta = platform.ExecutionMetaFor(platformID, op.CapabilityID, op.Flow, op.Operation)
	if adapter != "" {
		snap.ExecMeta.Adapter = adapter
	}
}

func decoratePlatformMutation(platformID platform.Platform, adapter platform.AdapterKind, op *git.PlatformExecInfo, result *platform.AdminMutationResult) {
	if result == nil || op == nil {
		return
	}
	meta := platform.ExecutionMetaFor(platformID, op.CapabilityID, op.Flow, op.Operation)
	if adapter != "" {
		meta.Adapter = adapter
	}
	result.ExecMeta = meta
	if result.Before != nil {
		result.Before.ExecMeta = meta
	}
	if result.After != nil {
		result.After.ExecMeta = meta
	}
	if strings.TrimSpace(result.LedgerID) == "" {
		result.LedgerID = platform.NewLedgerID(op.CapabilityID, op.Flow, op.Operation, firstNonEmpty(result.ResourceID, op.ResourceID), time.Now())
	}
}

func decoratePlatformValidation(platformID platform.Platform, adapter platform.AdapterKind, op *git.PlatformExecInfo, result *platform.AdminValidationResult) {
	if result == nil || op == nil {
		return
	}
	meta := platform.ExecutionMetaFor(platformID, op.CapabilityID, op.Flow, op.Operation)
	if adapter != "" {
		meta.Adapter = adapter
	}
	result.ExecMeta = meta
	if result.Snapshot != nil {
		result.Snapshot.ExecMeta = meta
	}
}

func decoratePlatformRollback(platformID platform.Platform, adapter platform.AdapterKind, op *git.PlatformExecInfo, result *platform.AdminRollbackResult) {
	if result == nil || op == nil {
		return
	}
	meta := platform.ExecutionMetaFor(platformID, op.CapabilityID, op.Flow, op.Operation)
	if adapter != "" {
		meta.Adapter = adapter
	}
	result.ExecMeta = meta
	if result.Snapshot != nil {
		result.Snapshot.ExecMeta = meta
	}
}

func buildPlatformLedgerEntry(platformID platform.Platform, req platformExecRequest, msg platformExecResultMsg, stepID string) platform.MutationLedgerEntry {
	meta := execMetaFromPlatformResult(msg)
	entry := platform.MutationLedgerEntry{
		ID:                 platform.NewLedgerID(req.Op.CapabilityID, req.Op.Flow, req.Op.Operation, firstNonEmpty(req.Op.ResourceID, resourceIDFromResult(msg)), time.Now()),
		At:                 time.Now(),
		Platform:           platformID.String(),
		CapabilityID:       strings.TrimSpace(req.Op.CapabilityID),
		Flow:               strings.TrimSpace(req.Op.Flow),
		Operation:          strings.TrimSpace(req.Op.Operation),
		ResourceID:         strings.TrimSpace(firstNonEmpty(resourceIDFromResult(msg), req.Op.ResourceID)),
		RequestRevision:    maxInt(0, req.Revision),
		ExecMeta:           meta,
		Request:            marshalPlatformRequest(req.Op),
		WorkflowStepID:     strings.TrimSpace(stepID),
		DiagnosticDecision: msg.Diagnostics.Decision,
		Diagnostics:        append([]platform.DiagnosticItem(nil), msg.Diagnostics.Items...),
	}
	if msg.Err != nil {
		entry.Summary = msg.Err.Error()
		entry.Failure = classifyPlatformFailure(msg.Err)
		return entry
	}
	switch {
	case msg.Mutation != nil:
		entry.Summary = strings.TrimSpace(firstNonEmpty(msg.Mutation.Operation, "mutation"))
		if msg.Mutation.Before != nil {
			entry.Before = cloneRaw(msg.Mutation.Before.State)
		}
		if msg.Mutation.After != nil {
			entry.After = cloneRaw(msg.Mutation.After.State)
		}
		if len(msg.Mutation.Metadata) > 0 {
			entry.Metadata = cloneStringMap(msg.Mutation.Metadata)
		}
		entry.ID = strings.TrimSpace(firstNonEmpty(msg.Mutation.LedgerID, entry.ID))
	case msg.Validation != nil:
		entry.Summary = strings.TrimSpace(firstNonEmpty(msg.Validation.Summary, "validation"))
		if len(msg.Validation.Metadata) > 0 {
			entry.Metadata = cloneStringMap(msg.Validation.Metadata)
		}
		if msg.Validation.Snapshot != nil {
			entry.Validate = cloneRaw(msg.Validation.Snapshot.State)
			if len(entry.Metadata) == 0 && len(msg.Validation.Snapshot.Metadata) > 0 {
				entry.Metadata = cloneStringMap(msg.Validation.Snapshot.Metadata)
			}
		}
	case msg.Rollback != nil:
		entry.Summary = strings.TrimSpace(firstNonEmpty(msg.Rollback.Summary, "rollback"))
		if len(msg.Rollback.Metadata) > 0 {
			entry.Metadata = cloneStringMap(msg.Rollback.Metadata)
		}
		if msg.Rollback.Snapshot != nil {
			entry.Rollback = cloneRaw(msg.Rollback.Snapshot.State)
			if len(entry.Metadata) == 0 && len(msg.Rollback.Snapshot.Metadata) > 0 {
				entry.Metadata = cloneStringMap(msg.Rollback.Snapshot.Metadata)
			}
		}
	case msg.Inspect != nil:
		entry.Summary = "inspect"
		entry.After = cloneRaw(msg.Inspect.State)
		if len(msg.Inspect.Metadata) > 0 {
			entry.Metadata = cloneStringMap(msg.Inspect.Metadata)
		}
	}
	return entry
}

func summarizeDiagnostics(set platform.DiagnosticSet) string {
	if len(set.Items) == 0 {
		return string(set.Decision)
	}
	parts := make([]string, 0, len(set.Items))
	for _, item := range set.Items {
		summary := strings.TrimSpace(item.Summary)
		if summary == "" {
			summary = item.Code
		}
		if summary != "" {
			parts = append(parts, summary)
		}
	}
	if len(parts) == 0 {
		return string(set.Decision)
	}
	return strings.Join(parts, "; ")
}

func marshalPlatformRequest(op *git.PlatformExecInfo) json.RawMessage {
	if op == nil {
		return nil
	}
	data, err := json.Marshal(op)
	if err != nil {
		return nil
	}
	return data
}

func resourceIDFromResult(msg platformExecResultMsg) string {
	switch {
	case msg.Mutation != nil:
		return strings.TrimSpace(msg.Mutation.ResourceID)
	case msg.Validation != nil:
		return strings.TrimSpace(msg.Validation.ResourceID)
	case msg.Rollback != nil && msg.Rollback.Snapshot != nil:
		return strings.TrimSpace(msg.Rollback.Snapshot.ResourceID)
	case msg.Inspect != nil:
		return strings.TrimSpace(msg.Inspect.ResourceID)
	default:
		return ""
	}
}

func classifyPlatformFailure(err error) platform.FailureTaxonomy {
	if err == nil {
		return ""
	}
	text := strings.ToLower(strings.TrimSpace(err.Error()))
	switch {
	case strings.Contains(text, "rate limit"), strings.Contains(text, "secondary rate limit"):
		return platform.FailureRateLimited
	case strings.Contains(text, "unauthorized"), strings.Contains(text, "forbidden"), strings.Contains(text, "token"), strings.Contains(text, "not configured"):
		return platform.FailureAuthMissing
	case strings.Contains(text, "adapter"):
		return platform.FailureAdapter
	case strings.Contains(text, "rollback"), strings.Contains(text, "cannot be rolled back"), strings.Contains(text, "not reversible"):
		return platform.FailureReversible
	case strings.Contains(text, "boundary"):
		return platform.FailureBoundary
	default:
		return platform.FailureExecutor
	}
}

func execMetaFromPlatformResult(msg platformExecResultMsg) platform.ExecutionMeta {
	if msg.Request.Op == nil {
		return platform.ExecutionMeta{}
	}
	meta := platform.ExecutionMetaFor(msg.Platform, msg.Request.Op.CapabilityID, msg.Request.Op.Flow, msg.Request.Op.Operation)
	switch {
	case msg.Inspect != nil && msg.Inspect.ExecMeta.Adapter != "":
		return msg.Inspect.ExecMeta
	case msg.Mutation != nil && msg.Mutation.ExecMeta.Adapter != "":
		return msg.Mutation.ExecMeta
	case msg.Validation != nil && msg.Validation.ExecMeta.Adapter != "":
		return msg.Validation.ExecMeta
	case msg.Rollback != nil && msg.Rollback.ExecMeta.Adapter != "":
		return msg.Rollback.ExecMeta
	default:
		return meta
	}
}

func (m Model) appendMutationLedger(entry platform.MutationLedgerEntry) Model {
	if strings.TrimSpace(entry.ID) == "" {
		return m
	}
	m.mutationLedger = append(m.mutationLedger, entry)
	if len(m.mutationLedger) > 60 {
		m.mutationLedger = m.mutationLedger[len(m.mutationLedger)-60:]
	}
	return m
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func compactStringList(values []string, limit int) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
