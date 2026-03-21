package collaboration

import (
	"context"
	"testing"
)

func TestMemoryContextStore_SaveAndGet(t *testing.T) {
	store := NewMemoryContextStore()
	ctx := context.Background()

	tc := &TaskContext{
		PrimaryObjectRef: "owner/repo#1",
		LinkedObjects:    []ObjectLink{},
		RelatedTasks:     []string{"task-1"},
		Notes:            "test notes",
	}
	err := store.SaveContext(ctx, tc)
	if err != nil {
		t.Fatalf("SaveContext: %v", err)
	}
	if tc.ContextID == "" {
		t.Error("ContextID should be set")
	}

	got, err := store.GetContext(ctx, tc.ContextID)
	if err != nil {
		t.Fatalf("GetContext: %v", err)
	}
	if got.PrimaryObjectRef != tc.PrimaryObjectRef {
		t.Errorf("PrimaryObjectRef = %q, want %q", got.PrimaryObjectRef, tc.PrimaryObjectRef)
	}
}

func TestMemoryContextStore_GetByObjectRef(t *testing.T) {
	store := NewMemoryContextStore()
	ctx := context.Background()

	tc := &TaskContext{
		PrimaryObjectRef: "owner/repo#42",
		LinkedObjects: []ObjectLink{
			{SourceRef: "owner/repo#42", TargetRef: "owner/repo#43", LinkType: LinkBlocks},
		},
	}
	err := store.SaveContext(ctx, tc)
	if err != nil {
		t.Fatalf("SaveContext: %v", err)
	}

	got, err := store.GetByObjectRef(ctx, "owner/repo#42")
	if err != nil {
		t.Fatalf("GetByObjectRef: %v", err)
	}
	if len(got.LinkedObjects) != 1 {
		t.Errorf("LinkedObjects len = %d, want 1", len(got.LinkedObjects))
	}
	if got.LinkedObjects[0].LinkType != LinkBlocks {
		t.Errorf("LinkType = %q, want blocks", got.LinkedObjects[0].LinkType)
	}
}

func TestMemoryContextStore_ListContexts(t *testing.T) {
	store := NewMemoryContextStore()
	ctx := context.Background()

	for i := 1; i <= 3; i++ {
		tc := &TaskContext{PrimaryObjectRef: "owner/repo#" + string(rune('0'+i))}
		_ = store.SaveContext(ctx, tc)
	}

	list, err := store.ListContexts(ctx)
	if err != nil {
		t.Fatalf("ListContexts: %v", err)
	}
	if len(list) < 3 {
		t.Errorf("ListContexts returned %d, want at least 3", len(list))
	}
}

func TestLinkType_Valid(t *testing.T) {
	valid := []LinkType{LinkBlocks, LinkBlockedBy, LinkRelatesTo, LinkDuplicateOf, LinkParentOf, LinkChildOf}
	for _, lt := range valid {
		if !lt.Valid() {
			t.Errorf("LinkType %q should be valid", lt)
		}
	}
	if (LinkType("invalid")).Valid() {
		t.Error("invalid link type should not be valid")
	}
}
