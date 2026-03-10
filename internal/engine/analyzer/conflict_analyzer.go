package analyzer

import (
	"context"
	"strings"

	"github.com/Joker-of-Gotham/gitdex/internal/git/cli"
)

// ConflictAnalyzer detects and analyzes merge conflicts.
type ConflictAnalyzer struct {
	cli cli.GitCLI
}

// NewConflictAnalyzer creates a new ConflictAnalyzer.
func NewConflictAnalyzer(gitCLI cli.GitCLI) *ConflictAnalyzer {
	return &ConflictAnalyzer{cli: gitCLI}
}

// ConflictFile represents a file with merge conflicts.
type ConflictFile struct {
	Path        string
	OursLines   int
	TheirsLines int
	Markers     int
}

// DetectConflicts finds files with unmerged conflicts (diff-filter=U).
func (a *ConflictAnalyzer) DetectConflicts(ctx context.Context) ([]ConflictFile, error) {
	stdout, _, err := a.cli.Exec(ctx, "diff", "--name-only", "--diff-filter=U")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(stdout) == "" {
		return nil, nil
	}

	var conflicts []ConflictFile
	for _, path := range strings.Split(strings.TrimSpace(stdout), "\n") {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		cf := ConflictFile{Path: path}
		content, _, _ := a.cli.Exec(ctx, "show", ":"+path)
		cf.Markers = strings.Count(content, "<<<<<<<")
		cf.TheirsLines = strings.Count(content, ">>>>>>>")
		cf.OursLines = strings.Count(content, "=======")
		conflicts = append(conflicts, cf)
	}
	return conflicts, nil
}

// PreMergeCheck lists potential conflicts before merge (dry-run merge with source into target).
func (a *ConflictAnalyzer) PreMergeCheck(ctx context.Context, source, target string) []string {
	if source == "" || target == "" {
		return nil
	}
	stdout, _, err := a.cli.Exec(ctx, "merge-tree", "--write-tree", target, source)
	if err != nil {
		return nil
	}
	// merge-tree returns conflict info in its output; parse for conflicted paths
	var conflicts []string
	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "changed in both") || strings.Contains(line, "CONFLICT") {
			// Extract path if possible; merge-tree output varies
			conflicts = append(conflicts, line)
		}
	}
	if len(conflicts) == 0 {
		// Alternative: try merge --no-commit --no-ff to detect conflicts
		_, stderr, err := a.cli.Exec(ctx, "merge", "--no-commit", "--no-ff", source)
		if err != nil && strings.Contains(stderr, "CONFLICT") {
			// Parse "CONFLICT (content): Merge conflict in <path>"
			for _, line := range strings.Split(stderr, "\n") {
				if strings.Contains(line, "Merge conflict in") {
					idx := strings.Index(line, "Merge conflict in")
					if idx >= 0 {
						rest := strings.TrimSpace(line[idx+len("Merge conflict in"):])
						rest = strings.Trim(rest, " \t\n")
						if rest != "" {
							conflicts = append(conflicts, rest)
						}
					}
				}
			}
			// Abort the merge since we only wanted to check
			_, _, _ = a.cli.Exec(ctx, "merge", "--abort")
		}
	}
	return conflicts
}

// ConfirmResolution checks if all conflict markers are resolved in the given file.
func (a *ConflictAnalyzer) ConfirmResolution(ctx context.Context, file string) bool {
	stdout, stderr, err := a.cli.Exec(ctx, "diff", "--check", "--", file)
	// git diff --check exits non-zero and prints to stderr when markers remain
	if err != nil {
		return false
	}
	combined := stdout + stderr
	return !strings.Contains(combined, "leftover conflict marker") && !strings.Contains(combined, file+":")
}
