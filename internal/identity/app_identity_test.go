package identity

import (
	"testing"
	"time"
)

func TestMemoryIdentityStore_SaveAndGet(t *testing.T) {
	store := NewMemoryIdentityStore()
	id := &AppIdentity{
		IdentityID:     "id_test",
		IdentityType:   IdentityTypeGitHubApp,
		AppID:          "123",
		InstallationID: "456",
		Capabilities:   []Capability{CapReadRepo, CapReadIssues},
		CreatedAt:      time.Now().UTC(),
	}

	if err := store.SaveIdentity(id); err != nil {
		t.Fatalf("save error: %v", err)
	}
	if id.IdentityID == "" {
		t.Error("expected IdentityID to be set by SaveIdentity")
	}

	got, err := store.GetIdentity(id.IdentityID)
	if err != nil {
		t.Fatalf("get error: %v", err)
	}
	if got.IdentityType != IdentityTypeGitHubApp {
		t.Errorf("got IdentityType %q, want github_app", got.IdentityType)
	}
	if got.AppID != "123" {
		t.Errorf("got AppID %q, want 123", got.AppID)
	}
	if len(got.Capabilities) != 2 {
		t.Errorf("got %d capabilities, want 2", len(got.Capabilities))
	}
}

func TestMemoryIdentityStore_GetNotFound(t *testing.T) {
	store := NewMemoryIdentityStore()
	_, err := store.GetIdentity("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent identity")
	}
}

func TestMemoryIdentityStore_ListIdentities(t *testing.T) {
	store := NewMemoryIdentityStore()
	_ = store.SaveIdentity(&AppIdentity{IdentityType: IdentityTypePAT, CreatedAt: time.Now().UTC()})
	_ = store.SaveIdentity(&AppIdentity{IdentityType: IdentityTypeToken, CreatedAt: time.Now().UTC()})

	list, err := store.ListIdentities()
	if err != nil {
		t.Fatalf("list error: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("got %d identities, want 2", len(list))
	}
}

func TestMemoryIdentityStore_GetCurrentIdentity(t *testing.T) {
	store := NewMemoryIdentityStore()
	id := &AppIdentity{IdentityType: IdentityTypeGitHubApp, AppID: "1", InstallationID: "2", CreatedAt: time.Now().UTC()}
	_ = store.SaveIdentity(id)

	current, err := store.GetCurrentIdentity()
	if err != nil {
		t.Fatalf("get current error: %v", err)
	}
	if current == nil {
		t.Fatal("expected current identity to be set after first save")
	}
	if current.IdentityID != id.IdentityID {
		t.Errorf("got current ID %q, want %q", current.IdentityID, id.IdentityID)
	}
}

func TestMemoryIdentityStore_SetCurrentIdentity(t *testing.T) {
	store := NewMemoryIdentityStore()
	id1 := &AppIdentity{IdentityType: IdentityTypePAT, CreatedAt: time.Now().UTC()}
	id2 := &AppIdentity{IdentityType: IdentityTypeToken, CreatedAt: time.Now().UTC()}
	_ = store.SaveIdentity(id1)
	_ = store.SaveIdentity(id2)

	if err := store.SetCurrentIdentity(id2.IdentityID); err != nil {
		t.Fatalf("set current error: %v", err)
	}
	current, _ := store.GetCurrentIdentity()
	if current.IdentityID != id2.IdentityID {
		t.Errorf("got current %q, want %q", current.IdentityID, id2.IdentityID)
	}
}

func TestMemoryIdentityStore_SaveNilFails(t *testing.T) {
	store := NewMemoryIdentityStore()
	err := store.SaveIdentity(nil)
	if err == nil {
		t.Fatal("expected error when saving nil identity")
	}
}
