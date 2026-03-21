package gitops

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeRemoteURL(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"https://github.com/owner/repo.git", "github.com/owner/repo"},
		{"https://github.com/owner/repo", "github.com/owner/repo"},
		{"git@github.com:owner/repo.git", "github.com/owner/repo"},
		{"git@github.com:owner/repo", "github.com/owner/repo"},
		{"ssh://git@github.com/owner/repo.git", "github.com/owner/repo"},
		{"http://github.com/owner/repo.git", "github.com/owner/repo"},
		{"git://github.com/owner/repo.git", "github.com/owner/repo"},
		{"https://github.com/Owner/Repo.git", "github.com/owner/repo"},
		{"git@github.com:Owner/Repo.git", "github.com/owner/repo"},
		{"https://www.github.com/owner/repo.git", "github.com/owner/repo"},
		{"https://github.com/owner/repo/", "github.com/owner/repo"},
		{"", ""},
		{"   ", ""},
	}

	for _, tt := range tests {
		got := NormalizeRemoteURL(tt.input)
		if got != tt.want {
			t.Errorf("NormalizeRemoteURL(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestNormalizeRemoteURL_SameRepo(t *testing.T) {
	urls := []string{
		"https://github.com/Joker-of-Gotham/test-git-tool.git",
		"git@github.com:Joker-of-Gotham/test-git-tool.git",
		"ssh://git@github.com/Joker-of-Gotham/test-git-tool.git",
		"https://github.com/joker-of-gotham/test-git-tool",
	}

	first := NormalizeRemoteURL(urls[0])
	for _, u := range urls[1:] {
		got := NormalizeRemoteURL(u)
		if got != first {
			t.Errorf("NormalizeRemoteURL(%q) = %q, want %q (same repo)", u, got, first)
		}
	}
}

func TestLocalIndex_LookupByOwnerName(t *testing.T) {
	idx := NewLocalIndex(NewGitExecutor())
	idx.index["github.com/alice/web-app"] = []WorktreeEntry{
		{Path: "/home/alice/web-app", RemoteName: "origin"},
		{Path: "/tmp/web-app-feature", RemoteName: "origin"},
	}

	paths := idx.LookupByOwnerName("Alice", "web-app")
	if len(paths) != 2 {
		t.Fatalf("expected 2 paths, got %d", len(paths))
	}
	if paths[0] != "/home/alice/web-app" {
		t.Errorf("path[0] = %q, want /home/alice/web-app", paths[0])
	}
}

func TestLocalIndex_Lookup(t *testing.T) {
	idx := NewLocalIndex(NewGitExecutor())
	idx.index["github.com/bob/api"] = []WorktreeEntry{{Path: "/code/api", RemoteName: "origin"}}

	paths := idx.Lookup("git@github.com:Bob/api.git")
	if len(paths) != 1 {
		t.Fatalf("expected 1 path, got %d", len(paths))
	}
	if paths[0] != "/code/api" {
		t.Errorf("path = %q, want /code/api", paths[0])
	}

	none := idx.Lookup("https://github.com/nobody/nothing")
	if len(none) != 0 {
		t.Error("should return nil for non-existent URL")
	}
}

func TestAppendUnique(t *testing.T) {
	result := appendUnique([]string{"/a", "/b"}, "/B", "/c")
	if len(result) != 3 {
		t.Fatalf("expected 3 unique paths, got %d: %v", len(result), result)
	}
}

func TestDiscoverScanRoots_PrefersExplicitWorkspaceRoots(t *testing.T) {
	rootA := t.TempDir()
	rootB := filepath.Join(rootA, "nested")
	if err := os.MkdirAll(rootB, 0o755); err != nil {
		t.Fatalf("MkdirAll(rootB) returned error: %v", err)
	}

	roots := discoverScanRoots([]string{rootA, rootB, rootA})
	if len(roots) != 2 {
		t.Fatalf("discoverScanRoots() len = %d, want 2 (%#v)", len(roots), roots)
	}
	if roots[0] != rootA || roots[1] != rootB {
		t.Fatalf("discoverScanRoots() = %#v, want [%q %q]", roots, rootA, rootB)
	}
}
