package gitops

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// WorktreeEntry associates a worktree path with the remote whose URL was used to index it.
type WorktreeEntry struct {
	Path       string
	RemoteName string
}

type LocalIndex struct {
	mu    sync.RWMutex
	index map[string][]WorktreeEntry // normalized remote URL → worktree entries
	exec  *GitExecutor
}

func NewLocalIndex(exec *GitExecutor) *LocalIndex {
	return &LocalIndex{
		exec:  exec,
		index: make(map[string][]WorktreeEntry),
	}
}

func (idx *LocalIndex) Build(ctx context.Context) {
	idx.BuildWithRoots(ctx, nil)
}

func (idx *LocalIndex) BuildWithRoots(ctx context.Context, preferredRoots []string) {
	roots := discoverScanRoots(preferredRoots)

	var gitDirs []string
	var gdMu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 4)

	for _, root := range roots {
		wg.Add(1)
		go func(r string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			found := walkForGitDirs(r)
			gdMu.Lock()
			gitDirs = append(gitDirs, found...)
			gdMu.Unlock()
		}(root)
	}
	wg.Wait()

	dedup := make(map[string]bool, len(gitDirs))
	var unique []string
	for _, d := range gitDirs {
		n := strings.ToLower(filepath.ToSlash(d))
		if !dedup[n] {
			dedup[n] = true
			unique = append(unique, d)
		}
	}

	newIndex := make(map[string][]WorktreeEntry)
	var idxMu sync.Mutex
	var wg2 sync.WaitGroup
	sem2 := make(chan struct{}, 8)

	for _, dir := range unique {
		wg2.Add(1)
		go func(repoDir string) {
			defer wg2.Done()
			sem2 <- struct{}{}
			defer func() { <-sem2 }()

			indexCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			entries := idx.indexRepo(indexCtx, repoDir)
			if len(entries) == 0 {
				return
			}

			idxMu.Lock()
			for url, wts := range entries {
				newIndex[url] = appendUniqueWorktree(newIndex[url], wts...)
			}
			idxMu.Unlock()
		}(dir)
	}
	wg2.Wait()

	idx.mu.Lock()
	idx.index = newIndex
	idx.mu.Unlock()
}

func (idx *LocalIndex) indexRepo(ctx context.Context, repoDir string) map[string][]WorktreeEntry {
	result := make(map[string][]WorktreeEntry)

	bareRes, err := idx.exec.Run(ctx, repoDir, "rev-parse", "--is-bare-repository")
	if err != nil {
		return nil
	}
	if strings.TrimSpace(bareRes.Stdout) == "true" {
		return nil
	}

	toplevelRes, err := idx.exec.Run(ctx, repoDir, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil
	}
	mainPath := filepath.ToSlash(strings.TrimSpace(toplevelRes.Stdout))
	if mainPath == "" {
		return nil
	}

	remotes := idx.getRemoteURLs(ctx, repoDir)
	if len(remotes) == 0 {
		return nil
	}

	wtRes, err := idx.exec.Run(ctx, repoDir, "worktree", "list", "--porcelain")
	if err != nil {
		for remoteName, rawURL := range remotes {
			normalized := NormalizeRemoteURL(rawURL)
			if normalized == "" {
				continue
			}
			result[normalized] = appendUniqueWorktree(result[normalized], WorktreeEntry{Path: mainPath, RemoteName: remoteName})
		}
		return result
	}

	var paths []string
	isBare := false
	currentPath := ""

	for _, line := range strings.Split(wtRes.Stdout, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "worktree ") {
			if currentPath != "" && !isBare {
				paths = append(paths, filepath.ToSlash(currentPath))
			}
			currentPath = strings.TrimPrefix(line, "worktree ")
			isBare = false
		} else if line == "bare" {
			isBare = true
		} else if line == "" && currentPath != "" {
			if !isBare {
				paths = append(paths, filepath.ToSlash(currentPath))
			}
			currentPath = ""
			isBare = false
		}
	}
	if currentPath != "" && !isBare {
		paths = append(paths, filepath.ToSlash(currentPath))
	}

	if len(paths) == 0 {
		paths = []string{mainPath}
	}

	for remoteName, rawURL := range remotes {
		normalized := NormalizeRemoteURL(rawURL)
		if normalized == "" {
			continue
		}
		for _, p := range paths {
			result[normalized] = appendUniqueWorktree(result[normalized], WorktreeEntry{Path: p, RemoteName: remoteName})
		}
	}
	return result
}

func (idx *LocalIndex) getRemoteURLs(ctx context.Context, repoDir string) map[string]string {
	urls := make(map[string]string)
	remotesRes, err := idx.exec.Run(ctx, repoDir, "remote")
	if err != nil {
		return urls
	}
	for _, name := range strings.Split(remotesRes.Stdout, "\n") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if res, err := idx.exec.Run(ctx, repoDir, "remote", "get-url", name); err == nil {
			if u := strings.TrimSpace(res.Stdout); u != "" {
				urls[name] = u
			}
		}
	}
	return urls
}

func (idx *LocalIndex) Lookup(remoteURL string) []string {
	return uniquePathsFromEntries(idx.LookupEntries(remoteURL))
}

// LookupEntries returns indexed worktrees for an exact normalized remote URL match.
func (idx *LocalIndex) LookupEntries(remoteURL string) []WorktreeEntry {
	key := NormalizeRemoteURL(remoteURL)
	if key == "" {
		return nil
	}
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	out := idx.index[key]
	if len(out) == 0 {
		return nil
	}
	cp := make([]WorktreeEntry, len(out))
	copy(cp, out)
	return cp
}

func (idx *LocalIndex) LookupByOwnerName(owner, name string) []string {
	return idx.LookupByRepo("github.com", owner, name)
}

func (idx *LocalIndex) LookupByRepo(host, owner, name string) []string {
	host = strings.TrimSpace(strings.ToLower(host))
	if host == "" {
		host = "github.com"
	}
	key := host + "/" + strings.ToLower(owner) + "/" + strings.ToLower(name)
	idx.mu.RLock()
	entries := idx.index[key]
	idx.mu.RUnlock()
	return uniquePathsFromEntries(entries)
}

// LookupByAnyRemote finds worktrees indexed under any remote URL whose normalized
// form ends with "/<owner>/<name>" (case-insensitive), scanning the full index
// rather than a single host-specific key.
func (idx *LocalIndex) LookupByAnyRemote(owner, name string) []WorktreeEntry {
	owner = strings.ToLower(strings.TrimSpace(owner))
	name = strings.ToLower(strings.TrimSpace(name))
	if owner == "" || name == "" {
		return nil
	}
	suffix := "/" + owner + "/" + name
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	var out []WorktreeEntry
	seen := make(map[string]bool)
	for key, entries := range idx.index {
		if !strings.HasSuffix(key, suffix) {
			continue
		}
		for _, e := range entries {
			dk := strings.ToLower(filepath.ToSlash(e.Path)) + "\x00" + e.RemoteName
			if seen[dk] {
				continue
			}
			seen[dk] = true
			out = append(out, e)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func NormalizeRemoteURL(rawURL string) string {
	u := strings.TrimSpace(rawURL)
	if u == "" {
		return ""
	}

	u = strings.TrimSuffix(u, "/")
	u = strings.TrimSuffix(u, ".git")

	if strings.HasPrefix(u, "git@") {
		u = strings.TrimPrefix(u, "git@")
		u = strings.Replace(u, ":", "/", 1)
	}

	for _, pfx := range []string{"https://", "http://", "git://", "ssh://"} {
		u = strings.TrimPrefix(u, pfx)
	}

	if at := strings.Index(u, "@"); at >= 0 {
		slash := strings.Index(u, "/")
		if slash < 0 || at < slash {
			u = u[at+1:]
		}
	}

	u = strings.TrimPrefix(u, "www.")

	return strings.ToLower(u)
}

var skipDirNames = map[string]bool{
	"node_modules": true, ".git": true, "vendor": true,
	".cache": true, ".npm": true, ".cargo": true, ".rustup": true,
	"__pycache__": true, ".venv": true, "venv": true,
	".tox": true, ".eggs": true, "dist": true, "build": true,
	".gradle": true, ".m2": true, ".ivy2": true,
	"Library": true, "AppData": true, "$Recycle.Bin": true,
	"System Volume Information": true, "Recovery": true,
	"PerfLogs": true,
}

var skipTopLevel = []string{
	"windows", "program files", "program files (x86)",
	"programdata", "msys64",
}

const maxScanDepth = 6

func walkForGitDirs(root string) []string {
	var results []string
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fs.SkipDir
		}
		if !d.IsDir() {
			return nil
		}

		name := d.Name()
		if skipDirNames[name] {
			return fs.SkipDir
		}

		rel, _ := filepath.Rel(root, path)
		depth := len(strings.Split(filepath.ToSlash(rel), "/"))
		if depth > maxScanDepth {
			return fs.SkipDir
		}

		if depth == 1 {
			lower := strings.ToLower(name)
			for _, pfx := range skipTopLevel {
				if lower == pfx {
					return fs.SkipDir
				}
			}
		}

		gitPath := filepath.Join(path, ".git")
		if info, serr := os.Stat(gitPath); serr == nil {
			if info.IsDir() || info.Mode().IsRegular() {
				results = append(results, path)
				return fs.SkipDir
			}
		}

		return nil
	})
	return results
}

func discoverScanRoots(preferredRoots []string) []string {
	var roots []string
	seen := make(map[string]bool)

	add := func(p string) {
		p = filepath.Clean(p)
		lower := strings.ToLower(p)
		if !seen[lower] {
			if _, err := os.Stat(p); err == nil {
				seen[lower] = true
				roots = append(roots, p)
			}
		}
	}

	for _, root := range preferredRoots {
		add(root)
	}

	if len(roots) > 0 {
		return roots
	}

	if home, err := os.UserHomeDir(); err == nil {
		add(home)
	}

	if runtime.GOOS == "windows" {
		for c := 'A'; c <= 'Z'; c++ {
			add(string(c) + ":\\")
		}
	} else {
		add("/")
	}

	return roots
}

func uniquePathsFromEntries(entries []WorktreeEntry) []string {
	if len(entries) == 0 {
		return nil
	}
	var out []string
	seen := make(map[string]bool)
	for _, e := range entries {
		key := strings.ToLower(filepath.ToSlash(e.Path))
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, e.Path)
	}
	return out
}

func appendUniqueWorktree(existing []WorktreeEntry, add ...WorktreeEntry) []WorktreeEntry {
	seen := make(map[string]bool, len(existing)+len(add))
	for _, e := range existing {
		seen[strings.ToLower(filepath.ToSlash(e.Path))+"\x00"+e.RemoteName] = true
	}
	for _, e := range add {
		k := strings.ToLower(filepath.ToSlash(e.Path)) + "\x00" + e.RemoteName
		if seen[k] {
			continue
		}
		seen[k] = true
		existing = append(existing, e)
	}
	return existing
}

func appendUnique(existing []string, add ...string) []string {
	set := make(map[string]bool, len(existing))
	for _, s := range existing {
		set[strings.ToLower(filepath.ToSlash(s))] = true
	}
	for _, s := range add {
		key := strings.ToLower(filepath.ToSlash(s))
		if !set[key] {
			set[key] = true
			existing = append(existing, s)
		}
	}
	return existing
}
