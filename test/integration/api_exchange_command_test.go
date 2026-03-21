package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/your-org/gitdex/internal/cli/command"
)

func TestAPIExchangeValidateRequiresFile(t *testing.T) {
	root := command.NewRootCommand()
	root.SetArgs([]string{"api", "exchange", "validate"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected error when file is missing")
	}
}

func TestAPIExchangeImportRequiresFile(t *testing.T) {
	root := command.NewRootCommand()
	root.SetArgs([]string{"api", "exchange", "import"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected error when file is missing")
	}
}

func TestAPIExchangeExportRuns(t *testing.T) {
	root := command.NewRootCommand()
	root.SetArgs([]string{"api", "exchange", "export", "--type", "plans", "--format", "json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("api exchange export failed: %v", err)
	}
}

func TestAPIExchangeValidateWithValidFile(t *testing.T) {
	dir := t.TempDir()
	fpath := filepath.Join(dir, "exchange.json")
	content := `{"format":"json","api_version":"v1","schema_version":"1","payload_type":"plans","data":{},"created_at":"2026-03-19T12:00:00Z"}`
	if err := os.WriteFile(fpath, []byte(content), 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	root := command.NewRootCommand()
	root.SetArgs([]string{"api", "exchange", "validate", "--file", fpath})
	if err := root.Execute(); err != nil {
		t.Fatalf("api exchange validate failed: %v", err)
	}
}
