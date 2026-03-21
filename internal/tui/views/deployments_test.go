package views

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/your-org/gitdex/internal/tui/theme"
)

func TestDeploymentsView_RenderAndDetail(t *testing.T) {
	th := theme.NewTheme(true)
	v := NewDeploymentsView(&th)
	v.SetSize(140, 40)
	v.SetDeployments([]DeploymentEntry{
		{ID: 1, Environment: "production", State: "success", Ref: "main", CreatedAt: "2026-03-21 12:00"},
	})

	out := v.Render()
	if !strings.Contains(out, "Deployments") {
		t.Fatalf("render missing title: %q", out)
	}

	_, cmd := v.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if !v.detail {
		t.Fatal("enter should toggle deployment detail")
	}
	if cmd == nil {
		t.Fatal("enter should emit deployment selection")
	}
	msg := cmd()
	if _, ok := msg.(DeploymentSelectedMsg); !ok {
		t.Fatalf("selection msg type = %T", msg)
	}

	out = v.Render()
	if !strings.Contains(out, "Inspector detail active") {
		t.Fatalf("detail render missing panel: %q", out)
	}
}
