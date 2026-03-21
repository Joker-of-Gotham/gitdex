package collaboration

import (
	"context"
	"sync"
	"testing"
)

func TestMemoryObjectStore_SaveAndGet(t *testing.T) {
	store := NewMemoryObjectStore()
	ctx := context.Background()

	obj := &CollaborationObject{
		ObjectType: ObjectTypeIssue,
		RepoOwner:  "owner",
		RepoName:   "repo",
		Number:     1,
		Title:      "Test issue",
		State:      "open",
		Author:     "alice",
	}

	if err := store.SaveObject(ctx, obj); err != nil {
		t.Fatalf("SaveObject: %v", err)
	}
	if obj.ObjectID == "" {
		t.Error("ObjectID should be set after save")
	}

	got, err := store.GetObject(ctx, obj.ObjectID)
	if err != nil {
		t.Fatalf("GetObject: %v", err)
	}
	if got.Title != obj.Title || got.Number != obj.Number {
		t.Errorf("got %+v, want Title=%q Number=%d", got, obj.Title, obj.Number)
	}
}

func TestMemoryObjectStore_GetByRepoAndNumber(t *testing.T) {
	store := NewMemoryObjectStore()
	ctx := context.Background()

	obj := &CollaborationObject{
		ObjectType: ObjectTypeIssue,
		RepoOwner:  "owner",
		RepoName:   "repo",
		Number:     42,
		Title:      "Issue 42",
	}
	if err := store.SaveObject(ctx, obj); err != nil {
		t.Fatalf("SaveObject: %v", err)
	}

	got, err := store.GetByRepoAndNumber(ctx, "owner", "repo", 42)
	if err != nil {
		t.Fatalf("GetByRepoAndNumber: %v", err)
	}
	if got.Number != 42 || got.Title != "Issue 42" {
		t.Errorf("got Number=%d Title=%q", got.Number, got.Title)
	}

	_, err = store.GetByRepoAndNumber(ctx, "owner", "repo", 99)
	if err == nil {
		t.Error("expected error for non-existent object")
	}
}

func TestMemoryObjectStore_ListObjects_FilterByType(t *testing.T) {
	store := NewMemoryObjectStore()
	ctx := context.Background()

	for i, ot := range []ObjectType{ObjectTypeIssue, ObjectTypePullRequest, ObjectTypeIssue} {
		obj := &CollaborationObject{
			ObjectType: ot,
			RepoOwner:  "o",
			RepoName:   "r",
			Number:     i + 1,
			Title:      "x",
		}
		if err := store.SaveObject(ctx, obj); err != nil {
			t.Fatalf("SaveObject: %v", err)
		}
	}

	list, err := store.ListObjects(ctx, &ObjectFilter{ObjectType: ObjectTypeIssue})
	if err != nil {
		t.Fatalf("ListObjects: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 issues, got %d", len(list))
	}
}

func TestMemoryObjectStore_ListObjects_FilterByState(t *testing.T) {
	store := NewMemoryObjectStore()
	ctx := context.Background()

	for i, state := range []string{"open", "closed", "open"} {
		obj := &CollaborationObject{
			ObjectType: ObjectTypeIssue,
			RepoOwner:  "o",
			RepoName:   "r",
			Number:     i + 1,
			State:      state,
		}
		if err := store.SaveObject(ctx, obj); err != nil {
			t.Fatalf("SaveObject: %v", err)
		}
	}

	open, _ := store.ListObjects(ctx, &ObjectFilter{State: "open"})
	if len(open) != 2 {
		t.Errorf("expected 2 open, got %d", len(open))
	}
	closed, _ := store.ListObjects(ctx, &ObjectFilter{State: "closed"})
	if len(closed) != 1 {
		t.Errorf("expected 1 closed, got %d", len(closed))
	}
}

func TestMemoryObjectStore_Concurrent(t *testing.T) {
	store := NewMemoryObjectStore()
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			obj := &CollaborationObject{
				ObjectType: ObjectTypeIssue,
				RepoOwner:  "o",
				RepoName:   "r",
				Number:     n + 100,
				Title:      "concurrent",
			}
			_ = store.SaveObject(ctx, obj)
		}(i)
	}
	wg.Wait()

	list, err := store.ListObjects(ctx, &ObjectFilter{RepoOwner: "o", RepoName: "r"})
	if err != nil {
		t.Fatalf("ListObjects: %v", err)
	}
	if len(list) != 10 {
		t.Errorf("expected 10 objects, got %d", len(list))
	}
}

func TestCollaborationObject_ZeroCreatedAt(t *testing.T) {
	store := NewMemoryObjectStore()
	ctx := context.Background()

	obj := &CollaborationObject{
		ObjectType: ObjectTypeIssue,
		RepoOwner:  "o",
		RepoName:   "r",
		Number:     1,
		Title:      "x",
	}
	if err := store.SaveObject(ctx, obj); err != nil {
		t.Fatalf("SaveObject: %v", err)
	}
	if obj.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
	if obj.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should be set")
	}
}

func TestObjectFilter_MatchLabels(t *testing.T) {
	store := NewMemoryObjectStore()
	ctx := context.Background()

	obj := &CollaborationObject{
		ObjectType: ObjectTypeIssue,
		RepoOwner:  "o",
		RepoName:   "r",
		Number:     1,
		Labels:     []string{"bug", "help wanted"},
	}
	if err := store.SaveObject(ctx, obj); err != nil {
		t.Fatalf("SaveObject: %v", err)
	}

	list, err := store.ListObjects(ctx, &ObjectFilter{Labels: []string{"bug"}})
	if err != nil {
		t.Fatalf("ListObjects: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 with label bug, got %d", len(list))
	}

	list2, _ := store.ListObjects(ctx, &ObjectFilter{Labels: []string{"nonexistent"}})
	if len(list2) != 0 {
		t.Errorf("expected 0 with nonexistent label, got %d", len(list2))
	}
}
