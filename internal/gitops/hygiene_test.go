package gitops

import (
	"testing"
)

func TestSupportedHygieneTasks(t *testing.T) {
	tasks := SupportedHygieneTasks()
	if len(tasks) != 4 {
		t.Fatalf("expected 4 tasks, got %d", len(tasks))
	}

	expected := map[HygieneAction]bool{
		HygienePruneRemoteBranches:  true,
		HygieneGCAggressive:         true,
		HygieneCleanUntracked:       true,
		HygieneRemoveMergedBranches: true,
	}
	for _, task := range tasks {
		if !expected[task.Action] {
			t.Errorf("unexpected action %q", task.Action)
		}
		if task.Description == "" {
			t.Errorf("task %q has empty description", task.Action)
		}
		if task.RiskLevel == "" {
			t.Errorf("task %q has empty risk_level", task.Action)
		}
		if task.EstimatedImpact == "" {
			t.Errorf("task %q has empty estimated_impact", task.Action)
		}
	}
}

func TestHygieneTask_AllActionsPresent(t *testing.T) {
	tasks := SupportedHygieneTasks()
	actions := []HygieneAction{
		HygienePruneRemoteBranches,
		HygieneGCAggressive,
		HygieneCleanUntracked,
		HygieneRemoveMergedBranches,
	}
	for _, a := range actions {
		found := false
		for _, t := range tasks {
			if t.Action == a {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("action %q not in SupportedHygieneTasks", a)
		}
	}
}
