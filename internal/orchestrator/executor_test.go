package orchestrator

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/your-org/gitdex/internal/planning"
)

func seedApprovedPlan(planStore planning.PlanStore, steps []planning.PlanStep) *planning.Plan {
	p := &planning.Plan{
		PlanID:    "plan_exec_test",
		Status:    planning.PlanApproved,
		RiskLevel: planning.RiskLow,
		Intent: planning.PlanIntent{
			Source:     "command",
			RawInput:   "test goal",
			ActionType: "test",
		},
		Scope: planning.PlanScope{Owner: "org", Repo: "repo"},
		Steps: steps,
	}
	_ = planStore.Save(p)
	return p
}

func initGitRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
		}
	}

	run("init", "-b", "main")
	run("config", "user.name", "Test User")
	run("config", "user.email", "test@example.com")

	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("seed\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	run("add", "README.md")
	run("commit", "-m", "seed")
	return root
}

func TestExecutor_StartFromPlan_Success(t *testing.T) {
	planStore := planning.NewMemoryPlanStore()
	taskStore := NewMemoryTaskStore()
	seedApprovedPlan(planStore, []planning.PlanStep{
		{Sequence: 1, Action: "git.status", Target: "", Description: "status", RiskLevel: planning.RiskLow, Reversible: true},
		{Sequence: 2, Action: "log", Target: "", Description: "show log", RiskLevel: planning.RiskLow, Reversible: true},
	})

	exec := NewExecutor(taskStore, planStore)
	task, err := exec.StartFromPlan(context.Background(), "plan_exec_test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if task.TaskID == "" {
		t.Error("expected non-empty task ID")
	}
	if task.CorrelationID == "" {
		t.Error("expected non-empty correlation ID")
	}
	if task.Status != TaskQueued {
		t.Errorf("got status %q, want %q", task.Status, TaskQueued)
	}
	if len(task.Steps) != 2 {
		t.Errorf("got %d steps, want 2", len(task.Steps))
	}
}

func TestExecutor_StartFromPlan_NotApproved(t *testing.T) {
	planStore := planning.NewMemoryPlanStore()
	taskStore := NewMemoryTaskStore()
	p := &planning.Plan{
		PlanID: "plan_not_approved",
		Status: planning.PlanReviewRequired,
		Intent: planning.PlanIntent{Source: "command", RawInput: "test", ActionType: "test"},
		Scope:  planning.PlanScope{Owner: "org", Repo: "repo"},
	}
	_ = planStore.Save(p)

	exec := NewExecutor(taskStore, planStore)
	_, err := exec.StartFromPlan(context.Background(), "plan_not_approved")
	if err == nil {
		t.Fatal("expected error for non-approved plan")
	}
}

func TestExecutor_Execute_Success(t *testing.T) {
	repoRoot := initGitRepo(t)
	planStore := planning.NewMemoryPlanStore()
	taskStore := NewMemoryTaskStore()
	seedApprovedPlan(planStore, []planning.PlanStep{
		{Sequence: 1, Action: "file.write", Target: "notes/todo.txt", Description: "hello orchestrator\n", RiskLevel: planning.RiskLow, Reversible: true},
		{Sequence: 2, Action: "git.add", Target: "notes/todo.txt", Description: "stage file", RiskLevel: planning.RiskLow, Reversible: true},
		{Sequence: 3, Action: "git.commit", Target: "add todo", Description: "commit file", RiskLevel: planning.RiskLow, Reversible: true},
	})

	exec := NewExecutorWithOptions(taskStore, planStore, ExecutorOptions{RepoRoot: repoRoot})
	task, _ := exec.StartFromPlan(context.Background(), "plan_exec_test")

	err := exec.Execute(context.Background(), task)
	if err != nil {
		t.Fatalf("execution error: %v", err)
	}

	latest, _ := taskStore.GetTask(task.TaskID)
	if latest.Status != TaskSucceeded {
		t.Errorf("got status %q, want %q", latest.Status, TaskSucceeded)
	}

	for _, s := range latest.Steps {
		if s.Status != StepSucceeded {
			t.Errorf("step %d: got %q, want %q", s.Sequence, s.Status, StepSucceeded)
		}
		if strings.TrimSpace(s.Output) == "" {
			t.Errorf("step %d: expected output", s.Sequence)
		}
	}

	content, err := os.ReadFile(filepath.Join(repoRoot, "notes", "todo.txt"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(content) != "hello orchestrator\n" {
		t.Fatalf("content = %q", string(content))
	}

	events, _ := taskStore.GetEvents(task.TaskID)
	if len(events) == 0 {
		t.Error("expected events to be recorded")
	}

	plan, _ := planStore.Get("plan_exec_test")
	if plan.Status != planning.PlanCompleted {
		t.Errorf("plan status: got %q, want %q", plan.Status, planning.PlanCompleted)
	}
}

func TestExecutor_Execute_Cancelled(t *testing.T) {
	repoRoot := initGitRepo(t)
	planStore := planning.NewMemoryPlanStore()
	taskStore := NewMemoryTaskStore()
	seedApprovedPlan(planStore, []planning.PlanStep{
		{Sequence: 1, Action: "git.status", Target: "", Description: "status", RiskLevel: planning.RiskLow, Reversible: true},
	})

	exec := NewExecutorWithOptions(taskStore, planStore, ExecutorOptions{RepoRoot: repoRoot})
	task, _ := exec.StartFromPlan(context.Background(), "plan_exec_test")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := exec.Execute(ctx, task)
	if err == nil {
		t.Fatal("expected cancellation error")
	}

	latest, _ := taskStore.GetTask(task.TaskID)
	if latest.Status != TaskCancelled {
		t.Errorf("got status %q, want %q", latest.Status, TaskCancelled)
	}
}

func TestExecutor_Execute_NotQueued(t *testing.T) {
	planStore := planning.NewMemoryPlanStore()
	taskStore := NewMemoryTaskStore()

	task := &Task{TaskID: "t_not_queued", Status: TaskExecuting}
	_ = taskStore.SaveTask(task)

	exec := NewExecutor(taskStore, planStore)
	err := exec.Execute(context.Background(), task)
	if err == nil {
		t.Fatal("expected error for non-queued task")
	}
}

func TestExecutor_StartFromPlan_ZeroSteps(t *testing.T) {
	planStore := planning.NewMemoryPlanStore()
	taskStore := NewMemoryTaskStore()

	p := &planning.Plan{
		PlanID:    "plan_no_steps",
		Status:    planning.PlanApproved,
		RiskLevel: planning.RiskLow,
		Intent:    planning.PlanIntent{Source: "command", RawInput: "test", ActionType: "test"},
		Scope:     planning.PlanScope{Owner: "org", Repo: "repo"},
		Steps:     []planning.PlanStep{},
	}
	_ = planStore.Save(p)

	exec := NewExecutor(taskStore, planStore)
	_, err := exec.StartFromPlan(context.Background(), "plan_no_steps")
	if err == nil {
		t.Fatal("expected error for plan with 0 steps")
	}
}

func TestExecutor_Execute_ConcurrentBlocked(t *testing.T) {
	repoRoot := initGitRepo(t)
	planStore := planning.NewMemoryPlanStore()
	taskStore := NewMemoryTaskStore()
	seedApprovedPlan(planStore, []planning.PlanStep{
		{Sequence: 1, Action: "git.status", Target: "", Description: "status", RiskLevel: planning.RiskLow, Reversible: true},
	})

	exec := NewExecutorWithOptions(taskStore, planStore, ExecutorOptions{RepoRoot: repoRoot})
	task, _ := exec.StartFromPlan(context.Background(), "plan_exec_test")

	exec.mu.Lock()
	exec.running[task.TaskID] = true
	exec.mu.Unlock()

	err := exec.Execute(context.Background(), task)
	if err == nil {
		t.Fatal("expected error for concurrent execution")
	}
}

func TestExecutor_Execute_UsesWorkspaceCloneDiscovery(t *testing.T) {
	workspaceRoot := t.TempDir()
	repoRoot := filepath.Join(workspaceRoot, "repo")

	if err := os.MkdirAll(repoRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	cmd := exec.Command("git", "init", "-b", "main")
	cmd.Dir = repoRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v\n%s", err, string(out))
	}
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = repoRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git config name failed: %v\n%s", err, string(out))
	}
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = repoRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git config email failed: %v\n%s", err, string(out))
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "README.md"), []byte("seed\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	cmd = exec.Command("git", "add", "README.md")
	cmd.Dir = repoRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git add failed: %v\n%s", err, string(out))
	}
	cmd = exec.Command("git", "commit", "-m", "seed")
	cmd.Dir = repoRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit failed: %v\n%s", err, string(out))
	}
	cmd = exec.Command("git", "remote", "add", "origin", "https://github.com/org/repo.git")
	cmd.Dir = repoRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git remote add failed: %v\n%s", err, string(out))
	}

	planStore := planning.NewMemoryPlanStore()
	taskStore := NewMemoryTaskStore()
	seedApprovedPlan(planStore, []planning.PlanStep{
		{Sequence: 1, Action: "git.status", Target: "", Description: "status", RiskLevel: planning.RiskLow, Reversible: true},
	})

	plan, _ := planStore.Get("plan_exec_test")
	plan.Scope.Owner = "org"
	plan.Scope.Repo = "repo"
	_ = planStore.Save(plan)

	exec := NewExecutorWithOptions(taskStore, planStore, ExecutorOptions{WorkspaceRoots: []string{workspaceRoot}})
	task, _ := exec.StartFromPlan(context.Background(), "plan_exec_test")
	if err := exec.Execute(context.Background(), task); err != nil {
		t.Fatalf("execution via workspace root failed: %v", err)
	}
}
