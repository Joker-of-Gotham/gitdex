package integration

import (
	"bytes"
	"strings"
	"testing"

	"github.com/your-org/gitdex/internal/cli/command"
)

func TestIdentityCommandRegistered(t *testing.T) {
	root := command.NewRootCommand()
	identityCmd, _, err := root.Find([]string{"identity"})
	if err != nil {
		t.Fatalf("identity command not found: %v", err)
	}

	subs := []string{"show", "list", "register"}
	for _, name := range subs {
		found := false
		for _, c := range identityCmd.Commands() {
			if c.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("subcommand %q not found under 'identity'", name)
		}
	}
}

func TestIdentityCommandHelp(t *testing.T) {
	root := command.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"identity", "--help"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("identity --help failed: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "show") {
		t.Error("help should mention show subcommand")
	}
	if !strings.Contains(output, "list") {
		t.Error("help should mention list subcommand")
	}
	if !strings.Contains(output, "register") {
		t.Error("help should mention register subcommand")
	}
}

func TestIdentityShowNoIdentity(t *testing.T) {
	root := command.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"identity", "show"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("identity show failed: %v", err)
	}
	if !strings.Contains(out.String(), "No identity") {
		t.Error("output should indicate no identity when none configured")
	}
}

func TestIdentityListEmpty(t *testing.T) {
	root := command.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"identity", "list"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("identity list failed: %v", err)
	}
	if !strings.Contains(out.String(), "No identities") {
		t.Error("output should indicate no identities when list is empty")
	}
}

func TestIdentityRegisterRequiresAppIDForGitHubApp(t *testing.T) {
	root := command.NewRootCommand()
	root.SetArgs([]string{"identity", "register", "--type", "github_app"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when github_app without app-id and installation-id")
	}
}

func TestIdentityRegisterGitHubAppHappyPath(t *testing.T) {
	root := command.NewRootCommand()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"identity", "register", "--type", "github_app", "--app-id", "123", "--installation-id", "456"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("identity register --type github_app --app-id 123 --installation-id 456 failed: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Identity registered") {
		t.Error("output should contain 'Identity registered'")
	}
	if !strings.Contains(output, "github_app") {
		t.Error("output should mention github_app type")
	}
	if !strings.Contains(output, "Set as current identity") {
		t.Error("output should indicate identity was set as current")
	}
}
