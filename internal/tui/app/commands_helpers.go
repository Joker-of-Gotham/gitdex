package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func parseOctalFileMode(s string) (os.FileMode, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty mode")
	}
	if len(s) == 3 && s[0] >= '0' && s[0] <= '7' {
		s = "0" + s
	}
	u64, err := strconv.ParseUint(s, 8, 32)
	if err != nil {
		return 0, err
	}
	return os.FileMode(u64), nil
}

func repoArchiveBasename(m *Model) string {
	if m.activeRepo != nil && strings.TrimSpace(m.activeRepo.Name) != "" {
		return strings.ReplaceAll(strings.TrimSpace(m.activeRepo.Name), "/", "-")
	}
	root := m.repoRoot()
	if root != "" {
		return filepath.Base(root)
	}
	return "repo"
}

func resolveOutputUnderRepo(root, outArg string) (string, error) {
	outArg = strings.TrimSpace(outArg)
	if outArg == "" {
		return "", fmt.Errorf("output path is required")
	}
	if filepath.IsAbs(outArg) {
		absRoot, err := filepath.Abs(root)
		if err != nil {
			return "", err
		}
		target := filepath.Clean(outArg)
		rel, err := filepath.Rel(absRoot, target)
		if err != nil {
			return "", err
		}
		if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			return "", fmt.Errorf("output escapes repository root")
		}
		return target, nil
	}
	return ensureRepoPath(root, filepath.ToSlash(outArg))
}

func formatPatchCmdErr(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
