package state

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	gitstate "github.com/your-org/gitdex/internal/platform/git"
	"github.com/your-org/gitdex/internal/state/repo"
)

func initTestRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}
	dir := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Test",
			"GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=Test",
			"GIT_COMMITTER_EMAIL=test@test.com",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %s\n%v", args, out, err)
		}
	}
	run("init", "-b", "main")
	run("config", "user.email", "test@test.com")
	run("config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	run("add", ".")
	run("commit", "-m", "init")
	return dir
}

func TestAssemble_LocalOnly_NoGitHub(t *testing.T) {
	dir := initTestRepo(t)

	assembler := NewAssembler(nil)
	summary, err := assembler.Assemble(context.Background(), "owner", "repo", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if summary.Local.Label != repo.Healthy {
		t.Errorf("local label = %q, want %q", summary.Local.Label, repo.Healthy)
	}
	if summary.Remote.Label != repo.Unknown {
		t.Errorf("remote label = %q, want %q", summary.Remote.Label, repo.Unknown)
	}
	if summary.Collaboration.Label != repo.Unknown {
		t.Errorf("collaboration label = %q, want %q", summary.Collaboration.Label, repo.Unknown)
	}
	if summary.Workflows.Label != repo.Unknown {
		t.Errorf("workflows label = %q, want %q", summary.Workflows.Label, repo.Unknown)
	}
	if summary.Deployments.Label != repo.Unknown {
		t.Errorf("deployments label = %q, want %q", summary.Deployments.Label, repo.Unknown)
	}
	if summary.OverallLabel != repo.Unknown {
		t.Errorf("overall label = %q, want %q", summary.OverallLabel, repo.Unknown)
	}
}

func TestAssemble_NoRepoPath(t *testing.T) {
	assembler := NewAssembler(nil)
	summary, err := assembler.Assemble(context.Background(), "owner", "repo", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary.Local.Label != repo.Unknown {
		t.Errorf("local label = %q, want %q", summary.Local.Label, repo.Unknown)
	}
}

func TestAssembleLocal_CleanUpToDate(t *testing.T) {
	gs := &gitstate.LocalGitState{
		Branch:  "main",
		HeadSHA: "abc123",
		IsClean: true,
	}
	local := assembleLocal(gs)
	if local.Label != repo.Healthy {
		t.Errorf("label = %q, want %q", local.Label, repo.Healthy)
	}
}

func TestAssembleLocal_Dirty(t *testing.T) {
	gs := &gitstate.LocalGitState{
		Branch:     "main",
		IsClean:    false,
		DirtyCount: 3,
	}
	local := assembleLocal(gs)
	if local.Label != repo.Drifting {
		t.Errorf("label = %q, want %q", local.Label, repo.Drifting)
	}
}

func TestAssembleLocal_Detached(t *testing.T) {
	gs := &gitstate.LocalGitState{
		Branch:     "abc123",
		IsDetached: true,
		IsClean:    true,
	}
	local := assembleLocal(gs)
	if local.Label != repo.Degraded {
		t.Errorf("label = %q, want %q", local.Label, repo.Degraded)
	}
}

func TestAssembleLocal_Behind(t *testing.T) {
	gs := &gitstate.LocalGitState{
		Branch:  "main",
		IsClean: true,
		Behind:  5,
	}
	local := assembleLocal(gs)
	if local.Label != repo.Drifting {
		t.Errorf("label = %q, want %q", local.Label, repo.Drifting)
	}
}

func TestAssembleLocal_Ahead(t *testing.T) {
	gs := &gitstate.LocalGitState{
		Branch:  "main",
		IsClean: true,
		Ahead:   3,
	}
	local := assembleLocal(gs)
	if local.Label != repo.Drifting {
		t.Errorf("label = %q, want %q", local.Label, repo.Drifting)
	}
}

func TestAssembleCollaboration_Healthy(t *testing.T) {
	cs := assembleCollaboration(nil)
	if cs.Label != repo.Healthy {
		t.Errorf("label = %q, want %q", cs.Label, repo.Healthy)
	}
}

func TestAssembleCollaboration_NeedsReview(t *testing.T) {
	prs := []repo.PullRequestSummary{
		{Number: 1, NeedsReview: true},
	}
	cs := assembleCollaboration(prs)
	if cs.Label != repo.Drifting {
		t.Errorf("label = %q, want %q", cs.Label, repo.Drifting)
	}
}

func TestAssembleCollaboration_Stale(t *testing.T) {
	prs := []repo.PullRequestSummary{
		{Number: 1, StaleDays: 30},
	}
	cs := assembleCollaboration(prs)
	if cs.Label != repo.Degraded {
		t.Errorf("label = %q, want %q", cs.Label, repo.Degraded)
	}
}

func TestAssembleWorkflows_AllGreen(t *testing.T) {
	runs := []repo.WorkflowRunSummary{
		{Name: "CI", Conclusion: "success"},
	}
	ws := assembleWorkflows(runs)
	if ws.Label != repo.Healthy {
		t.Errorf("label = %q, want %q", ws.Label, repo.Healthy)
	}
}

func TestAssembleWorkflows_Failure(t *testing.T) {
	runs := []repo.WorkflowRunSummary{
		{Name: "CI", Conclusion: "failure"},
	}
	ws := assembleWorkflows(runs)
	if ws.Label != repo.Degraded {
		t.Errorf("label = %q, want %q", ws.Label, repo.Degraded)
	}
}

func TestAssembleWorkflows_NoRuns(t *testing.T) {
	ws := assembleWorkflows(nil)
	if ws.Label != repo.Unknown {
		t.Errorf("label = %q, want %q", ws.Label, repo.Unknown)
	}
}

func TestAssembleDeployments_Healthy(t *testing.T) {
	deps := []repo.DeploymentSummary{
		{Environment: "prod", State: "success"},
	}
	ds := assembleDeployments(deps)
	if ds.Label != repo.Healthy {
		t.Errorf("label = %q, want %q", ds.Label, repo.Healthy)
	}
}

func TestAssembleDeployments_Failure(t *testing.T) {
	deps := []repo.DeploymentSummary{
		{Environment: "prod", State: "failure"},
	}
	ds := assembleDeployments(deps)
	if ds.Label != repo.Degraded {
		t.Errorf("label = %q, want %q", ds.Label, repo.Degraded)
	}
}

func TestAssembleDeployments_NoDeps(t *testing.T) {
	ds := assembleDeployments(nil)
	if ds.Label != repo.Unknown {
		t.Errorf("label = %q, want %q", ds.Label, repo.Unknown)
	}
}

func TestAssembleRisks_CIFailure(t *testing.T) {
	s := &repo.RepoSummary{
		Local:     repo.LocalState{Label: repo.Healthy, IsClean: true},
		Workflows: repo.WorkflowState{Label: repo.Degraded, Detail: "CI failed"},
	}
	risks := assembleRisks(s)

	found := false
	for _, r := range risks {
		if r.Severity == repo.RiskHigh && r.Description == "CI/CD pipeline failures" {
			found = true
		}
	}
	if !found {
		t.Error("expected high-severity CI risk")
	}
}

func TestAssembleNextActions_BehindUpstream(t *testing.T) {
	s := &repo.RepoSummary{
		Local: repo.LocalState{Label: repo.Drifting, IsClean: true, Behind: 3},
	}
	actions := assembleNextActions(s)

	found := false
	for _, a := range actions {
		if a.Action == "sync with upstream" {
			found = true
		}
	}
	if !found {
		t.Error("expected 'sync with upstream' action")
	}
}
