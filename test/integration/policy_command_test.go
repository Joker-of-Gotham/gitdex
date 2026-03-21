package integration

import (
	"bytes"
	"strings"
	"testing"

	"github.com/your-org/gitdex/internal/cli/command"
)

func TestPolicyCommandRegistered(t *testing.T) {
	root := command.NewRootCommand()
	policyCmd, _, err := root.Find([]string{"policy"})
	if err != nil {
		t.Fatalf("policy command not found: %v", err)
	}

	subs := []string{"show", "list", "create"}
	for _, name := range subs {
		found := false
		for _, c := range policyCmd.Commands() {
			if c.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("subcommand %q not found under 'policy'", name)
		}
	}
}

func TestPolicyCommandHelp(t *testing.T) {
	root := command.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"policy", "--help"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("policy --help failed: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "show") {
		t.Error("help should mention show subcommand")
	}
	if !strings.Contains(output, "list") {
		t.Error("help should mention list subcommand")
	}
	if !strings.Contains(output, "create") {
		t.Error("help should mention create subcommand")
	}
}

func TestPolicyShowNoBundle(t *testing.T) {
	root := command.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"policy", "show"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("policy show failed: %v", err)
	}
	if !strings.Contains(out.String(), "No policy") {
		t.Error("output should indicate no bundle when none configured")
	}
}

func TestPolicyListEmpty(t *testing.T) {
	root := command.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"policy", "list"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("policy list failed: %v", err)
	}
	if !strings.Contains(out.String(), "No policy") {
		t.Error("output should indicate no bundles when list is empty")
	}
}

func TestPolicyCreateRequiresName(t *testing.T) {
	root := command.NewRootCommand()
	root.SetArgs([]string{"policy", "create"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when create without --name")
	}
}

func TestPolicyCreateHappyPath(t *testing.T) {
	root := command.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"policy", "create", "--name", "my-policy"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("policy create --name my-policy failed: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Policy bundle created") {
		t.Error("output should contain 'Policy bundle created'")
	}
	if !strings.Contains(output, "my-policy") {
		t.Error("output should mention bundle name my-policy")
	}
	if !strings.Contains(output, "Set as active bundle") {
		t.Error("output should indicate bundle was set as active")
	}
}
