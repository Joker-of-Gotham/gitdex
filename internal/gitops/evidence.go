package gitops

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ExecutionEvidence records the outcome of a git operation for audit/debugging.
type ExecutionEvidence struct {
	TaskID        string             `json:"task_id"`
	CorrelationID string             `json:"correlation_id"`
	Action        string             `json:"action"`
	RepoPath      string             `json:"repo_path"`
	Timestamp     time.Time          `json:"timestamp"`
	Duration      time.Duration      `json:"duration"`
	GitCommands   []GitCommandRecord `json:"git_commands"`
	DiffBefore    string             `json:"diff_before,omitempty"`
	DiffAfter     string             `json:"diff_after,omitempty"`
	Result        string             `json:"result"`
	ErrorDetail   string             `json:"error_detail,omitempty"`
}

// GitCommandRecord records a single git command execution.
type GitCommandRecord struct {
	Command  string        `json:"command"`
	ExitCode int           `json:"exit_code"`
	Stdout   string        `json:"stdout,omitempty"`
	Stderr   string        `json:"stderr,omitempty"`
	Duration time.Duration `json:"duration"`
}

// EvidenceFilter filters evidence listings.
type EvidenceFilter struct {
	TaskID string
	Action string
	Limit  int
}

// EvidenceCollector writes and reads execution evidence to/from disk.
type EvidenceCollector struct {
	evidenceDir string
}

// NewEvidenceCollector creates a new EvidenceCollector.
func NewEvidenceCollector(evidenceDir string) *EvidenceCollector {
	return &EvidenceCollector{evidenceDir: evidenceDir}
}

// Collect marshals evidence to JSON and writes to <evidenceDir>/<task_id>.json.
func (c *EvidenceCollector) Collect(evidence *ExecutionEvidence) error {
	if evidence.TaskID == "" {
		return fmt.Errorf("task_id is required")
	}
	if err := os.MkdirAll(c.evidenceDir, 0755); err != nil {
		return fmt.Errorf("create evidence dir: %w", err)
	}
	path := filepath.Join(c.evidenceDir, evidence.TaskID+".json")
	data, err := json.MarshalIndent(evidence, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// Get reads and unmarshals the evidence file for the given taskID.
func (c *EvidenceCollector) Get(taskID string) (*ExecutionEvidence, error) {
	path := filepath.Join(c.evidenceDir, taskID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var e ExecutionEvidence
	if err := json.Unmarshal(data, &e); err != nil {
		return nil, err
	}
	return &e, nil
}

// List walks evidenceDir, reads all JSON files, filters in Go, and returns matching evidence.
func (c *EvidenceCollector) List(filter EvidenceFilter) ([]*ExecutionEvidence, error) {
	entries, err := os.ReadDir(c.evidenceDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var result []*ExecutionEvidence
	for _, ent := range entries {
		if ent.IsDir() || !strings.HasSuffix(ent.Name(), ".json") {
			continue
		}
		taskID := strings.TrimSuffix(ent.Name(), ".json")
		if filter.TaskID != "" && taskID != filter.TaskID {
			continue
		}
		ev, err := c.Get(taskID)
		if err != nil {
			continue
		}
		if filter.Action != "" && ev.Action != filter.Action {
			continue
		}
		result = append(result, ev)
		if filter.Limit > 0 && len(result) >= filter.Limit {
			break
		}
	}
	return result, nil
}
