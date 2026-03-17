package flow

import "testing"

func TestStateMachine_HappyPath(t *testing.T) {
	sm := NewStateMachine()
	steps := []FlowEvent{
		EventAnalyzeStart,
		EventAnalyzeSuccess,
		EventExecuteSuccess,
		EventRefreshDone,
	}
	for _, e := range steps {
		if err := sm.Transition(e); err != nil {
			t.Fatalf("unexpected transition error on %s: %v", e, err)
		}
	}
	if sm.State() != StateIdle {
		t.Fatalf("expected final state idle, got %s", sm.State())
	}
}

func TestStateMachine_ReplanPath(t *testing.T) {
	sm := NewStateMachine()
	steps := []FlowEvent{
		EventAnalyzeStart,
		EventAnalyzeSuccess,
		EventExecuteFail,
		EventReplan,
	}
	for _, e := range steps {
		if err := sm.Transition(e); err != nil {
			t.Fatalf("unexpected transition error on %s: %v", e, err)
		}
	}
	if sm.State() != StateAnalyzing {
		t.Fatalf("expected analyzing after replan, got %s", sm.State())
	}
}

