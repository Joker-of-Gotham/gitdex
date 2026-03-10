package status

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Joker-of-Gotham/gitdex/internal/git/cli"
)

type FileInspection struct {
	ImportantFiles []string       `json:"important_files,omitempty"`
	RecentModified []string       `json:"recent_modified,omitempty"`
	DiffStats      []FileDiffStat `json:"diff_stats,omitempty"`
	StagedStats    []FileDiffStat `json:"staged_stats,omitempty"`
	TotalFiles     int            `json:"total_files"`
}

type FileDiffStat struct {
	Path       string `json:"path"`
	Insertions int    `json:"insertions"`
	Deletions  int    `json:"deletions"`
}

var importantFileNames = map[string]bool{
	".gitignore": true, ".gitattributes": true, ".editorconfig": true,
	"readme.md": true, "readme": true, "readme.txt": true,
	"license": true, "license.md": true, "license.txt": true,
	"makefile": true, "cmakelists.txt": true, "justfile": true,
	"package.json": true, "package-lock.json": true,
	"go.mod": true, "go.sum": true,
	"cargo.toml": true, "cargo.lock": true,
	"pyproject.toml": true, "requirements.txt": true, "setup.py": true,
	"pom.xml": true, "build.gradle": true, "build.gradle.kts": true,
	"tsconfig.json": true, "webpack.config.js": true, "vite.config.ts": true,
	"dockerfile": true, "docker-compose.yml": true, "docker-compose.yaml": true,
	".env.example": true, ".dockerignore": true,
}

func enrichFileInspection(ctx context.Context, gitCLI cli.GitCLI, state *GitState) {
	fi := &FileInspection{}

	stdout, _, err := gitCLI.Exec(ctx, "ls-files", "--cached")
	if err == nil {
		for _, line := range splitLines(stdout) {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			fi.TotalFiles++
			base := strings.ToLower(filepath.Base(line))
			if importantFileNames[base] && fileExistsOnDisk(line) {
				fi.ImportantFiles = append(fi.ImportantFiles, line)
			}
		}
	}

	if !state.IsInitial {
		stdout, _, err = gitCLI.Exec(ctx, "log", "--oneline", "--name-only", "--diff-filter=M", "-5", "--format=")
		if err == nil {
			seen := make(map[string]bool)
			for _, line := range splitLines(stdout) {
				line = strings.TrimSpace(line)
				if line != "" && !seen[line] && len(fi.RecentModified) < 10 && fileExistsOnDisk(line) {
					fi.RecentModified = append(fi.RecentModified, line)
					seen[line] = true
				}
			}
		}
	}

	// Diff stats for working tree
	stdout, _, err = gitCLI.Exec(ctx, "diff", "--numstat")
	if err == nil {
		fi.DiffStats = parseDiffNumstat(stdout)
	}

	// Diff stats for staging area
	stdout, _, err = gitCLI.Exec(ctx, "diff", "--staged", "--numstat")
	if err == nil {
		fi.StagedStats = parseDiffNumstat(stdout)
	}

	state.FileInspect = fi
}

func parseDiffNumstat(output string) []FileDiffStat {
	var stats []FileDiffStat
	for _, line := range splitLines(output) {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		ins, _ := strconv.Atoi(fields[0])
		del, _ := strconv.Atoi(fields[1])
		path := fields[2]
		if len(fields) > 3 {
			path = strings.Join(fields[2:], " ")
		}
		stats = append(stats, FileDiffStat{
			Path:       path,
			Insertions: ins,
			Deletions:  del,
		})
		if len(stats) >= 50 {
			break
		}
	}
	return stats
}

func fileExistsOnDisk(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (fi *FileInspection) Summary() string {
	if fi == nil {
		return ""
	}
	var parts []string
	parts = append(parts, fmt.Sprintf("tracked_files=%d", fi.TotalFiles))
	if len(fi.ImportantFiles) > 0 {
		parts = append(parts, fmt.Sprintf("important=%s", strings.Join(fi.ImportantFiles, ",")))
	}
	if len(fi.DiffStats) > 0 {
		totalIns, totalDel := 0, 0
		for _, s := range fi.DiffStats {
			totalIns += s.Insertions
			totalDel += s.Deletions
		}
		parts = append(parts, fmt.Sprintf("working_diff=+%d/-%d(%d files)", totalIns, totalDel, len(fi.DiffStats)))
	}
	if len(fi.StagedStats) > 0 {
		totalIns, totalDel := 0, 0
		for _, s := range fi.StagedStats {
			totalIns += s.Insertions
			totalDel += s.Deletions
		}
		parts = append(parts, fmt.Sprintf("staged_diff=+%d/-%d(%d files)", totalIns, totalDel, len(fi.StagedStats)))
	}
	return strings.Join(parts, " | ")
}
