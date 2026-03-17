package flow

import "fmt"

type FlowState string

const (
	StateIdle      FlowState = "idle"
	StateAnalyzing FlowState = "analyzing"
	StateExecuting FlowState = "executing"
	StateRefreshing FlowState = "refreshing"
	StateReplanning FlowState = "replanning"
	StateCompleted FlowState = "completed"
	StateFailed    FlowState = "failed"
)

type FlowEvent string

const (
	EventAnalyzeStart   FlowEvent = "analyze_start"
	EventAnalyzeSuccess FlowEvent = "analyze_success"
	EventAnalyzeFail    FlowEvent = "analyze_fail"
	EventExecuteStart   FlowEvent = "execute_start"
	EventExecuteSuccess FlowEvent = "execute_success"
	EventExecuteFail    FlowEvent = "execute_fail"
	EventRefreshDone    FlowEvent = "refresh_done"
	EventReplan         FlowEvent = "replan"
	EventComplete       FlowEvent = "complete"
)

// StateMachine provides explicit state transitions for flow loops.
type StateMachine struct {
	state FlowState
}

func NewStateMachine() *StateMachine {
	return &StateMachine{state: StateIdle}
}

func (sm *StateMachine) State() FlowState {
	if sm == nil {
		return StateIdle
	}
	return sm.state
}

func (sm *StateMachine) Transition(event FlowEvent) error {
	if sm == nil {
		return nil
	}
	next, err := NextState(sm.state, event)
	if err != nil {
		return err
	}
	sm.state = next
	return nil
}

func NextState(current FlowState, event FlowEvent) (FlowState, error) {
	switch current {
	case StateIdle:
		if event == EventAnalyzeStart {
			return StateAnalyzing, nil
		}
	case StateAnalyzing:
		switch event {
		case EventAnalyzeSuccess:
			return StateExecuting, nil
		case EventAnalyzeFail:
			return StateFailed, nil
		}
	case StateExecuting:
		switch event {
		case EventExecuteSuccess:
			return StateRefreshing, nil
		case EventExecuteFail:
			return StateReplanning, nil
		}
	case StateRefreshing:
		switch event {
		case EventRefreshDone:
			return StateIdle, nil
		case EventComplete:
			return StateCompleted, nil
		}
	case StateReplanning:
		switch event {
		case EventReplan:
			return StateAnalyzing, nil
		case EventAnalyzeFail:
			return StateFailed, nil
		}
	case StateFailed:
		if event == EventReplan {
			return StateAnalyzing, nil
		}
	case StateCompleted:
		if event == EventAnalyzeStart {
			return StateAnalyzing, nil
		}
	}
	return current, fmt.Errorf("invalid transition: %s --(%s)--> ?", current, event)
}

