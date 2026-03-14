package platform

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStoreAndResolveReleaseAssetBytes(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", root)
	t.Setenv("USERPROFILE", root)
	t.Setenv("APPDATA", root)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, ".config"))

	entry, err := StoreReleaseAssetBytes("gitdex.txt", []byte("release bytes"), map[string]string{
		"content_type":             "text/plain",
		"source_kind":              "inline_text",
		"recoverability":           "reversible",
		"partial_restore_required": "false",
	})
	if err != nil {
		t.Fatal(err)
	}
	if entry.Ref == "" {
		t.Fatal("expected asset ref to be created")
	}
	if _, err := os.Stat(entry.BytesPath); err != nil {
		t.Fatalf("expected bytes to be persisted: %v", err)
	}

	resolved, err := ResolveReleaseAssetRef(entry.Ref)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.BytesPath != entry.BytesPath {
		t.Fatalf("expected resolved bytes path %q, got %q", entry.BytesPath, resolved.BytesPath)
	}
	if resolved.Recoverability != "reversible" {
		t.Fatalf("expected recoverability to persist, got %q", resolved.Recoverability)
	}
}
