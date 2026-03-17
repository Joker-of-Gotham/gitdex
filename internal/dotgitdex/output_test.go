package dotgitdex

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestOutputLogAppendAndReadRecent(t *testing.T) {
	tmp := t.TempDir()
	mgr := New(tmp)
	_ = mgr.Init()

	ol := NewOutputLog(mgr)

	r1 := Round{
		SessionID: "s1", RoundID: 1, Mode: "auto", Flow: "maintain",
		StartedAt: time.Now(), FinishedAt: time.Now(), Status: "success",
		Steps: []Step{
			{SequenceID: 1, Name: "fetch", Command: "git fetch", Success: true, StartedAt: time.Now(), FinishedAt: time.Now()},
		},
	}
	r2 := Round{
		SessionID: "s1", RoundID: 2, Mode: "auto", Flow: "goal",
		StartedAt: time.Now(), FinishedAt: time.Now(), Status: "partial-failure",
		Steps: []Step{
			{SequenceID: 1, Name: "push", Command: "git push", Success: false, Stderr: "rejected", StartedAt: time.Now(), FinishedAt: time.Now()},
		},
	}
	if err := ol.AppendRound(r1); err != nil {
		t.Fatal(err)
	}
	if err := ol.AppendRound(r2); err != nil {
		t.Fatal(err)
	}

	ol2 := NewOutputLog(mgr)
	text, err := ol2.ReadRecent(3)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "Round 1") || !strings.Contains(text, "Round 2") {
		t.Errorf("expected both rounds in output, got:\n%s", text)
	}
	if !strings.Contains(text, "FAIL") {
		t.Error("expected FAIL marker for round 2")
	}
}

func TestOutputLogSnapshotRecent(t *testing.T) {
	tmp := t.TempDir()
	mgr := New(tmp)
	_ = mgr.Init()
	ol := NewOutputLog(mgr)

	for i := 1; i <= 3; i++ {
		r := Round{
			SessionID: "s1", RoundID: i, Mode: "auto", Flow: "maintain",
			StartedAt: time.Now(), FinishedAt: time.Now(), Status: "success",
			Steps: []Step{
				{SequenceID: 1, Name: "step", Command: "echo ok", Success: true, StartedAt: time.Now(), FinishedAt: time.Now()},
			},
		}
		if err := ol.AppendRound(r); err != nil {
			t.Fatal(err)
		}
	}

	snap, err := ol.SnapshotRecent(2)
	if err != nil {
		t.Fatal(err)
	}
	if len(snap) != 2 {
		t.Fatalf("expected 2 snapshot rounds, got %d", len(snap))
	}
	if snap[0].RoundID != 2 || snap[1].RoundID != 3 {
		t.Fatalf("unexpected snapshot order: %+v", []int{snap[0].RoundID, snap[1].RoundID})
	}
}

func TestOutputLogBuildReplayScript(t *testing.T) {
	tmp := t.TempDir()
	mgr := New(tmp)
	_ = mgr.Init()
	ol := NewOutputLog(mgr)

	r := Round{
		SessionID: "s1", RoundID: 7, Mode: "manual", Flow: "goal",
		StartedAt: time.Now(), FinishedAt: time.Now(), Status: "partial-failure",
		Steps: []Step{
			{SequenceID: 1, Name: "fetch", Command: "git fetch --prune", Success: true, StartedAt: time.Now(), FinishedAt: time.Now()},
			{SequenceID: 2, Name: "write", FilePath: "README.md", FileOp: "update", Success: true, StartedAt: time.Now(), FinishedAt: time.Now()},
		},
	}
	if err := ol.AppendRound(r); err != nil {
		t.Fatal(err)
	}

	script, err := ol.BuildReplayScript(1)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(script, "git fetch --prune") {
		t.Fatalf("expected command in replay script, got:\n%s", script)
	}
	if !strings.Contains(script, "# file_update README.md") {
		t.Fatalf("expected file operation hint in replay script, got:\n%s", script)
	}
}

func TestOutputLogLoadCorruptedFileSelfHeals(t *testing.T) {
	tmp := t.TempDir()
	mgr := New(tmp)
	_ = mgr.Init()
	if err := os.WriteFile(mgr.OutputPath(), []byte("{invalid json"), 0o644); err != nil {
		t.Fatal(err)
	}

	ol := NewOutputLog(mgr)
	text, err := ol.ReadRecent(3)
	if err != nil {
		t.Fatalf("expected corrupted file to be healed, got err: %v", err)
	}
	if text != "" {
		t.Fatalf("expected empty output after healing corrupted file, got %q", text)
	}

	healed, err := os.ReadFile(mgr.OutputPath())
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(healed)) != "[]" {
		t.Fatalf("expected healed output file to be [], got %q", string(healed))
	}

	backups, err := filepath.Glob(mgr.OutputPath() + ".corrupt.*")
	if err != nil {
		t.Fatal(err)
	}
	if len(backups) == 0 {
		t.Fatal("expected a corrupt backup file to be created")
	}
}
