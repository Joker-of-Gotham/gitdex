package dotgitdex

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// Round captures one LLM-call window's execution log.
type Round struct {
	SessionID  string    `json:"session_id"`
	RoundID    int       `json:"round_id"`
	Mode       string    `json:"mode"`
	Flow       string    `json:"flow"` // maintain, goal, creative
	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at"`
	Status     string    `json:"status"` // success, partial-failure, skipped
	Steps      []Step    `json:"steps"`
}

// Step records one suggestion execution.
type Step struct {
	SequenceID int       `json:"sequence_id"`
	Name       string    `json:"name"`
	ActionType string    `json:"action_type,omitempty"`
	TraceID    string    `json:"trace_id,omitempty"`
	RoundID    string    `json:"round_id,omitempty"`
	AttemptID  string    `json:"attempt_id,omitempty"`
	SliceID    string    `json:"slice_id,omitempty"`
	Command    string    `json:"command,omitempty"`
	FilePath   string    `json:"file_path,omitempty"`
	FileOp     string    `json:"file_operation,omitempty"`
	Stdout     string    `json:"stdout,omitempty"`
	Stderr     string    `json:"stderr,omitempty"`
	ExitCode   int       `json:"exit_code"`
	Success    bool      `json:"success"`
	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at"`
}

// OutputLog manages maintain/output.txt as a rolling window of recent rounds.
type OutputLog struct {
	mgr    *Manager
	rounds []Round
}

// NewOutputLog creates an OutputLog backed by the manager's output path.
func NewOutputLog(mgr *Manager) *OutputLog {
	return &OutputLog{mgr: mgr}
}

// AppendRound adds a round to the log and persists.
func (o *OutputLog) AppendRound(r Round) error {
	o.rounds = append(o.rounds, r)
	return o.persist()
}

// ReadRecent returns at most maxRounds recent rounds formatted as text for LLM consumption.
func (o *OutputLog) ReadRecent(maxRounds int) (string, error) {
	if err := o.load(); err != nil {
		return "", err
	}
	start := 0
	if len(o.rounds) > maxRounds {
		start = len(o.rounds) - maxRounds
	}
	recent := o.rounds[start:]
	if len(recent) == 0 {
		return "", nil
	}

	var failedCmds []string
	var body strings.Builder

	for _, r := range recent {
		body.WriteString(fmt.Sprintf("--- Round %d [%s] flow=%s mode=%s status=%s ---\n",
			r.RoundID, r.StartedAt.Format(time.RFC3339), r.Flow, r.Mode, r.Status))
		for _, s := range r.Steps {
			if s.Success {
				body.WriteString(fmt.Sprintf("  [OK] %s", s.Name))
			} else {
				body.WriteString(fmt.Sprintf("  [FAIL] %s", s.Name))
				failDesc := buildFailedEntry(s)
				if failDesc != "" {
					failedCmds = append(failedCmds, failDesc)
				}
			}
			if s.TraceID != "" {
				body.WriteString(fmt.Sprintf("  trace=%s", s.TraceID))
			}
			if s.Command != "" {
				body.WriteString(fmt.Sprintf("  cmd=%s", s.Command))
			}
			if s.FilePath != "" {
				body.WriteString(fmt.Sprintf("  file=%s op=%s", s.FilePath, s.FileOp))
			}
			body.WriteString("\n")
			stdout := summarizeStdout(s)
			if stdout != "" {
				body.WriteString(fmt.Sprintf("    stdout: %s\n", stdout))
			}
			if s.Stderr != "" {
				body.WriteString(fmt.Sprintf("    stderr: %s\n", s.Stderr))
			}
		}
	}

	var b strings.Builder
	if len(failedCmds) > 0 {
		b.WriteString("=== FAILED COMMANDS (already attempted — try a different approach) ===\n")
		for _, fc := range failedCmds {
			b.WriteString("  ✗ " + fc + "\n")
		}
		b.WriteString("=== END FAILED COMMANDS ===\n\n")
	}
	b.WriteString(body.String())

	return b.String(), nil
}

// SnapshotRecent returns structured round snapshots for replay/diagnostics.
func (o *OutputLog) SnapshotRecent(maxRounds int) ([]Round, error) {
	if err := o.load(); err != nil {
		return nil, err
	}
	if len(o.rounds) == 0 {
		return nil, nil
	}
	if maxRounds <= 0 || maxRounds > len(o.rounds) {
		maxRounds = len(o.rounds)
	}
	start := len(o.rounds) - maxRounds
	return cloneRounds(o.rounds[start:]), nil
}

// BuildReplayScript emits a plain-text replay script from recent rounds.
// Command steps are emitted as executable lines; file ops are emitted as comments.
func (o *OutputLog) BuildReplayScript(maxRounds int) (string, error) {
	rounds, err := o.SnapshotRecent(maxRounds)
	if err != nil {
		return "", err
	}
	if len(rounds) == 0 {
		return "", nil
	}
	var b strings.Builder
	b.WriteString("# gitdex replay script (generated)\n")
	for _, r := range rounds {
		b.WriteString(fmt.Sprintf("# round=%d flow=%s mode=%s status=%s started=%s\n",
			r.RoundID, r.Flow, r.Mode, r.Status, r.StartedAt.Format(time.RFC3339)))
		for _, s := range r.Steps {
			switch {
			case strings.TrimSpace(s.Command) != "":
				b.WriteString(s.Command + "\n")
			case strings.TrimSpace(s.FilePath) != "":
				b.WriteString(fmt.Sprintf("# file_%s %s\n", s.FileOp, s.FilePath))
			default:
				b.WriteString(fmt.Sprintf("# step=%d name=%s (no replayable command)\n", s.SequenceID, s.Name))
			}
		}
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String()) + "\n", nil
}

func (o *OutputLog) persist() error {
	const maxKeptRounds = 10
	if len(o.rounds) > maxKeptRounds {
		o.rounds = o.rounds[len(o.rounds)-maxKeptRounds:]
	}
	data, err := json.MarshalIndent(o.rounds, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(o.mgr.OutputPath(), data, 0o644)
}

func (o *OutputLog) load() error {
	data, err := os.ReadFile(o.mgr.OutputPath())
	if err != nil {
		if os.IsNotExist(err) {
			o.rounds = nil
			return nil
		}
		return err
	}
	if err := json.Unmarshal(data, &o.rounds); err != nil {
		backup := o.mgr.OutputPath() + ".corrupt." + time.Now().Format("20060102T150405")
		_ = os.WriteFile(backup, data, 0o644)
		o.rounds = nil
		_ = os.WriteFile(o.mgr.OutputPath(), []byte("[]"), 0o644)
		return nil
	}
	return nil
}

// buildFailedEntry constructs a descriptive string for a failed step,
// covering both command-based and file-based actions.
func buildFailedEntry(s Step) string {
	if s.Command != "" {
		errMsg := s.Stderr
		if len(errMsg) > 200 {
			errMsg = errMsg[:200]
		}
		return fmt.Sprintf("%s (error: %s)", s.Command, errMsg)
	}
	if s.FilePath != "" {
		op := s.FileOp
		if op == "" {
			op = s.ActionType
		}
		if op == "" {
			op = "file_op"
		}
		errMsg := s.Stderr
		if len(errMsg) > 200 {
			errMsg = errMsg[:200]
		}
		return fmt.Sprintf("%s %s (error: %s)", op, s.FilePath, errMsg)
	}
	return ""
}

// summarizeStdout returns stdout as-is for most steps, but summarizes
// file_read output when it's very large to avoid context bloat.
func summarizeStdout(s Step) string {
	if s.Stdout == "" {
		return ""
	}
	if s.ActionType == "file_read" && len(s.Stdout) > 500 {
		lines := strings.SplitN(s.Stdout, "\n", 11)
		preview := lines
		if len(preview) > 10 {
			preview = preview[:10]
		}
		totalLines := strings.Count(s.Stdout, "\n") + 1
		return strings.Join(preview, "\n") + fmt.Sprintf("\n[file_read: %d lines total]", totalLines)
	}
	return s.Stdout
}

func cloneRounds(in []Round) []Round {
	out := make([]Round, len(in))
	for i := range in {
		out[i] = in[i]
		if len(in[i].Steps) > 0 {
			out[i].Steps = make([]Step, len(in[i].Steps))
			copy(out[i].Steps, in[i].Steps)
		}
	}
	return out
}
