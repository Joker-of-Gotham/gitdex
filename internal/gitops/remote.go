package gitops

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// RemoteManager handles remote and clone operations.
type RemoteManager struct {
	executor *GitExecutor
}

// NewRemoteManager creates a new RemoteManager with the given executor.
func NewRemoteManager(executor *GitExecutor) *RemoteManager {
	return &RemoteManager{executor: executor}
}

// CloneOptions configures clone behavior.
type CloneOptions struct {
	Mirror       bool
	Bare         bool
	Depth        int
	Branch       string
	SingleBranch bool
}

// FetchOptions configures fetch behavior.
type FetchOptions struct {
	Prune  bool
	Tags   bool
	Depth  int
	DryRun bool
}

// PushOptions configures push behavior.
type PushOptions struct {
	ForceWithLease bool
	Force          bool
	Tags           bool
	DryRun         bool
	SetUpstream    bool
}

// RemoteInfo holds remote metadata.
type RemoteInfo struct {
	Name     string
	FetchURL string
	PushURL  string
}

// RemoteRef represents a remote reference.
type RemoteRef struct {
	SHA  string
	Ref  string
	Type string
}

// Clone clones a repository from url into dir.
func (rm *RemoteManager) Clone(ctx context.Context, url string, dir string, opts CloneOptions) error {
	parentDir := filepath.Dir(dir)
	repoName := filepath.Base(dir)
	args := []string{"clone"}
	if opts.Mirror {
		args = append(args, "--mirror")
	} else if opts.Bare {
		args = append(args, "--bare")
	}
	if opts.Depth > 0 {
		args = append(args, "--depth", fmt.Sprintf("%d", opts.Depth))
	}
	if opts.Branch != "" {
		args = append(args, "--branch", opts.Branch)
	}
	if opts.SingleBranch {
		args = append(args, "--single-branch")
	}
	args = append(args, url, repoName)
	_, err := rm.executor.Run(ctx, parentDir, args...)
	return err
}

// Fetch fetches from the given remote with optional refspec.
func (rm *RemoteManager) Fetch(ctx context.Context, repoPath string, remote string, refspec string, opts FetchOptions) error {
	args := []string{"fetch"}
	if remote != "" {
		args = append(args, remote)
	}
	if refspec != "" {
		args = append(args, refspec)
	}
	if opts.Prune {
		args = append(args, "--prune")
	}
	if opts.Tags {
		args = append(args, "--tags")
	}
	if opts.Depth > 0 {
		args = append(args, "--depth", fmt.Sprintf("%d", opts.Depth))
	}
	if opts.DryRun {
		args = append(args, "--dry-run")
	}
	_, err := rm.executor.Run(ctx, repoPath, args...)
	return err
}

// FetchAll runs git fetch --all --prune.
func (rm *RemoteManager) FetchAll(ctx context.Context, repoPath string) error {
	_, err := rm.executor.Run(ctx, repoPath, "fetch", "--all", "--prune")
	return err
}

// Push pushes to the given remote with optional refspec.
func (rm *RemoteManager) Push(ctx context.Context, repoPath string, remote string, refspec string, opts PushOptions) error {
	args := []string{"push"}
	if remote != "" {
		args = append(args, remote)
	}
	if refspec != "" {
		args = append(args, refspec)
	}
	if opts.ForceWithLease {
		args = append(args, "--force-with-lease")
	}
	if opts.Force {
		args = append(args, "--force")
	}
	if opts.Tags {
		args = append(args, "--tags")
	}
	if opts.DryRun {
		args = append(args, "--dry-run")
	}
	if opts.SetUpstream {
		args = append(args, "--set-upstream")
	}
	_, err := rm.executor.Run(ctx, repoPath, args...)
	return err
}

// PushDelete deletes a ref on the remote.
func (rm *RemoteManager) PushDelete(ctx context.Context, repoPath string, remote string, ref string) error {
	if remote == "" || ref == "" {
		return fmt.Errorf("remote and ref are required")
	}
	_, err := rm.executor.Run(ctx, repoPath, "push", remote, "--delete", ref)
	return err
}

// ListRemotes parses git remote -v and returns RemoteInfo for each remote.
func (rm *RemoteManager) ListRemotes(ctx context.Context, repoPath string) ([]RemoteInfo, error) {
	lines, err := rm.executor.RunLines(ctx, repoPath, "remote", "-v")
	if err != nil {
		return nil, err
	}
	byName := make(map[string]*RemoteInfo)
	remoteRe := regexp.MustCompile(`^(\S+)\s+(\S+)\s+\((\w+)\)$`)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		m := remoteRe.FindStringSubmatch(line)
		if len(m) < 4 {
			continue
		}
		name, url, kind := m[1], m[2], m[3]
		if byName[name] == nil {
			byName[name] = &RemoteInfo{Name: name}
		}
		if kind == "fetch" {
			byName[name].FetchURL = url
		} else if kind == "push" {
			byName[name].PushURL = url
		}
	}
	var result []RemoteInfo
	for _, r := range byName {
		result = append(result, *r)
	}
	return result, nil
}

// AddRemote adds a remote.
func (rm *RemoteManager) AddRemote(ctx context.Context, repoPath string, name string, url string) error {
	_, err := rm.executor.Run(ctx, repoPath, "remote", "add", name, url)
	return err
}

// RemoveRemote removes a remote.
func (rm *RemoteManager) RemoveRemote(ctx context.Context, repoPath string, name string) error {
	_, err := rm.executor.Run(ctx, repoPath, "remote", "remove", name)
	return err
}

// SetRemoteURL sets the URL for a remote.
func (rm *RemoteManager) SetRemoteURL(ctx context.Context, repoPath string, name string, url string) error {
	_, err := rm.executor.Run(ctx, repoPath, "remote", "set-url", name, url)
	return err
}

// LsRemote lists refs on the remote.
func (rm *RemoteManager) LsRemote(ctx context.Context, repoPath string, remote string, pattern string) ([]RemoteRef, error) {
	args := []string{"ls-remote"}
	if remote != "" {
		args = append(args, remote)
	}
	if pattern != "" {
		args = append(args, pattern)
	}
	lines, err := rm.executor.RunLines(ctx, repoPath, args...)
	if err != nil {
		return nil, err
	}
	var result []RemoteRef
	refRe := regexp.MustCompile(`^([a-f0-9]{40})\s+(.+)`)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		m := refRe.FindStringSubmatch(line)
		if len(m) < 3 {
			continue
		}
		r := RemoteRef{SHA: m[1], Ref: m[2]}
		if strings.HasPrefix(m[2], "refs/heads/") {
			r.Type = "branch"
		} else if strings.HasPrefix(m[2], "refs/tags/") {
			r.Type = "tag"
		} else {
			r.Type = "ref"
		}
		result = append(result, r)
	}
	return result, nil
}

// SubmoduleInit runs git submodule init.
func (rm *RemoteManager) SubmoduleInit(ctx context.Context, repoPath string) error {
	_, err := rm.executor.Run(ctx, repoPath, "submodule", "init")
	return err
}

// SubmoduleUpdate runs git submodule update, with optional --recursive.
func (rm *RemoteManager) SubmoduleUpdate(ctx context.Context, repoPath string, recursive bool) error {
	args := []string{"submodule", "update"}
	if recursive {
		args = append(args, "--recursive")
	}
	_, err := rm.executor.Run(ctx, repoPath, args...)
	return err
}

// SubmoduleStatus returns the output of git submodule status.
func (rm *RemoteManager) SubmoduleStatus(ctx context.Context, repoPath string) (string, error) {
	result, err := rm.executor.Run(ctx, repoPath, "submodule", "status")
	if err != nil {
		return "", err
	}
	return result.Stdout, nil
}
