package views_test

import (
	"strings"
	"testing"

	"github.com/your-org/gitdex/internal/state/repo"
	"github.com/your-org/gitdex/internal/tui/theme"
	"github.com/your-org/gitdex/internal/tui/views"
)

func makeTheme() *theme.Theme {
	t := theme.NewTheme(true)
	return &t
}

func TestNewRouter(t *testing.T) {
	chatView := views.NewChatView(makeTheme())
	statusView := views.NewStatusView(makeTheme())
	r := views.NewRouter(views.ViewChat, chatView, statusView)
	if r == nil {
		t.Fatal("NewRouter() should return non-nil")
	}
}

func TestRouter_ActiveID(t *testing.T) {
	chatView := views.NewChatView(makeTheme())
	statusView := views.NewStatusView(makeTheme())
	r := views.NewRouter(views.ViewChat, chatView, statusView)
	if r.ActiveID() != views.ViewChat {
		t.Errorf("ActiveID() initially: got %s, want chat", r.ActiveID())
	}
}

func TestRouter_SwitchTo(t *testing.T) {
	chatView := views.NewChatView(makeTheme())
	statusView := views.NewStatusView(makeTheme())
	r := views.NewRouter(views.ViewChat, chatView, statusView)

	r.SwitchTo(views.ViewStatus)
	if r.ActiveID() != views.ViewStatus {
		t.Errorf("ActiveID() after SwitchTo(status): got %s", r.ActiveID())
	}
}

func TestRouter_Order(t *testing.T) {
	chatView := views.NewChatView(makeTheme())
	statusView := views.NewStatusView(makeTheme())
	r := views.NewRouter(views.ViewChat, chatView, statusView)

	order := r.Order()
	if len(order) != 2 {
		t.Errorf("Order() length: got %d, want 2", len(order))
	}
	if order[0] != views.ViewChat || order[1] != views.ViewStatus {
		t.Errorf("Order(): got %v", order)
	}
}

func TestRouter_ViewTitle(t *testing.T) {
	chatView := views.NewChatView(makeTheme())
	statusView := views.NewStatusView(makeTheme())
	r := views.NewRouter(views.ViewChat, chatView, statusView)

	if r.ViewTitle(views.ViewChat) != "Chat" {
		t.Errorf("ViewTitle(chat): got %q, want Chat", r.ViewTitle(views.ViewChat))
	}
	if r.ViewTitle(views.ViewStatus) != "Status" {
		t.Errorf("ViewTitle(status): got %q, want Status", r.ViewTitle(views.ViewStatus))
	}
}

func TestRouter_Render(t *testing.T) {
	chatView := views.NewChatView(makeTheme())
	statusView := views.NewStatusView(makeTheme())
	r := views.NewRouter(views.ViewChat, chatView, statusView)
	r.SetSize(100, 30)

	out := r.Render()
	if out == "" {
		t.Error("Render() should return non-empty string")
	}
	if !strings.Contains(out, "Gitdex chat is ready") || !strings.Contains(out, "/help") {
		t.Error("Render() should return active view (Chat) content")
	}
}

func TestRouter_Update_SwitchViewMsg(t *testing.T) {
	chatView := views.NewChatView(makeTheme())
	statusView := views.NewStatusView(makeTheme())
	r := views.NewRouter(views.ViewChat, chatView, statusView)
	r.SetSize(100, 30)

	r.Update(views.SwitchViewMsg{Target: views.ViewStatus})
	if r.ActiveID() != views.ViewStatus {
		t.Errorf("Update(SwitchViewMsg) should switch view, got %s", r.ActiveID())
	}
}

func TestRouter_SetSize(t *testing.T) {
	chatView := views.NewChatView(makeTheme())
	statusView := views.NewStatusView(makeTheme())
	r := views.NewRouter(views.ViewChat, chatView, statusView)

	r.SetSize(80, 25)
	out := r.Render()
	if out == "" {
		t.Error("SetSize then Render should return content")
	}
}

func TestNewChatView(t *testing.T) {
	v := views.NewChatView(makeTheme())
	if v == nil {
		t.Fatal("NewChatView() should return non-nil")
	}
	msgs := v.Messages()
	if len(msgs) == 0 {
		t.Error("NewChatView should have welcome message")
	}
	if !strings.Contains(msgs[0].Content, "Gitdex chat is ready") || !strings.Contains(msgs[0].Content, "/help") {
		t.Errorf("welcome message: got %q", msgs[0].Content)
	}
}

func TestChatView_AppendMessage(t *testing.T) {
	v := views.NewChatView(makeTheme())
	v.SetSize(80, 20)

	v.AppendMessage(views.Message{
		Role:    views.RoleUser,
		Content: "hello",
	})
	msgs := v.Messages()
	if len(msgs) != 2 {
		t.Errorf("Messages() length: got %d, want 2", len(msgs))
	}
	if msgs[1].Content != "hello" {
		t.Errorf("AppendMessage content: got %q", msgs[1].Content)
	}
}

func TestChatView_Messages(t *testing.T) {
	v := views.NewChatView(makeTheme())
	msgs := v.Messages()
	if len(msgs) < 1 {
		t.Error("Messages() should return at least welcome message")
	}
}

func TestChatView_Render(t *testing.T) {
	v := views.NewChatView(makeTheme())
	v.SetSize(80, 20)
	v.AppendMessage(views.Message{Role: views.RoleUser, Content: "test"})

	out := v.Render()
	if !strings.Contains(out, "test") {
		t.Errorf("Render() should contain message content, got %q", out)
	}
}

func TestChatView_SetSize(t *testing.T) {
	v := views.NewChatView(makeTheme())
	v.SetSize(60, 15)
	out := v.Render()
	if out == "" {
		t.Error("SetSize then Render should work")
	}
}

func TestChatView_ID(t *testing.T) {
	v := views.NewChatView(makeTheme())
	if v.ID() != views.ViewChat {
		t.Errorf("ID(): got %s, want chat", v.ID())
	}
}

func TestChatView_Title(t *testing.T) {
	v := views.NewChatView(makeTheme())
	if v.Title() != "Chat" {
		t.Errorf("Title(): got %q, want Chat", v.Title())
	}
}

func TestNewStatusView(t *testing.T) {
	v := views.NewStatusView(makeTheme())
	if v == nil {
		t.Fatal("NewStatusView() should return non-nil")
	}
}

func TestStatusView_SetSummary(t *testing.T) {
	v := views.NewStatusView(makeTheme())
	summary := &repo.RepoSummary{
		Owner:        "org",
		Repo:         "repo",
		OverallLabel: repo.Healthy,
	}
	v.SetSummary(summary)
	v.SetSize(80, 20)
	out := v.Render()
	if !strings.Contains(out, "org") || !strings.Contains(out, "repo") {
		t.Errorf("Render with SetSummary: got %q", out)
	}
}

func TestStatusView_Render_NilSummary(t *testing.T) {
	v := views.NewStatusView(makeTheme())
	v.SetSize(80, 20)
	out := v.Render()
	if !strings.Contains(out, "No repository data loaded") {
		t.Errorf("Render with nil summary should show placeholder, got %q", out)
	}
}

func TestStatusView_Render_WithData(t *testing.T) {
	v := views.NewStatusView(makeTheme())
	v.SetSummary(&repo.RepoSummary{
		Owner:        "my-org",
		Repo:         "my-repo",
		OverallLabel: repo.Healthy,
	})
	v.SetSize(80, 20)
	out := v.Render()
	if !strings.Contains(out, "my-org") || !strings.Contains(out, "my-repo") {
		t.Errorf("Render with data: got %q", out)
	}
}

func TestStatusView_ID(t *testing.T) {
	v := views.NewStatusView(makeTheme())
	if v.ID() != views.ViewStatus {
		t.Errorf("ID(): got %s, want status", v.ID())
	}
}

func TestStatusView_Update_StatusDataMsg(t *testing.T) {
	v := views.NewStatusView(makeTheme())
	summary := &repo.RepoSummary{
		Owner:        "updated",
		Repo:         "repo",
		OverallLabel: repo.Healthy,
	}
	updated, _ := v.Update(views.StatusDataMsg{Summary: summary})
	sv, ok := updated.(*views.StatusView)
	if !ok {
		t.Fatalf("Update should return *StatusView, got %T", updated)
	}
	sv.SetSize(80, 20)
	out := sv.Render()
	if !strings.Contains(out, "updated") {
		t.Errorf("Update(StatusDataMsg) should set summary, got %q", out)
	}
}

// Ensure Router Init doesn't panic and Update with non-SwitchViewMsg passes to active view
func TestRouter_Update_PassesToView(t *testing.T) {
	chatView := views.NewChatView(makeTheme())
	r := views.NewRouter(views.ViewChat, chatView)
	r.SetSize(80, 20)

	// AppendMessage via Update
	r.Update(views.AppendMessageMsg{
		Message: views.Message{Role: views.RoleUser, Content: "from update"},
	})
	out := r.Render()
	if !strings.Contains(out, "from update") {
		t.Errorf("Update(AppendMessageMsg) should reach ChatView, got %q", out)
	}
}
