package parser

import (
	"regexp"
	"strconv"
	"strings"
)

// DiffHunk represents a hunk in a unified diff.
type DiffHunk struct {
	OldStart int
	OldLines int
	NewStart int
	NewLines int
	Content  string
}

// FileDiff represents a file diff with hunks.
type FileDiff struct {
	Path   string
	Hunks  []DiffHunk
	Binary bool
}

var hunkRe = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)

// ParseDiff parses unified diff output into FileDiff slice.
func ParseDiff(output string) []FileDiff {
	if strings.TrimSpace(output) == "" {
		return nil
	}
	var files []FileDiff
	sections := splitDiffSections(output)
	for _, section := range sections {
		fd := parseFileDiff(section)
		if fd.Path != "" || len(fd.Hunks) > 0 {
			files = append(files, fd)
		}
	}
	return files
}

func splitDiffSections(output string) []string {
	var sections []string
	var current []string
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, "diff --git ") {
			if len(current) > 0 {
				sections = append(sections, strings.Join(current, "\n"))
			}
			current = []string{line}
		} else if len(current) > 0 {
			current = append(current, line)
		}
	}
	if len(current) > 0 {
		sections = append(sections, strings.Join(current, "\n"))
	}
	return sections
}

func parseFileDiff(section string) FileDiff {
	var fd FileDiff
	lines := strings.Split(section, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "diff --git ") {
			// diff --git a/path b/path
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				fd.Path = strings.TrimPrefix(parts[2], "a/")
			}
		}
		if strings.HasPrefix(line, "Binary files ") {
			fd.Binary = true
		}
		if m := hunkRe.FindStringSubmatch(line); len(m) >= 5 {
			oldStart, _ := strconv.Atoi(m[1])
			oldLines := 1
			if m[2] != "" {
				oldLines, _ = strconv.Atoi(m[2])
			}
			newStart, _ := strconv.Atoi(m[3])
			newLines := 1
			if m[4] != "" {
				newLines, _ = strconv.Atoi(m[4])
			}
			fd.Hunks = append(fd.Hunks, DiffHunk{
				OldStart: oldStart,
				OldLines: oldLines,
				NewStart: newStart,
				NewLines: newLines,
				Content:  line + "\n",
			})
		} else if len(fd.Hunks) > 0 && (strings.HasPrefix(line, " ") || strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-")) {
			fd.Hunks[len(fd.Hunks)-1].Content += line + "\n"
		}
	}
	return fd
}
