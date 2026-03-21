package gitops

import (
	"context"
	"testing"
)

func TestBranchManager_CreateBranch_ListBranches(t *testing.T) {
	dir := initTestRepo(t)
	ctx := context.Background()
	bm := NewBranchManager(NewGitExecutor())

	if err := bm.CreateBranch(ctx, dir, "feature", ""); err != nil {
		t.Fatalf("CreateBranch: %v", err)
	}

	branches, err := bm.ListBranches(ctx, dir, false)
	if err != nil {
		t.Fatalf("ListBranches: %v", err)
	}

	var found bool
	for _, b := range branches {
		// Name may be just "feature" or include extra data on some git versions
		if b.Name == "feature" || (len(b.Name) >= 7 && b.Name[:7] == "feature") {
			found = true
			break
		}
	}
	if !found {
		var names []string
		for _, b := range branches {
			names = append(names, b.Name)
		}
		t.Errorf("ListBranches: expected feature in %v", names)
	}
}

func TestBranchManager_DeleteBranch(t *testing.T) {
	dir := initTestRepo(t)
	ctx := context.Background()
	bm := NewBranchManager(NewGitExecutor())

	if err := bm.CreateBranch(ctx, dir, "to-delete", ""); err != nil {
		t.Fatalf("CreateBranch: %v", err)
	}
	if err := bm.DeleteBranch(ctx, dir, "to-delete", false); err != nil {
		t.Fatalf("DeleteBranch: %v", err)
	}

	branches, err := bm.ListBranches(ctx, dir, false)
	if err != nil {
		t.Fatalf("ListBranches: %v", err)
	}
	for _, b := range branches {
		if b.Name == "to-delete" {
			t.Error("DeleteBranch: branch should be gone")
			break
		}
	}
}

func TestBranchManager_SwitchBranch(t *testing.T) {
	dir := initTestRepo(t)
	ctx := context.Background()
	bm := NewBranchManager(NewGitExecutor())

	if err := bm.CreateBranch(ctx, dir, "switched", ""); err != nil {
		t.Fatalf("CreateBranch: %v", err)
	}
	if err := bm.SwitchBranch(ctx, dir, "switched"); err != nil {
		t.Fatalf("SwitchBranch: %v", err)
	}

	hi := NewHistoryInspector(NewGitExecutor())
	headSHA, err := hi.RevParse(ctx, dir, "HEAD")
	if err != nil {
		t.Fatalf("RevParse HEAD: %v", err)
	}
	// HEAD should point to same commit as switched branch (no new commits yet)
	branchSHA, err := hi.RevParse(ctx, dir, "switched")
	if err != nil {
		t.Fatalf("RevParse switched: %v", err)
	}
	if headSHA != branchSHA {
		t.Errorf("SwitchBranch: HEAD=%s, switched=%s", headSHA, branchSHA)
	}
}

func TestBranchManager_CreateTag_ListTags(t *testing.T) {
	dir := initTestRepo(t)
	ctx := context.Background()
	bm := NewBranchManager(NewGitExecutor())

	if err := bm.CreateTag(ctx, dir, "v1.0.0", "", true, "Release v1.0.0"); err != nil {
		t.Fatalf("CreateTag: %v", err)
	}

	tags, err := bm.ListTags(ctx, dir)
	if err != nil {
		t.Fatalf("ListTags: %v", err)
	}
	if !contains(tags, "v1.0.0") {
		t.Errorf("ListTags: expected v1.0.0 in %v", tags)
	}
}

func TestBranchManager_MergeBase(t *testing.T) {
	dir := initTestRepo(t)
	ctx := context.Background()
	bm := NewBranchManager(NewGitExecutor())

	// Create two branches from the same initial commit
	if err := bm.CreateBranch(ctx, dir, "a", ""); err != nil {
		t.Fatalf("CreateBranch a: %v", err)
	}
	if err := bm.CreateBranch(ctx, dir, "b", ""); err != nil {
		t.Fatalf("CreateBranch b: %v", err)
	}

	base, err := bm.MergeBase(ctx, dir, "a", "b")
	if err != nil {
		t.Fatalf("MergeBase: %v", err)
	}
	if base == "" {
		t.Error("MergeBase: expected non-empty SHA")
	}
	// Both branches point to initial commit, so merge base is that commit
	if len(base) != 40 {
		t.Errorf("MergeBase: expected 40-char SHA, got %q (len=%d)", base, len(base))
	}
}

func TestParseBranchTrack(t *testing.T) {
	tests := []struct {
		input      string
		wantAhead  int
		wantBehind int
	}{
		{input: "", wantAhead: 0, wantBehind: 0},
		{input: "[ahead 2]", wantAhead: 2, wantBehind: 0},
		{input: "[behind 3]", wantAhead: 0, wantBehind: 3},
		{input: "[ahead 4, behind 1]", wantAhead: 4, wantBehind: 1},
		{input: "[gone]", wantAhead: 0, wantBehind: 0},
	}

	for _, tt := range tests {
		ahead, behind := parseBranchTrack(tt.input)
		if ahead != tt.wantAhead || behind != tt.wantBehind {
			t.Fatalf("%q => ahead=%d behind=%d, want %d/%d", tt.input, ahead, behind, tt.wantAhead, tt.wantBehind)
		}
	}
}

func TestBranchManager_ListBranchesIncludesLastCommit(t *testing.T) {
	dir := initTestRepo(t)
	ctx := context.Background()
	bm := NewBranchManager(NewGitExecutor())

	branches, err := bm.ListBranches(ctx, dir, false)
	if err != nil {
		t.Fatalf("ListBranches: %v", err)
	}
	if len(branches) == 0 {
		t.Fatal("expected at least one branch")
	}
	if branches[0].LastCommit == "" {
		t.Fatal("expected last commit subject to be populated")
	}
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
