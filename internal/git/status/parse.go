package status

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Joker-of-Gotham/gitdex/internal/git"
)

// ParseStatusV2 parses the output of `git status --porcelain=v2 --branch`.
// See: https://git-scm.com/docs/git-status#_porcelain_format_version_2
func ParseStatusV2(output string) (*GitState, error) {
	state := &GitState{
		WorkingTree: []git.FileStatus{},
		StagingArea: []git.FileStatus{},
		LocalBranch: git.BranchInfo{},
		RemoteState: git.RemoteInfo{},
		StashStack:  []git.StashEntry{},
		Submodules:  []git.SubmoduleInfo{},
		RepoConfig:  git.RepoConfig{},
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSuffix(line, "\r")
		if line == "" {
			continue
		}

		// Branch headers
		if strings.HasPrefix(line, "# branch.") {
			parseBranchHeader(line, state)
			continue
		}

		// Stash header (with --show-stash)
		if strings.HasPrefix(line, "# stash ") {
			// Stash count is handled separately via GetStashCount
			continue
		}

		// Ordinary changed entry: 1 <XY> <sub> <mH> <mI> <mW> <hH> <hI> <path>
		if strings.HasPrefix(line, "1 ") && len(line) >= 4 {
			fs, err := parseOrdinaryEntry(line)
			if err != nil {
				return nil, err
			}
			addToState(state, fs)
			continue
		}

		// Renamed/copied entry: 2 <XY> <sub> ... <path><sep><origPath>
		if strings.HasPrefix(line, "2 ") && len(line) >= 4 {
			fs, err := parseRenamedEntry(line)
			if err != nil {
				return nil, err
			}
			addToState(state, fs)
			continue
		}

		// Unmerged entry: u <XY> <sub> ...
		if strings.HasPrefix(line, "u ") && len(line) >= 4 {
			fs, err := parseUnmergedEntry(line)
			if err != nil {
				return nil, err
			}
			addToState(state, fs)
			continue
		}

		// Untracked: ? <path>
		if strings.HasPrefix(line, "? ") {
			path := strings.TrimPrefix(line, "? ")
			path = unquotePath(path)
			fs := git.FileStatus{
				Path:         path,
				StagingCode:  git.StatusUntracked,
				WorktreeCode: git.StatusUntracked,
			}
			state.WorkingTree = append(state.WorkingTree, fs)
			continue
		}

		// Ignored: ! <path>
		if strings.HasPrefix(line, "! ") {
			path := strings.TrimPrefix(line, "! ")
			path = unquotePath(path)
			fs := git.FileStatus{
				Path:         path,
				StagingCode:  git.StatusIgnored,
				WorktreeCode: git.StatusIgnored,
			}
			state.WorkingTree = append(state.WorkingTree, fs)
			continue
		}
	}

	return state, nil
}

func parseBranchHeader(line string, state *GitState) {
	rest := strings.TrimPrefix(line, "# ")
	parts := strings.SplitN(rest, " ", 2)
	if len(parts) < 2 {
		return
	}
	key, val := parts[0], parts[1]

	switch key {
	case "branch.oid":
		if val != "(initial)" {
			state.HeadRef = val
		}
	case "branch.head":
		if val == "(detached)" {
			state.LocalBranch.IsDetached = true
		} else {
			state.LocalBranch.Name = val
		}
	case "branch.upstream":
		state.LocalBranch.Upstream = val
		if state.UpstreamState == nil {
			state.UpstreamState = &git.UpstreamInfo{}
		}
		state.UpstreamState.Name = val
	case "branch.ab":
		// Format: +3 -1 (ahead behind)
		ahead, behind := parseAheadBehind(val)
		state.LocalBranch.Ahead = ahead
		state.LocalBranch.Behind = behind
		if state.UpstreamState != nil {
			state.UpstreamState.Ahead = ahead
			state.UpstreamState.Behind = behind
		}
	}
}

func parseAheadBehind(s string) (ahead, behind int) {
	parts := strings.Fields(s)
	for _, p := range parts {
		if strings.HasPrefix(p, "+") {
			ahead, _ = strconv.Atoi(strings.TrimPrefix(p, "+"))
		} else if strings.HasPrefix(p, "-") {
			behind, _ = strconv.Atoi(strings.TrimPrefix(p, "-"))
		}
	}
	return ahead, behind
}

func parseOrdinaryEntry(line string) (*git.FileStatus, error) {
	// 1 <XY> <sub> <mH> <mI> <mW> <hH> <hI> <path>
	rest := strings.TrimPrefix(line, "1 ")
	parts := strings.SplitN(rest, " ", 8) // 7 metadata + path (path may contain spaces)
	if len(parts) < 8 {
		return nil, fmt.Errorf("invalid ordinary entry: %q", line)
	}
	xy := parts[0]
	path := parts[7]
	path = unquotePath(path)

	staging := charToStatusCode(xy[0])
	worktree := charToStatusCode(xy[1])

	return &git.FileStatus{
		Path:         path,
		StagingCode:  staging,
		WorktreeCode: worktree,
	}, nil
}

func parseRenamedEntry(line string) (*git.FileStatus, error) {
	// 2 <XY> <sub> <mH> <mI> <mW> <hH> <hI> <X><score> <path><sep><origPath>
	// sep is TAB without -z; path=target, origPath=source
	rest := strings.TrimPrefix(line, "2 ")
	parts := strings.SplitN(rest, " ", 9)
	if len(parts) < 9 {
		return nil, fmt.Errorf("invalid renamed entry: %q", line)
	}
	xy := parts[0]
	pathPart := parts[8]

	// pathPart is "path\torigPath" or "origPath\tpath" - spec says path then origPath, TAB sep
	var path, origPath string
	if idx := strings.Index(pathPart, "\t"); idx >= 0 {
		path = unquotePath(pathPart[idx+1:])   // target (after TAB)
		origPath = unquotePath(pathPart[:idx]) // source (before TAB)
	} else {
		path = unquotePath(pathPart)
	}

	staging := charToStatusCode(xy[0])
	worktree := charToStatusCode(xy[1])

	return &git.FileStatus{
		Path:         path,
		StagingCode:  staging,
		WorktreeCode: worktree,
		OrigPath:     origPath,
	}, nil
}

func parseUnmergedEntry(line string) (*git.FileStatus, error) {
	// u <XY> <sub> <m1> <m2> <m3> <mW> <h1> <h2> <h3> <path>
	rest := strings.TrimPrefix(line, "u ")
	parts := strings.SplitN(rest, " ", 10)
	if len(parts) < 10 {
		return nil, fmt.Errorf("invalid unmerged entry: %q", line)
	}
	xy := parts[0]
	path := parts[9]
	path = unquotePath(path)

	staging := charToStatusCode(xy[0])
	worktree := charToStatusCode(xy[1])

	return &git.FileStatus{
		Path:         path,
		StagingCode:  staging,
		WorktreeCode: worktree,
	}, nil
}

func charToStatusCode(c byte) git.FileStatusCode {
	switch c {
	case ' ', '.':
		return git.StatusUnmodified
	case 'M':
		return git.StatusModified
	case 'A':
		return git.StatusAdded
	case 'D':
		return git.StatusDeleted
	case 'R':
		return git.StatusRenamed
	case 'C':
		return git.StatusCopied
	case 'U':
		return git.StatusUnmerged
	case '?':
		return git.StatusUntracked
	case '!':
		return git.StatusIgnored
	case 'T':
		return git.StatusTypeChanged
	default:
		return git.FileStatusCode(c)
	}
}

func addToState(state *GitState, fs *git.FileStatus) {
	// Files with staging changes go to StagingArea (index)
	if fs.StagingCode != git.StatusUnmodified && fs.StagingCode != git.StatusUntracked && fs.StagingCode != git.StatusIgnored {
		state.StagingArea = append(state.StagingArea, *fs)
	}
	// Files with worktree changes (modified/deleted/untracked) go to WorkingTree
	if fs.WorktreeCode != git.StatusUnmodified {
		state.WorkingTree = append(state.WorkingTree, *fs)
	}
}

func unquotePath(p string) string {
	p = strings.TrimSpace(p)
	if len(p) >= 2 && p[0] == '"' && p[len(p)-1] == '"' {
		// C-style quoted string - basic unescape
		p = p[1 : len(p)-1]
		p = strings.ReplaceAll(p, "\\\"", "\"")
		p = strings.ReplaceAll(p, "\\n", "\n")
		p = strings.ReplaceAll(p, "\\t", "\t")
		p = strings.ReplaceAll(p, "\\\\", "\\")
	}
	return p
}
