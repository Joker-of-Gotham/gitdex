package conformance

import (
	"encoding/json"
	"testing"

	"github.com/your-org/gitdex/internal/gitops"
)

func TestHygieneTask_JSONRoundTrip(t *testing.T) {
	original := &gitops.HygieneTask{
		Action:          gitops.HygienePruneRemoteBranches,
		Description:     "Remove remote-tracking branches",
		RiskLevel:       "low",
		Reversible:      true,
		EstimatedImpact: "Reduces local ref clutter",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded gitops.HygieneTask
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.Action != original.Action {
		t.Errorf("Action: got %q, want %q", decoded.Action, original.Action)
	}
	if decoded.Description != original.Description {
		t.Errorf("Description: got %q, want %q", decoded.Description, original.Description)
	}
	if decoded.RiskLevel != original.RiskLevel {
		t.Errorf("RiskLevel: got %q, want %q", decoded.RiskLevel, original.RiskLevel)
	}
	if decoded.Reversible != original.Reversible {
		t.Errorf("Reversible: got %v, want %v", decoded.Reversible, original.Reversible)
	}
}

func TestHygieneResult_JSONRoundTrip(t *testing.T) {
	original := &gitops.HygieneResult{
		Success:          true,
		Action:           gitops.HygieneCleanUntracked,
		FilesAffected:    3,
		BranchesAffected: 0,
		Summary:          "Completed successfully",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded gitops.HygieneResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.Success != original.Success {
		t.Errorf("Success: got %v, want %v", decoded.Success, original.Success)
	}
	if decoded.Action != original.Action {
		t.Errorf("Action: got %q, want %q", decoded.Action, original.Action)
	}
	if decoded.FilesAffected != original.FilesAffected {
		t.Errorf("FilesAffected: got %d, want %d", decoded.FilesAffected, original.FilesAffected)
	}
	if decoded.BranchesAffected != original.BranchesAffected {
		t.Errorf("BranchesAffected: got %d, want %d", decoded.BranchesAffected, original.BranchesAffected)
	}
}

func TestHygieneResult_Failure_JSONContract(t *testing.T) {
	original := &gitops.HygieneResult{
		Success:      false,
		Action:       gitops.HygieneGCAggressive,
		ErrorMessage: "repository path is required",
		Summary:      "Retry with a valid repo path",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded gitops.HygieneResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.Success != false {
		t.Errorf("Success: got %v, want false", decoded.Success)
	}
	if decoded.ErrorMessage != original.ErrorMessage {
		t.Errorf("ErrorMessage: got %q, want %q", decoded.ErrorMessage, original.ErrorMessage)
	}
	if decoded.Summary != original.Summary {
		t.Errorf("Summary: got %q, want %q", decoded.Summary, original.Summary)
	}
}
