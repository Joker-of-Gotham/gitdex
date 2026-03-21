package orchestrator

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/your-org/gitdex/internal/gitops"
	"github.com/your-org/gitdex/internal/platform/config"
	"github.com/your-org/gitdex/internal/planning"
)

func (e *Executor) resolveRepoRoot(ctx context.Context, plan *planning.Plan) (string, error) {
	if e.repoRootResolver != nil {
		return e.repoRootResolver(ctx, plan)
	}
	if strings.TrimSpace(e.repoRoot) != "" {
		return filepath.Clean(strings.TrimSpace(e.repoRoot)), nil
	}
	if plan != nil && strings.TrimSpace(plan.Scope.Owner) != "" && strings.TrimSpace(plan.Scope.Repo) != "" {
		idx := gitops.NewLocalIndex(e.gitExecutor)
		idx.BuildWithRoots(ctx, e.workspaceRoots)
		paths := idx.LookupByOwnerName(plan.Scope.Owner, plan.Scope.Repo)
		if len(paths) > 0 {
			return filepath.Clean(paths[0]), nil
		}
	}
	root, err := config.ResolveRepositoryRoot("")
	if err == nil && strings.TrimSpace(root) != "" {
		return filepath.Clean(root), nil
	}
	return "", fmt.Errorf("no local repository found for execution")
}

func (e *Executor) defaultExecuteStep(ctx context.Context, plan *planning.Plan, repoRoot string, step *StepResult) (string, error) {
	action := normalizeAction(step.Action)
	switch action {
	case "status", "git.status":
		return e.gitRun(ctx, repoRoot, "status", "--short", "--branch")
	case "fetch", "git.fetch":
		if step.Target != "" {
			return e.gitRun(ctx, repoRoot, "fetch", step.Target)
		}
		return e.gitRun(ctx, repoRoot, "fetch")
	case "pull", "git.pull":
		if fields := strings.Fields(step.Target); len(fields) > 0 {
			return e.gitRun(ctx, repoRoot, append([]string{"pull"}, fields...)...)
		}
		return e.gitRun(ctx, repoRoot, "pull")
	case "push", "git.push":
		if fields := strings.Fields(step.Target); len(fields) > 0 {
			return e.gitRun(ctx, repoRoot, append([]string{"push"}, fields...)...)
		}
		return e.gitRun(ctx, repoRoot, "push")
	case "add", "git.add":
		target := strings.TrimSpace(step.Target)
		if target == "" {
			target = "."
		}
		return e.gitRun(ctx, repoRoot, "add", target)
	case "reset", "git.reset":
		target := strings.TrimSpace(step.Target)
		if target == "" {
			return e.gitRun(ctx, repoRoot, "reset")
		}
		return e.gitRun(ctx, repoRoot, "reset", "--", target)
	case "restore", "git.restore":
		if strings.TrimSpace(step.Target) == "" {
			return "", fmt.Errorf("restore requires a target path")
		}
		return e.gitRun(ctx, repoRoot, "restore", step.Target)
	case "commit", "git.commit":
		msg := strings.TrimSpace(step.Target)
		if msg == "" {
			msg = strings.TrimSpace(step.Description)
		}
		if msg == "" {
			return "", fmt.Errorf("commit requires a message in target or description")
		}
		return e.gitRun(ctx, repoRoot, "commit", "-m", msg)
	case "amend", "git.commit.amend":
		return e.gitRun(ctx, repoRoot, "commit", "--amend", "--no-edit")
	case "branch.create", "git.branch.create", "branch":
		if strings.TrimSpace(step.Target) == "" {
			return "", fmt.Errorf("branch create requires a branch name")
		}
		return e.gitRun(ctx, repoRoot, "branch", step.Target)
	case "branch.delete", "git.branch.delete":
		if strings.TrimSpace(step.Target) == "" {
			return "", fmt.Errorf("branch delete requires a branch name")
		}
		return e.gitRun(ctx, repoRoot, "branch", "-d", step.Target)
	case "branch.rename", "git.branch.rename":
		oldName, newName, err := parseSourceTarget(step.Target)
		if err != nil {
			return "", err
		}
		return e.gitRun(ctx, repoRoot, "branch", "-m", oldName, newName)
	case "checkout", "git.checkout", "switch", "git.switch":
		if strings.TrimSpace(step.Target) == "" {
			return "", fmt.Errorf("checkout requires a branch name")
		}
		return e.gitRun(ctx, repoRoot, "checkout", step.Target)
	case "merge", "git.merge":
		if strings.TrimSpace(step.Target) == "" {
			return "", fmt.Errorf("merge requires a source branch")
		}
		return e.gitRun(ctx, repoRoot, "merge", step.Target)
	case "rebase", "git.rebase":
		if strings.TrimSpace(step.Target) == "" {
			return "", fmt.Errorf("rebase requires a branch")
		}
		return e.gitRun(ctx, repoRoot, "rebase", step.Target)
	case "cherry-pick", "git.cherry-pick":
		if strings.TrimSpace(step.Target) == "" {
			return "", fmt.Errorf("cherry-pick requires a commit hash")
		}
		return e.gitRun(ctx, repoRoot, "cherry-pick", step.Target)
	case "stash", "git.stash":
		fields := strings.Fields(step.Target)
		if len(fields) == 0 {
			return e.gitRun(ctx, repoRoot, "stash")
		}
		return e.gitRun(ctx, repoRoot, append([]string{"stash"}, fields...)...)
	case "tag", "git.tag":
		if strings.TrimSpace(step.Target) == "" {
			return "", fmt.Errorf("tag requires a name")
		}
		return e.gitRun(ctx, repoRoot, "tag", step.Target)
	case "gc", "git.gc":
		return e.gitRun(ctx, repoRoot, "gc")
	case "clean", "git.clean":
		return e.gitRun(ctx, repoRoot, "clean", "-fd")
	case "log", "git.log":
		return e.gitRun(ctx, repoRoot, "log", "--oneline", "-20")
	case "mkdir", "file.mkdir":
		target, err := ensureRepoPath(repoRoot, step.Target)
		if err != nil {
			return "", err
		}
		if err := os.MkdirAll(target, 0o755); err != nil {
			return "", err
		}
		return "created directory " + filepath.ToSlash(target), nil
	case "file.write", "write_file":
		target, err := ensureRepoPath(repoRoot, step.Target)
		if err != nil {
			return "", err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return "", err
		}
		if err := os.WriteFile(target, []byte(step.Description), 0o644); err != nil {
			return "", err
		}
		return "wrote file " + filepath.ToSlash(target), nil
	case "file.append", "append_file":
		target, err := ensureRepoPath(repoRoot, step.Target)
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
		if _, err := f.WriteString(step.Description); err != nil {
			return "", err
		}
		return "appended file " + filepath.ToSlash(target), nil
	case "file.delete", "delete_path", "rm":
		target, err := ensureRepoPath(repoRoot, step.Target)
		if err != nil {
			return "", err
		}
		if err := os.RemoveAll(target); err != nil {
			return "", err
		}
		return "deleted " + filepath.ToSlash(target), nil
	case "file.move", "move_path", "mv":
		source, target, err := parseRepoSourceTarget(repoRoot, step.Target)
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
	case "file.copy", "copy_path", "cp":
		source, target, err := parseRepoSourceTarget(repoRoot, step.Target)
		if err != nil {
			return "", err
		}
		if err := copyPath(source, target); err != nil {
			return "", err
		}
		return fmt.Sprintf("copied %s -> %s", filepath.ToSlash(source), filepath.ToSlash(target)), nil
	default:
		return "", fmt.Errorf("unsupported step action %q", step.Action)
	}
}

func (e *Executor) gitRun(ctx context.Context, repoRoot string, args ...string) (string, error) {
	result, err := e.gitExecutor.Run(ctx, repoRoot, args...)
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

func normalizeAction(action string) string {
	return strings.ToLower(strings.TrimSpace(action))
}

func ensureRepoPath(root, rel string) (string, error) {
	if strings.TrimSpace(root) == "" {
		return "", fmt.Errorf("repository root unavailable")
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

func parseSourceTarget(raw string) (string, string, error) {
	value := strings.TrimSpace(raw)
	for _, sep := range []string{"=>", "->", "::"} {
		if idx := strings.Index(value, sep); idx >= 0 {
			left := strings.TrimSpace(value[:idx])
			right := strings.TrimSpace(value[idx+len(sep):])
			if left == "" || right == "" {
				return "", "", fmt.Errorf("source and target are required")
			}
			return left, right, nil
		}
	}
	parts := strings.Fields(value)
	if len(parts) == 2 {
		return parts[0], parts[1], nil
	}
	return "", "", fmt.Errorf("expected <source> <target>")
}

func parseRepoSourceTarget(repoRoot, raw string) (string, string, error) {
	sourceRaw, targetRaw, err := parseSourceTarget(raw)
	if err != nil {
		return "", "", err
	}
	source, err := ensureRepoPath(repoRoot, sourceRaw)
	if err != nil {
		return "", "", err
	}
	target, err := ensureRepoPath(repoRoot, targetRaw)
	if err != nil {
		return "", "", err
	}
	return source, target, nil
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
