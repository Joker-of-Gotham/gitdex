package executor

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CmdObj is a Builder-pattern command descriptor inspired by lazygit's
// oscommands.CmdObj. It separates command construction from execution,
// making commands testable, loggable, and cross-platform safe.
type CmdObj struct {
	binary   string
	args     []string
	workDir  string
	env      []string
	dontLog  bool
	stream   bool
	tempFile string
	platform *Platform
}

func newCmdObj(p *Platform, binary string, args ...string) *CmdObj {
	return &CmdObj{
		binary:   binary,
		args:     args,
		platform: p,
	}
}

// NewGitCmd creates a CmdObj for a git command.
func NewGitCmd(p *Platform, args ...string) *CmdObj {
	return newCmdObj(p, p.GitBin, args...)
}

// NewGhCmd creates a CmdObj for a GitHub CLI command.
func NewGhCmd(p *Platform, args ...string) *CmdObj {
	return newCmdObj(p, p.GhBin, args...)
}

// NewShellCmd creates a CmdObj for an arbitrary binary.
func NewShellCmd(p *Platform, binary string, args ...string) *CmdObj {
	return newCmdObj(p, binary, args...)
}

// SetWd sets the working directory.
func (c *CmdObj) SetWd(dir string) *CmdObj {
	c.workDir = dir
	return c
}

// AddEnv appends an environment variable (KEY=VALUE).
func (c *CmdObj) AddEnv(key, value string) *CmdObj {
	c.env = append(c.env, key+"="+value)
	return c
}

// DontLog marks this command as sensitive (suppresses logging).
func (c *CmdObj) DontLog() *CmdObj {
	c.dontLog = true
	return c
}

// Stream enables streaming stdout (for long-running commands).
func (c *CmdObj) Stream() *CmdObj {
	c.stream = true
	return c
}

// WithMessageFile writes content to a temporary file and appends
// the appropriate flag (e.g., -F <path>) to the argument list.
// The temp file is auto-cleaned after Run().
func (c *CmdObj) WithMessageFile(content string, flag string) *CmdObj {
	if flag == "" {
		flag = "-F"
	}
	tmpDir := os.TempDir()
	if c.workDir != "" {
		gitdexTmp := filepath.Join(c.workDir, ".gitdex", "tmp")
		if err := os.MkdirAll(gitdexTmp, 0o755); err == nil {
			tmpDir = gitdexTmp
		}
	}

	cleaned := StripTrailingWhitespace(content)
	f, err := os.CreateTemp(tmpDir, "gitdex-msg-*.txt")
	if err != nil {
		return c
	}
	f.WriteString(cleaned)
	f.Close()

	c.tempFile = f.Name()
	c.args = append(c.args, flag, f.Name())
	return c
}

// String returns the full command string for logging.
func (c *CmdObj) String() string {
	parts := make([]string, 0, 1+len(c.args))
	parts = append(parts, c.binary)
	parts = append(parts, c.args...)
	return strings.Join(parts, " ")
}

// Run executes the command and returns stdout, stderr, and error.
func (c *CmdObj) Run(ctx context.Context) (stdout string, stderr string, err error) {
	defer c.cleanup()

	cmd := exec.CommandContext(ctx, c.binary, c.args...)
	if c.workDir != "" {
		cmd.Dir = c.workDir
	}
	if len(c.env) > 0 {
		cmd.Env = append(os.Environ(), c.env...)
	}

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err = cmd.Run()
	return outBuf.String(), errBuf.String(), err
}

// CombinedRun executes and returns combined output.
func (c *CmdObj) CombinedRun(ctx context.Context) (string, error) {
	defer c.cleanup()

	cmd := exec.CommandContext(ctx, c.binary, c.args...)
	if c.workDir != "" {
		cmd.Dir = c.workDir
	}
	if len(c.env) > 0 {
		cmd.Env = append(os.Environ(), c.env...)
	}

	out, err := cmd.CombinedOutput()
	return string(out), err
}

func (c *CmdObj) cleanup() {
	if c.tempFile != "" {
		os.Remove(c.tempFile)
		c.tempFile = ""
	}
}

// StripTrailingWhitespace removes trailing whitespace from each line
// and normalizes line endings. Preserves a single trailing newline
// (POSIX text file convention required by many git hooks).
func StripTrailingWhitespace(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t\v\f\u00A0")
	}
	for len(lines) > 1 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	result := strings.Join(lines, "\n")
	if result != "" {
		result += "\n"
	}
	return result
}

// CmdObjRunner is the interface for executing CmdObj commands.
// Enables mock-based testing without real shell execution.
type CmdObjRunner interface {
	RunCmd(ctx context.Context, cmd *CmdObj) (string, string, error)
}

type defaultCmdObjRunner struct{}

func (d defaultCmdObjRunner) RunCmd(ctx context.Context, cmd *CmdObj) (string, string, error) {
	return cmd.Run(ctx)
}

// DefaultRunner returns the real CmdObj executor.
func DefaultRunner() CmdObjRunner {
	return defaultCmdObjRunner{}
}

// ShouldLog returns whether this command should be logged.
func (c *CmdObj) ShouldLog() bool {
	return !c.dontLog
}

// Binary returns the command binary name.
func (c *CmdObj) Binary() string {
	return c.binary
}

// Args returns a copy of the command arguments.
func (c *CmdObj) Args() []string {
	cp := make([]string, len(c.args))
	copy(cp, c.args)
	return cp
}

// WorkDir returns the command working directory.
func (c *CmdObj) WorkDir() string {
	return c.workDir
}

// IsGitCommit returns true if this is a git commit command.
func (c *CmdObj) IsGitCommit() bool {
	return len(c.args) > 0 && c.args[0] == "commit"
}

// NeedsTempFile checks whether the command has text content
// that should go through a temp file instead of inline args.
func NeedsTempFile(actionType, command string) bool {
	if actionType != "git_command" && actionType != "github_op" {
		return false
	}
	lower := strings.ToLower(command)

	gitPatterns := []string{
		"git commit -m ",
		"git commit -am ",
		"git tag -m ",
		"git tag -a ",
		"git notes add -m ",
		"git notes append -m ",
	}
	for _, p := range gitPatterns {
		if strings.HasPrefix(lower, p) {
			return true
		}
	}

	ghPatterns := []string{
		"gh issue create",
		"gh pr create",
		"gh release create",
		"gh issue comment",
		"gh pr comment",
		"gh issue edit",
		"gh pr edit",
		"gh release edit",
	}
	for _, p := range ghPatterns {
		if strings.HasPrefix(lower, p) {
			return strings.Contains(lower, "--body ") || strings.Contains(lower, "-b ")
		}
	}

	return false
}

// ConvertToTempFile takes a command string that uses -m or --body inline
// and returns a CmdObj that uses -F with a temp file instead.
func ConvertToTempFile(p *Platform, workDir, actionType, command string) *CmdObj {
	args := ParseCommand(command)
	if len(args) == 0 {
		return nil
	}

	binary := args[0]
	cmdArgs := args[1:]

	if strings.ToLower(filepath.Base(binary)) == "git" || binary == p.GitBin {
		return convertGitToTempFile(p, workDir, cmdArgs)
	}
	if strings.ToLower(filepath.Base(binary)) == "gh" || binary == p.GhBin {
		return convertGhToTempFile(p, workDir, cmdArgs)
	}
	return nil
}

func convertGitToTempFile(p *Platform, workDir string, args []string) *CmdObj {
	var message string
	var cleanArgs []string

	for i := 0; i < len(args); i++ {
		if args[i] == "-m" && i+1 < len(args) {
			message = args[i+1]
			i++
			continue
		}
		if strings.HasPrefix(args[i], "-m") && len(args[i]) > 2 {
			message = args[i][2:]
			continue
		}
		if args[i] == "-am" && i+1 < len(args) {
			message = args[i+1]
			cleanArgs = append(cleanArgs, "-a")
			i++
			continue
		}
		cleanArgs = append(cleanArgs, args[i])
	}

	if message == "" {
		cmd := NewGitCmd(p, args...)
		cmd.SetWd(workDir)
		return cmd
	}

	cmd := NewGitCmd(p, cleanArgs...)
	cmd.SetWd(workDir)
	cmd.WithMessageFile(message, "-F")
	return cmd
}

func convertGhToTempFile(p *Platform, workDir string, args []string) *CmdObj {
	var body string
	var cleanArgs []string

	for i := 0; i < len(args); i++ {
		if (args[i] == "--body" || args[i] == "-b") && i+1 < len(args) {
			body = args[i+1]
			i++
			continue
		}
		if strings.HasPrefix(args[i], "--body=") {
			body = strings.TrimPrefix(args[i], "--body=")
			continue
		}
		cleanArgs = append(cleanArgs, args[i])
	}

	if body == "" {
		cmd := NewGhCmd(p, args...)
		cmd.SetWd(workDir)
		return cmd
	}

	cmd := NewGhCmd(p, cleanArgs...)
	cmd.SetWd(workDir)
	cmd.WithMessageFile(body, "-F")
	return cmd
}

// ParseCommand splits a command string into arguments, respecting quotes.
func ParseCommand(cmd string) []string {
	var args []string
	var current strings.Builder
	inSingle := false
	inDouble := false
	runes := []rune(cmd)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if r == '\\' {
			if inDouble && i+1 < len(runes) {
				next := runes[i+1]
				if next == '"' || next == '\\' {
					current.WriteRune(next)
					i++
					continue
				}
			}
			current.WriteRune(r)
			continue
		}
		if r == '\'' && !inDouble {
			inSingle = !inSingle
			continue
		}
		if r == '"' && !inSingle {
			inDouble = !inDouble
			continue
		}
		if (r == ' ' || r == '\t') && !inSingle && !inDouble {
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
			continue
		}
		current.WriteRune(r)
	}
	if current.Len() > 0 {
		args = append(args, current.String())
	}
	return args
}

// Describe returns a human-readable description for logging.
func (c *CmdObj) Describe() string {
	if c.dontLog {
		return fmt.Sprintf("[%s] <redacted>", c.binary)
	}
	return fmt.Sprintf("[%s] %s", c.binary, strings.Join(c.args, " "))
}
