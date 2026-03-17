package compliance

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestNoReferenceProjectInRuntimeSource(t *testing.T) {
	root := repoRoot(t)
	files := collectRuntimeSourceFiles(t, root)
	var hits []string
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		text := string(data)
		if strings.Contains(text, "reference_project/") || strings.Contains(text, "reference_project\\") {
			hits = append(hits, rel(root, f))
		}
	}
	if len(hits) > 0 {
		t.Fatalf("runtime source must not reference reference_project code: %v", hits)
	}
}

func TestNoHostAbsolutePathHardcodingInRuntimeSource(t *testing.T) {
	root := repoRoot(t)
	files := collectRuntimeSourceFiles(t, root)
	var hits []string
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		text := string(data)
		if strings.Contains(text, `C:\Users\`) ||
			strings.Contains(text, `/Users/`) ||
			strings.Contains(text, `/home/`) ||
			strings.Contains(text, `workspaceStorage`) {
			hits = append(hits, rel(root, f))
		}
	}
	if len(hits) > 0 {
		t.Fatalf("runtime source contains host-specific absolute path hardcoding: %v", hits)
	}
}

func TestNoLiteralGitExecInRuntimeSource(t *testing.T) {
	root := repoRoot(t)
	files := collectRuntimeSourceFiles(t, root)
	var hits []string
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		text := string(data)
		if strings.Contains(text, `exec.Command("git"`) ||
			strings.Contains(text, `exec.CommandContext(ctx, "git"`) ||
			strings.Contains(text, `exec.LookPath("git")`) {
			hits = append(hits, rel(root, f))
		}
	}
	if len(hits) > 0 {
		t.Fatalf("runtime source must not invoke literal git binary directly: %v", hits)
	}
}

func collectRuntimeSourceFiles(t *testing.T, root string) []string {
	t.Helper()
	var files []string
	targets := []string{
		filepath.Join(root, "internal"),
		filepath.Join(root, "cmd"),
	}
	for _, target := range targets {
		err := filepath.WalkDir(target, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if filepath.Ext(path) != ".go" {
				return nil
			}
			if strings.HasSuffix(path, "_test.go") {
				return nil
			}
			files = append(files, path)
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", target, err)
		}
	}
	return files
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	dir := filepath.Dir(file)
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		next := filepath.Dir(dir)
		if next == dir {
			break
		}
		dir = next
	}
	t.Fatal("cannot locate repository root from compliance test")
	return ""
}

func rel(root, abs string) string {
	p, err := filepath.Rel(root, abs)
	if err != nil {
		return abs
	}
	return filepath.ToSlash(p)
}
