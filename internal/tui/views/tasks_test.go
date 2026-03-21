package views_test

import (
	"strings"
	"testing"

	"github.com/your-org/gitdex/internal/tui/views"
)

func TestNewTasksView(t *testing.T) {
	v := views.NewTasksView(makeTheme())
	if v == nil {
		t.Fatal("NewTasksView should return non-nil")
	}
	if v.ID() != views.ViewTasks {
		t.Errorf("ID(): got %v, want ViewTasks", v.ID())
	}
}

func TestTasksView_Render_Empty(t *testing.T) {
	v := views.NewTasksView(makeTheme())
	v.SetSize(80, 20)
	out := v.Render()
	if out == "" {
		t.Error("Render() should not return empty")
	}
	if !strings.Contains(out, "No tasks are currently queued") {
		t.Error("Render() empty state should show tasks empty state text")
	}
}
