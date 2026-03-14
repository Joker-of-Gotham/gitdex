package platform

import (
	"context"
	"encoding/json"
	"testing"
)

type fakeAdminExecutor struct{}

func (fakeAdminExecutor) CapabilityID() string { return "pages" }

func (fakeAdminExecutor) Inspect(context.Context, AdminInspectRequest) (*AdminSnapshot, error) {
	return &AdminSnapshot{CapabilityID: "pages", ResourceID: "site"}, nil
}

func (fakeAdminExecutor) Mutate(context.Context, AdminMutationRequest) (*AdminMutationResult, error) {
	return &AdminMutationResult{CapabilityID: "pages", Operation: "update", ResourceID: "site"}, nil
}

func (fakeAdminExecutor) Validate(context.Context, AdminValidationRequest) (*AdminValidationResult, error) {
	return &AdminValidationResult{OK: true, Summary: "ok", ResourceID: "site"}, nil
}

func (fakeAdminExecutor) Rollback(context.Context, AdminRollbackRequest) (*AdminRollbackResult, error) {
	return &AdminRollbackResult{OK: true, Summary: "rolled back"}, nil
}

func TestDirectAdapterExecutorDelegatesToAdminExecutor(t *testing.T) {
	adapter := NewDirectAdapterExecutor(AdapterCLI)
	exec := fakeAdminExecutor{}
	if !adapter.CanHandle("pages") {
		t.Fatalf("expected adapter to handle capability")
	}
	if adapter.Kind() != AdapterCLI {
		t.Fatalf("expected CLI adapter kind, got %s", adapter.Kind())
	}
	if _, err := adapter.Inspect(context.Background(), exec, AdminInspectRequest{}); err != nil {
		t.Fatalf("inspect failed: %v", err)
	}
	if _, err := adapter.Mutate(context.Background(), exec, AdminMutationRequest{Operation: "update"}); err != nil {
		t.Fatalf("mutate failed: %v", err)
	}
	if _, err := adapter.Validate(context.Background(), exec, AdminValidationRequest{}); err != nil {
		t.Fatalf("validate failed: %v", err)
	}
	if _, err := adapter.RollbackOrCompensate(context.Background(), exec, AdminRollbackRequest{}); err != nil {
		t.Fatalf("rollback failed: %v", err)
	}
}

func TestCLIAdapterExecutorAnnotatesAuditMetadata(t *testing.T) {
	adapter := NewCLIAdapterExecutor("gh-custom")
	exec := fakeAdminExecutor{}

	snapshot, err := adapter.Inspect(context.Background(), exec, AdminInspectRequest{ResourceID: "site"})
	if err != nil {
		t.Fatalf("inspect failed: %v", err)
	}
	if snapshot.Metadata["adapter_backed"] != string(AdapterCLI) || snapshot.Metadata["adapter_binary"] != "gh-custom" {
		t.Fatalf("expected cli metadata on snapshot, got %+v", snapshot.Metadata)
	}

	validation, err := adapter.Validate(context.Background(), exec, AdminValidationRequest{ResourceID: "site"})
	if err != nil {
		t.Fatalf("validate failed: %v", err)
	}
	if validation.Metadata["adapter_binary"] != "gh-custom" {
		t.Fatalf("expected cli binary metadata on validation, got %+v", validation.Metadata)
	}

	rollback, err := adapter.RollbackOrCompensate(context.Background(), exec, AdminRollbackRequest{})
	if err != nil {
		t.Fatalf("rollback failed: %v", err)
	}
	if rollback.Metadata["adapter_transport"] != "cli" {
		t.Fatalf("expected cli transport metadata on rollback, got %+v", rollback.Metadata)
	}
}

func TestResolveBrowserStubDriverSupportsAliases(t *testing.T) {
	driver := ResolveBrowserStubDriver("stub-driver")
	if driver.Name() != "default" {
		t.Fatalf("expected stub-driver alias to resolve to default, got %s", driver.Name())
	}
	playwright := ResolveBrowserStubDriver("playwright")
	if playwright.Name() != "playwright" {
		t.Fatalf("expected playwright driver, got %s", playwright.Name())
	}
}

func TestBrowserStubAdapterExecutorProducesAuditFriendlyResults(t *testing.T) {
	adapter := NewBrowserStubAdapterExecutor("playwright")
	exec := NewStubAdminExecutor("pages")
	if !adapter.CanHandle("pages") {
		t.Fatal("expected browser stub to handle capability")
	}
	if adapter.Kind() != AdapterBrowser {
		t.Fatalf("expected browser kind, got %s", adapter.Kind())
	}

	mutation, err := adapter.Mutate(context.Background(), exec, AdminMutationRequest{
		Operation:  "update",
		ResourceID: "site",
	})
	if err != nil {
		t.Fatalf("mutate failed: %v", err)
	}
	if mutation.Metadata["adapter_backed"] != string(AdapterBrowser) {
		t.Fatalf("expected browser metadata, got %+v", mutation.Metadata)
	}

	validation, err := adapter.Validate(context.Background(), exec, AdminValidationRequest{
		ResourceID: "site",
		Mutation:   mutation,
	})
	if err != nil {
		t.Fatalf("validate failed: %v", err)
	}
	if validation.OK {
		t.Fatalf("expected browser stub validation to require operator action, got %+v", validation)
	}
	if validation.Metadata["operator_validation_required"] != "true" {
		t.Fatalf("expected validation metadata, got %+v", validation.Metadata)
	}

	rollback, err := adapter.RollbackOrCompensate(context.Background(), exec, AdminRollbackRequest{Mutation: mutation})
	if err != nil {
		t.Fatalf("rollback failed: %v", err)
	}
	if rollback.Compensation == nil || rollback.Compensation.Kind != "manual_restore_required" {
		t.Fatalf("expected manual compensation, got %+v", rollback.Compensation)
	}
	if rollback.Metadata["browser_driver"] != "playwright" {
		t.Fatalf("expected rollback metadata to retain driver, got %+v", rollback.Metadata)
	}

	var state map[string]any
	if err := json.Unmarshal(mutation.After.State, &state); err != nil {
		t.Fatalf("decode stub state: %v", err)
	}
	if state["browser_driver"] != "playwright" {
		t.Fatalf("expected driver in stub state, got %+v", state)
	}
}
