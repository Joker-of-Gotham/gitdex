package tui

import (
	"errors"
	"testing"

	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/Joker-of-Gotham/gitdex/internal/tui/oplog"
	"github.com/stretchr/testify/assert"
)

func TestCollectRecentOps_IgnoresUserActions(t *testing.T) {
	log := oplog.New(oplog.DefaultMaxEntries)
	log.Add(oplog.Entry{
		Type:    oplog.EntryUserAction,
		Summary: "Accepted input suggestion: Resolve Commit Failure",
	})
	log.Add(oplog.Entry{
		Type:    oplog.EntryCmdFail,
		Summary: "Command failed: git commit -m \"Fix commit failure\"",
		Detail:  "nothing added to commit",
	})

	m := Model{opLog: log}
	ops := m.collectRecentOps()

	assert.Len(t, ops, 1)
	assert.Equal(t, "git commit -m \"Fix commit failure\"", ops[0].Command)
	assert.Equal(t, "failed", ops[0].Result)
	assert.Equal(t, "nothing added to commit", ops[0].Output)
}

func TestBestErrorDetail_UsesStdoutWhenStderrEmpty(t *testing.T) {
	result := &git.ExecutionResult{
		Stdout:   "On branch master\nnothing added to commit but untracked files present",
		Stderr:   "",
		ExitCode: 1,
		Success:  false,
	}

	detail := bestErrorDetail(errors.New("exit status 1"), result)
	assert.Contains(t, detail, "nothing added to commit")
}

func TestCollectRecentOps_IncludesSkipCancelAndModeSignals(t *testing.T) {
	log := oplog.New(oplog.DefaultMaxEntries)
	log.Add(oplog.Entry{Type: oplog.EntryUserAction, Summary: "Skipped suggestion: Push changes"})
	log.Add(oplog.Entry{Type: oplog.EntryUserAction, Summary: "Input cancelled"})
	log.Add(oplog.Entry{Type: oplog.EntryUserAction, Summary: "Switched AI mode to full"})

	m := Model{opLog: log}
	ops := m.collectRecentOps()

	assert.Len(t, ops, 3)
	assert.Equal(t, "skipped", ops[0].Type)
	assert.Equal(t, "cancelled", ops[1].Type)
	assert.Equal(t, "mode_switch", ops[2].Type)
	assert.Equal(t, "full", ops[2].Mode)
}

func TestCollectRecentOps_IncludesViewedAdvisory(t *testing.T) {
	log := oplog.New(oplog.DefaultMaxEntries)
	log.Add(oplog.Entry{Type: oplog.EntryUserAction, Summary: "Viewed advisory: Review .gitignore", Detail: "Inspect ignore rules"})

	m := Model{opLog: log}
	ops := m.collectRecentOps()

	assert.Len(t, ops, 1)
	assert.Equal(t, "viewed", ops[0].Type)
	assert.Equal(t, "Review .gitignore", ops[0].Action)
	assert.Equal(t, "viewed", ops[0].Result)
}
