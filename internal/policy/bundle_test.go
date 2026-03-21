package policy

import (
	"testing"
	"time"
)

func TestMemoryBundleStore_SaveAndGet(t *testing.T) {
	store := NewMemoryBundleStore()
	bundle := &PolicyBundle{
		Name:      "test-bundle",
		Version:   "1.0.0",
		CreatedAt: time.Now().UTC(),
	}

	if err := store.SaveBundle(bundle); err != nil {
		t.Fatalf("save error: %v", err)
	}
	if bundle.BundleID == "" {
		t.Error("expected BundleID to be set by SaveBundle")
	}

	got, err := store.GetBundle(bundle.BundleID)
	if err != nil {
		t.Fatalf("get error: %v", err)
	}
	if got.Name != "test-bundle" {
		t.Errorf("got Name %q, want test-bundle", got.Name)
	}
	if got.Version != "1.0.0" {
		t.Errorf("got Version %q, want 1.0.0", got.Version)
	}
}

func TestMemoryBundleStore_GetNotFound(t *testing.T) {
	store := NewMemoryBundleStore()
	_, err := store.GetBundle("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent bundle")
	}
}

func TestMemoryBundleStore_ListBundles(t *testing.T) {
	store := NewMemoryBundleStore()
	_ = store.SaveBundle(&PolicyBundle{Name: "b1", CreatedAt: time.Now().UTC()})
	_ = store.SaveBundle(&PolicyBundle{Name: "b2", CreatedAt: time.Now().UTC()})

	list, err := store.ListBundles()
	if err != nil {
		t.Fatalf("list error: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("got %d bundles, want 2", len(list))
	}
}

func TestMemoryBundleStore_GetActiveBundle(t *testing.T) {
	store := NewMemoryBundleStore()
	bundle := &PolicyBundle{Name: "active", CreatedAt: time.Now().UTC()}
	_ = store.SaveBundle(bundle)

	active, err := store.GetActiveBundle()
	if err != nil {
		t.Fatalf("get active error: %v", err)
	}
	if active == nil {
		t.Fatal("expected active bundle to be set after first save")
	}
	if active.BundleID != bundle.BundleID {
		t.Errorf("got active ID %q, want %q", active.BundleID, bundle.BundleID)
	}
}

func TestMemoryBundleStore_SetActiveBundle(t *testing.T) {
	store := NewMemoryBundleStore()
	b1 := &PolicyBundle{Name: "b1", CreatedAt: time.Now().UTC()}
	b2 := &PolicyBundle{Name: "b2", CreatedAt: time.Now().UTC()}
	_ = store.SaveBundle(b1)
	_ = store.SaveBundle(b2)

	if err := store.SetActiveBundle(b2.BundleID); err != nil {
		t.Fatalf("set active error: %v", err)
	}
	active, _ := store.GetActiveBundle()
	if active.BundleID != b2.BundleID {
		t.Errorf("got active %q, want %q", active.BundleID, b2.BundleID)
	}
}

func TestMemoryBundleStore_SaveNilFails(t *testing.T) {
	store := NewMemoryBundleStore()
	err := store.SaveBundle(nil)
	if err == nil {
		t.Fatal("expected error when saving nil bundle")
	}
}
