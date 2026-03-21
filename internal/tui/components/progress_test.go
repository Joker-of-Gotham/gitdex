package components_test

import (
	"strings"
	"testing"

	"github.com/your-org/gitdex/internal/tui/components"
	"github.com/your-org/gitdex/internal/tui/theme"
)

func TestNewProgressBar(t *testing.T) {
	tk := theme.NewTheme(true)
	p := components.NewProgressBar(&tk)
	if p == nil {
		t.Fatal("NewProgressBar() should return non-nil")
	}
}

func TestProgressBar_SetPercent_Render(t *testing.T) {
	tk := theme.NewTheme(true)
	p := components.NewProgressBar(&tk)
	p.SetPercent(0.5)
	out := p.Render()
	if !strings.Contains(out, "50%") {
		t.Errorf("Render at 50%% should contain \"50%%\", got %q", out)
	}
}

func TestProgressBar_SetPercent_100(t *testing.T) {
	tk := theme.NewTheme(true)
	p := components.NewProgressBar(&tk)
	p.SetPercent(1.0)
	out := p.Render()
	if !strings.Contains(out, "100%") {
		t.Errorf("Render at 100%% should contain \"100%%\", got %q", out)
	}
}
