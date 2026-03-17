package knowledge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Joker-of-Gotham/gitdex/internal/dotgitdex"
)

func TestExtract(t *testing.T) {
	tmp := t.TempDir()
	store := dotgitdex.New(tmp)
	if err := store.Init(); err != nil {
		t.Fatal(err)
	}

	if err := Extract(store); err != nil {
		t.Fatal(err)
	}

	files, err := os.ReadDir(store.KnowledgeDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("expected knowledge files to be extracted")
	}

	for _, f := range files {
		if f.Name() == "index.yaml" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(store.KnowledgeDir(), f.Name()))
		if err != nil {
			t.Errorf("failed to read %s: %v", f.Name(), err)
		}
		if len(data) == 0 {
			t.Errorf("%s is empty", f.Name())
		}
	}

	entries, err := store.ReadIndex()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Fatal("expected index entries to be generated")
	}

	for _, e := range entries {
		if e.KnowledgeID == "" || e.Path == "" || e.Domain == "" {
			t.Errorf("incomplete index entry: %+v", e)
		}
	}
}

func TestReader(t *testing.T) {
	tmp := t.TempDir()
	testFile := filepath.Join(tmp, "test.yaml")
	if err := os.WriteFile(testFile, []byte("test content here"), 0o644); err != nil {
		t.Fatal(err)
	}

	reader := NewReader(tmp)
	content, err := reader.ReadByPaths([]string{testFile})
	if err != nil {
		t.Fatal(err)
	}
	if content == "" {
		t.Fatal("expected non-empty content")
	}

	content2, err := reader.ReadByPaths([]string{filepath.Join(tmp, "nonexistent.yaml")})
	if err != nil {
		t.Fatal(err)
	}
	if content2 == "" {
		t.Fatal("expected fallback message for missing file")
	}
}

func TestReaderCacheRefreshAndInvalidate(t *testing.T) {
	tmp := t.TempDir()
	testFile := filepath.Join(tmp, "cache.yaml")
	if err := os.WriteFile(testFile, []byte("v1"), 0o644); err != nil {
		t.Fatal(err)
	}

	reader := NewReader(tmp)
	got1, err := reader.ReadByPaths([]string{testFile})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got1, "v1") {
		t.Fatalf("expected first read to contain v1, got %q", got1)
	}

	// Different size should invalidate cache entry automatically on next read.
	if err := os.WriteFile(testFile, []byte("v2-long"), 0o644); err != nil {
		t.Fatal(err)
	}
	got2, err := reader.ReadByPaths([]string{testFile})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got2, "v2-long") {
		t.Fatalf("expected refreshed content after file change, got %q", got2)
	}

	// Same-size updates can still be forced via explicit invalidation.
	if err := os.WriteFile(testFile, []byte("v3-long"), 0o644); err != nil {
		t.Fatal(err)
	}
	reader.Invalidate(testFile)
	got3, err := reader.ReadByPaths([]string{testFile})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got3, "v3-long") {
		t.Fatalf("expected invalidated cache to reload new content, got %q", got3)
	}
}
