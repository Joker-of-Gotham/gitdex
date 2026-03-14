package platform

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type AdapterExecutor interface {
	Kind() AdapterKind
	CanHandle(capabilityID string) bool
	Inspect(ctx context.Context, exec AdminExecutor, req AdminInspectRequest) (*AdminSnapshot, error)
	Mutate(ctx context.Context, exec AdminExecutor, req AdminMutationRequest) (*AdminMutationResult, error)
	Validate(ctx context.Context, exec AdminExecutor, req AdminValidationRequest) (*AdminValidationResult, error)
	RollbackOrCompensate(ctx context.Context, exec AdminExecutor, req AdminRollbackRequest) (*AdminRollbackResult, error)
}

type directAdapterExecutor struct {
	kind     AdapterKind
	metadata map[string]string
}

type cliAdapterExecutor struct {
	binary   string
	metadata map[string]string
}

func NewDirectAdapterExecutor(kind AdapterKind) AdapterExecutor {
	if kind == "" {
		kind = AdapterAPI
	}
	return directAdapterExecutor{
		kind:     kind,
		metadata: adapterAuditMetadata(kind, ""),
	}
}

func NewAPIAdapterExecutor() AdapterExecutor {
	return directAdapterExecutor{
		kind:     AdapterAPI,
		metadata: adapterAuditMetadata(AdapterAPI, ""),
	}
}

func NewCLIAdapterExecutor(binary string) AdapterExecutor {
	binary = strings.TrimSpace(binary)
	if binary == "" {
		binary = "gh"
	}
	return cliAdapterExecutor{
		binary:   binary,
		metadata: adapterAuditMetadata(AdapterCLI, binary),
	}
}

func NewBrowserStubAdapterExecutor(driver string) AdapterExecutor {
	return browserStubAdapterExecutor{driver: ResolveBrowserStubDriver(driver)}
}

func (d directAdapterExecutor) Kind() AdapterKind {
	return d.kind
}

func (d directAdapterExecutor) CanHandle(capabilityID string) bool {
	return capabilityID != ""
}

func (d directAdapterExecutor) Inspect(ctx context.Context, exec AdminExecutor, req AdminInspectRequest) (*AdminSnapshot, error) {
	if exec == nil {
		return nil, fmt.Errorf("%s adapter has no bound executor", d.kind)
	}
	result, err := exec.Inspect(ctx, req)
	annotateAdapterSnapshot(result, d.metadata)
	return result, err
}

func (d directAdapterExecutor) Mutate(ctx context.Context, exec AdminExecutor, req AdminMutationRequest) (*AdminMutationResult, error) {
	if exec == nil {
		return nil, fmt.Errorf("%s adapter has no bound executor", d.kind)
	}
	result, err := exec.Mutate(ctx, req)
	annotateAdapterMutation(result, d.metadata)
	return result, err
}

func (d directAdapterExecutor) Validate(ctx context.Context, exec AdminExecutor, req AdminValidationRequest) (*AdminValidationResult, error) {
	if exec == nil {
		return nil, fmt.Errorf("%s adapter has no bound executor", d.kind)
	}
	result, err := exec.Validate(ctx, req)
	annotateAdapterValidation(result, d.metadata)
	return result, err
}

func (d directAdapterExecutor) RollbackOrCompensate(ctx context.Context, exec AdminExecutor, req AdminRollbackRequest) (*AdminRollbackResult, error) {
	if exec == nil {
		return nil, fmt.Errorf("%s adapter has no bound executor", d.kind)
	}
	result, err := exec.Rollback(ctx, req)
	annotateAdapterRollback(result, d.metadata)
	return result, err
}

func (c cliAdapterExecutor) Kind() AdapterKind {
	return AdapterCLI
}

func (c cliAdapterExecutor) CanHandle(capabilityID string) bool {
	return strings.TrimSpace(capabilityID) != ""
}

func (c cliAdapterExecutor) Inspect(ctx context.Context, exec AdminExecutor, req AdminInspectRequest) (*AdminSnapshot, error) {
	if exec == nil {
		return nil, fmt.Errorf("%s adapter has no bound executor", c.Kind())
	}
	result, err := exec.Inspect(ctx, req)
	annotateAdapterSnapshot(result, c.metadata)
	return result, err
}

func (c cliAdapterExecutor) Mutate(ctx context.Context, exec AdminExecutor, req AdminMutationRequest) (*AdminMutationResult, error) {
	if exec == nil {
		return nil, fmt.Errorf("%s adapter has no bound executor", c.Kind())
	}
	result, err := exec.Mutate(ctx, req)
	annotateAdapterMutation(result, c.metadata)
	return result, err
}

func (c cliAdapterExecutor) Validate(ctx context.Context, exec AdminExecutor, req AdminValidationRequest) (*AdminValidationResult, error) {
	if exec == nil {
		return nil, fmt.Errorf("%s adapter has no bound executor", c.Kind())
	}
	result, err := exec.Validate(ctx, req)
	annotateAdapterValidation(result, c.metadata)
	return result, err
}

func (c cliAdapterExecutor) RollbackOrCompensate(ctx context.Context, exec AdminExecutor, req AdminRollbackRequest) (*AdminRollbackResult, error) {
	if exec == nil {
		return nil, fmt.Errorf("%s adapter has no bound executor", c.Kind())
	}
	result, err := exec.Rollback(ctx, req)
	annotateAdapterRollback(result, c.metadata)
	return result, err
}

type browserStubAdapterExecutor struct {
	driver BrowserStubDriver
}

func (b browserStubAdapterExecutor) Kind() AdapterKind {
	return AdapterBrowser
}

func (b browserStubAdapterExecutor) CanHandle(capabilityID string) bool {
	return strings.TrimSpace(capabilityID) != ""
}

func (b browserStubAdapterExecutor) Inspect(_ context.Context, exec AdminExecutor, req AdminInspectRequest) (*AdminSnapshot, error) {
	capabilityID := capabilityIDFromExecutor(exec)
	resourceID := strings.TrimSpace(req.ResourceID)
	return &AdminSnapshot{
		CapabilityID: capabilityID,
		ResourceID:   resourceID,
		State:        b.stubState(capabilityID, "inspect", resourceID),
		Metadata:     adapterAuditMetadata(AdapterBrowser, b.driverName()),
	}, nil
}

func (b browserStubAdapterExecutor) Mutate(_ context.Context, exec AdminExecutor, req AdminMutationRequest) (*AdminMutationResult, error) {
	capabilityID := capabilityIDFromExecutor(exec)
	resourceID := strings.TrimSpace(req.ResourceID)
	return &AdminMutationResult{
		CapabilityID: capabilityID,
		Operation:    strings.TrimSpace(req.Operation),
		ResourceID:   resourceID,
		After: &AdminSnapshot{
			CapabilityID: capabilityID,
			ResourceID:   resourceID,
			State:        b.stubState(capabilityID, "mutate", resourceID),
			Metadata:     adapterAuditMetadata(AdapterBrowser, b.driverName()),
		},
		Metadata: adapterAuditMetadata(AdapterBrowser, b.driverName(),
			"manual_completion_required", "true",
			"rollback_grade", "manual restore required",
		),
	}, nil
}

func (b browserStubAdapterExecutor) Validate(_ context.Context, exec AdminExecutor, req AdminValidationRequest) (*AdminValidationResult, error) {
	capabilityID := capabilityIDFromExecutor(exec)
	resourceID := strings.TrimSpace(firstNonEmptyAdmin(req.ResourceID, resourceIDFromMutation(req.Mutation)))
	return &AdminValidationResult{
		OK:         false,
		Summary:    "browser-backed stub requires operator validation",
		ResourceID: resourceID,
		Metadata: adapterAuditMetadata(AdapterBrowser, b.driverName(),
			"manual_completion_required", "true",
			"operator_validation_required", "true",
		),
		Snapshot: &AdminSnapshot{
			CapabilityID: capabilityID,
			ResourceID:   resourceID,
			State:        b.stubState(capabilityID, "validate", resourceID),
			Metadata:     adapterAuditMetadata(AdapterBrowser, b.driverName()),
		},
	}, nil
}

func (b browserStubAdapterExecutor) RollbackOrCompensate(_ context.Context, exec AdminExecutor, req AdminRollbackRequest) (*AdminRollbackResult, error) {
	capabilityID := capabilityIDFromExecutor(exec)
	resourceID := resourceIDFromMutation(req.Mutation)
	return &AdminRollbackResult{
		OK:      false,
		Summary: "browser-backed stub recorded manual recovery path",
		Metadata: adapterAuditMetadata(AdapterBrowser, b.driverName(),
			"manual_completion_required", "true",
			"rollback_grade", "manual restore required",
		),
		Snapshot: &AdminSnapshot{
			CapabilityID: capabilityID,
			ResourceID:   resourceID,
			State:        b.stubState(capabilityID, "rollback", resourceID),
			Metadata:     adapterAuditMetadata(AdapterBrowser, b.driverName()),
		},
		Compensation: &CompensationAction{
			Kind:        "manual_restore_required",
			Summary:     "complete the browser-backed recovery manually and retain the audit chain",
			OperatorRef: "browser:" + b.driverName(),
			LedgerChain: compactLedgerRefs(resourceID),
			Scope: map[string]string{
				"adapter":     string(AdapterBrowser),
				"driver":      b.driverName(),
				"resource_id": resourceID,
			},
		},
	}, nil
}

func (b browserStubAdapterExecutor) stubState(capabilityID, flow, resourceID string) json.RawMessage {
	if b.driver == nil {
		return ResolveBrowserStubDriver("default").StubState(capabilityID, flow, resourceID)
	}
	return b.driver.StubState(capabilityID, flow, resourceID)
}

func (b browserStubAdapterExecutor) driverName() string {
	if b.driver == nil {
		return "default"
	}
	return strings.TrimSpace(b.driver.Name())
}

func capabilityIDFromExecutor(exec AdminExecutor) string {
	if exec == nil {
		return ""
	}
	return strings.TrimSpace(exec.CapabilityID())
}

func resourceIDFromMutation(mutation *AdminMutationResult) string {
	if mutation == nil {
		return ""
	}
	return strings.TrimSpace(mutation.ResourceID)
}

func firstNonEmptyAdmin(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func compactLedgerRefs(values ...string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
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
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func adapterAuditMetadata(kind AdapterKind, identity string, extras ...string) map[string]string {
	metadata := map[string]string{
		"adapter_backed": string(kind),
	}
	switch kind {
	case AdapterAPI:
		metadata["adapter_transport"] = "api"
	case AdapterCLI:
		metadata["adapter_transport"] = "cli"
	case AdapterBrowser:
		metadata["adapter_transport"] = "browser_stub"
	}
	identity = strings.TrimSpace(identity)
	switch kind {
	case AdapterCLI:
		if identity != "" {
			metadata["adapter_binary"] = identity
		}
	case AdapterBrowser:
		if identity != "" {
			metadata["browser_driver"] = identity
		}
	}
	for idx := 0; idx+1 < len(extras); idx += 2 {
		key := strings.TrimSpace(extras[idx])
		value := strings.TrimSpace(extras[idx+1])
		if key == "" || value == "" {
			continue
		}
		metadata[key] = value
	}
	return metadata
}

func annotateAdapterSnapshot(snapshot *AdminSnapshot, metadata map[string]string) {
	if snapshot == nil {
		return
	}
	snapshot.Metadata = mergeMetadata(snapshot.Metadata, metadata)
}

func annotateAdapterMutation(result *AdminMutationResult, metadata map[string]string) {
	if result == nil {
		return
	}
	result.Metadata = mergeMetadata(result.Metadata, metadata)
	annotateAdapterSnapshot(result.Before, metadata)
	annotateAdapterSnapshot(result.After, metadata)
}

func annotateAdapterValidation(result *AdminValidationResult, metadata map[string]string) {
	if result == nil {
		return
	}
	result.Metadata = mergeMetadata(result.Metadata, metadata)
	annotateAdapterSnapshot(result.Snapshot, metadata)
}

func annotateAdapterRollback(result *AdminRollbackResult, metadata map[string]string) {
	if result == nil {
		return
	}
	result.Metadata = mergeMetadata(result.Metadata, metadata)
	annotateAdapterSnapshot(result.Snapshot, metadata)
	if result.Compensation != nil {
		if result.Compensation.Scope == nil {
			result.Compensation.Scope = map[string]string{}
		}
		for key, value := range metadata {
			if _, exists := result.Compensation.Scope[key]; exists {
				continue
			}
			result.Compensation.Scope[key] = value
		}
	}
}

func mergeMetadata(base, extra map[string]string) map[string]string {
	if len(base) == 0 && len(extra) == 0 {
		return nil
	}
	out := CloneStringMap(base)
	if out == nil {
		out = map[string]string{}
	}
	for key, value := range extra {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		if _, exists := out[key]; exists {
			continue
		}
		out[key] = value
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
