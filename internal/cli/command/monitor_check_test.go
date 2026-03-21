package command

import (
	"bytes"
	"strings"
	"testing"

	"github.com/your-org/gitdex/internal/app/bootstrap"
	"github.com/your-org/gitdex/internal/autonomy"
)

func TestMonitorCheckAppendsEventsForRepo(t *testing.T) {
	repoRoot := initAutonomyRepo(t)
	app := autonomyTestApp(t, repoRoot)
	defer func() { _ = app.StorageProvider.Close() }()

	var out bytes.Buffer
	cmd := newMonitorGroupCommand(&runtimeOptions{}, func() bootstrap.App { return app })
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"check", "--repo", "owner/repo"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("monitor check failed: %v\n%s", err, out.String())
	}

	events, err := app.StorageProvider.MonitorStore().ListEvents(autonomy.MonitorEventFilter{
		RepoOwner: "owner",
		RepoName:  "repo",
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("list monitor events: %v", err)
	}
	if len(events) != 5 {
		t.Fatalf("monitor event count = %d, want 5", len(events))
	}
	if !strings.Contains(out.String(), "Monitor ad-hoc (owner/repo)") {
		t.Fatalf("unexpected output:\n%s", out.String())
	}
}
