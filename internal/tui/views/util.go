package views

import (
	"os/exec"
	"strings"

	"github.com/mattn/go-runewidth"
)

// runGit runs `git -C dir` with the given git subcommand arguments.
func runGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	b, err := cmd.CombinedOutput()
	return string(b), err
}

func trimGitOut(s string) string { return strings.TrimSpace(s) }

func truncate(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if runewidth.StringWidth(s) <= maxWidth {
		return s
	}
	return runewidth.Truncate(s, maxWidth, "...")
}
