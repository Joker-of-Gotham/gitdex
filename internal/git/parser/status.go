package parser

import (
	"regexp"
	"strings"

	"github.com/Joker-of-Gotham/gitdex/internal/git"
)

// submoduleStatusRe matches lines like:
// [-+]? <sha1> <path> (<describe>)
// e.g. " 51ebf55... foo (v2.25.0)" or "+cb5918a... bar (v2.15.0)"
var submoduleStatusRe = regexp.MustCompile(`^([-+U]?)\s*([0-9a-fA-F]{40})\s+(.+?)\s+\((.+)\)\s*$`)

// ParseSubmodules parses the output of `git submodule status` into SubmoduleInfo slice.
func ParseSubmodules(output string) []git.SubmoduleInfo {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var result []git.SubmoduleInfo
	for _, line := range lines {
		line = strings.TrimSuffix(strings.TrimSpace(line), "\r")
		if line == "" {
			continue
		}
		info := parseSubmoduleLine(line)
		if info.Path != "" || info.Commit != "" {
			result = append(result, info)
		}
	}
	return result
}

func parseSubmoduleLine(line string) git.SubmoduleInfo {
	// Try regex first for "sha path (describe)" format
	matches := submoduleStatusRe.FindStringSubmatch(line)
	if len(matches) >= 5 {
		prefix := matches[1]
		commit := matches[2]
		path := strings.TrimSpace(matches[3])
		_ = strings.TrimSpace(matches[4]) // describe output, not stored in SubmoduleInfo

		status := "ok"
		switch prefix {
		case "-":
			status = "not initialized"
		case "+":
			status = "different commit"
		case "U":
			status = "merge conflict"
		}

		// Extract name (last path component)
		name := path
		if idx := strings.LastIndex(path, "/"); idx >= 0 {
			name = path[idx+1:]
		}

		return git.SubmoduleInfo{
			Name:   name,
			Path:   path,
			Commit: commit,
			Status: status,
			URL:    "", // git submodule status doesn't include URL
		}
	}

	// Fallback: minimal format "prefix? sha path" or "sha path"
	parts := strings.Fields(line)
	if len(parts) >= 2 {
		var commit, path string
		status := "ok"
		if len(parts[0]) == 1 && (parts[0] == "-" || parts[0] == "+" || parts[0] == "U") {
			switch parts[0] {
			case "-":
				status = "not initialized"
			case "+":
				status = "different commit"
			case "U":
				status = "merge conflict"
			}
			if len(parts) >= 3 {
				commit = parts[1]
				path = parts[2]
			}
		} else if len(parts[0]) == 40 && len(parts) >= 2 {
			commit = parts[0]
			path = parts[1]
		}
		if path != "" {
			name := path
			if idx := strings.LastIndex(path, "/"); idx >= 0 {
				name = path[idx+1:]
			}
			return git.SubmoduleInfo{Name: name, Path: path, Commit: commit, Status: status}
		}
	}
	return git.SubmoduleInfo{}
}
