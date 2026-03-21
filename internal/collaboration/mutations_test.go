package collaboration

import (
	"context"
	"testing"
)

func TestMutationEngine_ObjectStoreSaveAndGet(t *testing.T) {
	store := NewMemoryObjectStore()
	ctx := context.Background()

	obj := &CollaborationObject{
		ObjectType: ObjectTypeIssue,
		RepoOwner:  "owner",
		RepoName:   "repo",
		Number:     1,
		Title:      "Test issue",
		Body:       "Body text",
		State:      "open",
		Author:     "alice",
	}

	if err := store.SaveObject(ctx, obj); err != nil {
		t.Fatalf("SaveObject: %v", err)
	}

	got, err := store.GetByRepoAndNumber(ctx, "owner", "repo", 1)
	if err != nil {
		t.Fatalf("GetByRepoAndNumber: %v", err)
	}
	if got.Number != 1 || got.Title != "Test issue" {
		t.Errorf("persisted object: Number=%d Title=%q", got.Number, got.Title)
	}
}

func TestMutationEngine_ObjectStoreGetNonExistent(t *testing.T) {
	store := NewMemoryObjectStore()
	ctx := context.Background()

	_, err := store.GetByRepoAndNumber(ctx, "owner", "repo", 99)
	if err == nil {
		t.Fatal("expected error for non-existent object")
	}
}

func TestMutationEngine_ObjectStoreUpdateState(t *testing.T) {
	store := NewMemoryObjectStore()
	ctx := context.Background()

	obj := &CollaborationObject{
		ObjectType: ObjectTypeIssue,
		RepoOwner:  "owner",
		RepoName:   "repo",
		Number:     7,
		Title:      "To close",
		State:      "open",
		Author:     "bob",
	}
	if err := store.SaveObject(ctx, obj); err != nil {
		t.Fatalf("SaveObject: %v", err)
	}

	got, _ := store.GetByRepoAndNumber(ctx, "owner", "repo", 7)
	got.State = "closed"
	if err := store.SaveObject(ctx, got); err != nil {
		t.Fatalf("SaveObject (update): %v", err)
	}

	updated, _ := store.GetByRepoAndNumber(ctx, "owner", "repo", 7)
	if updated.State != "closed" {
		t.Errorf("State = %q, want closed", updated.State)
	}
}

func TestGitHubMutationEngine_NilClient(t *testing.T) {
	engine := NewGitHubMutationEngine(nil)
	ctx := context.Background()

	req := &MutationRequest{
		MutationType: MutationCreate,
		ObjectType:   ObjectTypeIssue,
		RepoOwner:    "owner",
		RepoName:     "repo",
		Title:        "Test",
	}

	_, err := engine.Execute(ctx, req)
	if err == nil {
		t.Fatal("expected error with nil client")
	}
}

func TestMutationRequest_Validation(t *testing.T) {
	req := &MutationRequest{
		MutationType: MutationCreate,
		ObjectType:   ObjectTypeIssue,
		RepoOwner:    "",
		RepoName:     "repo",
		Title:        "Test",
	}

	if req.RepoOwner != "" {
		t.Error("expected empty RepoOwner")
	}

	req.RepoOwner = "owner"
	if req.RepoOwner != "owner" {
		t.Error("expected owner to be set")
	}
}
