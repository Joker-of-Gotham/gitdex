package context

import (
	"fmt"
	"strings"
)

// CompressFileList compresses a file list beyond maxItems into a summary.
func CompressFileList(files []string, maxItems int) string {
	if len(files) <= maxItems {
		return strings.Join(files, "\n")
	}
	shown := files[:maxItems]
	return strings.Join(shown, "\n") + fmt.Sprintf("\n... and %d more files", len(files)-maxItems)
}

// CompressCommitHistory keeps the most recent commits and summarizes the rest.
func CompressCommitHistory(commits []string, keepRecent int) string {
	if len(commits) <= keepRecent {
		return strings.Join(commits, "\n")
	}
	recent := commits[:keepRecent]
	return strings.Join(recent, "\n") + fmt.Sprintf("\n... (%d older commits omitted)", len(commits)-keepRecent)
}

// CompressOperationLog keeps detailed recent ops and summarizes older ones.
func CompressOperationLog(ops []string, keepDetailed int) string {
	if len(ops) <= keepDetailed {
		return strings.Join(ops, "\n")
	}
	detailed := ops[:keepDetailed]
	olderCount := len(ops) - keepDetailed

	execCount, skipCount, cancelCount := 0, 0, 0
	for _, op := range ops[keepDetailed:] {
		lower := strings.ToLower(op)
		switch {
		case strings.Contains(lower, "executed") || strings.Contains(lower, "\"type\":\"executed\""):
			execCount++
		case strings.Contains(lower, "skipped") || strings.Contains(lower, "\"type\":\"skipped\""):
			skipCount++
		case strings.Contains(lower, "cancelled") || strings.Contains(lower, "\"type\":\"cancelled\""):
			cancelCount++
		default:
			execCount++
		}
	}

	summary := fmt.Sprintf("... %d older operations (exec=%d, skip=%d, cancel=%d)",
		olderCount, execCount, skipCount, cancelCount)
	return strings.Join(detailed, "\n") + "\n" + summary
}

// CompressDiffStats summarizes diff stats into a compact form.
func CompressDiffStats(stats []struct {
	Path     string
	Ins, Del int
}, maxFiles int) string {
	if len(stats) == 0 {
		return ""
	}
	var lines []string
	totalIns, totalDel := 0, 0
	for i, s := range stats {
		totalIns += s.Ins
		totalDel += s.Del
		if i < maxFiles {
			lines = append(lines, fmt.Sprintf("  %s: +%d/-%d", s.Path, s.Ins, s.Del))
		}
	}
	header := fmt.Sprintf("Diff: %d files, +%d/-%d lines", len(stats), totalIns, totalDel)
	if len(stats) > maxFiles {
		lines = append(lines, fmt.Sprintf("  ... and %d more files", len(stats)-maxFiles))
	}
	return header + "\n" + strings.Join(lines, "\n")
}
