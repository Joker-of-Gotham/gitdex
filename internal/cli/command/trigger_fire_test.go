package command

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/your-org/gitdex/internal/app/bootstrap"
	"github.com/your-org/gitdex/internal/autonomy"
	"github.com/your-org/gitdex/internal/llm/adapter"
)

func TestTriggerFireExecuteWritesFileAndRecordsEvent(t *testing.T) {
	repoRoot := initAutonomyRepo(t)
	app := autonomyTestApp(t, repoRoot)
	defer func() { _ = app.StorageProvider.Close() }()

	restore := setAutonomyProviderForTest(&adapter.MockProvider{
		ChatCompletionFn: func(ctx context.Context, req adapter.ChatRequest) (*adapter.ChatResponse, error) {
			return &adapter.ChatResponse{
				Content: `{
  "description": "add trigger note",
  "steps": [
    {"order": 1, "action": "file.write", "args": {"path": "notes/trigger.txt", "content": "triggered\n"}, "reversible": true, "description": "write trigger file"},
    {"order": 2, "action": "git.add", "args": {"path": "notes/trigger.txt"}, "reversible": true, "description": "stage trigger file"},
    {"order": 3, "action": "git.commit", "args": {"message": "add trigger file"}, "reversible": false, "description": "commit trigger file"}
  ],
  "risk_level": "high",
  "rationale": "test trigger execution"
}`,
			}, nil
		},
	})
	defer restore()

	cfg := &autonomy.TriggerConfig{
		TriggerID:      "tr_manual",
		TriggerType:    autonomy.TriggerOperator,
		Name:           "manual-fire",
		Source:         "owner/repo",
		ActionTemplate: "add trigger note",
		Enabled:        true,
	}
	if err := app.StorageProvider.TriggerStore().SaveTrigger(cfg); err != nil {
		t.Fatalf("save trigger: %v", err)
	}

	var out bytes.Buffer
	cmd := newTriggerGroupCommand(&runtimeOptions{}, func() bootstrap.App { return app })
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"fire", "tr_manual", "--repo", "owner/repo", "--execute", "--auto-threshold", "high"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("trigger fire failed: %v\n%s", err, out.String())
	}

	content, err := os.ReadFile(filepath.Join(repoRoot, "notes", "trigger.txt"))
	if err != nil {
		t.Fatalf("expected trigger file to be written: %v", err)
	}
	if string(content) != "triggered\n" {
		t.Fatalf("unexpected trigger file content: %q", string(content))
	}

	events, err := app.StorageProvider.TriggerStore().ListTriggerEvents("tr_manual", 10)
	if err != nil {
		t.Fatalf("list trigger events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("trigger event count = %d, want 1", len(events))
	}
	if events[0].ResultingTaskID == "" {
		t.Fatal("trigger event should record resulting cycle id")
	}
}
