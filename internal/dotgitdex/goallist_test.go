package dotgitdex

import (
	"testing"
)

func TestParseGoalList(t *testing.T) {
	input := `# Goal-List

- [ ] Goal A
  - [ ] Todo A-1
  - [x] Todo A-2
- [x] Goal B
  - [x] Todo B-1
`

	goals, err := ParseGoalList(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(goals) != 2 {
		t.Fatalf("got %d goals, want 2", len(goals))
	}

	if goals[0].Title != "Goal A" || goals[0].Completed {
		t.Errorf("goal[0] = %+v", goals[0])
	}
	if len(goals[0].Todos) != 2 {
		t.Fatalf("goal[0] has %d todos, want 2", len(goals[0].Todos))
	}
	if goals[0].Todos[0].Completed {
		t.Error("Todo A-1 should not be completed")
	}
	if !goals[0].Todos[1].Completed {
		t.Error("Todo A-2 should be completed")
	}
	if !goals[1].Completed {
		t.Error("Goal B should be completed")
	}
}

func TestPendingGoals(t *testing.T) {
	goals := []Goal{
		{Title: "A", Completed: false, Todos: []Todo{{Title: "t1", Completed: true}, {Title: "t2"}}},
		{Title: "B", Completed: true},
	}
	pending := PendingGoals(goals)
	if len(pending) != 1 {
		t.Fatalf("got %d pending, want 1", len(pending))
	}
	if pending[0].Title != "A" {
		t.Errorf("pending[0].Title = %q, want A", pending[0].Title)
	}
	if len(pending[0].Todos) != 1 {
		t.Errorf("pending[0] should have 1 incomplete todo, got %d", len(pending[0].Todos))
	}
}

func TestWriteReadGoalList(t *testing.T) {
	tmp := t.TempDir()
	mgr := New(tmp)
	_ = mgr.Init()

	goals := []Goal{
		{Title: "Create PR", Completed: false, Todos: []Todo{{Title: "Write code"}, {Title: "Push", Completed: true}}},
		{Title: "Deploy", Completed: true},
	}
	if err := mgr.WriteGoalList(goals); err != nil {
		t.Fatal(err)
	}
	got, err := mgr.ReadGoalList()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("read %d goals, want 2", len(got))
	}
	if got[0].Title != "Create PR" {
		t.Errorf("got[0].Title = %q", got[0].Title)
	}
	if !got[1].Completed {
		t.Error("got[1] should be completed")
	}
}
