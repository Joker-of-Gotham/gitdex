package autonomyexec

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/your-org/gitdex/internal/autonomy"
	"github.com/your-org/gitdex/internal/gitops"
	ghclient "github.com/your-org/gitdex/internal/platform/github"
)

func buildToolRegistry(repoRoot string, ghClient *ghclient.Client, owner, repoName string) *autonomy.ToolRegistry {
	registry := autonomy.NewToolRegistry()
	registerFileTools(registry, repoRoot)
	registerGitTools(registry, repoRoot)
	registerGitHubTools(registry, ghClient, owner, repoName)
	return registry
}

func registerFileTools(registry *autonomy.ToolRegistry, repoRoot string) {
	register := func(name, desc string, params map[string]autonomy.ToolParam, handler autonomy.ActionHandler) {
		registry.Register(autonomy.Tool{
			Name:        name,
			Description: desc,
			Parameters:  params,
			Handler:     handler,
		})
	}

	register("file.mkdir", "Create a directory inside the local clone", map[string]autonomy.ToolParam{
		"path": {Name: "path", Type: "string", Description: "Directory path", Required: true},
	}, func(_ context.Context, args map[string]string) (string, error) {
		target, err := ensureRepoPath(repoRoot, args["path"])
		if err != nil {
			return "", err
		}
		if err := os.MkdirAll(target, 0o755); err != nil {
			return "", err
		}
		return "created directory " + filepath.ToSlash(target), nil
	})

	register("file.write", "Write a file inside the local clone", map[string]autonomy.ToolParam{
		"path":    {Name: "path", Type: "string", Description: "File path", Required: true},
		"content": {Name: "content", Type: "string", Description: "File content", Required: true},
	}, func(_ context.Context, args map[string]string) (string, error) {
		target, err := ensureRepoPath(repoRoot, args["path"])
		if err != nil {
			return "", err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return "", err
		}
		if err := os.WriteFile(target, []byte(args["content"]), 0o644); err != nil {
			return "", err
		}
		return "wrote file " + filepath.ToSlash(target), nil
	})

	register("file.append", "Append content to a file inside the local clone", map[string]autonomy.ToolParam{
		"path":    {Name: "path", Type: "string", Description: "File path", Required: true},
		"content": {Name: "content", Type: "string", Description: "Appended content", Required: true},
	}, func(_ context.Context, args map[string]string) (string, error) {
		target, err := ensureRepoPath(repoRoot, args["path"])
		if err != nil {
			return "", err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return "", err
		}
		f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return "", err
		}
		defer f.Close()
		if _, err := f.WriteString(args["content"]); err != nil {
			return "", err
		}
		return "appended file " + filepath.ToSlash(target), nil
	})

	register("file.delete", "Delete a file or directory inside the local clone", map[string]autonomy.ToolParam{
		"path": {Name: "path", Type: "string", Description: "Path to delete", Required: true},
	}, func(_ context.Context, args map[string]string) (string, error) {
		target, err := ensureRepoPath(repoRoot, args["path"])
		if err != nil {
			return "", err
		}
		if err := os.RemoveAll(target); err != nil {
			return "", err
		}
		return "deleted " + filepath.ToSlash(target), nil
	})

	register("file.move", "Move or rename a path inside the local clone", map[string]autonomy.ToolParam{
		"source": {Name: "source", Type: "string", Description: "Source path", Required: true},
		"target": {Name: "target", Type: "string", Description: "Target path", Required: true},
	}, func(_ context.Context, args map[string]string) (string, error) {
		source, err := ensureRepoPath(repoRoot, args["source"])
		if err != nil {
			return "", err
		}
		target, err := ensureRepoPath(repoRoot, args["target"])
		if err != nil {
			return "", err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return "", err
		}
		if err := os.Rename(source, target); err != nil {
			return "", err
		}
		return fmt.Sprintf("moved %s -> %s", filepath.ToSlash(source), filepath.ToSlash(target)), nil
	})

	register("file.copy", "Copy a file or directory inside the local clone", map[string]autonomy.ToolParam{
		"source": {Name: "source", Type: "string", Description: "Source path", Required: true},
		"target": {Name: "target", Type: "string", Description: "Target path", Required: true},
	}, func(_ context.Context, args map[string]string) (string, error) {
		source, err := ensureRepoPath(repoRoot, args["source"])
		if err != nil {
			return "", err
		}
		target, err := ensureRepoPath(repoRoot, args["target"])
		if err != nil {
			return "", err
		}
		if err := copyPath(source, target); err != nil {
			return "", err
		}
		return fmt.Sprintf("copied %s -> %s", filepath.ToSlash(source), filepath.ToSlash(target)), nil
	})
}

func registerGitTools(registry *autonomy.ToolRegistry, repoRoot string) {
	register := func(name, desc string, params map[string]autonomy.ToolParam, handler autonomy.ActionHandler) {
		registry.Register(autonomy.Tool{
			Name:        name,
			Description: desc,
			Parameters:  params,
			Handler:     handler,
		})
	}

	run := func(ctx context.Context, args ...string) (string, error) {
		if strings.TrimSpace(repoRoot) == "" {
			return "", fmt.Errorf("no local clone available for git action")
		}
		result, err := gitops.NewGitExecutor().Run(ctx, repoRoot, args...)
		if err != nil {
			return "", err
		}
		out := strings.TrimSpace(result.Stdout)
		if out == "" {
			out = strings.TrimSpace(result.Stderr)
		}
		if out == "" {
			out = "ok"
		}
		return out, nil
	}

	register("git.status", "Show repository status", nil, func(ctx context.Context, _ map[string]string) (string, error) {
		return run(ctx, "status", "--short", "--branch")
	})
	register("git.fetch", "Fetch remote updates", map[string]autonomy.ToolParam{
		"remote": {Name: "remote", Type: "string", Description: "Remote name", Required: false},
	}, func(ctx context.Context, args map[string]string) (string, error) {
		if remote := strings.TrimSpace(args["remote"]); remote != "" {
			return run(ctx, "fetch", remote)
		}
		return run(ctx, "fetch")
	})
	register("git.pull", "Pull upstream changes", map[string]autonomy.ToolParam{
		"remote": {Name: "remote", Type: "string", Description: "Remote name", Required: false},
		"branch": {Name: "branch", Type: "string", Description: "Branch name", Required: false},
	}, func(ctx context.Context, args map[string]string) (string, error) {
		cmdArgs := []string{"pull"}
		if remote := strings.TrimSpace(args["remote"]); remote != "" {
			cmdArgs = append(cmdArgs, remote)
		}
		if branch := strings.TrimSpace(args["branch"]); branch != "" {
			cmdArgs = append(cmdArgs, branch)
		}
		return run(ctx, cmdArgs...)
	})
	register("git.push", "Push local changes", map[string]autonomy.ToolParam{
		"remote": {Name: "remote", Type: "string", Description: "Remote name", Required: false},
		"branch": {Name: "branch", Type: "string", Description: "Branch name", Required: false},
	}, func(ctx context.Context, args map[string]string) (string, error) {
		cmdArgs := []string{"push"}
		if remote := strings.TrimSpace(args["remote"]); remote != "" {
			cmdArgs = append(cmdArgs, remote)
		}
		if branch := strings.TrimSpace(args["branch"]); branch != "" {
			cmdArgs = append(cmdArgs, branch)
		}
		return run(ctx, cmdArgs...)
	})
	register("git.add", "Stage a file or path", map[string]autonomy.ToolParam{
		"path": {Name: "path", Type: "string", Description: "Path to stage", Required: false},
	}, func(ctx context.Context, args map[string]string) (string, error) {
		path := strings.TrimSpace(args["path"])
		if path == "" {
			path = "."
		}
		return run(ctx, "add", path)
	})
	register("git.reset", "Unstage changes", map[string]autonomy.ToolParam{
		"path": {Name: "path", Type: "string", Description: "Path to reset", Required: false},
	}, func(ctx context.Context, args map[string]string) (string, error) {
		path := strings.TrimSpace(args["path"])
		if path == "" {
			return run(ctx, "reset")
		}
		return run(ctx, "reset", "--", path)
	})
	register("git.restore", "Restore a file from HEAD", map[string]autonomy.ToolParam{
		"path": {Name: "path", Type: "string", Description: "Path to restore", Required: true},
	}, func(ctx context.Context, args map[string]string) (string, error) {
		return run(ctx, "restore", strings.TrimSpace(args["path"]))
	})
	register("git.commit", "Create a commit", map[string]autonomy.ToolParam{
		"message": {Name: "message", Type: "string", Description: "Commit message", Required: true},
	}, func(ctx context.Context, args map[string]string) (string, error) {
		return run(ctx, "commit", "-m", strings.TrimSpace(args["message"]))
	})
	register("git.commit.amend", "Amend the previous commit", nil, func(ctx context.Context, _ map[string]string) (string, error) {
		return run(ctx, "commit", "--amend", "--no-edit")
	})
	register("git.branch.create", "Create a branch", map[string]autonomy.ToolParam{
		"name": {Name: "name", Type: "string", Description: "Branch name", Required: true},
	}, func(ctx context.Context, args map[string]string) (string, error) {
		return run(ctx, "branch", strings.TrimSpace(args["name"]))
	})
	register("git.branch.delete", "Delete a branch", map[string]autonomy.ToolParam{
		"name": {Name: "name", Type: "string", Description: "Branch name", Required: true},
	}, func(ctx context.Context, args map[string]string) (string, error) {
		return run(ctx, "branch", "-d", strings.TrimSpace(args["name"]))
	})
	register("git.branch.rename", "Rename a branch", map[string]autonomy.ToolParam{
		"old_name": {Name: "old_name", Type: "string", Description: "Old branch name", Required: true},
		"new_name": {Name: "new_name", Type: "string", Description: "New branch name", Required: true},
	}, func(ctx context.Context, args map[string]string) (string, error) {
		return run(ctx, "branch", "-m", strings.TrimSpace(args["old_name"]), strings.TrimSpace(args["new_name"]))
	})
	register("git.checkout", "Checkout a branch or ref", map[string]autonomy.ToolParam{
		"branch": {Name: "branch", Type: "string", Description: "Branch or ref", Required: true},
	}, func(ctx context.Context, args map[string]string) (string, error) {
		return run(ctx, "checkout", strings.TrimSpace(args["branch"]))
	})
	register("git.merge", "Merge a branch", map[string]autonomy.ToolParam{
		"branch": {Name: "branch", Type: "string", Description: "Branch to merge", Required: true},
	}, func(ctx context.Context, args map[string]string) (string, error) {
		return run(ctx, "merge", strings.TrimSpace(args["branch"]))
	})
	register("git.rebase", "Rebase onto a branch", map[string]autonomy.ToolParam{
		"branch": {Name: "branch", Type: "string", Description: "Target branch", Required: true},
	}, func(ctx context.Context, args map[string]string) (string, error) {
		return run(ctx, "rebase", strings.TrimSpace(args["branch"]))
	})
	register("git.cherry-pick", "Cherry-pick a commit", map[string]autonomy.ToolParam{
		"commit": {Name: "commit", Type: "string", Description: "Commit SHA", Required: true},
	}, func(ctx context.Context, args map[string]string) (string, error) {
		return run(ctx, "cherry-pick", strings.TrimSpace(args["commit"]))
	})
	register("git.stash", "Create a stash entry", map[string]autonomy.ToolParam{
		"message": {Name: "message", Type: "string", Description: "Optional stash message", Required: false},
	}, func(ctx context.Context, args map[string]string) (string, error) {
		message := strings.TrimSpace(args["message"])
		if message == "" {
			return run(ctx, "stash")
		}
		return run(ctx, "stash", "push", "-m", message)
	})
	register("git.tag", "Create a tag", map[string]autonomy.ToolParam{
		"name": {Name: "name", Type: "string", Description: "Tag name", Required: true},
	}, func(ctx context.Context, args map[string]string) (string, error) {
		return run(ctx, "tag", strings.TrimSpace(args["name"]))
	})
	register("git.gc", "Run git garbage collection", nil, func(ctx context.Context, _ map[string]string) (string, error) {
		return run(ctx, "gc")
	})
	register("git.clean", "Delete untracked files", nil, func(ctx context.Context, _ map[string]string) (string, error) {
		return run(ctx, "clean", "-fd")
	})
	register("git.log", "Show recent commit log", nil, func(ctx context.Context, _ map[string]string) (string, error) {
		return run(ctx, "log", "--oneline", "-20")
	})
}

func ensureRepoPath(root, rel string) (string, error) {
	if strings.TrimSpace(root) == "" {
		return "", fmt.Errorf("no local clone available for file action")
	}
	if strings.TrimSpace(rel) == "" {
		return "", fmt.Errorf("path is required")
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	target := filepath.Join(absRoot, filepath.Clean(rel))
	relative, err := filepath.Rel(absRoot, target)
	if err != nil {
		return "", err
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("path escapes repository root")
	}
	return target, nil
}

func copyPath(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		if err := os.MkdirAll(dst, info.Mode().Perm()); err != nil {
			return err
		}
		entries, err := os.ReadDir(src)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if err := copyPath(filepath.Join(src, entry.Name()), filepath.Join(dst, entry.Name())); err != nil {
				return err
			}
		}
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode().Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}
