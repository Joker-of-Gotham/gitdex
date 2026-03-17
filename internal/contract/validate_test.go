package contract

import "testing"

func TestValidateSuggestion_OK(t *testing.T) {
	s := SuggestionItem{
		Name:   "Commit staged changes",
		Reason: "Persist progress",
		Action: ActionSpec{
			Type:    "git_command",
			Command: `git commit -m "test"`,
		},
	}
	if err := ValidateSuggestion(s); err != nil {
		t.Fatalf("expected valid suggestion, got error: %v", err)
	}
}

func TestValidateSuggestion_InvalidAction(t *testing.T) {
	s := SuggestionItem{
		Name:   "bad",
		Reason: "bad",
		Action: ActionSpec{
			Type: "file_write",
		},
	}
	if err := ValidateSuggestion(s); err == nil {
		t.Fatalf("expected error for invalid file_write action")
	}
}

func TestValidateAction_FileWriteAppendRequiresContent(t *testing.T) {
	a := ActionSpec{
		Type:     "file_write",
		FilePath: "README.md",
		FileOp:   "append",
	}
	if err := ValidateAction(a); err == nil {
		t.Fatalf("expected validation error")
	}
}

func TestValidateAction_CommandPlaceholderRejected(t *testing.T) {
	a := ActionSpec{
		Type:    "shell_command",
		Command: "mkdir .github/...",
	}
	if err := ValidateAction(a); err == nil {
		t.Fatalf("expected validation error for placeholder command")
	}
}

func TestValidateAction_GoWildcardAllowed(t *testing.T) {
	a := ActionSpec{
		Type:    "shell_command",
		Command: "go test ./...",
	}
	if err := ValidateAction(a); err != nil {
		t.Fatalf("go wildcard should remain valid, got error: %v", err)
	}
}

func TestValidateAction_FilePathPlaceholderRejected(t *testing.T) {
	a := ActionSpec{
		Type:        "file_write",
		FilePath:    ".github/...",
		FileOp:      "create",
		FileContent: "name: ci\n",
	}
	if err := ValidateAction(a); err == nil {
		t.Fatalf("expected validation error for placeholder file path")
	}
}
