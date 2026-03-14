package prompt

import (
	"strings"
	"testing"
)

func TestAnalyzeSystemDocumentsPlatformExecSchema(t *testing.T) {
	required := []string{
		`"platform_exec"`,
		`"capability_id"`,
		`"flow"`,
		`"validate_payload"`,
		`"rollback_payload"`,
	}
	for _, fragment := range required {
		if !strings.Contains(analyzeSystem("en"), fragment) {
			t.Fatalf("missing platform_exec fragment %q", fragment)
		}
	}
}
