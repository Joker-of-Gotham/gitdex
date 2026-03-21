package collaboration

import (
	"context"
	"testing"
	"time"
)

func TestRuleBasedTriageEngine_Triage_NilObject(t *testing.T) {
	engine := NewRuleBasedTriageEngine()
	_, err := engine.Triage(context.Background(), nil)
	if err == nil {
		t.Error("expected error for nil object")
	}
}

func TestRuleBasedTriageEngine_Triage_SecurityLabel(t *testing.T) {
	engine := NewRuleBasedTriageEngine()
	obj := &CollaborationObject{
		RepoOwner: "owner",
		RepoName:  "repo",
		Number:    1,
		Labels:    []string{"security"},
		State:     "open",
	}
	res, err := engine.Triage(context.Background(), obj)
	if err != nil {
		t.Fatalf("Triage: %v", err)
	}
	if res.Priority != TriageCritical {
		t.Errorf("priority = %q, want critical", res.Priority)
	}
	if res.Reason != "security label" {
		t.Errorf("reason = %q, want security label", res.Reason)
	}
}

func TestRuleBasedTriageEngine_Triage_BugLabel(t *testing.T) {
	engine := NewRuleBasedTriageEngine()
	obj := &CollaborationObject{
		RepoOwner: "owner",
		RepoName:  "repo",
		Number:    2,
		Labels:    []string{"bug"},
		State:     "open",
	}
	res, err := engine.Triage(context.Background(), obj)
	if err != nil {
		t.Fatalf("Triage: %v", err)
	}
	if res.Priority != TriageHigh {
		t.Errorf("priority = %q, want high", res.Priority)
	}
}

func TestRuleBasedTriageEngine_Triage_StaleLabel(t *testing.T) {
	engine := NewRuleBasedTriageEngine()
	obj := &CollaborationObject{
		RepoOwner: "owner",
		RepoName:  "repo",
		Number:    3,
		Labels:    []string{"stale"},
		State:     "open",
	}
	res, err := engine.Triage(context.Background(), obj)
	if err != nil {
		t.Fatalf("Triage: %v", err)
	}
	if res.Priority != TriageLow {
		t.Errorf("priority = %q, want low", res.Priority)
	}
}

func TestRuleBasedTriageEngine_Triage_DefaultMedium(t *testing.T) {
	engine := NewRuleBasedTriageEngine()
	obj := &CollaborationObject{
		RepoOwner: "owner",
		RepoName:  "repo",
		Number:    4,
		Labels:    []string{"enhancement"},
		State:     "open",
	}
	res, err := engine.Triage(context.Background(), obj)
	if err != nil {
		t.Fatalf("Triage: %v", err)
	}
	if res.Priority != TriageMedium {
		t.Errorf("priority = %q, want medium", res.Priority)
	}
}

func TestRuleBasedTriageEngine_Summarize(t *testing.T) {
	engine := NewRuleBasedTriageEngine()
	objects := []*CollaborationObject{
		{RepoOwner: "o", RepoName: "r", Number: 1, Labels: []string{"bug"}, State: "open"},
		{RepoOwner: "o", RepoName: "r", Number: 2, Labels: []string{"docs"}, State: "open"},
	}
	summary, err := engine.Summarize(context.Background(), objects, "7d")
	if err != nil {
		t.Fatalf("Summarize: %v", err)
	}
	if summary.TotalObjects != 2 {
		t.Errorf("total_objects = %d, want 2", summary.TotalObjects)
	}
	if summary.Period != "7d" {
		t.Errorf("period = %q, want 7d", summary.Period)
	}
	if summary.RepoOwner != "o" || summary.RepoName != "r" {
		t.Errorf("repo = %s/%s, want o/r", summary.RepoOwner, summary.RepoName)
	}
	if len(summary.ByPriority) == 0 {
		t.Error("by_priority should not be empty")
	}
	if summary.GeneratedAt.IsZero() {
		t.Error("generated_at should be set")
	}
}

func TestObjectRef(t *testing.T) {
	obj := &CollaborationObject{RepoOwner: "o", RepoName: "r", Number: 42}
	if got := ObjectRef(obj); got != "o/r#42" {
		t.Errorf("ObjectRef(issue) = %q, want o/r#42", got)
	}

	pr := &CollaborationObject{RepoOwner: "o", RepoName: "r", Number: 42, ObjectType: ObjectTypePullRequest}
	if got := ObjectRef(pr); got != "o/r#pr/42" {
		t.Errorf("ObjectRef(pr) = %q, want o/r#pr/42", got)
	}

	if got := ObjectRef(nil); got != "" {
		t.Errorf("ObjectRef(nil) = %q, want empty", got)
	}
}

func TestActivitySummary_Structure(t *testing.T) {
	s := ActivitySummary{
		RepoOwner:    "o",
		RepoName:     "r",
		Period:       "7d",
		TotalObjects: 5,
		ByType:       map[string]int{"issue": 3, "pull_request": 2},
		ByPriority:   map[string]int{"high": 1, "medium": 4},
		TopItems:     []TriageResult{{ObjectRef: "o/r#1", Priority: TriageHigh}},
		GeneratedAt:  time.Now().UTC(),
	}
	if s.TotalObjects != 5 {
		t.Error("ActivitySummary structure check failed")
	}
}
