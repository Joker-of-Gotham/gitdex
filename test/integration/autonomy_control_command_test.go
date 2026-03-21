package integration

import (
	"testing"

	"github.com/your-org/gitdex/internal/cli/command"
)

func TestAutonomyControlCommandRegistered(t *testing.T) {
	root := command.NewRootCommand()
	autCmd, _, err := root.Find([]string{"autonomy"})
	if err != nil {
		t.Fatalf("autonomy command not found: %v", err)
	}

	controlSubs := []string{"pause", "resume", "cancel", "takeover"}
	for _, name := range controlSubs {
		found := false
		for _, c := range autCmd.Commands() {
			if c.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("subcommand %q not found under 'autonomy'", name)
		}
	}
}

func TestAutonomyPauseRequiresTaskID(t *testing.T) {
	root := command.NewRootCommand()
	root.SetArgs([]string{"autonomy", "pause"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when task_id is missing")
	}
}

func TestAutonomyResumeRequiresTaskID(t *testing.T) {
	root := command.NewRootCommand()
	root.SetArgs([]string{"autonomy", "resume"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when task_id is missing")
	}
}

func TestAutonomyPauseRuns(t *testing.T) {
	root := command.NewRootCommand()
	root.SetArgs([]string{"autonomy", "pause", "task_ctrl_1"})
	err := root.Execute()
	if err != nil {
		t.Fatalf("autonomy pause failed: %v", err)
	}
}

func TestAutonomyResumeRuns(t *testing.T) {
	root := command.NewRootCommand()
	root.SetArgs([]string{"autonomy", "resume", "task_ctrl_1"})
	err := root.Execute()
	if err != nil {
		t.Fatalf("autonomy resume failed: %v", err)
	}
}

func TestAutonomyCancelRuns(t *testing.T) {
	root := command.NewRootCommand()
	root.SetArgs([]string{"autonomy", "cancel", "task_ctrl_2"})
	err := root.Execute()
	if err != nil {
		t.Fatalf("autonomy cancel failed: %v", err)
	}
}

func TestAutonomyTakeoverRuns(t *testing.T) {
	root := command.NewRootCommand()
	root.SetArgs([]string{"autonomy", "takeover", "task_ctrl_3"})
	err := root.Execute()
	if err != nil {
		t.Fatalf("autonomy takeover failed: %v", err)
	}
}
