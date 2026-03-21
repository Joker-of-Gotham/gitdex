package integration_test

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/your-org/gitdex/internal/autonomy"
	"github.com/your-org/gitdex/internal/state/repo"
	"github.com/your-org/gitdex/internal/tui/components"
	"github.com/your-org/gitdex/internal/tui/theme"
	"github.com/your-org/gitdex/internal/tui/views"
)

func assertContains(t *testing.T, output, substr, desc string) {
	t.Helper()
	if !strings.Contains(output, substr) {
		t.Fatalf("%s: output should contain %q (got %d chars)", desc, substr, len(output))
	}
}

func TestE2E_ChatView_StreamingFlow(t *testing.T) {
	th := theme.NewTheme(true)
	v := views.NewChatView(&th)
	v.SetSize(120, 32)

	v.AppendMessage(views.Message{Role: views.RoleUser, Content: "hello", Timestamp: time.Now()})
	assertContains(t, v.Render(), "hello", "chat should show user message")

	v.BeginStream()
	if !v.IsStreaming() {
		t.Fatal("BeginStream should enable streaming")
	}

	v.Update(views.StreamChunkMsg{Content: "hello, ", Done: false})
	v.Update(views.StreamChunkMsg{Content: "I am Gitdex AI", Done: false})

	output := v.Render()
	assertContains(t, output, "hello,", "chat should show first stream chunk")
	assertContains(t, output, "Gitdex AI", "chat should show second stream chunk")
	assertContains(t, output, "...", "chat should show stream continuation marker")

	v.Update(views.StreamChunkMsg{Content: ".", Done: true})
	v.EndStream()
	if v.IsStreaming() {
		t.Fatal("EndStream should disable streaming")
	}
	if strings.Contains(v.Render(), "...") {
		t.Fatal("stream continuation marker should be removed after stream end")
	}
}

func TestE2E_ComposerPasteAndSubmit(t *testing.T) {
	th := theme.NewTheme(true)
	c := components.NewComposer(&th)
	c.SetWidth(120)
	c.SetFocused(true)

	content := "line1\nline2\nline3"
	c.Update(tea.PasteMsg{Content: content})
	if c.Value() != content {
		t.Fatalf("composer paste: got %q want %q", c.Value(), content)
	}

	cmd := c.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("composer enter should dispatch submit command")
	}
	msg, ok := cmd().(components.SubmitMsg)
	if !ok {
		t.Fatalf("submit message type = %T", cmd())
	}
	if msg.Input != content {
		t.Fatalf("submit input = %q want %q", msg.Input, content)
	}
}

func TestE2E_ReposView_EmptyState(t *testing.T) {
	th := theme.NewTheme(true)
	v := views.NewReposView(&th)
	v.SetSize(120, 40)

	assertContains(t, v.Render(), "Loading repositories", "empty repos should show loading placeholder")
	if v.SelectedRepo() != nil {
		t.Fatal("selected repo should be nil while loading")
	}

	v.SetItems(nil)
	assertContains(t, v.Render(), "No repositories detected yet", "zero-item repos should show discovery placeholder")
}

func TestE2E_ReposView_LocalStatusMarkers(t *testing.T) {
	th := theme.NewTheme(true)
	v := views.NewReposView(&th)
	v.SetSize(160, 40)

	repos := []views.RepoListItem{
		{Name: "local-repo", FullName: "x/local-repo", IsLocal: true, LocalPaths: []string{"/path/to/local"}},
		{Name: "remote-only", FullName: "x/remote-only", IsLocal: false},
	}
	v.SetItems(repos)

	output := v.Render()
	assertContains(t, output, "local", "repos should show local availability marker")
	assertContains(t, output, "remote", "repos should show remote availability marker")
}

func TestE2E_DashboardOverview_Render(t *testing.T) {
	th := theme.NewTheme(true)
	v := views.NewDashboardView(&th)
	v.SetSize(140, 40)

	v.Update(views.StatusDataMsg{Summary: &repo.RepoSummary{
		Owner:        "acme",
		Repo:         "gitdex",
		OverallLabel: repo.Healthy,
		Timestamp:    time.Now(),
		Local:        repo.LocalState{Label: repo.Healthy, Detail: "workspace is clean", Branch: "main"},
		Remote:       repo.RemoteState{Label: repo.Healthy, Detail: "remote reachable", DefaultBranch: "main"},
		Collaboration: repo.CollaborationSignals{
			Label:          repo.Healthy,
			Detail:         "collaboration healthy",
			OpenPRCount:    2,
			OpenIssueCount: 3,
		},
		Workflows:   repo.WorkflowState{Label: repo.Healthy, Detail: "runs passing"},
		Deployments: repo.DeploymentState{Label: repo.Unknown, Detail: "deployment unknown"},
	}})

	output := v.Render()
	assertContains(t, output, "Health Matrix", "dashboard overview should show health matrix")
	assertContains(t, output, "acme/gitdex", "dashboard overview should show repo name")
	assertContains(t, output, "workspace is clean", "dashboard overview should show full local detail")
}

func TestE2E_WorkspaceTabs_Render(t *testing.T) {
	th := theme.NewTheme(true)
	v := views.NewWorkspaceView(&th)
	v.SetSize(120, 32)

	v.Plans().SetPlans([]views.PlanSummary{{
		Title:          "Stabilize release train",
		Status:         "active",
		Scope:          "repo health and release hygiene",
		StepCount:      5,
		CompletedSteps: 2,
		RiskLevel:      "medium",
	}})
	v.Tasks().SetTasks([]views.TaskItem{{
		ID:           "TASK-1",
		Title:        "Resolve flaky release job",
		Status:       "open",
		AssignedPlan: "Stabilize release train",
		Priority:     1,
	}})
	v.Evidence().SetEntries([]views.EvidenceEntry{{
		Timestamp: time.Now(),
		Action:    "git status",
		Result:    "clean",
		Detail:    "working tree is clean",
		Success:   true,
	}})

	assertContains(t, v.Render(), "Execution Plans", "workspace should render plans tab by default")

	v.Update(tea.KeyPressMsg{Code: '2'})
	assertContains(t, v.Render(), "Task Queue", "workspace should render tasks tab")

	v.Update(tea.KeyPressMsg{Code: '3'})
	assertContains(t, v.Render(), "Evidence Stream", "workspace should render evidence tab")
}

func TestE2E_ApprovalQueueView_DataFlow(t *testing.T) {
	th := theme.NewTheme(true)
	v := views.NewApprovalQueueView(&th)
	v.SetSize(160, 50)

	plans := []autonomy.ActionPlan{
		{
			ID:          "plan-1",
			Description: "clean merged branches",
			RiskLevel:   autonomy.RiskMedium,
			Steps: []autonomy.PlanStep{
				{Order: 1, Action: "git.branch.delete", Args: map[string]string{"name": "feature/old"}},
			},
		},
		{
			ID:          "plan-2",
			Description: "merge dependency PR",
			RiskLevel:   autonomy.RiskHigh,
			Steps: []autonomy.PlanStep{
				{Order: 1, Action: "github.pr.merge", Args: map[string]string{"number": "55"}},
			},
		},
	}

	v.SetPendingPlans(plans)
	output := v.Render()
	assertContains(t, output, "clean merged branches", "approval queue should show first plan")
	assertContains(t, output, "merge dependency PR", "approval queue should show second plan")
}

func TestE2E_FileSystemCommandPatterns(t *testing.T) {
	th := theme.NewTheme(true)
	v := views.NewChatView(&th)
	v.SetSize(160, 60)

	fileCommands := []struct {
		cmd  string
		desc string
	}{
		{"/new README.md", "create file"},
		{"/edit main.go", "edit file"},
		{"/rm old-file.txt", "remove file"},
		{"/diff", "show diff"},
		{"/search TODO", "search content"},
		{"/find config.yaml", "find file"},
	}

	for _, fc := range fileCommands {
		v.AppendMessage(views.Message{Role: views.RoleUser, Content: fc.cmd})
		v.AppendMessage(views.Message{Role: views.RoleAssistant, Content: fc.desc + " command registered"})
	}

	messages := v.Messages()
	joined := make([]string, 0, len(messages))
	for _, msg := range messages {
		joined = append(joined, msg.Content)
	}
	all := strings.Join(joined, "\n")
	for _, fc := range fileCommands {
		assertContains(t, all, fc.cmd, "chat should retain file command: "+fc.cmd)
	}
}
