package status

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/Joker-of-Gotham/gitdex/internal/git/cli"
)

type ConfigState struct {
	UserName           string   `json:"user_name,omitempty"`
	UserEmail          string   `json:"user_email,omitempty"`
	IdentityConfigured bool     `json:"identity_configured"`
	DefaultBranch      string   `json:"default_branch,omitempty"`
	CredentialHelper   string   `json:"credential_helper,omitempty"`
	SSHKeyFiles        []string `json:"ssh_key_files,omitempty"`
	Hooks              []string `json:"hooks,omitempty"`
	CoreAutoCRLF       string   `json:"core_autocrlf,omitempty"`
	PullRebase         string   `json:"pull_rebase,omitempty"`
}

func enrichConfigState(ctx context.Context, gitCLI cli.GitCLI, state *GitState) {
	cs := &ConfigState{}

	cs.UserName = gitConfigGet(ctx, gitCLI, "user.name")
	cs.UserEmail = gitConfigGet(ctx, gitCLI, "user.email")
	cs.IdentityConfigured = cs.UserName != "" && cs.UserEmail != ""
	cs.DefaultBranch = gitConfigGet(ctx, gitCLI, "init.defaultbranch")
	cs.CredentialHelper = gitConfigGet(ctx, gitCLI, "credential.helper")
	cs.CoreAutoCRLF = gitConfigGet(ctx, gitCLI, "core.autocrlf")
	cs.PullRebase = gitConfigGet(ctx, gitCLI, "pull.rebase")

	cs.SSHKeyFiles = detectSSHKeys()
	cs.Hooks = detectHooks(ctx, gitCLI)

	state.ConfigInfo = cs
}

func gitConfigGet(ctx context.Context, gitCLI cli.GitCLI, key string) string {
	stdout, _, err := gitCLI.Exec(ctx, "config", "--get", key)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(stdout)
}

func detectSSHKeys() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	sshDir := filepath.Join(home, ".ssh")
	entries, err := os.ReadDir(sshDir)
	if err != nil {
		return nil
	}
	var keys []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".pub") {
			keys = append(keys, name)
		}
	}
	return keys
}

func detectHooks(ctx context.Context, gitCLI cli.GitCLI) []string {
	stdout, _, err := gitCLI.Exec(ctx, "rev-parse", "--git-dir")
	if err != nil {
		return nil
	}
	hooksDir := filepath.Join(strings.TrimSpace(stdout), "hooks")
	entries, err := os.ReadDir(hooksDir)
	if err != nil {
		return nil
	}
	var hooks []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".sample") {
			continue
		}
		hooks = append(hooks, name)
	}
	return hooks
}
