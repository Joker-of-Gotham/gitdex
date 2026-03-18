package executor

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Joker-of-Gotham/gitdex/internal/config"
	"github.com/Joker-of-Gotham/gitdex/internal/contract"
	"github.com/Joker-of-Gotham/gitdex/internal/dotgitdex"
	"github.com/Joker-of-Gotham/gitdex/internal/git/cli"
	"github.com/Joker-of-Gotham/gitdex/internal/observability"
	"github.com/Joker-of-Gotham/gitdex/internal/planner"
)

// ExecutionResult is the runtime result contract shared across layers.
type ExecutionResult = contract.ActionResult

// validActionTypes lists the 5 tool types accepted by the executor.
// Aligned with promptv2.Tools definitions.
var validActionTypes = map[string]bool{
	"git_command":   true,
	"shell_command": true,
	"file_write":    true,
	"file_read":     true,
	"github_op":     true,
}

// Runner executes suggestion items.
type Runner struct {
	gitCLI   cli.GitCLI
	store    *dotgitdex.Manager
	repoRoot string
	logger   *ExecutionLogger
	cmdExec  CommandExecutor
	platform *Platform

	gitAdapter    GitAdapter
	githubAdapter GitHubAdapter
	fileAdapter   FileAdapter

	idempotencyMu   sync.Mutex
	executedActions map[string]bool

	roundMu         sync.Mutex
	recentFileReads map[string]bool
	roundSignatures map[string]bool
}

var (
	secretLiteralPattern = regexp.MustCompile(`(?i)\b(?:gh[pousr]_[a-z0-9_]{20,}|github_pat_[a-z0-9_]{20,}|sk-[a-z0-9_\-]{20,}|dsk-[a-z0-9_\-]{20,}|xox[baprs]-[a-z0-9\-]{20,})\b`)
	secretAssignPattern  = regexp.MustCompile(`(?i)\b([A-Z][A-Z0-9_]*(?:TOKEN|SECRET|API_KEY|PASSWORD|PASSWD))\s*=\s*("[^"]*"|'[^']*'|[^\s]+)`)
	authHeaderPattern    = regexp.MustCompile(`(?i)(authorization:\s*bearer\s+)([^\s]+)`)
	secretQueryPattern   = regexp.MustCompile(`(?i)([?&](?:access_token|token|api_key)=)([^&\s]+)`)
)

// NewRunner creates a Runner.
func NewRunner(gitCLI cli.GitCLI, store *dotgitdex.Manager, logger *ExecutionLogger) *Runner {
	root := ""
	if store != nil {
		root = store.RepoRoot
	}
	r := &Runner{
		gitCLI:          gitCLI,
		store:           store,
		repoRoot:        root,
		logger:          logger,
		cmdExec:         osCommandExecutor{},
		platform:        DetectPlatform(),
		executedActions: make(map[string]bool),
		recentFileReads: make(map[string]bool),
		roundSignatures: make(map[string]bool),
	}
	r.gitAdapter = runnerGitAdapter{runner: r}
	r.githubAdapter = runnerGitHubAdapter{runner: r}
	r.fileAdapter = runnerFileAdapter{runner: r}
	return r
}

// ResetRound clears per-round deduplication state. Call at the start of each
// LLM planning round.
func (r *Runner) ResetRound() {
	r.roundMu.Lock()
	defer r.roundMu.Unlock()
	r.recentFileReads = make(map[string]bool)
	r.roundSignatures = make(map[string]bool)
}

// ExecuteSuggestion dispatches based on ActionSpec type.
func (r *Runner) ExecuteSuggestion(ctx context.Context, seqID int, item planner.SuggestionItem) *ExecutionResult {
	trace, ok := observability.TraceFromContext(ctx)
	if !ok {
		trace = contract.TraceMetadata{TraceID: observability.NewTraceID()}
	}

	if !validActionTypes[item.Action.Type] {
		result := &ExecutionResult{
			Stderr:  fmt.Sprintf("invalid action type %q — valid types: git_command, shell_command, file_write, file_read, github_op", item.Action.Type),
			Success: false,
			Trace:   trace,
		}
		observability.RecordCommand(false)
		if r.logger != nil {
			r.logger.LogStep(seqID, item, result)
		}
		return result
	}

	if msg := r.preflightAction(item.Action); msg != "" {
		result := &ExecutionResult{Stderr: msg, Success: false, Trace: trace}
		observability.RecordCommand(false)
		if r.logger != nil {
			r.logger.LogStep(seqID, item, result)
		}
		return result
	}

	if reason := r.checkGitdexProtection(item); reason != "" {
		result := &ExecutionResult{
			Stderr:  reason,
			Success: false,
			Trace:   trace,
		}
		observability.RecordCommand(false)
		if r.logger != nil {
			r.logger.LogStep(seqID, item, result)
		}
		return result
	}

	sig := r.actionSignature(item)
	if r.wasRoundDuplicate(sig) {
		result := &ExecutionResult{
			Command: item.Action.Command,
			Stdout:  "action signature duplicate in this round, skipping",
			Success: true,
			Trace:   trace,
		}
		result.RecoverBy = contract.RecoveryAction{Type: contract.RecoverySkip, Reason: "round-level signature dedup"}
		observability.RecordCommand(true)
		if r.logger != nil {
			r.logger.LogStep(seqID, item, result)
		}
		return result
	}

	if item.Action.Type == "file_read" {
		if r.wasFileRecentlyRead(item.Action.FilePath) {
			result := &ExecutionResult{
				FilePath: item.Action.FilePath,
				Stdout:   "file already read this round, skipping consecutive read",
				Success:  true,
				Trace:    trace,
			}
			result.RecoverBy = contract.RecoveryAction{Type: contract.RecoverySkip, Reason: "consecutive file_read blocked"}
			observability.RecordCommand(true)
			if r.logger != nil {
				r.logger.LogStep(seqID, item, result)
			}
			return result
		}
	}

	start := time.Now()
	var result *ExecutionResult
	key := r.idempotencyKey(item)
	if key != "" && r.wasActionExecuted(key) {
		result = &ExecutionResult{
			Command: item.Action.Command,
			Stdout:  "idempotency preflight: action already executed, skipping duplicate",
			Success: true,
			Trace:   trace,
		}
		result.RecoverBy = contract.RecoveryAction{Type: contract.RecoverySkip, Reason: "duplicate action key"}
		result.Duration = time.Since(start)
		observability.RecordCommand(true)
		if r.logger != nil {
			r.logger.LogStep(seqID, item, result)
		}
		return result
	}

	switch item.Action.Type {
	case "git_command":
		result = r.gitAdapter.ExecGit(ctx, item.Action.Command)
	case "shell_command":
		result = r.execShellCommand(ctx, item.Action.Command)
	case "file_write":
		result = r.fileAdapter.Write(item.Action)
	case "file_read":
		result = r.fileAdapter.Read(item.Action)
	case "github_op":
		result = r.githubAdapter.ExecGitHub(ctx, item.Action.Command)
	default:
		result = &ExecutionResult{
			Stderr:  fmt.Sprintf("unsupported action type: %s", item.Action.Type),
			Success: false,
		}
	}

	result.Duration = time.Since(start)
	result.Trace = trace
	redactActionResult(result)
	result.RecoverBy = r.recommendRecovery(result)
	observability.RecordCommand(result.Success)
	if result.Success && key != "" {
		r.markActionExecuted(key)
	}
	r.markRoundSignature(sig)
	if item.Action.Type == "file_read" && result.Success {
		r.markFileRead(item.Action.FilePath)
	}
	if r.logger != nil {
		r.logger.LogStep(seqID, item, result)
	}
	return result
}

func (r *Runner) preflightAction(action planner.ActionSpec) string {
	if err := contract.ValidateAction(action); err != nil {
		return "preflight validation failed: " + err.Error()
	}
	return ""
}

func (r *Runner) idempotencyKey(item planner.SuggestionItem) string {
	typ := item.Action.Type
	switch typ {
	case "file_write":
		seed := strings.Join([]string{
			typ,
			item.Action.FileOp,
			item.Action.FilePath,
			item.Action.FileContent,
		}, "|")
		return hashKey(seed)
	case "file_read":
		return hashKey("file_read|" + item.Action.FilePath)
	case "github_op":
		seed := typ + "|" + strings.TrimSpace(item.Action.Command)
		return hashKey(seed)
	default:
		return ""
	}
}

func (r *Runner) actionSignature(item planner.SuggestionItem) string {
	seed := strings.Join([]string{
		item.Action.Type,
		strings.TrimSpace(item.Action.Command),
		strings.TrimSpace(item.Action.FilePath),
		strings.TrimSpace(item.Action.FileOp),
	}, "|")
	return hashKey(seed)
}

func (r *Runner) wasRoundDuplicate(sig string) bool {
	r.roundMu.Lock()
	defer r.roundMu.Unlock()
	return r.roundSignatures[sig]
}

func (r *Runner) markRoundSignature(sig string) {
	r.roundMu.Lock()
	defer r.roundMu.Unlock()
	r.roundSignatures[sig] = true
}

func (r *Runner) wasFileRecentlyRead(path string) bool {
	r.roundMu.Lock()
	defer r.roundMu.Unlock()
	return r.recentFileReads[path]
}

func (r *Runner) markFileRead(path string) {
	r.roundMu.Lock()
	defer r.roundMu.Unlock()
	r.recentFileReads[path] = true
}

func (r *Runner) wasActionExecuted(key string) bool {
	r.idempotencyMu.Lock()
	defer r.idempotencyMu.Unlock()
	return r.executedActions[key]
}

func (r *Runner) markActionExecuted(key string) {
	r.idempotencyMu.Lock()
	defer r.idempotencyMu.Unlock()
	r.executedActions[key] = true
}

func hashKey(s string) string {
	sum := sha1.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}

func redactActionResult(result *ExecutionResult) {
	if result == nil {
		return
	}
	result.Command = redactSensitiveText(result.Command)
	result.Stdout = redactSensitiveText(result.Stdout)
	result.Stderr = redactSensitiveText(result.Stderr)
}

func redactSensitiveText(s string) string {
	if strings.TrimSpace(s) == "" {
		return s
	}
	out := secretLiteralPattern.ReplaceAllString(s, "[REDACTED]")
	out = secretAssignPattern.ReplaceAllString(out, "$1=[REDACTED]")
	out = authHeaderPattern.ReplaceAllString(out, "${1}[REDACTED]")
	out = secretQueryPattern.ReplaceAllString(out, "${1}[REDACTED]")
	return out
}

func (r *Runner) recommendRecovery(result *ExecutionResult) contract.RecoveryAction {
	if result == nil {
		return contract.RecoveryAction{Type: contract.RecoveryAbort, Reason: "nil result"}
	}
	if result.Success {
		return contract.RecoveryAction{Type: contract.RecoverySkip, Reason: "already successful"}
	}
	low := strings.ToLower(result.Stderr)
	switch {
	case strings.Contains(low, "already exists"),
		strings.Contains(low, "nothing to commit"),
		strings.Contains(low, "duplicate"):
		return contract.RecoveryAction{Type: contract.RecoverySkip, Reason: "non-fatal duplicate state"}
	case strings.Contains(low, "timeout"),
		strings.Contains(low, "connection"),
		strings.Contains(low, "temporarily"),
		strings.Contains(low, "http 500"),
		strings.Contains(low, "http 502"),
		strings.Contains(low, "http 503"):
		return contract.RecoveryAction{Type: contract.RecoveryRetry, Reason: "transient network/server failure"}
	case strings.Contains(low, "permission denied"),
		strings.Contains(low, "http 401"),
		strings.Contains(low, "http 403"),
		strings.Contains(low, "forbidden"),
		strings.Contains(low, "unauthorized"):
		return contract.RecoveryAction{Type: contract.RecoveryManual, Reason: "authorization or permission failure"}
	default:
		return contract.RecoveryAction{Type: contract.RecoveryAbort, Reason: "non-recoverable failure"}
	}
}

// checkGitdexProtection rejects commands that would modify or delete .gitdex/.
func (r *Runner) checkGitdexProtection(item planner.SuggestionItem) string {
	switch item.Action.Type {
	case "git_command", "shell_command", "github_op":
		if containsGitdex(item.Action.Command) {
			return "BLOCKED: command targets .gitdex/ directory which is protected system state"
		}
	case "file_write", "file_read":
		if containsGitdex(item.Action.FilePath) {
			return "BLOCKED: file operation targets .gitdex/ directory which is protected system state"
		}
	}
	return ""
}

func containsGitdex(s string) bool {
	lower := strings.ToLower(s)
	return strings.Contains(lower, ".gitdex/") || strings.Contains(lower, ".gitdex\\") ||
		lower == ".gitdex" || strings.HasSuffix(lower, "/.gitdex") || strings.HasSuffix(lower, "\\.gitdex")
}

func (r *Runner) execGitCommand(ctx context.Context, cmdStr string) *ExecutionResult {
	if cmdStr == "" {
		return &ExecutionResult{Stderr: "empty command", Success: false}
	}
	if reason := rejectShellOperators(cmdStr); reason != "" {
		return &ExecutionResult{Command: cmdStr, Stderr: reason, Success: false}
	}
	args := ParseCommand(cmdStr)
	if len(args) == 0 {
		return &ExecutionResult{Command: cmdStr, Stderr: "empty command after parsing", Success: false}
	}
	gitArgs := args
	if gitArgs[0] == "git" {
		gitArgs = gitArgs[1:]
	}
	if len(gitArgs) == 0 {
		return &ExecutionResult{Command: cmdStr, Stderr: "no git subcommand", Success: false}
	}

	// Auto-fix trailing whitespace before commit to prevent pre-commit hook failures.
	if isGitCommitCommand(gitArgs) {
		r.autoFixStagedWhitespace(ctx)
	}

	// Also clean files being staged via `git add` to prevent later commit failures.
	if isGitAddCommand(gitArgs) {
		r.autoFixFilesBeingAdded(ctx, gitArgs)
	}

	// Convert -m/-am to -F <tempfile> for commit/tag/notes to avoid whitespace issues.
	if NeedsTempFile("git_command", cmdStr) {
		cmdObj := ConvertToTempFile(r.platform, r.repoRoot, "git_command", cmdStr)
		if cmdObj != nil {
			stdout, stderr, err := cmdObj.Run(ctx)
			result := &ExecutionResult{
				Command: cmdStr,
				Stdout:  stdout,
				Stderr:  stderr,
				Success: err == nil,
			}
			if err != nil {
				result.ExitCode = 1
				result.Stderr = classifyGitError(stderr, cmdStr)
			}
			return result
		}
	}

	stdout, stderr, err := r.gitCLI.Exec(ctx, gitArgs...)
	result := &ExecutionResult{
		Command: cmdStr,
		Stdout:  stdout,
		Stderr:  stderr,
		Success: err == nil,
	}
	if err != nil {
		result.ExitCode = 1
		result.Stderr = classifyGitError(stderr, cmdStr)
	}
	return result
}

func isGitCommitCommand(gitArgs []string) bool {
	if len(gitArgs) == 0 {
		return false
	}
	return gitArgs[0] == "commit"
}

func isGitAddCommand(gitArgs []string) bool {
	if len(gitArgs) == 0 {
		return false
	}
	return gitArgs[0] == "add"
}

// autoFixFilesBeingAdded cleans trailing whitespace from files listed in a
// `git add` command before they are staged.
func (r *Runner) autoFixFilesBeingAdded(ctx context.Context, gitArgs []string) {
	if r.repoRoot == "" || len(gitArgs) < 2 {
		return
	}
	for _, arg := range gitArgs[1:] {
		if strings.HasPrefix(arg, "-") {
			continue
		}
		if arg == "." {
			// `git add .` — clean all modified tracked files.
			r.autoFixStagedWhitespace(ctx)
			return
		}
		if isBinaryFileName(arg) {
			continue
		}
		fullPath := filepath.Join(r.repoRoot, arg)
		data, err := os.ReadFile(fullPath)
		if err != nil {
			continue
		}
		original := string(data)
		cleaned := StripTrailingWhitespace(original)
		if cleaned != original {
			os.WriteFile(fullPath, []byte(cleaned), 0o644)
		}
	}
}

// autoFixStagedWhitespace strips trailing whitespace from all staged and
// modified text files before a git commit, preventing pre-commit hook
// rejections. It combines both staged (--cached) and unstaged (working tree)
// file lists to handle `git commit -a` which auto-stages modified tracked files.
func (r *Runner) autoFixStagedWhitespace(ctx context.Context) {
	if r.repoRoot == "" {
		return
	}

	fileSet := make(map[string]bool)

	// Staged files.
	if out, _, err := r.gitCLI.Exec(ctx, "diff", "--cached", "--name-only", "--diff-filter=ACMR"); err == nil {
		for _, name := range strings.Split(strings.TrimSpace(out), "\n") {
			name = strings.TrimSpace(name)
			if name != "" {
				fileSet[name] = true
			}
		}
	}

	// Unstaged modified tracked files (covers `git commit -a`).
	if out, _, err := r.gitCLI.Exec(ctx, "diff", "--name-only", "--diff-filter=ACMR"); err == nil {
		for _, name := range strings.Split(strings.TrimSpace(out), "\n") {
			name = strings.TrimSpace(name)
			if name != "" {
				fileSet[name] = true
			}
		}
	}

	if len(fileSet) == 0 {
		return
	}

	var fixedFiles []string
	for name := range fileSet {
		if isBinaryFileName(name) {
			continue
		}
		fullPath := filepath.Join(r.repoRoot, name)
		data, err := os.ReadFile(fullPath)
		if err != nil {
			continue
		}
		original := string(data)
		cleaned := StripTrailingWhitespace(original)
		if cleaned != original {
			if err := os.WriteFile(fullPath, []byte(cleaned), 0o644); err == nil {
				fixedFiles = append(fixedFiles, name)
			}
		}
	}

	if len(fixedFiles) > 0 {
		args := append([]string{"add"}, fixedFiles...)
		r.gitCLI.Exec(ctx, args...)
	}
}

func isBinaryFileName(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	binaryExts := map[string]bool{
		".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".bmp": true,
		".ico": true, ".svg": true, ".webp": true,
		".zip": true, ".gz": true, ".tar": true, ".7z": true, ".rar": true,
		".exe": true, ".dll": true, ".so": true, ".dylib": true,
		".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true,
		".woff": true, ".woff2": true, ".ttf": true, ".eot": true, ".otf": true,
		".mp3": true, ".mp4": true, ".avi": true, ".mov": true,
		".bin": true, ".dat": true, ".db": true, ".sqlite": true,
	}
	return binaryExts[ext]
}

// classifyGitError turns raw git stderr into actionable LLM-friendly feedback.
func classifyGitError(stderr string, cmdStr string) string {
	low := strings.ToLower(stderr)
	var hint string
	switch {
	case strings.Contains(low, "trailing whitespace"):
		hint = "Pre-commit hook rejected: trailing whitespace detected. Use file_read to read the file, then file_write with 'update' to rewrite it without trailing whitespace. Do NOT use sed/awk/perl."
	case strings.Contains(low, "nothing to commit"):
		hint = "Nothing to commit. All changes are already committed or the working tree is clean. Skip this step."
	case strings.Contains(low, "conflict") && (strings.Contains(low, "merge") || strings.Contains(low, "rebase")):
		hint = "Merge/rebase conflict detected. Use file_read to read conflicted files, then file_write to resolve conflicts manually, then 'git add' to mark resolved."
	case strings.Contains(low, "not a git repository"):
		hint = "Not in a git repository. Check the working directory."
	case strings.Contains(low, "permission denied") || strings.Contains(low, "access denied"):
		hint = "Permission denied. Check file permissions or authentication."
	case strings.Contains(low, "could not read from remote") || strings.Contains(low, "connection"):
		hint = "Network error: cannot reach remote repository. Check SSH keys, network, or try HTTPS."
	case strings.Contains(low, "already exists"):
		hint = "Branch or tag already exists. Skip creation or use a different name."
	case strings.Contains(low, "diverged") || strings.Contains(low, "non-fast-forward"):
		hint = "Branches have diverged. Try 'git pull --rebase' or 'git fetch' before pushing."
	case strings.Contains(low, "detached head"):
		hint = "In detached HEAD state. Create a branch with 'git checkout -b <name>' to preserve work."
	case strings.Contains(low, "pathspec") && strings.Contains(low, "did not match"):
		if strings.Contains(strings.ToLower(cmdStr), "git commit -m ") {
			hint = "File path not found. If this came from 'git commit -m', quote the full commit message (example: git commit -m \"Fix trailing whitespace\")."
		} else {
			hint = "File path not found. Verify the file exists in the repository."
		}
	case strings.Contains(low, "failed to push") || strings.Contains(low, "rejected"):
		hint = "Push rejected. The remote has changes not present locally. Try 'git pull --rebase origin <branch>' first."
	case strings.Contains(low, "no changes added to commit"):
		hint = "No changes staged. Use 'git add <file>' first, or use 'git commit -a' to auto-stage tracked files."
	}
	if hint != "" {
		return stderr + "\n\n[GITDEX DIAGNOSIS] " + hint
	}
	return stderr
}

// rejectShellOperators scans cmdStr for unquoted shell operators
// (&, &&, ||, |, ;) and returns an error message if found.
// Returns "" if the command is safe (no operators outside quotes).
func rejectShellOperators(cmdStr string) string {
	inSingle := false
	inDouble := false
	escaped := false
	runes := []rune(cmdStr)
	n := len(runes)

	for i := 0; i < n; i++ {
		r := runes[i]
		if escaped {
			escaped = false
			continue
		}
		if r == '\\' && !inSingle {
			escaped = true
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
		if inSingle || inDouble {
			continue
		}
		switch r {
		case ';':
			return "shell operator ';' detected. Commands are NOT run through a shell. Each suggestion must contain exactly ONE command. Split into separate suggestions."
		case '|':
			if i+1 < n && runes[i+1] == '|' {
				return "shell operator '||' detected. Commands are NOT run through a shell. Each suggestion must contain exactly ONE command. Split into separate suggestions."
			}
			return "shell operator '|' (pipe) detected. Commands are NOT run through a shell. Each suggestion must contain exactly ONE command. Split into separate suggestions."
		case '&':
			if i+1 < n && runes[i+1] == '&' {
				return "shell operator '&&' detected. Commands are NOT run through a shell. Each suggestion must contain exactly ONE command. Split into separate suggestions."
			}
			return "shell operator '&' (background) detected. Commands are NOT run through a shell. Each suggestion must contain exactly ONE command. Split into separate suggestions."
		}
	}
	return ""
}

// parseCommand delegates to the exported ParseCommand in cmdobj.go.
func parseCommand(cmd string) []string {
	return ParseCommand(cmd)
}

// Command allowlists and alternative hints are now centralized in platform.go
// (CrossPlatformCommands, UnixOnlyCommands, WindowsOnlyCommands, ToolAlternatives).
// Platform.IsCommandAllowed() and AlternativeHint() provide the unified API.

func (r *Runner) execShellCommand(ctx context.Context, cmdStr string) *ExecutionResult {
	if cmdStr == "" {
		return &ExecutionResult{Stderr: "empty shell command", Success: false}
	}

	if reason := rejectShellOperators(cmdStr); reason != "" {
		return &ExecutionResult{Command: cmdStr, Stderr: reason, Success: false}
	}

	args := ParseCommand(cmdStr)
	if len(args) == 0 {
		return &ExecutionResult{Command: cmdStr, Stderr: "empty shell command after parsing", Success: false}
	}

	base := strings.ToLower(filepath.Base(args[0]))
	base = strings.TrimSuffix(base, ".exe")

	// Auto-route git/gh to their dedicated executors with full protection logic.
	if base == "git" {
		return r.execGitCommand(ctx, cmdStr)
	}
	if base == "gh" {
		return r.execGitHubOp(ctx, cmdStr)
	}

	subShells := map[string]bool{
		"cmd": true, "powershell": true, "pwsh": true,
		"bash": true, "sh": true, "zsh": true, "fish": true, "csh": true,
	}
	if subShells[base] {
		return &ExecutionResult{
			Command: cmdStr,
			Stderr:  fmt.Sprintf("shell_command: sub-shell %q is blocked — run the target command directly", base),
			Success: false,
		}
	}

	if !r.platform.IsCommandAllowed(base) {
		errMsg := fmt.Sprintf("shell_command: %q is not available on this platform (%s)", base, r.platform.OS)
		if hint := AlternativeHint(base); hint != "" {
			errMsg += fmt.Sprintf(". Alternative: %s", hint)
		}
		return &ExecutionResult{
			Command: cmdStr,
			Stderr:  errMsg,
			Success: false,
		}
	}

	cmdObj := NewShellCmd(r.platform, args[0], args[1:]...)
	cmdObj.SetWd(r.repoRoot)
	stdout, stderr, err := cmdObj.Run(ctx)
	result := &ExecutionResult{
		Command: cmdStr,
		Stdout:  stdout,
		Stderr:  stderr,
		Success: err == nil,
	}
	if err != nil {
		result.ExitCode = 1
	}
	return result
}

var validGHSubcommands = map[string]bool{
	"issue": true, "pr": true, "release": true, "label": true,
	"repo": true, "workflow": true, "run": true, "api": true,
	"auth": true, "browse": true, "gist": true, "secret": true,
	"variable": true, "codespace": true, "cache": true,
}

// ghAuthOKByBinary caches `gh auth status` results per configured binary.
var (
	ghAuthMu             sync.Mutex
	ghAuthOKByBinary     = make(map[string]bool)
	ghAuthScopesByBinary = make(map[string]map[string]bool)
)

func checkGHAuth(ctx context.Context, cmdExec CommandExecutor, repoRoot, ghBin string) error {
	if strings.TrimSpace(ghBin) == "" {
		ghBin = "gh"
	}
	if cmdExec == nil {
		cmdExec = osCommandExecutor{}
	}
	ghAuthMu.Lock()
	cached, seen := ghAuthOKByBinary[ghBin]
	ghAuthMu.Unlock()
	if seen {
		if cached {
			return nil
		}
		return fmt.Errorf("%s auth previously failed — run '%s auth login' first", ghBin, ghBin)
	}
	out, err := cmdExec.CombinedOutput(ctx, ghBin, []string{"auth", "status"}, repoRoot)
	ok := err == nil
	ghAuthMu.Lock()
	ghAuthOKByBinary[ghBin] = ok
	if ok {
		ghAuthScopesByBinary[ghBin] = parseGHScopes(string(out))
	}
	ghAuthMu.Unlock()
	if !ok {
		return fmt.Errorf("%s is not authenticated — run '%s auth login' first", ghBin, ghBin)
	}
	return nil
}

func parseGHScopes(statusOutput string) map[string]bool {
	scopes := make(map[string]bool)
	for _, line := range strings.Split(statusOutput, "\n") {
		lower := strings.ToLower(line)
		idx := strings.Index(lower, "token scopes:")
		if idx < 0 {
			continue
		}
		raw := line[idx+len("token scopes:"):]
		parts := strings.FieldsFunc(raw, func(r rune) bool {
			return r == ',' || r == ' ' || r == '\t' || r == '\'' || r == '"' || r == '[' || r == ']'
		})
		for _, p := range parts {
			p = strings.ToLower(strings.TrimSpace(p))
			if p == "" {
				continue
			}
			scopes[p] = true
		}
	}
	return scopes
}

func requiredGHScopes(ghArgs []string) []string {
	if len(ghArgs) < 2 {
		return nil
	}
	sub := strings.ToLower(strings.TrimSpace(ghArgs[0]))
	verb := strings.ToLower(strings.TrimSpace(ghArgs[1]))

	// Read-only verbs do not need scope pre-check.
	mutatingVerbs := map[string]bool{
		"create": true, "edit": true, "delete": true, "set": true,
		"close": true, "reopen": true, "merge": true, "enable": true,
		"disable": true, "run": true, "rerun": true, "cancel": true,
		"upload": true, "update": true, "remove": true, "restore": true,
	}
	if sub != "api" && !mutatingVerbs[verb] {
		return nil
	}

	required := map[string]bool{}
	switch sub {
	case "workflow", "run":
		required["repo"] = true
		required["workflow"] = true
	case "issue", "pr", "release", "label", "repo", "secret", "variable":
		required["repo"] = true
	case "api":
		joined := strings.ToLower(strings.Join(ghArgs, " "))
		if strings.Contains(joined, "/actions/") || strings.Contains(joined, "/workflows/") {
			required["workflow"] = true
		}
		if strings.Contains(joined, "/repos/") || strings.Contains(joined, "/orgs/") ||
			strings.Contains(joined, "/issues") || strings.Contains(joined, "/pulls") {
			required["repo"] = true
		}
	default:
		// Unknown or non-mutating subcommand: no-op.
	}

	if len(required) == 0 {
		return nil
	}
	out := make([]string, 0, len(required))
	for scope := range required {
		out = append(out, scope)
	}
	return out
}

func checkGHScopes(ghBin string, ghArgs []string) error {
	req := requiredGHScopes(ghArgs)
	if len(req) == 0 {
		return nil
	}
	ghAuthMu.Lock()
	scopes := ghAuthScopesByBinary[ghBin]
	ghAuthMu.Unlock()
	// Older gh versions may not print scopes; avoid false negatives.
	if len(scopes) == 0 {
		return nil
	}
	missing := make([]string, 0, len(req))
	for _, scope := range req {
		if hasGHScope(scopes, scope) {
			continue
		}
		missing = append(missing, scope)
	}
	if len(missing) > 0 {
		return fmt.Errorf("%s token missing required scope(s): %s", ghBin, strings.Join(missing, ", "))
	}
	return nil
}

func hasGHScope(scopes map[string]bool, required string) bool {
	switch required {
	case "repo":
		return scopes["repo"] || scopes["public_repo"]
	default:
		return scopes[required]
	}
}

// ghPreflightCreate runs a quick existence check before `gh <resource> create`.
// Returns a non-empty error string if the resource already exists or the check
// reveals a problem. Returns "" if creation should proceed.
func (r *Runner) ghPreflightCreate(ctx context.Context, ghArgs []string) string {
	if len(ghArgs) < 2 || ghArgs[1] != "create" {
		return ""
	}
	resource := ghArgs[0]
	switch resource {
	case "release":
		tag := ghFlagValueOrPositional(ghArgs[2:], "", 0)
		if tag == "" {
			return ""
		}
		out, _, err := r.runGH(ctx, "release", "view", tag)
		if err == nil && out != "" {
			return fmt.Sprintf("ALREADY EXISTS: release %q already exists. Use 'gh release edit %s' to modify or skip this step.", tag, tag)
		}
	case "label":
		name := ghFlagValueOrPositional(ghArgs[2:], "--name", 0)
		if name == "" {
			name = ghFlagValueOrPositional(ghArgs[2:], "", 0)
		}
		if name == "" {
			return ""
		}
		out, _, err := r.runGH(ctx, "label", "list", "--search", name)
		if err == nil && containsLabelName(out, name) {
			return fmt.Sprintf("ALREADY EXISTS: label %q already exists. Skip this step.", name)
		}
	case "issue":
		title := ghFlagValueOrPositional(ghArgs[2:], "--title", -1)
		if title != "" {
			out, _, err := r.runGH(ctx, "issue", "list", "--search", title, "--state", "all", "--limit", "5")
			if err == nil && strings.Contains(out, title) {
				return fmt.Sprintf("LIKELY DUPLICATE: issue with title %q may already exist. Check first with 'gh issue list --search %q'.", title, title)
			}
		}
	case "pr":
		// gh pr create will fail if one already exists for the branch; proceed.
	case "secret":
		name := ghFlagValueOrPositional(ghArgs[2:], "--name", 0)
		if name == "" {
			return ""
		}
		out, _, err := r.runGH(ctx, "secret", "list")
		if err == nil && strings.Contains(out, name) {
			return fmt.Sprintf("ALREADY EXISTS: secret %q already exists. Use 'gh secret set %s' to update.", name, name)
		}
	case "variable":
		name := ghFlagValueOrPositional(ghArgs[2:], "--name", 0)
		if name == "" {
			return ""
		}
		out, _, err := r.runGH(ctx, "variable", "list")
		if err == nil && strings.Contains(out, name) {
			return fmt.Sprintf("ALREADY EXISTS: variable %q already exists. Use 'gh variable set %s' to update.", name, name)
		}
	}
	return ""
}

func containsLabelName(listOutput, name string) bool {
	for _, line := range strings.Split(listOutput, "\n") {
		fields := strings.Fields(line)
		if len(fields) > 0 && strings.EqualFold(fields[0], name) {
			return true
		}
		if strings.Contains(line, name) {
			return true
		}
	}
	return false
}

// ghFlagValueOrPositional extracts a value from args. If flagName is non-empty,
// looks for --flag value. Otherwise returns the positional arg at posIdx.
func ghFlagValueOrPositional(args []string, flagName string, posIdx int) string {
	if flagName != "" {
		for i, a := range args {
			if a == flagName && i+1 < len(args) {
				return args[i+1]
			}
			if strings.HasPrefix(a, flagName+"=") {
				return strings.TrimPrefix(a, flagName+"=")
			}
		}
		return ""
	}
	idx := 0
	for _, a := range args {
		if strings.HasPrefix(a, "-") {
			continue
		}
		if idx == posIdx {
			return a
		}
		idx++
	}
	return ""
}

func (r *Runner) runGH(ctx context.Context, args ...string) (string, string, error) {
	cmdObj := NewGhCmd(r.platform, args...)
	cmdObj.SetWd(r.repoRoot)
	return cmdObj.Run(ctx)
}

func (r *Runner) ghBinary() string {
	if cfg := config.Get(); cfg != nil {
		if bin := strings.TrimSpace(cfg.Adapters.GitHub.GH.Binary); bin != "" {
			return bin
		}
	}
	return "gh"
}

func isGitHubBinaryToken(token, configured string) bool {
	tokenBase := strings.ToLower(strings.TrimSuffix(filepath.Base(strings.TrimSpace(token)), ".exe"))
	cfgBase := strings.ToLower(strings.TrimSuffix(filepath.Base(strings.TrimSpace(configured)), ".exe"))
	if cfgBase == "" {
		cfgBase = "gh"
	}
	return tokenBase == cfgBase
}

// classifyGHError turns raw gh stderr into actionable LLM-friendly feedback.
func classifyGHError(stderr string, cmdStr string) string {
	low := strings.ToLower(stderr)
	var hints []string
	if strings.Contains(low, "http 422") {
		hints = append(hints, "HTTP 422 (Validation Failed): the resource likely already exists, or a required field is missing/invalid. Do NOT retry with the same arguments. Skip this step entirely.")
	}
	if strings.Contains(low, "http 404") {
		hints = append(hints, "HTTP 404 (Not Found): the repository, tag, or resource does not exist. Verify all identifiers (repo name, owner, tag, branch). Do NOT retry — fix the identifier or skip.")
	}
	if strings.Contains(low, "http 403") {
		hints = append(hints, "HTTP 403 (Forbidden): insufficient permissions. The token may lack the required scope. STOP all github_op commands.")
	}
	if strings.Contains(low, "http 401") {
		hints = append(hints, "HTTP 401 (Unauthorized): authentication failed. STOP all github_op commands. Run 'gh auth login' first.")
	}
	if strings.Contains(low, "http 409") {
		hints = append(hints, "HTTP 409 (Conflict): the resource conflicts with existing state. Skip this step.")
	}
	if strings.Contains(low, "http 429") {
		hints = append(hints, "HTTP 429 (Rate Limited): GitHub API rate limit exceeded. STOP all github_op commands for this round.")
	}
	if strings.Contains(low, "http 400") {
		hints = append(hints, "HTTP 400 (Bad Request): the request is malformed. Check command syntax and arguments.")
	}
	if strings.Contains(low, "http 500") || strings.Contains(low, "http 502") || strings.Contains(low, "http 503") {
		hints = append(hints, "HTTP 5xx (Server Error): GitHub is temporarily unavailable. Skip github_op commands for now.")
	}
	if strings.Contains(low, "not logged in") || strings.Contains(low, "authentication") {
		hints = append(hints, "gh is not authenticated. STOP all github_op commands.")
	}
	if strings.Contains(low, "could not resolve") || strings.Contains(low, "connection refused") || strings.Contains(low, "timeout") {
		hints = append(hints, "Network error: cannot reach GitHub API. STOP all github_op commands.")
	}
	if strings.Contains(low, "already exists") {
		hints = append(hints, "The resource already exists. Skip this step entirely. Do NOT retry.")
	}
	if strings.Contains(low, "no such") || strings.Contains(low, "not found") {
		if !strings.Contains(low, "http 404") {
			hints = append(hints, "Resource not found. Verify all identifiers before retrying.")
		}
	}
	if len(hints) > 0 {
		return stderr + "\n\n[GITDEX DIAGNOSIS] " + strings.Join(hints, " | ")
	}
	return stderr
}

func (r *Runner) execGitHubOp(ctx context.Context, cmdStr string) *ExecutionResult {
	if cmdStr == "" {
		return &ExecutionResult{Stderr: "empty github_op command", Success: false}
	}
	if reason := rejectShellOperators(cmdStr); reason != "" {
		return &ExecutionResult{Command: cmdStr, Stderr: reason, Success: false}
	}
	args := ParseCommand(cmdStr)
	if len(args) == 0 {
		return &ExecutionResult{Command: cmdStr, Stderr: "empty github_op after parsing", Success: false}
	}
	ghBin := r.ghBinary()
	ghArgs := args
	if isGitHubBinaryToken(ghArgs[0], ghBin) {
		ghArgs = ghArgs[1:]
	}
	if len(ghArgs) == 0 {
		return &ExecutionResult{Command: cmdStr, Stderr: "github_op: no subcommand after 'gh'", Success: false}
	}

	if !validGHSubcommands[ghArgs[0]] {
		return &ExecutionResult{
			Command: cmdStr,
			Stderr:  fmt.Sprintf("github_op: invalid gh subcommand %q. Valid subcommands: issue, pr, release, label, repo, workflow, run, api, auth, secret, variable", ghArgs[0]),
			Success: false,
		}
	}

	// Auth pre-check (cached)
	if err := checkGHAuth(ctx, r.cmdExec, r.repoRoot, ghBin); err != nil {
		return &ExecutionResult{
			Command: cmdStr,
			Stderr:  fmt.Sprintf("github_op: %v. Do NOT retry gh commands until authenticated.", err),
			Success: false,
		}
	}
	if err := checkGHScopes(ghBin, ghArgs); err != nil {
		return &ExecutionResult{
			Command: cmdStr,
			Stderr:  fmt.Sprintf("github_op: %v. Fix token scopes first (gh auth refresh -s %s).", err, strings.Join(requiredGHScopes(ghArgs), ",")),
			Success: false,
		}
	}

	// Pre-flight existence check for create operations
	if reason := r.ghPreflightCreate(ctx, ghArgs); reason != "" {
		return &ExecutionResult{
			Command: cmdStr,
			Stderr:  reason,
			Success: false,
		}
	}

	// Convert --body to temp file for commands that include body text.
	if NeedsTempFile("github_op", cmdStr) {
		cmdObj := ConvertToTempFile(r.platform, r.repoRoot, "github_op", cmdStr)
		if cmdObj != nil {
			stdout, stderr, err := cmdObj.Run(ctx)
			result := &ExecutionResult{
				Command: cmdStr,
				Stdout:  stdout,
				Stderr:  stderr,
				Success: err == nil,
			}
			if err != nil {
				result.ExitCode = 1
				result.Stderr = classifyGHError(stderr, cmdStr)
			}
			return result
		}
	}

	cmdObj := NewGhCmd(r.platform, ghArgs...)
	cmdObj.SetWd(r.repoRoot)
	stdout, stderr, err := cmdObj.Run(ctx)
	result := &ExecutionResult{
		Command: cmdStr,
		Stdout:  stdout,
		Stderr:  stderr,
		Success: err == nil,
	}
	if err != nil {
		result.ExitCode = 1
		result.Stderr = classifyGHError(stderr, cmdStr)
	}
	return result
}

func (r *Runner) execFileWrite(action planner.ActionSpec) *ExecutionResult {
	result := &ExecutionResult{
		FilePath: action.FilePath,
	}

	safePath, err := r.resolveInsideRepo(action.FilePath)
	if err != nil {
		result.Stderr = fmt.Sprintf("BLOCKED: %v", err)
		result.ExitCode = 1
		return result
	}

	switch action.FileOp {
	case "create", "update":
		if dir := filepath.Dir(safePath); dir != "." && dir != "" {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				result.Stderr = "mkdir parent: " + err.Error()
				result.ExitCode = 1
				return result
			}
		}
		content := StripTrailingWhitespace(action.FileContent)
		err := os.WriteFile(safePath, []byte(content), 0o644)
		result.Success = err == nil
		if err != nil {
			result.Stderr = err.Error()
			result.ExitCode = 1
		}
	case "append":
		if dir := filepath.Dir(safePath); dir != "." && dir != "" {
			_ = os.MkdirAll(dir, 0o755)
		}
		f, err := os.OpenFile(safePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			result.Stderr = err.Error()
			result.ExitCode = 1
			return result
		}
		content := StripTrailingWhitespace(action.FileContent)
		_, err = f.WriteString(content)
		f.Close()
		result.Success = err == nil
		if err != nil {
			result.Stderr = err.Error()
			result.ExitCode = 1
		}
	case "delete":
		err := os.Remove(safePath)
		result.Success = err == nil
		if err != nil {
			result.Stderr = err.Error()
			result.ExitCode = 1
		}
	case "mkdir":
		err := os.MkdirAll(safePath, 0o755)
		result.Success = err == nil
		if err != nil {
			result.Stderr = err.Error()
			result.ExitCode = 1
		}
	default:
		result.Stderr = fmt.Sprintf("unknown file_operation: %s", action.FileOp)
		result.ExitCode = 1
	}

	return result
}

func (r *Runner) execFileRead(action planner.ActionSpec) *ExecutionResult {
	result := &ExecutionResult{FilePath: action.FilePath}

	if action.FilePath == "" {
		result.Stderr = "file_read: file_path is required"
		result.ExitCode = 1
		return result
	}

	safePath, err := r.resolveInsideRepo(action.FilePath)
	if err != nil {
		result.Stderr = fmt.Sprintf("BLOCKED: %v", err)
		result.ExitCode = 1
		return result
	}

	data, err := os.ReadFile(safePath)
	if err != nil {
		result.Stderr = fmt.Sprintf("file_read: %v", err)
		result.ExitCode = 1
		return result
	}

	result.Stdout = string(data)
	result.Success = true
	return result
}

// resolveInsideRepo ensures the given path resolves to a location inside repoRoot.
func (r *Runner) resolveInsideRepo(path string) (string, error) {
	if r.repoRoot == "" {
		return path, nil
	}
	candidate := path
	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(r.repoRoot, candidate)
	}
	abs, err := filepath.Abs(candidate)
	if err != nil {
		return "", fmt.Errorf("file_write: cannot resolve path %q: %w", path, err)
	}
	abs = filepath.Clean(abs)
	root := filepath.Clean(r.repoRoot)
	if !strings.HasPrefix(abs, root+string(filepath.Separator)) && abs != root {
		return "", fmt.Errorf("file_write: path %q escapes repository root", path)
	}
	return abs, nil
}

// stripTrailingWhitespace delegates to the exported StripTrailingWhitespace in cmdobj.go.
func stripTrailingWhitespace(s string) string {
	return StripTrailingWhitespace(s)
}
