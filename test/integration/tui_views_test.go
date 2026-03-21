package integration_test

import (
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/your-org/gitdex/internal/llm/adapter"
	"github.com/your-org/gitdex/internal/state/repo"
	"github.com/your-org/gitdex/internal/tui/app"
	"github.com/your-org/gitdex/internal/tui/components"
	"github.com/your-org/gitdex/internal/tui/panes"
	"github.com/your-org/gitdex/internal/tui/theme"
	"github.com/your-org/gitdex/internal/tui/views"
)

var ansiReIntegration = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSIIntegration(s string) string {
	return ansiReIntegration.ReplaceAllString(s, "")
}

func TestE2E_SettingsView_FullWorkflow(t *testing.T) {
	th := theme.NewTheme(true)
	v := views.NewSettingsView(&th)
	v.SetSize(120, 40)
	v.Init()

	v.LoadFromConfig(map[string]string{
		"llm.provider": "openai",
		"llm.model":    "gpt-4",
		"llm.api_key":  "sk-test",
	})

	values := v.GetFieldValues()
	if values["llm.provider"] != "openai" {
		t.Errorf("provider = %q, want openai", values["llm.provider"])
	}

	v.LoadFromConfig(map[string]string{
		"llm.provider": "deepseek",
	})
	values = v.GetFieldValues()
	if values["llm.provider"] != "deepseek" {
		t.Errorf("after load deepseek: provider = %q, want deepseek", values["llm.provider"])
	}

	v.LoadFromConfig(map[string]string{
		"llm.provider": "ollama",
	})
	values = v.GetFieldValues()
	if values["llm.provider"] != "ollama" {
		t.Errorf("after load ollama: provider = %q, want ollama", values["llm.provider"])
	}

	output := v.Render()
	plain := stripANSIIntegration(output)
	if !strings.Contains(plain, "ollama") && !strings.Contains(plain, "Settings") {
		t.Error("Render should show settings form content")
	}
}

func TestE2E_LLMProviderFactory_AllProviders(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		wantErr  bool
	}{
		{"openai", "openai", false},
		{"deepseek", "deepseek", false},
		{"ollama", "ollama", false},
		{"empty defaults to openai", "", false},
		{"unsupported", "anthropic", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := adapter.NewProviderFromConfig(tt.provider, "", "", "")
			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if p == nil {
				t.Fatal("provider should not be nil")
			}
		})
	}
}

func TestE2E_PullsView_DataFlow(t *testing.T) {
	th := theme.NewTheme(true)
	v := views.NewPullsView(&th)
	v.SetSize(160, 50)

	prs := []repo.PullRequestSummary{
		{Number: 1, Title: "feat: add dark mode", Author: "alice", Labels: []string{"enhancement"}, IsDraft: false, NeedsReview: true},
		{Number: 2, Title: "fix: memory leak in parser", Author: "bob", IsDraft: true, StaleDays: 7},
		{Number: 3, Title: "chore: update dependencies", Author: "charlie", Labels: []string{"chore", "deps"}},
	}

	v.Update(views.PullsDataMsg{Items: prs})

	output := v.Render()
	if !strings.Contains(output, "#1") {
		t.Error("should show PR #1")
	}
	if !strings.Contains(output, "alice") {
		t.Error("should show author")
	}
}

func TestE2E_IssuesView_DataFlow(t *testing.T) {
	th := theme.NewTheme(true)
	v := views.NewIssuesView(&th)
	v.SetSize(160, 50)

	issues := []repo.IssueSummary{
		{Number: 10, Title: "Bug: crash on startup", Author: "dev", State: "OPEN", Comments: 5, Labels: []string{"bug", "critical"}},
		{Number: 11, Title: "Feature: dark theme", Author: "designer", State: "OPEN", Comments: 2},
	}

	v.Update(views.IssuesDataMsg{Items: issues})

	output := v.Render()
	if !strings.Contains(output, "#10") {
		t.Error("should show issue #10")
	}
	if !strings.Contains(output, "OPEN") {
		t.Error("should show OPEN state")
	}
}

func TestE2E_FilesView_TreeAndCode(t *testing.T) {
	th := theme.NewTheme(true)
	v := views.NewFilesView(&th)
	v.SetSize(120, 40)

	entries := []string{"cmd/main.go", "internal/app.go", "go.mod", "README.md"}
	root := views.BuildFileTree(entries)
	v.SetTree(root)

	output := v.Render()
	if !strings.Contains(output, "File Explorer") {
		t.Error("should show file explorer title")
	}

	v.Update(views.FileContentMsg{
		Path:    "cmd/main.go",
		Content: "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n",
	})

	output = v.Render()
	plain := stripANSIIntegration(output)
	if !strings.Contains(plain, "package") && !strings.Contains(plain, "main") {
		t.Error("code view should show file content")
	}
}

func TestE2E_DashboardView_SubTabs(t *testing.T) {
	th := theme.NewTheme(true)
	v := views.NewDashboardView(&th)
	v.SetSize(120, 40)

	output := v.Render()
	if !strings.Contains(output, "Overview") {
		t.Error("should show Overview sub-tab")
	}
	if !strings.Contains(output, "Health") {
		t.Error("should show Health sub-tab")
	}
	if !strings.Contains(output, "Repos") {
		t.Error("should show Repos sub-tab")
	}

	v.Update(tea.KeyPressMsg{Code: '2'})
	output = v.Render()
	if output == "" {
		t.Error("Health tab should render content")
	}

	v.Update(tea.KeyPressMsg{Code: '3'})
	output = v.Render()
	if output == "" {
		t.Error("Repos tab should render content")
	}

	v.Update(tea.KeyPressMsg{Code: '1'})
	output = v.Render()
	if output == "" {
		t.Error("Overview tab should render content")
	}
}

func TestE2E_ExplorerView_SubTabs(t *testing.T) {
	th := theme.NewTheme(true)
	v := views.NewExplorerView(&th)
	v.SetSize(120, 40)

	if v.ActiveTab() != 0 {
		t.Error("initial tab should be 0 (PRs)")
	}

	v.Update(tea.KeyPressMsg{Code: '2'})
	if v.ActiveTab() != 1 {
		t.Error("tab should be 1 (Issues) after pressing '2' in GitHub mega-group")
	}

	v.Update(tea.KeyPressMsg{Code: '3'})
	if v.ActiveTab() != 5 {
		t.Error("tab should be 5 (Workflows) after pressing '3' in GitHub mega-group")
	}

	v.Update(tea.KeyPressMsg{Code: '4'})
	if v.ActiveTab() != 6 {
		t.Error("tab should be 6 (Deployments) after pressing '4'")
	}

	v.Update(tea.KeyPressMsg{Code: '5'})
	if v.ActiveTab() != 7 {
		t.Error("tab should be 7 (Releases) after pressing '5'")
	}

	output := v.Render()
	if !strings.Contains(output, "Releases") {
		t.Error("should show Releases sub-tab")
	}

	v.Update(tea.KeyPressMsg{Code: ']'})
	if v.ActiveTab() != 2 {
		t.Errorf("] should jump to Git mega first tab (Files), got %d", v.ActiveTab())
	}
	v.Update(tea.KeyPressMsg{Code: '2'})
	if v.ActiveTab() != 3 {
		t.Error("Git mega: '2' should select Commits")
	}
	v.Update(tea.KeyPressMsg{Code: '3'})
	if v.ActiveTab() != 4 {
		t.Error("Git mega: '3' should select Branches")
	}
}

func TestE2E_WorkspaceView_SubTabs(t *testing.T) {
	th := theme.NewTheme(true)
	v := views.NewWorkspaceView(&th)
	v.SetSize(120, 40)

	output := v.Render()
	if !strings.Contains(output, "Plans") {
		t.Error("should show Plans sub-tab")
	}

	v.Update(tea.KeyPressMsg{Code: '2'})
	output = v.Render()
	if !strings.Contains(output, "Tasks") {
		t.Error("should show Tasks sub-tab")
	}

	v.Update(tea.KeyPressMsg{Code: '3'})
	output = v.Render()
	if !strings.Contains(output, "Evidence") {
		t.Error("should show Evidence sub-tab")
	}

	v.Update(tea.KeyPressMsg{Code: '4'})
	output = v.Render()
	if !strings.Contains(output, "Cruise") {
		t.Error("should show Cruise sub-tab")
	}

	v.Update(tea.KeyPressMsg{Code: '5'})
	output = v.Render()
	if !strings.Contains(output, "Approvals") {
		t.Error("should show Approvals sub-tab")
	}
}

func TestE2E_AppIntegration_FKeyRouting(t *testing.T) {
	m := app.New()
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	model := m2.(app.Model)

	fkeys := []tea.KeyPressMsg{
		{Code: tea.KeyF1},
		{Code: tea.KeyF2},
		{Code: tea.KeyF3},
		{Code: tea.KeyF4},
		{Code: tea.KeyF5},
	}

	for _, k := range fkeys {
		m2, _ = model.Update(k)
		model = m2.(app.Model)
	}
}

func TestE2E_IconDetection_DefaultUnicode(t *testing.T) {
	t.Setenv("GITDEX_NERD_FONT", "")
	if theme.DetectNerdFont() {
		t.Error("default (unset) should return false for safety")
	}

	t.Setenv("GITDEX_NERD_FONT", "1")
	if !theme.DetectNerdFont() {
		t.Error("GITDEX_NERD_FONT=1 should return true")
	}

	t.Setenv("GITDEX_NERD_FONT", "0")
	if theme.DetectNerdFont() {
		t.Error("GITDEX_NERD_FONT=0 should return false")
	}
}

func TestE2E_ComposerPaste(t *testing.T) {
	th := theme.NewTheme(true)
	composer := components.NewComposer(&th)
	composer.SetFocused(true)

	composer.Update(tea.PasteMsg{Content: "hello world"})
	if composer.Value() != "hello world" {
		t.Errorf("after paste: Value() = %q, want 'hello world'", composer.Value())
	}
}

func TestE2E_ComposerPaste_MultilineAndSpecial(t *testing.T) {
	th := theme.NewTheme(true)
	composer := components.NewComposer(&th)
	composer.SetFocused(true)

	multiline := "line1\nline2\n第三行"
	composer.Update(tea.PasteMsg{Content: multiline})
	if composer.Value() != multiline {
		t.Errorf("multi-line paste: got %q, want %q", composer.Value(), multiline)
	}
}

func TestE2E_SettingsView_GitConfig(t *testing.T) {
	th := theme.NewTheme(true)
	v := views.NewSettingsView(&th)
	v.SetSize(120, 40)

	v.LoadFromConfig(map[string]string{
		"git.user_name":       "Alice",
		"git.user_email":      "alice@example.com",
		"git.ssh_key_path":    "~/.ssh/id_ed25519",
		"identity.github_pat": "ghp_test123",
	})

	values := v.GetFieldValues()
	if values["git.user_name"] != "Alice" {
		t.Errorf("git.user_name = %q, want Alice", values["git.user_name"])
	}
	if values["git.user_email"] != "alice@example.com" {
		t.Errorf("git.user_email = %q", values["git.user_email"])
	}
	if values["identity.github_pat"] != "ghp_test123" {
		t.Errorf("identity.github_pat = %q", values["identity.github_pat"])
	}

	output := v.Render()
	plain := stripANSIIntegration(output)
	if !strings.Contains(plain, "Settings") {
		t.Error("should render settings view")
	}
	_ = plain
}

func TestE2E_NavPanelIcons_ConsolidatedPaths(t *testing.T) {
	th := theme.NewTheme(true)
	styles := theme.NewStyles(th)

	items := []panes.NavItem{
		{Label: "Dashboard", Path: "dashboard"},
		{Label: "Chat", Path: "chat"},
		{Label: "Explorer", Path: "explorer"},
		{Label: "Workspace", Path: "workspace"},
		{Label: "Settings", Path: "settings"},
	}
	nav := panes.NewNavPane(&th, styles, items)
	nav.SetSize(40, 30)
	nav.SetFocused(true)

	output := nav.View()
	for _, item := range items {
		if !strings.Contains(output, item.Label) {
			t.Errorf("NavPane should show %q label", item.Label)
		}
	}
}

func TestE2E_FieldsToFileConfig_WithGit(t *testing.T) {
	th := theme.NewTheme(true)
	v := views.NewSettingsView(&th)
	v.SetSize(120, 40)

	v.LoadFromConfig(map[string]string{
		"llm.provider":        "ollama",
		"git.user_name":       "Bob",
		"git.user_email":      "bob@dev.io",
		"git.ssh_key_path":    "/keys/id_rsa",
		"identity.github_pat": "ghp_xxx",
		"output":              "json",
	})

	fc := views.FieldsToFileConfig(v.GetFields())
	if fc.LLM.Provider != "ollama" {
		t.Errorf("LLM.Provider = %q", fc.LLM.Provider)
	}
	if fc.Git.UserName != "Bob" {
		t.Errorf("Git.UserName = %q", fc.Git.UserName)
	}
	if fc.Git.UserEmail != "bob@dev.io" {
		t.Errorf("Git.UserEmail = %q", fc.Git.UserEmail)
	}
	if fc.Identity.GitHubPAT != "ghp_xxx" {
		t.Errorf("Identity.GitHubPAT = %q", fc.Identity.GitHubPAT)
	}
	if fc.Output != "json" {
		t.Errorf("Output = %q", fc.Output)
	}
}

func TestE2E_FocusNavigation_ContentAreaAccessible(t *testing.T) {
	m := app.New()
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	model := m2.(app.Model)

	m2, _ = model.Update(tea.KeyPressMsg{Code: '\t'})
	model = m2.(app.Model)

	m2, _ = model.Update(tea.KeyPressMsg{Code: 'q'})
	_ = m2.(app.Model)
}

func TestE2E_SettingsView_ProviderAutoFill(t *testing.T) {
	th := theme.NewTheme(true)
	v := views.NewSettingsView(&th)
	v.SetSize(120, 40)

	for _, provider := range adapter.SupportedProviders {
		v.LoadFromConfig(map[string]string{"llm.provider": provider})
		values := v.GetFieldValues()
		if values["llm.provider"] != provider {
			t.Errorf("provider = %q, want %q", values["llm.provider"], provider)
		}
	}
}

// --- RV boundary coverage: sub-tab Left/Right navigation ---

func TestE2E_DashboardView_LeftRightNavigation(t *testing.T) {
	th := theme.NewTheme(true)
	v := views.NewDashboardView(&th)
	v.SetSize(120, 40)

	if v.ActiveTab() != 0 {
		t.Fatal("initial tab should be 0")
	}

	v.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	if v.ActiveTab() != 0 {
		t.Error("Left at tab 0 should stay at 0 (boundary)")
	}

	v.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if v.ActiveTab() != 1 {
		t.Error("Right from tab 0 should go to 1")
	}

	v.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if v.ActiveTab() != 2 {
		t.Error("Right from tab 1 should go to 2 (Repos)")
	}

	v.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if v.ActiveTab() != 2 {
		t.Error("Right at last tab should stay at last (boundary)")
	}

	v.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	if v.ActiveTab() != 1 {
		t.Error("Left from tab 2 should go to 1")
	}

	v.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	if v.ActiveTab() != 0 {
		t.Error("Left from tab 1 should go to 0")
	}
}

func Old_TestE2E_ExplorerView_LeftRightNavigation(t *testing.T) {
	th := theme.NewTheme(true)
	v := views.NewExplorerView(&th)
	v.SetSize(120, 40)

	v.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	if v.ActiveTab() != 0 {
		t.Error("Left at tab 0 should stay at 0")
	}

	v.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if v.ActiveTab() != 1 {
		t.Error("Right: 0 → 1")
	}
	v.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if v.ActiveTab() != 2 {
		t.Error("Right: 1 → 2")
	}
	v.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if v.ActiveTab() != 2 {
		t.Error("Right at tab 2 should stay at 2 (boundary)")
	}
	v.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	if v.ActiveTab() != 1 {
		t.Error("Left: 2 → 1")
	}
}

func TestE2E_WorkspaceView_LeftRightNavigation(t *testing.T) {
	th := theme.NewTheme(true)
	v := views.NewWorkspaceView(&th)
	v.SetSize(120, 40)

	v.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if v.ActiveTab() != 1 {
		t.Error("Right: 0 → 1")
	}
	v.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if v.ActiveTab() != 2 {
		t.Error("Right: 1 → 2")
	}
	v.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if v.ActiveTab() != 3 {
		t.Error("Right: 2 → 3")
	}
	v.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if v.ActiveTab() != 4 {
		t.Error("Right: 3 → 4")
	}
	v.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if v.ActiveTab() != 4 {
		t.Error("Right at last tab should clamp")
	}
	v.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	v.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	v.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	v.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	if v.ActiveTab() != 0 {
		t.Error("Left×4 from 4 should reach 0")
	}
	v.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	if v.ActiveTab() != 0 {
		t.Error("Left at 0 should clamp")
	}
}

// --- RV boundary: invalid key presses on composite views ---

func Old_TestE2E_ExplorerView_InvalidKeyIgnored(t *testing.T) {
	th := theme.NewTheme(true)
	v := views.NewExplorerView(&th)
	v.SetSize(120, 40)

	v.Update(tea.KeyPressMsg{Code: '4'})
	if v.ActiveTab() != 0 {
		t.Error("pressing '4' on 3-tab view should be ignored")
	}

	v.Update(tea.KeyPressMsg{Code: '0'})
	if v.ActiveTab() != 0 {
		t.Error("pressing '0' should be ignored")
	}

	v.Update(tea.KeyPressMsg{Code: 'x'})
	if v.ActiveTab() != 0 {
		t.Error("pressing 'x' should be ignored at tab level")
	}
}

func TestE2E_ExplorerView_LeftRightNavigation(t *testing.T) {
	th := theme.NewTheme(true)
	v := views.NewExplorerView(&th)
	v.SetSize(120, 40)

	// GitHub mega-group cycles: PRs → Issues → Workflows → Deployments → Releases → wrap.
	v.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if v.ActiveTab() != 1 {
		t.Errorf("Right from PRs: got tab %d, want 1 (Issues)", v.ActiveTab())
	}
	v.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if v.ActiveTab() != 5 {
		t.Errorf("Right from Issues: got tab %d, want 5 (Workflows)", v.ActiveTab())
	}
	v.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if v.ActiveTab() != 6 {
		t.Errorf("Right from Workflows: got tab %d, want 6 (Deployments)", v.ActiveTab())
	}
	v.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if v.ActiveTab() != 7 {
		t.Errorf("Right from Deployments: got tab %d, want 7 (Releases)", v.ActiveTab())
	}
	v.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if v.ActiveTab() != 0 {
		t.Errorf("Right from Releases should wrap to PRs, got %d", v.ActiveTab())
	}

	v.Update(tea.KeyPressMsg{Code: ']'})
	if v.ActiveTab() != 2 {
		t.Errorf("] should jump to Git mega (Files), got %d", v.ActiveTab())
	}
	v.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if v.ActiveTab() != 3 {
		t.Errorf("Git mega: Right Files → Commits, got %d", v.ActiveTab())
	}
	v.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if v.ActiveTab() != 4 {
		t.Errorf("Git mega: Right Commits → Branches, got %d", v.ActiveTab())
	}
	v.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if v.ActiveTab() != 2 {
		t.Errorf("Git mega: Right from Branches should wrap to Files, got %d", v.ActiveTab())
	}
}

func TestE2E_ExplorerView_InvalidKeyIgnored(t *testing.T) {
	th := theme.NewTheme(true)
	v := views.NewExplorerView(&th)
	v.SetSize(120, 40)

	v.Update(tea.KeyPressMsg{Code: '8'})
	if v.ActiveTab() != 0 {
		t.Error("pressing '8' outside current mega-group slot range should be ignored")
	}

	v.Update(tea.KeyPressMsg{Code: '0'})
	if v.ActiveTab() != 0 {
		t.Error("pressing '0' should be ignored")
	}

	v.Update(tea.KeyPressMsg{Code: 'x'})
	if v.ActiveTab() != 0 {
		t.Error("pressing 'x' should be ignored at tab level")
	}
}

// --- RV boundary: very small window rendering ---

func TestE2E_DashboardView_VerySmallWindow(t *testing.T) {
	th := theme.NewTheme(true)
	v := views.NewDashboardView(&th)

	v.SetSize(20, 10)
	output := v.Render()
	if output == "" {
		t.Error("very small window should still render something")
	}
	if !strings.Contains(output, "1") {
		t.Error("minimal mode should show tab number '1'")
	}
}

func TestE2E_ExplorerView_CompactWindow(t *testing.T) {
	th := theme.NewTheme(true)
	v := views.NewExplorerView(&th)

	v.SetSize(45, 15)
	output := v.Render()
	if output == "" {
		t.Error("compact window should render")
	}
}

func TestE2E_WorkspaceView_MinimalWindow(t *testing.T) {
	th := theme.NewTheme(true)
	v := views.NewWorkspaceView(&th)

	v.SetSize(25, 8)
	output := v.Render()
	if output == "" {
		t.Error("minimal window should still render")
	}
}

func TestE2E_SubTabs_ZeroWidth(t *testing.T) {
	th := theme.NewTheme(true)
	v := views.NewDashboardView(&th)
	v.SetSize(0, 0)
	output := v.Render()
	if output != "" {
		t.Error("zero-size should return empty string")
	}
}

// --- RV boundary: sub-tab at already-active position ---

func TestE2E_ExplorerView_SameTabReselect(t *testing.T) {
	th := theme.NewTheme(true)
	v := views.NewExplorerView(&th)
	v.SetSize(120, 40)

	v.Update(tea.KeyPressMsg{Code: '1'})
	if v.ActiveTab() != 0 {
		t.Error("pressing '1' when already on tab 0 should stay at 0")
	}

	v.Update(tea.KeyPressMsg{Code: '2'})
	v.Update(tea.KeyPressMsg{Code: '2'})
	if v.ActiveTab() != 1 {
		t.Error("pressing '2' twice should stay at tab 1")
	}
}

// --- RV isolation: composite view data routing (no external deps) ---

func TestE2E_ExplorerView_DataRouting_Isolated(t *testing.T) {
	th := theme.NewTheme(true)
	v := views.NewExplorerView(&th)
	v.SetSize(120, 40)

	v.Update(views.PullsDataMsg{Items: []repo.PullRequestSummary{
		{Number: 42, Title: "Test PR", Author: "tester"},
	}})
	v.Update(tea.KeyPressMsg{Code: '1'})
	output := v.Render()
	if !strings.Contains(output, "#42") {
		t.Error("PRs tab should show routed PR data")
	}

	v.Update(views.IssuesDataMsg{Items: []repo.IssueSummary{
		{Number: 99, Title: "Test Issue", Author: "dev", State: "OPEN"},
	}})
	v.Update(tea.KeyPressMsg{Code: '2'})
	output = v.Render()
	if !strings.Contains(output, "#99") {
		t.Error("Issues tab should show routed issue data")
	}

	root := views.BuildFileTree([]string{"src/main.go", "README.md"})
	v.Update(views.FileTreeDataMsg{Root: root})
	v.Update(tea.KeyPressMsg{Code: ']'})
	output = v.Render()
	if !strings.Contains(output, "File Explorer") {
		t.Error("Files tab should show file tree")
	}

	v.Update(views.WorkflowRunsDataMsg{Runs: []views.WorkflowRunEntry{
		{RunID: 123, WorkflowID: 99, Name: "CI", Status: "completed", Conclusion: "success", Branch: "main"},
	}})
	v.Update(tea.KeyPressMsg{Code: '['})
	v.Update(tea.KeyPressMsg{Code: '3'})
	output = v.Render()
	if !strings.Contains(output, "Workflows") {
		t.Error("Workflows tab should show workflow runs")
	}

	v.Update(views.DeploymentDataMsg{Deployments: []views.DeploymentEntry{
		{ID: 1, Environment: "production", State: "success", Ref: "main"},
	}})
	v.Update(tea.KeyPressMsg{Code: '4'})
	output = v.Render()
	if !strings.Contains(output, "Deployments") {
		t.Error("Deployments tab should show deployment records")
	}

	v.Update(tea.KeyPressMsg{Code: '5'})
	output = v.Render()
	if !strings.Contains(output, "Releases") {
		t.Error("Releases tab should render")
	}
}

func TestE2E_DashboardView_DataRouting_Isolated(t *testing.T) {
	th := theme.NewTheme(true)
	v := views.NewDashboardView(&th)
	v.SetSize(120, 40)

	v.Update(views.StatusDataMsg{Summary: &repo.RepoSummary{
		Owner:        "test-org",
		Repo:         "test-repo",
		OverallLabel: repo.Healthy,
	}})

	v.Update(tea.KeyPressMsg{Code: '1'})
	output := v.Render()
	if output == "" {
		t.Error("Overview tab should render after data routing")
	}

	v.Update(tea.KeyPressMsg{Code: '2'})
	output = v.Render()
	if output == "" {
		t.Error("Health tab should render after data routing")
	}
}

// --- RV: StatusBar mode-specific colors ---

func TestE2E_StatusBar_ModeBadgeColors(t *testing.T) {
	th := theme.NewTheme(true)
	sb := components.NewStatusBar(&th)
	sb.SetWidth(120)

	modes := []string{"INSERT", "NORMAL", "NAV", "INSPECT", "COMMAND"}
	for _, mode := range modes {
		sb.SetMode(mode)
		output := sb.Render()
		if !strings.Contains(output, mode) {
			t.Errorf("StatusBar should contain mode text %q", mode)
		}
		if output == "" {
			t.Errorf("StatusBar should render for mode %q", mode)
		}
	}
}

// --- RV: help text updated for new key bindings ---

func TestE2E_App_HelpContainsNewInfo(t *testing.T) {
	m := app.New()
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	model := m2.(app.Model)

	m2, _ = model.Update(tea.KeyPressMsg{Code: '?'})
	model = m2.(app.Model)

	v := model.View()
	if !strings.Contains(v.Content, "Dashboard") {
		t.Error("help should mention Dashboard")
	}
	if !strings.Contains(v.Content, "Explorer") {
		t.Error("help should mention Explorer")
	}
	if !strings.Contains(v.Content, "Workspace") {
		t.Error("help should mention Workspace")
	}
	if !strings.Contains(v.Content, "1-7") {
		t.Error("help should mention sub-tab switching keys")
	}
}
