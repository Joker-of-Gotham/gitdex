package app

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/your-org/gitdex/internal/gitops"
	"github.com/your-org/gitdex/internal/state/repo"
	"github.com/your-org/gitdex/internal/tui/views"
)

func (m *Model) registerRepoCommands() {
	m.registerFileCommands()
	m.registerGitCommands()
	m.registerGitHubCommands()
}

func (m *Model) requireRepo(fn func(args string) string) CommandHandler {
	return func(args string) string {
		if m.activeRepo == nil {
			return "尚未进入仓库。请先在 Dashboard > Repos 中选择仓库。"
		}
		return fn(args)
	}
}

func (m *Model) requireWritable(fn func(args string) string) CommandHandler {
	return func(args string) string {
		if m.activeRepo == nil {
			return "尚未进入仓库。请先在 Dashboard > Repos 中选择仓库。"
		}
		if m.activeRepo.IsReadOnly {
			return "当前仓库处于只读模式。请先切换到本地克隆后再执行写操作。"
		}
		return fn(args)
	}
}

func (m *Model) requireLocalPath(fn func(args string) string) CommandHandler {
	return func(args string) string {
		if m.activeRepo == nil {
			return "尚未进入仓库。请先在 Dashboard > Repos 中选择仓库。"
		}
		if strings.TrimSpace(m.repoRoot()) == "" {
			return "当前仓库没有本地路径。请先克隆或选择本地仓库后再执行该命令。"
		}
		return fn(args)
	}
}

func (m *Model) repoRoot() string {
	if m.activeRepo != nil && m.activeRepo.LocalPath() != "" {
		return m.activeRepo.LocalPath()
	}
	return ""
}

func (m *Model) activeRepoCoordinates() (string, string) {
	if m.activeRepo == nil {
		return "", ""
	}
	owner := strings.TrimSpace(m.activeRepo.Owner)
	name := strings.TrimSpace(m.activeRepo.Name)
	if owner != "" && name != "" {
		return owner, name
	}
	if root := m.repoRoot(); root != "" {
		return parseRemoteOwnerRepo(root)
	}
	return "", name
}

func gitRun(root string, args ...string) (string, error) {
	executor := gitops.NewGitExecutor()
	result, err := executor.Run(context.Background(), root, args...)
	if err != nil {
		return "", err
	}
	out := result.Stdout
	if out == "" {
		out = result.Stderr
	}
	return strings.TrimSpace(out), nil
}

func parseRemoteOwnerRepo(root string) (string, string) {
	if root == "" {
		return "", ""
	}
	url, err := gitRun(root, "config", "--get", "remote.origin.url")
	if err != nil || strings.TrimSpace(url) == "" {
		return "", ""
	}
	url = strings.TrimSpace(strings.TrimSuffix(url, ".git"))
	if strings.Contains(url, "://") {
		parts := strings.Split(url, "/")
		if len(parts) >= 2 {
			return parts[len(parts)-2], parts[len(parts)-1]
		}
	}
	if idx := strings.LastIndex(url, ":"); idx >= 0 {
		parts := strings.Split(strings.TrimPrefix(url[idx+1:], "/"), "/")
		if len(parts) >= 2 {
			return parts[len(parts)-2], parts[len(parts)-1]
		}
	}
	return "", ""
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

func splitCommandBody(raw string) (string, string) {
	if idx := strings.Index(raw, " -- "); idx >= 0 {
		return strings.TrimSpace(raw[:idx]), strings.TrimSpace(raw[idx+4:])
	}
	return strings.TrimSpace(raw), ""
}

func parseEditArgs(raw string) (path string, appendMode bool, content string, ok bool) {
	if idx := strings.Index(raw, " ++ "); idx >= 0 {
		return strings.TrimSpace(raw[:idx]), true, raw[idx+4:], true
	}
	if idx := strings.Index(raw, " -- "); idx >= 0 {
		return strings.TrimSpace(raw[:idx]), false, raw[idx+4:], true
	}
	return "", false, "", false
}

func parseNumberAndBody(raw, usage string) (int, string, string) {
	head, body := splitCommandBody(raw)
	parts := strings.Fields(head)
	if len(parts) == 0 {
		return 0, "", usage
	}
	number, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, "", usage
	}
	text := strings.TrimSpace(strings.TrimPrefix(head, parts[0]))
	if body != "" {
		text = body
	}
	return number, strings.TrimSpace(text), ""
}

func currentBranch(root string) (string, error) {
	branch, err := gitRun(root, "branch", "--show-current")
	if err != nil {
		return "", err
	}
	branch = strings.TrimSpace(branch)
	if branch == "" {
		return "", fmt.Errorf("current branch is empty")
	}
	return branch, nil
}

func relativeRepoPath(root, target string) string {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return target
	}
	return filepath.ToSlash(rel)
}

func normalizeListArg(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func previewURL(url string) string {
	if strings.TrimSpace(url) == "" {
		return ""
	}
	return "\n" + url
}

func nonEmptyRepoFullName(activeRepo *repo.RepoContext, owner, name string) string {
	if activeRepo != nil && strings.TrimSpace(activeRepo.FullName) != "" {
		return activeRepo.FullName
	}
	if owner != "" && name != "" {
		return owner + "/" + name
	}
	return name
}

func defaultActiveBranch(activeRepo *repo.RepoContext) string {
	if activeRepo == nil {
		return ""
	}
	return strings.TrimSpace(activeRepo.DefaultBranch)
}

func (m *Model) requireGitHubRepo() (string, string, string) {
	if m.ghClient == nil {
		return "", "", "GitHub 身份未配置或初始化失败。请先在 Settings 中完成 GitHub App/PAT 配置。"
	}
	owner, repo := m.activeRepoCoordinates()
	if owner == "" || repo == "" {
		return "", "", "无法确定 GitHub 仓库坐标。请确保当前仓库存在 origin remote 或已正确选择远端仓库。"
	}
	return owner, repo, ""
}

func (m *Model) registerFileCommands() {
	m.cmdHandlers["clone"] = m.requireRepo(func(args string) string {
		if m.activeRepo != nil && m.activeRepo.IsLocal && m.activeRepo.LocalPath() != "" {
			return fmt.Sprintf("当前仓库已经有本地克隆: %s", m.activeRepo.LocalPath())
		}
		owner, repoName, issue := m.requireGitHubRepo()
		if issue != "" {
			return issue
		}
		item := views.RepoListItem{
			Owner:         owner,
			Name:          repoName,
			FullName:      nonEmptyRepoFullName(m.activeRepo, owner, repoName),
			DefaultBranch: defaultActiveBranch(m.activeRepo),
		}
		target := strings.TrimSpace(args)
		if target == "" {
			target = m.defaultCloneTarget(item)
		}
		m.queuePostCommand(m.cloneRepo(item, target))
		return fmt.Sprintf("开始克隆 %s 到 %s", item.FullName, target)
	})

	m.cmdHandlers["new"] = m.requireWritable(func(args string) string {
		path := strings.TrimSpace(args)
		if path == "" {
			return "用法: /new <path>"
		}
		root := m.repoRoot()
		target, err := ensureRepoPath(root, path)
		if err != nil {
			return fmt.Sprintf("创建失败: %v", err)
		}
		if strings.HasSuffix(path, "/") || strings.HasSuffix(path, "\\") {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return fmt.Sprintf("创建目录失败: %v", err)
			}
			return fmt.Sprintf("已创建目录: %s", relativeRepoPath(root, target))
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fmt.Sprintf("创建父目录失败: %v", err)
		}
		file, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err != nil {
			if os.IsExist(err) {
				return fmt.Sprintf("文件已存在: %s", relativeRepoPath(root, target))
			}
			return fmt.Sprintf("创建文件失败: %v", err)
		}
		_ = file.Close()
		return fmt.Sprintf("已创建文件: %s", relativeRepoPath(root, target))
	})

	m.cmdHandlers["mkdir"] = m.requireWritable(func(args string) string {
		path := strings.TrimSpace(args)
		if path == "" {
			return "Usage: /mkdir <path>"
		}
		root := m.repoRoot()
		target, err := ensureRepoPath(root, path)
		if err != nil {
			return fmt.Sprintf("Create directory failed: %v", err)
		}
		if err := os.MkdirAll(target, 0o755); err != nil {
			return fmt.Sprintf("Create directory failed: %v", err)
		}
		return fmt.Sprintf("Created directory %s", relativeRepoPath(root, target))
	})

	m.cmdHandlers["edit"] = m.requireWritable(func(args string) string {
		path, appendMode, content, ok := parseEditArgs(args)
		if !ok || strings.TrimSpace(path) == "" {
			return "用法: /edit <path> -- <content>\n追加: /edit <path> ++ <content>"
		}
		root := m.repoRoot()
		target, err := ensureRepoPath(root, path)
		if err != nil {
			return fmt.Sprintf("编辑失败: %v", err)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fmt.Sprintf("创建父目录失败: %v", err)
		}
		if appendMode {
			file, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
			if err != nil {
				return fmt.Sprintf("追加失败: %v", err)
			}
			defer file.Close()
			if _, err := file.WriteString(content); err != nil {
				return fmt.Sprintf("追加失败: %v", err)
			}
			return fmt.Sprintf("已追加内容: %s", relativeRepoPath(root, target))
		}
		if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
			return fmt.Sprintf("写入失败: %v", err)
		}
		return fmt.Sprintf("已写入文件: %s", relativeRepoPath(root, target))
	})

	m.cmdHandlers["rm"] = m.requireWritable(func(args string) string {
		raw := strings.TrimSpace(args)
		if raw == "" {
			return "用法: /rm --confirm <path>"
		}
		if !strings.HasPrefix(raw, "--confirm ") {
			return fmt.Sprintf("删除已拦截。请确认后重试: /rm --confirm %s", raw)
		}
		path := strings.TrimSpace(strings.TrimPrefix(raw, "--confirm "))
		if path == "" {
			return "用法: /rm --confirm <path>"
		}
		root := m.repoRoot()
		target, err := ensureRepoPath(root, path)
		if err != nil {
			return fmt.Sprintf("删除失败: %v", err)
		}
		if _, err := os.Stat(target); err != nil {
			if os.IsNotExist(err) {
				return fmt.Sprintf("路径不存在: %s", relativeRepoPath(root, target))
			}
			return fmt.Sprintf("删除失败: %v", err)
		}
		if err := os.RemoveAll(target); err != nil {
			return fmt.Sprintf("删除失败: %v", err)
		}
		return fmt.Sprintf("已删除: %s", relativeRepoPath(root, target))
	})

	m.cmdHandlers["mv"] = m.requireWritable(func(args string) string {
		parts := strings.Fields(args)
		if len(parts) != 2 {
			return "Usage: /mv <source> <target>"
		}
		root := m.repoRoot()
		source, err := ensureRepoPath(root, parts[0])
		if err != nil {
			return fmt.Sprintf("Move failed: %v", err)
		}
		target, err := ensureRepoPath(root, parts[1])
		if err != nil {
			return fmt.Sprintf("Move failed: %v", err)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fmt.Sprintf("Move failed: %v", err)
		}
		if err := os.Rename(source, target); err != nil {
			return fmt.Sprintf("Move failed: %v", err)
		}
		return fmt.Sprintf("Moved %s -> %s", relativeRepoPath(root, source), relativeRepoPath(root, target))
	})

	m.cmdHandlers["cp"] = m.requireWritable(func(args string) string {
		parts := strings.Fields(args)
		if len(parts) != 2 {
			return "Usage: /cp <source> <target>"
		}
		root := m.repoRoot()
		source, err := ensureRepoPath(root, parts[0])
		if err != nil {
			return fmt.Sprintf("Copy failed: %v", err)
		}
		target, err := ensureRepoPath(root, parts[1])
		if err != nil {
			return fmt.Sprintf("Copy failed: %v", err)
		}
		if err := copyPath(source, target); err != nil {
			return fmt.Sprintf("Copy failed: %v", err)
		}
		return fmt.Sprintf("Copied %s -> %s", relativeRepoPath(root, source), relativeRepoPath(root, target))
	})

	m.cmdHandlers["diff"] = m.requireRepo(func(args string) string {
		root := m.repoRoot()
		if root == "" {
			return "无法确定仓库根目录"
		}
		executor := gitops.NewGitExecutor()
		pm := gitops.NewPatchManager(executor)
		opts := &gitops.DiffOptions{}
		path := strings.TrimSpace(args)
		if path != "" {
			opts.Paths = []string{path}
		}
		diff, err := pm.Diff(context.Background(), root, opts)
		if err != nil {
			return fmt.Sprintf("diff 失败: %v", err)
		}
		if diff == "" {
			return "(无变更)"
		}
		return diff
	})

	m.cmdHandlers["search"] = m.requireRepo(func(args string) string {
		pattern := strings.TrimSpace(args)
		if pattern == "" {
			return "用法: /search <pattern>"
		}
		root := m.repoRoot()
		if root == "" {
			return "无法确定仓库根目录"
		}
		query := strings.ToLower(pattern)
		var matches []string
		_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				if d.Name() == ".git" {
					return filepath.SkipDir
				}
				return nil
			}
			if len(matches) >= 50 {
				return fs.SkipAll
			}
			file, err := os.Open(path)
			if err != nil {
				return nil
			}
			defer file.Close()
			scanner := bufio.NewScanner(file)
			lineNo := 0
			for scanner.Scan() {
				lineNo++
				line := scanner.Text()
				if strings.Contains(strings.ToLower(line), query) {
					matches = append(matches, fmt.Sprintf("%s:%d: %s", relativeRepoPath(root, path), lineNo, line))
					if len(matches) >= 50 {
						return fs.SkipAll
					}
				}
			}
			return nil
		})
		if len(matches) == 0 {
			return "未找到匹配内容"
		}
		if len(matches) >= 50 {
			matches = append(matches, "... (仅显示前 50 条匹配)")
		}
		return strings.Join(matches, "\n")
	})

	m.cmdHandlers["find"] = m.requireRepo(func(args string) string {
		name := strings.TrimSpace(args)
		if name == "" {
			return "用法: /find <name>"
		}
		root := m.repoRoot()
		if root == "" {
			return "无法确定仓库根目录"
		}
		query := strings.ToLower(name)
		var matches []string
		_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				if d.Name() == ".git" {
					return filepath.SkipDir
				}
				return nil
			}
			if strings.Contains(strings.ToLower(d.Name()), query) {
				matches = append(matches, relativeRepoPath(root, path))
			}
			return nil
		})
		if len(matches) == 0 {
			return "未找到匹配文件"
		}
		return strings.Join(matches, "\n")
	})

	m.cmdHandlers["chmod"] = m.requireWritable(m.requireLocalPath(func(args string) string {
		parts := strings.Fields(args)
		if len(parts) != 2 {
			return "Usage: /chmod <mode> <file>\nExample: /chmod 644 README.md"
		}
		mode, err := parseOctalFileMode(parts[0])
		if err != nil {
			return fmt.Sprintf("Invalid mode %q: %v", parts[0], err)
		}
		root := m.repoRoot()
		target, err := ensureRepoPath(root, parts[1])
		if err != nil {
			return fmt.Sprintf("chmod failed: %v", err)
		}
		if err := os.Chmod(target, mode); err != nil {
			return fmt.Sprintf("chmod failed: %v", err)
		}
		return fmt.Sprintf("chmod %s: %s", parts[0], relativeRepoPath(root, target))
	}))

	m.cmdHandlers["symlink"] = m.requireWritable(m.requireLocalPath(func(args string) string {
		parts := strings.Fields(args)
		if len(parts) < 2 {
			return "Usage: /symlink <target> <linkname>"
		}
		target := parts[0]
		linkRel := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(args), parts[0]))
		linkRel = strings.TrimSpace(linkRel)
		if linkRel == "" {
			return "Usage: /symlink <target> <linkname>"
		}
		root := m.repoRoot()
		linkAbs, err := ensureRepoPath(root, filepath.ToSlash(linkRel))
		if err != nil {
			return fmt.Sprintf("symlink failed: %v", err)
		}
		if err := os.Symlink(target, linkAbs); err != nil {
			return fmt.Sprintf("symlink failed: %v", err)
		}
		return fmt.Sprintf("symlink %s -> %s", relativeRepoPath(root, linkAbs), target)
	}))

	m.cmdHandlers["archive"] = m.requireRepo(m.requireLocalPath(func(args string) string {
		parts := strings.Fields(args)
		if len(parts) < 1 {
			return "Usage: /archive <tar.gz|zip> [output]"
		}
		format := parts[0]
		if format != "tar.gz" && format != "zip" {
			return "Format must be tar.gz or zip"
		}
		root := m.repoRoot()
		outArg := ""
		if len(parts) >= 2 {
			outArg = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(args), parts[0]))
		}
		if outArg == "" {
			outArg = repoArchiveBasename(m) + "." + format
		}
		outAbs, err := resolveOutputUnderRepo(root, outArg)
		if err != nil {
			return fmt.Sprintf("archive failed: %v", err)
		}
		executor := gitops.NewGitExecutor()
		_, err = executor.Run(context.Background(), root, "archive", "--format="+format, "--output="+outAbs, "HEAD")
		if err != nil {
			return fmt.Sprintf("git archive failed: %v", err)
		}
		return fmt.Sprintf("Created archive: %s", relativeRepoPath(root, outAbs))
	}))

	m.cmdHandlers["patch"] = m.requireWritable(m.requireLocalPath(func(args string) string {
		args = strings.TrimSpace(args)
		if args == "" {
			return "Usage: /patch apply|check|reverse <file>"
		}
		idx := strings.IndexByte(args, ' ')
		if idx < 0 {
			return "Usage: /patch apply|check|reverse <file>"
		}
		sub := strings.ToLower(strings.TrimSpace(args[:idx]))
		patchRel := strings.TrimSpace(args[idx+1:])
		if patchRel == "" {
			return "Usage: /patch apply|check|reverse <file>"
		}
		root := m.repoRoot()
		patchAbs, err := ensureRepoPath(root, filepath.ToSlash(patchRel))
		if err != nil {
			return fmt.Sprintf("patch failed: %v", err)
		}
		executor := gitops.NewGitExecutor()
		patchMgr := gitops.NewPatchManager(executor)
		ctx := context.Background()
		switch sub {
		case "apply":
			if err := patchMgr.ApplyPatch(ctx, root, patchAbs, false); err != nil {
				return fmt.Sprintf("git apply failed: %s", formatPatchCmdErr(err))
			}
			return "Patch applied successfully."
		case "check":
			if err := patchMgr.ApplyPatch(ctx, root, patchAbs, true); err != nil {
				return fmt.Sprintf("git apply --check failed: %s", formatPatchCmdErr(err))
			}
			return "Patch can be applied cleanly"
		case "reverse":
			if err := patchMgr.ApplyPatchReverse(ctx, root, patchAbs); err != nil {
				return fmt.Sprintf("git apply -R failed: %s", formatPatchCmdErr(err))
			}
			return "Patch reversed successfully."
		default:
			return "Usage: /patch apply|check|reverse <file>"
		}
	}))
}

func (m *Model) registerGitCommands() {
	m.cmdHandlers["add"] = m.requireWritable(func(args string) string {
		root := m.repoRoot()
		path := strings.TrimSpace(args)
		if path == "" {
			path = "."
		}
		if _, err := gitRun(root, "add", path); err != nil {
			return fmt.Sprintf("git add 失败: %v", err)
		}
		return fmt.Sprintf("已暂存: %s", path)
	})

	m.cmdHandlers["reset"] = m.requireWritable(func(args string) string {
		root := m.repoRoot()
		path := strings.TrimSpace(args)
		var err error
		if path != "" {
			_, err = gitRun(root, "reset", "--", path)
		} else {
			_, err = gitRun(root, "reset")
		}
		if err != nil {
			return fmt.Sprintf("git reset 失败: %v", err)
		}
		return "已取消暂存"
	})

	m.cmdHandlers["restore"] = m.requireWritable(func(args string) string {
		path := strings.TrimSpace(args)
		if path == "" {
			return "用法: /restore <path>"
		}
		root := m.repoRoot()
		if _, err := gitRun(root, "restore", path); err != nil {
			return fmt.Sprintf("git restore 失败: %v", err)
		}
		return fmt.Sprintf("已恢复: %s", path)
	})

	m.cmdHandlers["status"] = m.requireRepo(func(_ string) string {
		root := m.repoRoot()
		out, err := gitRun(root, "status", "--short", "--branch")
		if err != nil {
			return fmt.Sprintf("git status 失败: %v", err)
		}
		if out == "" {
			return "工作区干净"
		}
		return out
	})

	m.cmdHandlers["commit"] = m.requireWritable(func(args string) string {
		msg := strings.TrimSpace(args)
		if msg == "" {
			return "用法: /commit <message>"
		}
		root := m.repoRoot()
		out, err := gitRun(root, "commit", "-m", msg)
		if err != nil {
			return fmt.Sprintf("git commit 失败: %v", err)
		}
		return out
	})

	m.cmdHandlers["amend"] = m.requireWritable(func(_ string) string {
		root := m.repoRoot()
		out, err := gitRun(root, "commit", "--amend", "--no-edit")
		if err != nil {
			return fmt.Sprintf("git amend 失败: %v", err)
		}
		return out
	})

	m.cmdHandlers["branch"] = m.requireRepo(func(args string) string {
		root := m.repoRoot()
		args = strings.TrimSpace(args)

		if args == "" {
			out, err := gitRun(root, "branch", "-a")
			if err != nil {
				return fmt.Sprintf("git branch 失败: %v", err)
			}
			return out
		}

		if strings.HasPrefix(args, "-d ") {
			name := strings.TrimSpace(strings.TrimPrefix(args, "-d "))
			if _, err := gitRun(root, "branch", "-d", name); err != nil {
				return fmt.Sprintf("删除分支失败: %v", err)
			}
			return fmt.Sprintf("已删除分支: %s", name)
		}

		if _, err := gitRun(root, "branch", args); err != nil {
			return fmt.Sprintf("创建分支失败: %v", err)
		}
		return fmt.Sprintf("已创建分支: %s", args)
	})

	m.cmdHandlers["checkout"] = m.requireWritable(func(args string) string {
		name := strings.TrimSpace(args)
		if name == "" {
			return "用法: /checkout <branch-name>"
		}
		root := m.repoRoot()
		if _, err := gitRun(root, "checkout", name); err != nil {
			return fmt.Sprintf("checkout 失败: %v", err)
		}
		return fmt.Sprintf("已切换到分支: %s", name)
	})

	m.cmdHandlers["merge"] = m.requireWritable(func(args string) string {
		name := strings.TrimSpace(args)
		if name == "" {
			return "用法: /merge <branch-name>"
		}
		root := m.repoRoot()
		out, err := gitRun(root, "merge", name)
		if err != nil {
			return fmt.Sprintf("merge 失败: %v", err)
		}
		return out
	})

	m.cmdHandlers["rebase"] = m.requireWritable(func(args string) string {
		name := strings.TrimSpace(args)
		if name == "" {
			return "用法: /rebase <branch-name>"
		}
		root := m.repoRoot()
		out, err := gitRun(root, "rebase", name)
		if err != nil {
			return fmt.Sprintf("rebase 失败: %v", err)
		}
		return out
	})

	m.cmdHandlers["fetch"] = m.requireRepo(func(args string) string {
		root := m.repoRoot()
		remote := strings.TrimSpace(args)
		var (
			out string
			err error
		)
		if remote != "" {
			out, err = gitRun(root, "fetch", remote)
		} else {
			out, err = gitRun(root, "fetch")
		}
		if err != nil {
			return fmt.Sprintf("fetch 失败: %v", err)
		}
		if out == "" {
			return "fetch 完成"
		}
		return out
	})

	m.cmdHandlers["pull"] = m.requireWritable(func(args string) string {
		root := m.repoRoot()
		cmdArgs := []string{"pull"}
		cmdArgs = append(cmdArgs, strings.Fields(args)...)
		out, err := gitRun(root, cmdArgs...)
		if err != nil {
			return fmt.Sprintf("pull 失败: %v", err)
		}
		return out
	})

	m.cmdHandlers["push"] = m.requireWritable(func(args string) string {
		root := m.repoRoot()
		cmdArgs := []string{"push"}
		cmdArgs = append(cmdArgs, strings.Fields(args)...)
		out, err := gitRun(root, cmdArgs...)
		if err != nil {
			return fmt.Sprintf("push 失败: %v", err)
		}
		if out == "" {
			return "push 完成"
		}
		return out
	})

	m.cmdHandlers["remote"] = m.requireRepo(func(_ string) string {
		root := m.repoRoot()
		out, err := gitRun(root, "remote", "-v")
		if err != nil {
			return fmt.Sprintf("remote 失败: %v", err)
		}
		return out
	})

	m.cmdHandlers["stash"] = m.requireWritable(m.requireLocalPath(func(args string) string {
		parts := strings.Fields(strings.TrimSpace(args))
		if len(parts) == 0 {
			return "用法: /stash list|push|pop|apply|drop [args]"
		}
		sub := parts[0]
		rest := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(args), sub))
		root := m.repoRoot()
		var cmdArgs []string
		switch sub {
		case "list":
			cmdArgs = []string{"stash", "list"}
		case "push":
			cmdArgs = []string{"stash", "push"}
			if rest != "" {
				cmdArgs = append(cmdArgs, strings.Fields(rest)...)
			}
		case "pop":
			cmdArgs = []string{"stash", "pop"}
			if rest != "" {
				cmdArgs = append(cmdArgs, strings.Fields(rest)...)
			}
		case "apply":
			cmdArgs = []string{"stash", "apply"}
			if rest != "" {
				cmdArgs = append(cmdArgs, strings.Fields(rest)...)
			}
		case "drop":
			cmdArgs = []string{"stash", "drop"}
			if rest != "" {
				cmdArgs = append(cmdArgs, strings.Fields(rest)...)
			}
		default:
			return fmt.Sprintf("未知 stash 子命令 %q。用法: /stash list|push|pop|apply|drop [args]", sub)
		}
		out, err := gitRun(root, cmdArgs...)
		if err != nil {
			return fmt.Sprintf("stash 失败: %v", err)
		}
		if out == "" {
			return "stash 完成"
		}
		return out
	}))

	m.cmdHandlers["log"] = m.requireRepo(func(args string) string {
		root := m.repoRoot()
		cmdArgs := []string{"log", "--oneline", "-20"}
		if strings.TrimSpace(args) != "" {
			cmdArgs = append([]string{"log"}, strings.Fields(args)...)
		}
		out, err := gitRun(root, cmdArgs...)
		if err != nil {
			return fmt.Sprintf("log 失败: %v", err)
		}
		return out
	})

	m.cmdHandlers["blame"] = m.requireRepo(func(args string) string {
		path := strings.TrimSpace(args)
		if path == "" {
			return "用法: /blame <file-path>"
		}
		root := m.repoRoot()
		out, err := gitRun(root, "blame", "--date=short", path)
		if err != nil {
			return fmt.Sprintf("blame 失败: %v", err)
		}
		return out
	})

	m.cmdHandlers["tag"] = m.requireLocalPath(func(args string) string {
		root := m.repoRoot()
		parts := strings.Fields(strings.TrimSpace(args))
		if len(parts) == 0 {
			return "用法: /tag list|create <name>|delete <name>"
		}
		switch parts[0] {
		case "list":
			out, err := gitRun(root, "tag", "-l")
			if err != nil {
				return fmt.Sprintf("列出标签失败: %v", err)
			}
			if out == "" {
				return "无标签"
			}
			return out
		case "create":
			if len(parts) < 2 {
				return "用法: /tag create <name>"
			}
			if m.activeRepo != nil && m.activeRepo.IsReadOnly {
				return "当前仓库处于只读模式。请先切换到本地克隆后再执行写操作。"
			}
			name := parts[1]
			if _, err := gitRun(root, "tag", name); err != nil {
				return fmt.Sprintf("创建标签失败: %v", err)
			}
			return fmt.Sprintf("已创建标签: %s", name)
		case "delete":
			if len(parts) < 2 {
				return "用法: /tag delete <name>"
			}
			if m.activeRepo != nil && m.activeRepo.IsReadOnly {
				return "当前仓库处于只读模式。请先切换到本地克隆后再执行写操作。"
			}
			if _, err := gitRun(root, "tag", "-d", parts[1]); err != nil {
				return fmt.Sprintf("删除标签失败: %v", err)
			}
			return fmt.Sprintf("已删除标签: %s", parts[1])
		default:
			return "用法: /tag list|create <name>|delete <name>"
		}
	})

	m.cmdHandlers["worktree"] = m.requireLocalPath(func(args string) string {
		root := m.repoRoot()
		parts := strings.Fields(strings.TrimSpace(args))
		if len(parts) == 0 {
			return "用法: /worktree list|add <path> [branch]|remove <path>"
		}
		switch parts[0] {
		case "list":
			out, err := gitRun(root, "worktree", "list")
			if err != nil {
				return fmt.Sprintf("worktree 失败: %v", err)
			}
			return out
		case "add":
			if len(parts) < 2 {
				return "用法: /worktree add <path> [branch]"
			}
			if m.activeRepo != nil && m.activeRepo.IsReadOnly {
				return "当前仓库处于只读模式。请先切换到本地克隆后再执行写操作。"
			}
			cmdArgs := append([]string{"worktree", "add"}, parts[1:]...)
			out, err := gitRun(root, cmdArgs...)
			if err != nil {
				return fmt.Sprintf("worktree add 失败: %v", err)
			}
			if out == "" {
				return "worktree add 完成"
			}
			return out
		case "remove":
			if len(parts) < 2 {
				return "用法: /worktree remove <path>"
			}
			if m.activeRepo != nil && m.activeRepo.IsReadOnly {
				return "当前仓库处于只读模式。请先切换到本地克隆后再执行写操作。"
			}
			cmdArgs := append([]string{"worktree", "remove"}, parts[1:]...)
			out, err := gitRun(root, cmdArgs...)
			if err != nil {
				return fmt.Sprintf("worktree remove 失败: %v", err)
			}
			if out == "" {
				return "worktree remove 完成"
			}
			return out
		default:
			return "用法: /worktree list|add <path> [branch]|remove <path>"
		}
	})

	m.cmdHandlers["cherry-pick"] = m.requireWritable(m.requireLocalPath(func(args string) string {
		commit := strings.TrimSpace(args)
		if commit == "" {
			return "用法: /cherry-pick <commit>"
		}
		root := m.repoRoot()
		out, err := gitRun(root, "cherry-pick", commit)
		if err != nil {
			return fmt.Sprintf("cherry-pick 失败: %v", err)
		}
		if out == "" {
			return "cherry-pick 完成"
		}
		return out
	}))

	m.cmdHandlers["reflog"] = m.requireLocalPath(func(_ string) string {
		root := m.repoRoot()
		m.switchView(views.ViewReflog)
		m.reflogView.SetRepoPath(root)
		m.queuePostCommand(views.LoadReflogCmd(root))
		return "已切换到 Reflog 视图并加载 reflog。"
	})

	m.cmdHandlers["bisect"] = m.requireWritable(m.requireLocalPath(func(args string) string {
		parts := strings.Fields(strings.TrimSpace(args))
		if len(parts) == 0 {
			return "用法: /bisect start|good|bad|skip|reset [args]"
		}
		sub := parts[0]
		rest := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(args), sub))
		root := m.repoRoot()
		switch sub {
		case "start", "good", "bad", "skip", "reset":
		default:
			return fmt.Sprintf("未知 bisect 子命令 %q。用法: /bisect start|good|bad|skip|reset [args]", sub)
		}
		cmdArgs := []string{"bisect", sub}
		if rest != "" {
			cmdArgs = append(cmdArgs, strings.Fields(rest)...)
		}
		out, err := gitRun(root, cmdArgs...)
		if err != nil {
			return fmt.Sprintf("bisect 失败: %v", err)
		}
		if out == "" {
			return "bisect 完成"
		}
		return out
	}))

	m.cmdHandlers["submodule"] = m.requireWritable(m.requireLocalPath(func(args string) string {
		parts := strings.Fields(strings.TrimSpace(args))
		if len(parts) == 0 {
			return "用法: /submodule init|update|sync [args]"
		}
		sub := parts[0]
		rest := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(args), sub))
		switch sub {
		case "init", "update", "sync":
		default:
			return fmt.Sprintf("未知 submodule 子命令 %q。用法: /submodule init|update|sync [args]", sub)
		}
		root := m.repoRoot()
		cmdArgs := []string{"submodule", sub}
		if rest != "" {
			cmdArgs = append(cmdArgs, strings.Fields(rest)...)
		}
		out, err := gitRun(root, cmdArgs...)
		if err != nil {
			return fmt.Sprintf("submodule 失败: %v", err)
		}
		if out == "" {
			return "submodule 完成"
		}
		return out
	}))

	m.cmdHandlers["gc"] = m.requireWritable(func(_ string) string {
		root := m.repoRoot()
		out, err := gitRun(root, "gc")
		if err != nil {
			return fmt.Sprintf("gc 失败: %v", err)
		}
		if out == "" {
			return "gc 完成"
		}
		return out
	})

	m.cmdHandlers["clean"] = m.requireWritable(func(args string) string {
		if !strings.Contains(args, "--confirm") {
			return "清理已拦截。使用 /clean --confirm 确认删除未跟踪文件。"
		}
		root := m.repoRoot()
		out, err := gitRun(root, "clean", "-fd")
		if err != nil {
			return fmt.Sprintf("clean 失败: %v", err)
		}
		return out
	})
}

func (m *Model) registerGitHubCommands() {
	m.cmdHandlers["pr"] = m.requireRepo(func(args string) string {
		parts := strings.Fields(args)
		if len(parts) == 0 {
			return "用法: /pr <create|merge|close|comment|review> [args]"
		}
		sub := parts[0]
		subArgs := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(args), sub))

		switch sub {
		case "create":
			return m.handlePRCreate(subArgs)
		case "merge":
			return m.handlePRMerge(subArgs)
		case "close":
			return m.handlePRClose(subArgs)
		case "comment":
			return m.handlePRComment(subArgs)
		case "review":
			return m.handlePRReview(subArgs)
		default:
			return fmt.Sprintf("未知 PR 子命令: %s\n可用: create, merge, close, comment, review", sub)
		}
	})

	m.cmdHandlers["issue"] = m.requireRepo(func(args string) string {
		parts := strings.Fields(args)
		if len(parts) == 0 {
			return "用法: /issue <create|close|reopen|comment|label|assign> [args]"
		}
		sub := parts[0]
		subArgs := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(args), sub))

		switch sub {
		case "create":
			return m.handleIssueCreate(subArgs)
		case "close":
			return m.handleIssueClose(subArgs)
		case "reopen":
			return m.handleIssueReopen(subArgs)
		case "comment":
			return m.handleIssueComment(subArgs)
		case "label":
			return m.handleIssueLabel(subArgs)
		case "assign":
			return m.handleIssueAssign(subArgs)
		default:
			return fmt.Sprintf("未知 Issue 子命令: %s\n可用: create, close, reopen, comment, label, assign", sub)
		}
	})

	m.cmdHandlers["actions"] = m.requireRepo(func(args string) string {
		parts := strings.Fields(args)
		if len(parts) == 0 {
			return "用法: /actions run <workflow-id> [ref]\n查看运行状态请切换到 Explorer 的 Workflows 区域。"
		}
		switch parts[0] {
		case "run":
			if m.ghClient == nil {
				return "GitHub 身份未配置或初始化失败。请先在 Settings 中完成 GitHub App/PAT 配置。"
			}
			if len(parts) < 2 {
				return "用法: /actions run <workflow-id> [ref]"
			}
			workflowID, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return "workflow-id 必须是数字。"
			}
			owner, repo, usageErr := m.requireGitHubRepo()
			if usageErr != "" {
				return usageErr
			}
			ref := ""
			if len(parts) > 2 {
				ref = parts[2]
			}
			if ref == "" {
				if root := m.repoRoot(); root != "" {
					ref, _ = currentBranch(root)
				}
			}
			if ref == "" && m.activeRepo != nil {
				ref = strings.TrimSpace(m.activeRepo.DefaultBranch)
			}
			if ref == "" {
				ref = "main"
			}
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			if err := m.ghClient.TriggerWorkflow(ctx, owner, repo, workflowID, ref); err != nil {
				return fmt.Sprintf("触发 workflow 失败: %v", err)
			}
			return fmt.Sprintf("已触发 workflow %d on %s/%s @ %s", workflowID, owner, repo, ref)
		default:
			return fmt.Sprintf("未知 actions 子命令: %s\n可用: run", parts[0])
		}
	})

	m.cmdHandlers["release"] = m.requireRepo(func(args string) string {
		parts := strings.Fields(args)
		if len(parts) == 0 || parts[0] != "create" {
			return "用法: /release create <tag> <name> [-- <body>]"
		}
		return m.handleReleaseCreate(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(args), "create")))
	})

	m.cmdHandlers["deploy"] = m.requireRepo(func(_ string) string {
		return "Deployment 目前提供实时观测与健康汇总。查看状态请切换到 Dashboard > Health。"
	})
}

func (m *Model) handlePRCreate(args string) string {
	owner, repo, usageErr := m.requireGitHubRepo()
	if usageErr != "" {
		return usageErr
	}

	headSpec, body := splitCommandBody(args)
	parts := strings.Fields(headSpec)
	if len(parts) < 2 {
		return "用法: /pr create <base-branch> <title> [-- <body>]\n或: /pr create <head>:<base> <title> [-- <body>]"
	}

	branchSpec := parts[0]
	title := strings.TrimSpace(strings.TrimPrefix(headSpec, branchSpec))
	if title == "" {
		return "PR 标题不能为空。"
	}

	head := ""
	base := ""
	if strings.Contains(branchSpec, ":") {
		segments := strings.SplitN(branchSpec, ":", 2)
		head = strings.TrimSpace(segments[0])
		base = strings.TrimSpace(segments[1])
	} else {
		base = branchSpec
		root := m.repoRoot()
		if root == "" {
			return "当前没有本地仓库上下文，无法推断 head 分支。请使用 <head>:<base> 语法。"
		}
		var err error
		head, err = currentBranch(root)
		if err != nil {
			return fmt.Sprintf("无法确定当前分支: %v", err)
		}
	}
	if head == "" || base == "" {
		return "PR 分支参数无效。"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pr, err := m.ghClient.CreatePullRequest(ctx, owner, repo, title, body, head, base)
	if err != nil {
		return fmt.Sprintf("创建 PR 失败: %v", err)
	}
	return fmt.Sprintf("已创建 PR #%d: %s%s", pr.GetNumber(), pr.GetTitle(), previewURL(pr.GetHTMLURL()))
}

func (m *Model) handlePRMerge(args string) string {
	owner, repo, usageErr := m.requireGitHubRepo()
	if usageErr != "" {
		return usageErr
	}

	head, commitMsg := splitCommandBody(args)
	parts := strings.Fields(head)
	if len(parts) == 0 {
		return "用法: /pr merge <number> [merge|squash|rebase] [-- <commit-message>]"
	}
	number, err := strconv.Atoi(parts[0])
	if err != nil {
		return "用法: /pr merge <number> [merge|squash|rebase] [-- <commit-message>]"
	}
	method := "merge"
	if len(parts) > 1 {
		method = strings.TrimSpace(parts[1])
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	result, err := m.ghClient.MergePullRequest(ctx, owner, repo, number, commitMsg, method)
	if err != nil {
		return fmt.Sprintf("合并 PR 失败: %v", err)
	}
	if !result.GetMerged() {
		return fmt.Sprintf("PR #%d 未完成合并: %s", number, result.GetMessage())
	}
	return fmt.Sprintf("已合并 PR #%d (%s)", number, method)
}

func (m *Model) handlePRClose(args string) string {
	owner, repo, usageErr := m.requireGitHubRepo()
	if usageErr != "" {
		return usageErr
	}
	number, err := strconv.Atoi(strings.TrimSpace(args))
	if err != nil {
		return "用法: /pr close <number>"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := m.ghClient.CloseIssue(ctx, owner, repo, number); err != nil {
		return fmt.Sprintf("关闭 PR 失败: %v", err)
	}
	return fmt.Sprintf("已关闭 PR #%d", number)
}

func (m *Model) handlePRComment(args string) string {
	owner, repo, usageErr := m.requireGitHubRepo()
	if usageErr != "" {
		return usageErr
	}
	number, body, parseErr := parseNumberAndBody(args, "用法: /pr comment <number> -- <text>")
	if parseErr != "" || strings.TrimSpace(body) == "" {
		return "用法: /pr comment <number> -- <text>"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	comment, err := m.ghClient.CreateComment(ctx, owner, repo, number, body)
	if err != nil {
		return fmt.Sprintf("评论 PR 失败: %v", err)
	}
	return fmt.Sprintf("已评论 PR #%d%s", number, previewURL(comment.GetHTMLURL()))
}

func (m *Model) handlePRReview(args string) string {
	owner, repo, usageErr := m.requireGitHubRepo()
	if usageErr != "" {
		return usageErr
	}
	head, body := splitCommandBody(args)
	parts := strings.Fields(head)
	if len(parts) < 2 {
		return "用法: /pr review <number> <approve|request-changes|comment> [-- <body>]"
	}
	number, err := strconv.Atoi(parts[0])
	if err != nil {
		return "用法: /pr review <number> <approve|request-changes|comment> [-- <body>]"
	}

	action := strings.ToLower(strings.TrimSpace(parts[1]))
	event := ""
	switch action {
	case "approve":
		event = "APPROVE"
	case "request-changes":
		event = "REQUEST_CHANGES"
	case "comment":
		event = "COMMENT"
	default:
		return "review 动作仅支持 approve、request-changes、comment。"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	review, err := m.ghClient.SubmitPRReview(ctx, owner, repo, number, event, body)
	if err != nil {
		return fmt.Sprintf("提交 PR review 失败: %v", err)
	}
	return fmt.Sprintf("已提交 PR #%d review: %s%s", number, action, previewURL(review.GetHTMLURL()))
}

func (m *Model) handleIssueCreate(args string) string {
	owner, repo, usageErr := m.requireGitHubRepo()
	if usageErr != "" {
		return usageErr
	}

	title, body := splitCommandBody(args)
	title = strings.TrimSpace(title)
	if title == "" {
		return "用法: /issue create <title> [-- <body>]"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	issue, err := m.ghClient.CreateIssue(ctx, owner, repo, title, body, nil, nil)
	if err != nil {
		return fmt.Sprintf("创建 Issue 失败: %v", err)
	}
	return fmt.Sprintf("已创建 Issue #%d: %s%s", issue.GetNumber(), issue.GetTitle(), previewURL(issue.GetHTMLURL()))
}

func (m *Model) handleIssueClose(args string) string {
	owner, repo, usageErr := m.requireGitHubRepo()
	if usageErr != "" {
		return usageErr
	}
	number, err := strconv.Atoi(strings.TrimSpace(args))
	if err != nil {
		return "用法: /issue close <number>"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := m.ghClient.CloseIssue(ctx, owner, repo, number); err != nil {
		return fmt.Sprintf("关闭 Issue 失败: %v", err)
	}
	return fmt.Sprintf("已关闭 Issue #%d", number)
}

func (m *Model) handleIssueReopen(args string) string {
	owner, repo, usageErr := m.requireGitHubRepo()
	if usageErr != "" {
		return usageErr
	}
	number, err := strconv.Atoi(strings.TrimSpace(args))
	if err != nil {
		return "用法: /issue reopen <number>"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := m.ghClient.ReopenIssue(ctx, owner, repo, number); err != nil {
		return fmt.Sprintf("重新打开 Issue 失败: %v", err)
	}
	return fmt.Sprintf("已重新打开 Issue #%d", number)
}

func (m *Model) handleIssueComment(args string) string {
	owner, repo, usageErr := m.requireGitHubRepo()
	if usageErr != "" {
		return usageErr
	}
	number, body, parseErr := parseNumberAndBody(args, "用法: /issue comment <number> -- <text>")
	if parseErr != "" || strings.TrimSpace(body) == "" {
		return "用法: /issue comment <number> -- <text>"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	comment, err := m.ghClient.CreateComment(ctx, owner, repo, number, body)
	if err != nil {
		return fmt.Sprintf("评论 Issue 失败: %v", err)
	}
	return fmt.Sprintf("已评论 Issue #%d%s", number, previewURL(comment.GetHTMLURL()))
}

func (m *Model) handleIssueLabel(args string) string {
	owner, repo, usageErr := m.requireGitHubRepo()
	if usageErr != "" {
		return usageErr
	}
	number, labelSpec, parseErr := parseNumberAndBody(args, "用法: /issue label <number> <label1,label2>")
	if parseErr != "" || strings.TrimSpace(labelSpec) == "" {
		return "用法: /issue label <number> <label1,label2>"
	}
	labels := normalizeListArg(labelSpec)
	if len(labels) == 0 {
		return "至少需要一个 label。"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := m.ghClient.AddLabels(ctx, owner, repo, number, labels); err != nil {
		return fmt.Sprintf("添加 label 失败: %v", err)
	}
	return fmt.Sprintf("已为 Issue #%d 添加标签: %s", number, strings.Join(labels, ", "))
}

func (m *Model) handleIssueAssign(args string) string {
	owner, repo, usageErr := m.requireGitHubRepo()
	if usageErr != "" {
		return usageErr
	}
	number, assigneeSpec, parseErr := parseNumberAndBody(args, "用法: /issue assign <number> <user1,user2>")
	if parseErr != "" || strings.TrimSpace(assigneeSpec) == "" {
		return "用法: /issue assign <number> <user1,user2>"
	}
	assignees := normalizeListArg(assigneeSpec)
	if len(assignees) == 0 {
		return "至少需要一个 assignee。"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := m.ghClient.SetAssignees(ctx, owner, repo, number, assignees); err != nil {
		return fmt.Sprintf("设置 assignee 失败: %v", err)
	}
	return fmt.Sprintf("已为 Issue #%d 设置 assignee: %s", number, strings.Join(assignees, ", "))
}

func (m *Model) handleReleaseCreate(args string) string {
	owner, repo, usageErr := m.requireGitHubRepo()
	if usageErr != "" {
		return usageErr
	}

	head, body := splitCommandBody(args)
	parts := strings.Fields(head)
	if len(parts) < 2 {
		return "用法: /release create <tag> <name> [-- <body>]"
	}
	tag := parts[0]
	name := strings.TrimSpace(strings.TrimPrefix(head, tag))
	if name == "" {
		return "Release 名称不能为空。"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	release, err := m.ghClient.CreateRelease(ctx, owner, repo, tag, name, body, false, false)
	if err != nil {
		return fmt.Sprintf("创建 Release 失败: %v", err)
	}
	return fmt.Sprintf("已创建 Release %s%s", release.TagName, previewURL(release.HTMLURL))
}

func (m *Model) registerHelpUpdate() {
	m.cmdHandlers["help"] = func(_ string) string {
		var b strings.Builder
		b.WriteString("可用命令:\n\n")
		b.WriteString("  === 导航 ===\n")
		b.WriteString("  /help            显示帮助\n")
		b.WriteString("  /dashboard       切换到 Dashboard\n")
		b.WriteString("  /chat            切换到 Chat\n")
		b.WriteString("  /explorer        切换到 Explorer\n")
		b.WriteString("  /workspace       切换到 Workspace\n")
		b.WriteString("  /settings        打开 Settings\n")
		b.WriteString("  /clear           清空聊天记录\n")
		b.WriteString("  /theme [name]    切换主题\n")
		b.WriteString("  /quit            退出\n\n")

		if m.activeRepo != nil {
			b.WriteString("  === 文件系统 ===\n")
			b.WriteString("  /new <path>                    创建文件或目录\n")
			b.WriteString("  /edit <path> -- <content>      覆盖写入文件\n")
			b.WriteString("  /edit <path> ++ <content>      追加写入文件\n")
			b.WriteString("  /rm --confirm <path>           删除文件或目录\n")
			b.WriteString("  /diff [path]                   查看 diff\n")
			b.WriteString("  /search <pattern>              搜索文件内容\n")
			b.WriteString("  /find <name>                   查找文件\n")
			b.WriteString("  /chmod <mode> <file>           设置权限 (如 755、644)\n")
			b.WriteString("  /symlink <target> <linkname>   创建符号链接\n")
			b.WriteString("  /archive <tar.gz|zip> [out]    git archive HEAD\n")
			b.WriteString("  /patch apply|check|reverse <f> git apply\n\n")

			b.WriteString("  === Git ===\n")
			b.WriteString("  /status                        查看工作区状态\n")
			b.WriteString("  /add [path|.]                  暂存文件\n")
			b.WriteString("  /reset [path]                  取消暂存\n")
			b.WriteString("  /restore <path>                恢复文件\n")
			b.WriteString("  /commit <message>              提交更改\n")
			b.WriteString("  /amend                         修改上次提交\n")
			b.WriteString("  /branch [name|-d name]         列出/创建/删除分支\n")
			b.WriteString("  /checkout <name>               切换分支\n")
			b.WriteString("  /merge <name>                  合并分支\n")
			b.WriteString("  /rebase <name>                 rebase 到指定分支\n")
			b.WriteString("  /fetch [remote]                拉取远端更新\n")
			b.WriteString("  /pull [args]                   拉取并合并\n")
			b.WriteString("  /push [args]                   推送到远端\n")
			b.WriteString("  /remote                        查看 remote\n")
			b.WriteString("  /stash list|push|pop|apply|drop [args]  管理 stash\n")
			b.WriteString("  /log [opts]                    查看提交日志\n")
			b.WriteString("  /blame <path>                  行级追溯\n")
			b.WriteString("  /tag list|create <name>|delete <name>   列出/创建/删除标签\n")
			b.WriteString("  /worktree list|add <path> [branch]|remove <path>  管理 worktree\n")
			b.WriteString("  /cherry-pick <commit>          cherry-pick 提交\n")
			b.WriteString("  /reflog                        打开 Reflog 视图\n")
			b.WriteString("  /bisect start|good|bad|skip|reset [args]  bisect 流程\n")
			b.WriteString("  /submodule init|update|sync [args]  子模块操作\n")
			b.WriteString("  /gc                            仓库整理\n")
			b.WriteString("  /clean --confirm               删除未跟踪文件\n\n")

			b.WriteString("  === GitHub ===\n")
			b.WriteString("  /pr create <base> <title> [-- <body>]\n")
			b.WriteString("  /pr merge <number> [merge|squash|rebase] [-- <commit-message>]\n")
			b.WriteString("  /pr close <number>\n")
			b.WriteString("  /pr comment <number> -- <text>\n")
			b.WriteString("  /pr review <number> <approve|request-changes|comment> [-- <body>]\n")
			b.WriteString("  /issue create <title> [-- <body>]\n")
			b.WriteString("  /issue close <number>\n")
			b.WriteString("  /issue reopen <number>\n")
			b.WriteString("  /issue comment <number> -- <text>\n")
			b.WriteString("  /issue label <number> <label1,label2>\n")
			b.WriteString("  /issue assign <number> <user1,user2>\n")
			b.WriteString("  /actions run <workflow-id> [ref]\n")
			b.WriteString("  /release create <tag> <name> [-- <body>]\n\n")
		}

		b.WriteString("直接输入自然语言即可与 Gitdex 对话。")
		return b.String()
	}
}

// suppress unused import warnings
var _ = views.StreamChunkMsg{}
