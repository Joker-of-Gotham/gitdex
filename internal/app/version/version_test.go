package version_test

import (
	"testing"

	"github.com/your-org/gitdex/internal/app/version"
)

func TestStarterVersionConstantsAreDefined(t *testing.T) {
	if version.CLIName == "" {
		t.Fatal("CLIName should not be empty")
	}
	if version.DaemonName == "" {
		t.Fatal("DaemonName should not be empty")
	}
	if version.Version == "" {
		t.Fatal("Version should not be empty")
	}
}
