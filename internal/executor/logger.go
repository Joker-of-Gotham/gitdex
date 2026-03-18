package executor

import (
	"time"

	"github.com/Joker-of-Gotham/gitdex/internal/dotgitdex"
	"github.com/Joker-of-Gotham/gitdex/internal/planner"
)

// ExecutionLogger collects execution steps within a round and flushes to output.txt.
type ExecutionLogger struct {
	store     *dotgitdex.Manager
	outputLog *dotgitdex.OutputLog
	sessionID string
	roundID   int
	mode      string
	flow      string
	steps     []dotgitdex.Step
	startedAt time.Time
	hasError  bool
}

// NewExecutionLogger creates a new logger for a session.
func NewExecutionLogger(store *dotgitdex.Manager, sessionID, mode string) *ExecutionLogger {
	return &ExecutionLogger{
		store:     store,
		outputLog: dotgitdex.NewOutputLog(store),
		sessionID: sessionID,
		mode:      mode,
	}
}

// SetFlow sets the current flow name for logging.
func (l *ExecutionLogger) SetFlow(flow string) {
	l.flow = flow
}

// LogStep records one suggestion execution step.
func (l *ExecutionLogger) LogStep(seqID int, item planner.SuggestionItem, result *ExecutionResult) {
	if l.startedAt.IsZero() {
		l.startedAt = time.Now()
	}
	step := dotgitdex.Step{
		SequenceID:   seqID,
		Name:         item.Name,
		TraceID:      result.Trace.TraceID,
		RoundID:      result.Trace.RoundID,
		AttemptID:    result.Trace.AttemptID,
		SliceID:      result.Trace.SliceID,
		Command:      item.Action.Command,
		FilePath:     item.Action.FilePath,
		FileOp:       item.Action.FileOp,
		Stdout:       result.Stdout,
		Stderr:       result.Stderr,
		ExitCode:     result.ExitCode,
		Success:      result.Success,
		RecoveryType: string(result.RecoverBy.Type),
		RecoveryNote: result.RecoverBy.Reason,
		StartedAt:    time.Now().Add(-result.Duration),
		FinishedAt:   time.Now(),
	}
	if !result.Success {
		l.hasError = true
	}
	l.steps = append(l.steps, step)
}

// Flush persists the current round to output.txt and prepares for the next round.
func (l *ExecutionLogger) Flush() error {
	if len(l.steps) == 0 {
		return nil
	}

	status := "success"
	if l.hasError {
		status = "partial-failure"
	}

	round := dotgitdex.Round{
		SessionID:  l.sessionID,
		RoundID:    l.roundID,
		Mode:       l.mode,
		Flow:       l.flow,
		StartedAt:  l.startedAt,
		FinishedAt: time.Now(),
		Status:     status,
		Steps:      l.steps,
	}

	err := l.outputLog.AppendRound(round)
	l.NextRound()
	return err
}

// NextRound resets the step buffer and increments the round counter.
func (l *ExecutionLogger) NextRound() {
	l.roundID++
	l.steps = nil
	l.hasError = false
	l.startedAt = time.Time{}
}

// ReadRecentOutput reads the recent execution log text.
func (l *ExecutionLogger) ReadRecentOutput(maxRounds int) (string, error) {
	return l.outputLog.ReadRecent(maxRounds)
}
