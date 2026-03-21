package repocontext

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/your-org/gitdex/internal/app/bootstrap"
	"github.com/your-org/gitdex/internal/platform/config"
)

func TestParseRepoFlag(t *testing.T) {
	tests := []struct {
		input     string
		wantOwner string
		wantRepo  string
	}{
		{"owner/repo", "owner", "repo"},
		{" owner / repo ", "owner", "repo"},
		{"org/my-project", "org", "my-project"},
		{"", "", ""},
		{"noslash", "", ""},
		{"a/b/c", "a", "b/c"},
	}
	for _, tt := range tests {
		owner, repo := ParseRepoFlag(tt.input)
		if owner != tt.wantOwner || repo != tt.wantRepo {
			t.Errorf("ParseRepoFlag(%q) = (%q, %q), want (%q, %q)", tt.input, owner, repo, tt.wantOwner, tt.wantRepo)
		}
	}
}

func TestParseRemoteURL(t *testing.T) {
	tests := []struct {
		input    string
		wantHost string
		wantOwn  string
		wantRepo string
	}{
		{"https://github.com/owner/repo.git", "github.com", "owner", "repo"},
		{"https://github.com/owner/repo", "github.com", "owner", "repo"},
		{"git@github.com:owner/repo.git", "github.com", "owner", "repo"},
		{"", "", "", ""},
	}
	for _, tt := range tests {
		host, owner, repo := ParseRemoteURL(tt.input)
		if host != tt.wantHost || owner != tt.wantOwn || repo != tt.wantRepo {
			t.Errorf("ParseRemoteURL(%q) = (%q, %q, %q), want (%q, %q, %q)",
				tt.input, host, owner, repo, tt.wantHost, tt.wantOwn, tt.wantRepo)
		}
	}
}

func TestClassifyRemoteRole(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"origin", "origin"},
		{"ORIGIN", "origin"},
		{"upstream", "upstream"},
		{"fork", "fork"},
		{"mirror", "mirror"},
		{"myremote", "secondary"},
		{"  origin ", "origin"},
	}
	for _, tt := range tests {
		got := classifyRemoteRole(tt.name)
		if got != tt.want {
			t.Errorf("classifyRemoteRole(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestHostFromTopology(t *testing.T) {
	got := hostFromTopology(RemoteTopology{})
	if got != "" {
		t.Errorf("empty topology host = %q, want empty", got)
	}

	topo := RemoteTopology{
		Canonical: RemoteBinding{URL: "https://github.com/owner/repo.git"},
	}
	got = hostFromTopology(topo)
	if got != "github.com" {
		t.Errorf("topology host = %q, want github.com", got)
	}
}

func TestNormalizePaths(t *testing.T) {
	paths := normalizePaths([]string{"/a/b", "/a/b", "/c/d", "", "  ", "/A/B"})
	if len(paths) != 2 {
		t.Fatalf("normalizePaths dedup expected 2 unique paths, got %d: %v", len(paths), paths)
	}
	for _, p := range paths {
		if strings.Contains(p, "\\") {
			t.Errorf("path %q contains backslash, expected forward slash", p)
		}
	}
}

func TestNormalizePaths_Empty(t *testing.T) {
	paths := normalizePaths(nil)
	if len(paths) != 0 {
		t.Errorf("normalizePaths(nil) = %v, want empty", paths)
	}
	paths = normalizePaths([]string{"", "  "})
	if len(paths) != 0 {
		t.Errorf("normalizePaths with blanks = %v, want empty", paths)
	}
}

func TestAppendUniquePaths(t *testing.T) {
	base := []string{"/a/b"}
	result := appendUniquePaths(base, "/a/b", "/c/d")
	if len(result) != 2 {
		t.Errorf("appendUniquePaths expected 2 unique, got %d: %v", len(result), result)
	}
}

func TestCleanPath(t *testing.T) {
	if cleanPath("") != "" {
		t.Error("cleanPath('') should return empty")
	}
	if cleanPath("   ") != "" {
		t.Error("cleanPath('   ') should return empty")
	}
	got := cleanPath("  /a/b/../c  ")
	want := filepath.Clean("/a/c")
	if got != want {
		t.Errorf("cleanPath = %q, want %q", got, want)
	}
}

func TestFirstNonEmpty(t *testing.T) {
	if firstNonEmpty("", "  ", "hello", "world") != "hello" {
		t.Error("firstNonEmpty should return 'hello'")
	}
	if firstNonEmpty("", "", "") != "" {
		t.Error("firstNonEmpty all empty should return ''")
	}
	if firstNonEmpty("first") != "first" {
		t.Error("firstNonEmpty single should return 'first'")
	}
}

func TestParseRepoInputs_FromSpec(t *testing.T) {
	host, owner, repo := parseRepoInputs(ResolveOptions{RepoSpec: "myorg/myrepo"})
	if owner != "myorg" || repo != "myrepo" {
		t.Errorf("parseRepoInputs from spec = (%q, %q, %q), want ('', 'myorg', 'myrepo')", host, owner, repo)
	}
}

func TestParseRepoInputs_FromFields(t *testing.T) {
	host, owner, repo := parseRepoInputs(ResolveOptions{Owner: "o", Repo: "r"})
	if owner != "o" || repo != "r" {
		t.Errorf("parseRepoInputs from fields = (%q, %q, %q), want ('', 'o', 'r')", host, owner, repo)
	}
}

func TestParseRepoInputs_FromRemoteURL(t *testing.T) {
	host, owner, repo := parseRepoInputs(ResolveOptions{RemoteURL: "https://github.com/octo/cat.git"})
	if host != "github.com" || owner != "octo" || repo != "cat" {
		t.Errorf("parseRepoInputs from URL = (%q, %q, %q), want ('github.com', 'octo', 'cat')", host, owner, repo)
	}
}

func TestParseRepoInputs_SpecOverridesFields(t *testing.T) {
	_, owner, repo := parseRepoInputs(ResolveOptions{
		RepoSpec: "spec-owner/spec-repo",
		Owner:    "field-owner",
		Repo:     "field-repo",
	})
	if owner != "spec-owner" || repo != "spec-repo" {
		t.Errorf("spec should override fields: got (%q, %q)", owner, repo)
	}
}

func TestDefaultCloneDir_WithWorkspaceRoot(t *testing.T) {
	app := bootstrap.App{
		Config: config.Config{
			FileConfig: config.FileConfig{
				Git: config.GitConfig{
					WorkspaceRoots: []string{"/home/user/code"},
				},
			},
		},
	}
	got := DefaultCloneDir(app, "myrepo")
	want := filepath.Join("/home/user/code", "myrepo")
	if got != want {
		t.Errorf("DefaultCloneDir with root = %q, want %q", got, want)
	}
}

func TestDefaultCloneDir_WithoutWorkspaceRoot(t *testing.T) {
	app := bootstrap.App{
		Config: config.Config{
			Paths: config.ConfigPaths{WorkingDir: "/tmp/work"},
		},
	}
	got := DefaultCloneDir(app, "myrepo")
	want := filepath.Join("/tmp/work", "myrepo")
	if got != want {
		t.Errorf("DefaultCloneDir without root = %q, want %q", got, want)
	}
}

func TestDefaultCloneDir_Fallback(t *testing.T) {
	app := bootstrap.App{Config: config.Config{}}
	got := DefaultCloneDir(app, "myrepo")
	want := filepath.Join(".", "myrepo")
	if got != want {
		t.Errorf("DefaultCloneDir fallback = %q, want %q", got, want)
	}
}

func TestAccessModeConstants(t *testing.T) {
	if AccessModeUnknown != "unknown" {
		t.Error("AccessModeUnknown should be 'unknown'")
	}
	if AccessModeRemoteOnly != "remote_only" {
		t.Error("AccessModeRemoteOnly should be 'remote_only'")
	}
	if AccessModeLocalWritable != "local_writable" {
		t.Error("AccessModeLocalWritable should be 'local_writable'")
	}
}

func TestRepoContext_FieldsInitialization(t *testing.T) {
	rc := RepoContext{
		Host:            "github.com",
		Owner:           "test",
		Repo:            "repo",
		CanonicalRemote: "github.com/test/repo",
		LocalPaths:      []string{"/home/user/repo"},
		ActiveLocalPath: "/home/user/repo",
		AccessMode:      AccessModeLocalWritable,
	}
	if rc.Host != "github.com" {
		t.Errorf("Host = %q", rc.Host)
	}
	if len(rc.LocalPaths) != 1 {
		t.Errorf("LocalPaths len = %d", len(rc.LocalPaths))
	}
	if rc.AccessMode != AccessModeLocalWritable {
		t.Errorf("AccessMode = %q", rc.AccessMode)
	}
}

func TestRemoteTopology_CanonicalSelection(t *testing.T) {
	topo := RemoteTopology{
		Remotes: []RemoteBinding{
			{Name: "upstream", URL: "https://github.com/upstream/repo.git", Normalized: "github.com/upstream/repo"},
			{Name: "origin", URL: "https://github.com/fork/repo.git", Normalized: "github.com/fork/repo"},
		},
	}
	for _, b := range topo.Remotes {
		if b.Name == "origin" {
			topo.Canonical = b
			break
		}
	}
	if topo.Canonical.Name != "origin" {
		t.Errorf("canonical should be origin, got %q", topo.Canonical.Name)
	}
}

func TestRemoteBinding_RoleField(t *testing.T) {
	b := RemoteBinding{
		Name:       "origin",
		URL:        "https://github.com/owner/repo",
		Normalized: "github.com/owner/repo",
		Role:       classifyRemoteRole("origin"),
	}
	if b.Role != "origin" {
		t.Errorf("Role = %q, want 'origin'", b.Role)
	}
}
