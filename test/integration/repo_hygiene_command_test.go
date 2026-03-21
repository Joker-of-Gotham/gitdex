package integration

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/your-org/gitdex/internal/cli/command"
)

func TestRepoHygieneCommandRegistered(t *testing.T) {
	root := command.NewRootCommand()
	repoCmd, _, err := root.Find([]string{"repo"})
	if err != nil {
		t.Fatalf("repo command not found: %v", err)
	}

	hygieneCmd, _, err := repoCmd.Find([]string{"hygiene"})
	if err != nil {
		t.Fatalf("repo hygiene command not found: %v", err)
	}

	subs := []string{"list", "run"}
	for _, name := range subs {
		found := false
		for _, c := range hygieneCmd.Commands() {
			if c.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("subcommand %q not found under 'repo hygiene'", name)
		}
	}
}

func TestRepoHygieneList(t *testing.T) {
	root := command.NewRootCommand()
	root.SetArgs([]string{"repo", "hygiene", "list"})
	var out bytes.Buffer
	root.SetOut(&out)
	err := root.Execute()
	if err != nil {
		t.Fatalf("repo hygiene list failed: %v", err)
	}
	if out.Len() == 0 {
		t.Error("expected non-empty output from hygiene list")
	}
}

func TestRepoHygieneList_JSON(t *testing.T) {
	root := command.NewRootCommand()
	root.SetArgs([]string{"repo", "hygiene", "list", "--output", "json"})
	var out bytes.Buffer
	root.SetOut(&out)
	err := root.Execute()
	if err != nil {
		t.Fatalf("repo hygiene list --output json failed: %v", err)
	}

	var decoded struct {
		Tasks []struct {
			Action          string `json:"action"`
			Description     string `json:"description"`
			RiskLevel       string `json:"risk_level"`
			Reversible      bool   `json:"reversible"`
			EstimatedImpact string `json:"estimated_impact"`
		} `json:"tasks"`
	}
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	if len(decoded.Tasks) != 4 {
		t.Errorf("expected 4 tasks in JSON, got %d", len(decoded.Tasks))
	}
}

func TestRepoHygieneRun_RequiresAction(t *testing.T) {
	root := command.NewRootCommand()
	root.SetArgs([]string{"repo", "hygiene", "run"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when action not provided")
	}
}
