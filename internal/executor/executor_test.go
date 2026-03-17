package executor

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Joker-of-Gotham/gitdex/internal/dotgitdex"
	"github.com/Joker-of-Gotham/gitdex/internal/git/cli"
	"github.com/Joker-of-Gotham/gitdex/internal/planner"
)

func TestNewRunner(t *testing.T) {
	r := NewRunner(nil, nil, nil)
	if r == nil {
		t.Fatal("NewRunner returned nil")
	}
}

func TestExecuteSuggestion_UnsupportedActionType(t *testing.T) {
	r := NewRunner(nil, nil, nil)
	ctx := context.Background()
	item := planner.SuggestionItem{
		Name:   "unsupported",
		Action: planner.ActionSpec{Type: "unknown_type"},
	}
	result := r.ExecuteSuggestion(ctx, 1, item)
	if result == nil {
		t.Fatal("ExecuteSuggestion returned nil")
	}
	if result.Success {
		t.Error("expected Success=false for unsupported action type")
	}
	if result.Stderr == "" {
		t.Error("expected non-empty Stderr for unsupported action type")
	}
}

func TestExecuteSuggestion_EmptyGitCommand(t *testing.T) {
	r := NewRunner(cli.GitCLI(nil), nil, nil)
	ctx := context.Background()
	item := planner.SuggestionItem{
		Name:   "empty-cmd",
		Action: planner.ActionSpec{Type: "git_command", Command: ""},
	}
	result := r.ExecuteSuggestion(ctx, 1, item)
	if result == nil {
		t.Fatal("ExecuteSuggestion returned nil")
	}
	if result.Success {
		t.Error("expected Success=false for empty command")
	}
	if !findSubstring(result.Stderr, "preflight validation failed") {
		t.Errorf("expected preflight validation error, got %q", result.Stderr)
	}
}

func TestExecuteSuggestion_FileWriteCreate(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "create-test.txt")
	r := NewRunner(nil, nil, nil)
	ctx := context.Background()
	content := "hello create"
	item := planner.SuggestionItem{
		Name:   "create-file",
		Action: planner.ActionSpec{Type: "file_write", FilePath: filePath, FileContent: content, FileOp: "create"},
	}
	result := r.ExecuteSuggestion(ctx, 1, item)
	if result == nil {
		t.Fatal("ExecuteSuggestion returned nil")
	}
	if !result.Success {
		t.Fatalf("expected Success=true for create, got Stderr=%q", result.Stderr)
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read created file: %v", err)
	}
	expected := content + "\n"
	if string(data) != expected {
		t.Errorf("expected content %q, got %q", expected, string(data))
	}
}

func TestExecuteSuggestion_FileWriteAppend(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "append-test.txt")
	initial := "initial\n"
	if err := os.WriteFile(filePath, []byte(initial), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	r := NewRunner(nil, nil, nil)
	ctx := context.Background()
	appendContent := "appended\n"
	item := planner.SuggestionItem{
		Name:   "append-file",
		Action: planner.ActionSpec{Type: "file_write", FilePath: filePath, FileContent: appendContent, FileOp: "append"},
	}
	result := r.ExecuteSuggestion(ctx, 1, item)
	if result == nil {
		t.Fatal("ExecuteSuggestion returned nil")
	}
	if !result.Success {
		t.Fatalf("expected Success=true for append, got Stderr=%q", result.Stderr)
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read file after append: %v", err)
	}
	expected := initial + appendContent
	if string(data) != expected {
		t.Errorf("expected content %q, got %q", expected, string(data))
	}
}

func TestExecuteSuggestion_FileWriteDelete(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "delete-test.txt")
	if err := os.WriteFile(filePath, []byte("to be deleted"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	r := NewRunner(nil, nil, nil)
	ctx := context.Background()
	item := planner.SuggestionItem{
		Name:   "delete-file",
		Action: planner.ActionSpec{Type: "file_write", FilePath: filePath, FileOp: "delete"},
	}
	result := r.ExecuteSuggestion(ctx, 1, item)
	if result == nil {
		t.Fatal("ExecuteSuggestion returned nil")
	}
	if !result.Success {
		t.Fatalf("expected Success=true for delete, got Stderr=%q", result.Stderr)
	}
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Errorf("file should not exist after delete: %v", err)
	}
}

func TestExecuteSuggestion_FileWriteUpdate(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "update-test.txt")
	initial := "old content"
	if err := os.WriteFile(filePath, []byte(initial), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	r := NewRunner(nil, nil, nil)
	ctx := context.Background()
	newContent := "new content"
	item := planner.SuggestionItem{
		Name:   "update-file",
		Action: planner.ActionSpec{Type: "file_write", FilePath: filePath, FileContent: newContent, FileOp: "update"},
	}
	result := r.ExecuteSuggestion(ctx, 1, item)
	if result == nil {
		t.Fatal("ExecuteSuggestion returned nil")
	}
	if !result.Success {
		t.Fatalf("expected Success=true for update, got Stderr=%q", result.Stderr)
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read file after update: %v", err)
	}
	expected := newContent + "\n"
	if string(data) != expected {
		t.Errorf("expected content %q, got %q", expected, string(data))
	}
}

func TestExecuteSuggestion_FileWriteMkdir(t *testing.T) {
	dir := t.TempDir()
	newDir := filepath.Join(dir, "sub", "deep")
	r := NewRunner(nil, nil, nil)
	ctx := context.Background()
	item := planner.SuggestionItem{
		Name:   "make-directory",
		Action: planner.ActionSpec{Type: "file_write", FilePath: newDir, FileOp: "mkdir"},
	}
	result := r.ExecuteSuggestion(ctx, 1, item)
	if !result.Success {
		t.Fatalf("expected Success=true for mkdir, got Stderr=%q", result.Stderr)
	}
	info, err := os.Stat(newDir)
	if err != nil {
		t.Fatalf("directory should exist: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected a directory")
	}
}

func TestExecuteSuggestion_FileWriteAutoCreateParentDir(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "new-parent", "child.txt")
	r := NewRunner(nil, nil, nil)
	ctx := context.Background()
	item := planner.SuggestionItem{
		Name:   "create-with-parent",
		Action: planner.ActionSpec{Type: "file_write", FilePath: filePath, FileContent: "content", FileOp: "create"},
	}
	result := r.ExecuteSuggestion(ctx, 1, item)
	if !result.Success {
		t.Fatalf("expected Success=true, parent dir auto-created, got Stderr=%q", result.Stderr)
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("file should exist: %v", err)
	}
	if string(data) != "content\n" {
		t.Errorf("expected 'content\\n', got %q", string(data))
	}
}

func TestExecuteSuggestion_ShellCommandEmpty(t *testing.T) {
	r := NewRunner(nil, nil, nil)
	ctx := context.Background()
	item := planner.SuggestionItem{
		Name:   "empty-shell",
		Action: planner.ActionSpec{Type: "shell_command", Command: ""},
	}
	result := r.ExecuteSuggestion(ctx, 1, item)
	if result.Success {
		t.Error("expected failure for empty shell command")
	}
}

func TestExecuteSuggestion_GitHubOpEmpty(t *testing.T) {
	r := NewRunner(nil, nil, nil)
	ctx := context.Background()
	item := planner.SuggestionItem{
		Name:   "empty-gh",
		Action: planner.ActionSpec{Type: "github_op", Command: ""},
	}
	result := r.ExecuteSuggestion(ctx, 1, item)
	if result.Success {
		t.Error("expected failure for empty github_op command")
	}
}

func TestGitdexProtection_AllActionTypes(t *testing.T) {
	r := NewRunner(nil, nil, nil)
	ctx := context.Background()
	types := []string{"git_command", "shell_command", "github_op"}
	for _, tp := range types {
		item := planner.SuggestionItem{
			Name:   "blocked-" + tp,
			Action: planner.ActionSpec{Type: tp, Command: "rm -rf .gitdex/"},
		}
		result := r.ExecuteSuggestion(ctx, 1, item)
		if result.Success {
			t.Errorf("expected %s targeting .gitdex to be blocked", tp)
		}
		if result.Stderr == "" || !contains(result.Stderr, "BLOCKED") {
			t.Errorf("expected BLOCKED in stderr for %s, got %q", tp, result.Stderr)
		}
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && findSubstring(s, sub))
}
func findSubstring(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestToolLabel(t *testing.T) {
	cases := map[string]string{
		"git_command":   "GIT",
		"shell_command": "SHELL",
		"file_write":    "FILE",
		"file_read":     "READ",
		"github_op":     "GITHUB",
		"other":         "UNKNOWN",
	}
	for tp, expected := range cases {
		a := planner.ActionSpec{Type: tp}
		if got := a.ToolLabel(); got != expected {
			t.Errorf("ToolLabel(%q) = %q, want %q", tp, got, expected)
		}
	}
}

func TestNewExecutionLogger(t *testing.T) {
	dir := t.TempDir()
	mgr := dotgitdex.New(dir)
	if err := mgr.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	logger := NewExecutionLogger(mgr, "sess-1", "maintain")
	if logger == nil {
		t.Fatal("NewExecutionLogger returned nil")
	}
}

func TestExecutionLogger_LogStepAddsStep(t *testing.T) {
	dir := t.TempDir()
	mgr := dotgitdex.New(dir)
	if err := mgr.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	logger := NewExecutionLogger(mgr, "sess-1", "maintain")
	item := planner.SuggestionItem{
		Name:   "step-one",
		Action: planner.ActionSpec{Type: "file_write", FilePath: "/tmp/x", FileOp: "create"},
	}
	result := &ExecutionResult{Success: true}
	logger.LogStep(1, item, result)
	if len(logger.steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(logger.steps))
	}
	if logger.steps[0].Name != "step-one" {
		t.Errorf("expected step name step-one, got %q", logger.steps[0].Name)
	}
	if logger.steps[0].SequenceID != 1 {
		t.Errorf("expected seq 1, got %d", logger.steps[0].SequenceID)
	}
}

func TestExecutionLogger_FlushWithNoStepsIsNoop(t *testing.T) {
	dir := t.TempDir()
	mgr := dotgitdex.New(dir)
	if err := mgr.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	logger := NewExecutionLogger(mgr, "sess-1", "maintain")
	err := logger.Flush()
	if err != nil {
		t.Fatalf("Flush with no steps should not error: %v", err)
	}
}

// === Shell operator detection tests ===

func TestRejectShellOperators_Ampersand(t *testing.T) {
	msg := rejectShellOperators(`gh label create bug -c "#d73a4a" & gh label create enhancement`)
	if msg == "" {
		t.Fatal("expected error for '&' operator, got empty")
	}
	if !findSubstring(msg, "'&'") {
		t.Errorf("expected mention of '&' in error, got %q", msg)
	}
}

func TestRejectShellOperators_DoubleAmpersand(t *testing.T) {
	msg := rejectShellOperators("git add . && git commit -m 'fix'")
	if msg == "" {
		t.Fatal("expected error for '&&' operator, got empty")
	}
	if !findSubstring(msg, "'&&'") {
		t.Errorf("expected mention of '&&' in error, got %q", msg)
	}
}

func TestRejectShellOperators_Pipe(t *testing.T) {
	msg := rejectShellOperators("cat file.txt | grep error")
	if msg == "" {
		t.Fatal("expected error for '|' operator, got empty")
	}
}

func TestRejectShellOperators_DoublePipe(t *testing.T) {
	msg := rejectShellOperators("make build || echo failed")
	if msg == "" {
		t.Fatal("expected error for '||' operator, got empty")
	}
}

func TestRejectShellOperators_Semicolon(t *testing.T) {
	msg := rejectShellOperators("cd /tmp; ls")
	if msg == "" {
		t.Fatal("expected error for ';' operator, got empty")
	}
}

func TestRejectShellOperators_InsideQuotes_Allowed(t *testing.T) {
	safe := []string{
		`git commit -m "fix & improve"`,
		`echo "a && b || c"`,
		`git commit -m 'fix; update'`,
		`gh label create bug -c "#d73a4a"`,
		`echo "hello | world"`,
	}
	for _, cmd := range safe {
		if msg := rejectShellOperators(cmd); msg != "" {
			t.Errorf("expected no error for %q inside quotes, got %q", cmd, msg)
		}
	}
}

func TestRejectShellOperators_CleanCommands(t *testing.T) {
	safe := []string{
		"git fetch --all --prune",
		"npm run build",
		"go test ./...",
		`gh label create bug -c "#d73a4a" -d "Bug report"`,
		"echo hello",
		"python3 -c 'print(1)'",
		"curl -c cookies.txt https://example.com",
	}
	for _, cmd := range safe {
		if msg := rejectShellOperators(cmd); msg != "" {
			t.Errorf("expected no error for clean command %q, got %q", cmd, msg)
		}
	}
}

func TestParseCommand_QuotedCommitMessage(t *testing.T) {
	args := parseCommand(`git commit -m "Fix trailing whitespace in templates"`)
	if len(args) != 4 {
		t.Fatalf("expected 4 args, got %d: %#v", len(args), args)
	}
	if args[2] != "-m" || args[3] != "Fix trailing whitespace in templates" {
		t.Fatalf("commit message parsing mismatch: %#v", args)
	}
}

func TestParseCommand_WindowsPathPreserved(t *testing.T) {
	args := parseCommand(`python "C:\Users\demo\scripts\tool.py"`)
	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %d: %#v", len(args), args)
	}
	if args[1] != `C:\Users\demo\scripts\tool.py` {
		t.Fatalf("windows path should be preserved, got %q", args[1])
	}
}

func TestParseGHScopes(t *testing.T) {
	out := `
github.com
  ✓ Logged in to github.com account demo
  - Token scopes: 'repo', 'workflow', 'read:org'
`
	scopes := parseGHScopes(out)
	if !scopes["repo"] || !scopes["workflow"] || !scopes["read:org"] {
		t.Fatalf("unexpected parsed scopes: %#v", scopes)
	}
}

func TestRequiredGHScopes(t *testing.T) {
	if got := requiredGHScopes([]string{"issue", "list"}); len(got) != 0 {
		t.Fatalf("issue list should not require scopes precheck, got %v", got)
	}
	got := requiredGHScopes([]string{"workflow", "enable"})
	if len(got) != 2 {
		t.Fatalf("workflow enable should require repo+workflow, got %v", got)
	}
}

func TestCheckGHScopes_MissingWorkflow(t *testing.T) {
	ghAuthMu.Lock()
	orig := ghAuthScopesByBinary
	ghAuthScopesByBinary = map[string]map[string]bool{
		"gh": {"repo": true},
	}
	ghAuthMu.Unlock()
	t.Cleanup(func() {
		ghAuthMu.Lock()
		ghAuthScopesByBinary = orig
		ghAuthMu.Unlock()
	})

	err := checkGHScopes("gh", []string{"workflow", "enable"})
	if err == nil || !findSubstring(err.Error(), "workflow") {
		t.Fatalf("expected missing workflow scope error, got %v", err)
	}
}

func TestRedactSensitiveText(t *testing.T) {
	raw := `Authorization: Bearer ghp_abcdefghijklmnopqrstuvwxyz012345 OPENAI_API_KEY=sk-abcdefghijklmnopqrstuvwxyz1234 https://api.example.com?token=topsecret`
	got := redactSensitiveText(raw)
	if findSubstring(strings.ToLower(got), "ghp_") || findSubstring(strings.ToLower(got), "sk-") || findSubstring(strings.ToLower(got), "topsecret") {
		t.Fatalf("sensitive fragments should be redacted, got %q", got)
	}
	if !findSubstring(got, "[REDACTED]") {
		t.Fatalf("expected redaction marker, got %q", got)
	}
}

type fakeCommandExecutor struct {
	out   []byte
	err   error
	calls int
}

func (f *fakeCommandExecutor) CombinedOutput(_ context.Context, _ string, _ []string, _ string) ([]byte, error) {
	f.calls++
	return f.out, f.err
}

func TestCheckGHAuth_UsesInjectedExecutorAndCaches(t *testing.T) {
	ghAuthMu.Lock()
	origAuth := ghAuthOKByBinary
	origScopes := ghAuthScopesByBinary
	ghAuthOKByBinary = map[string]bool{}
	ghAuthScopesByBinary = map[string]map[string]bool{}
	ghAuthMu.Unlock()
	t.Cleanup(func() {
		ghAuthMu.Lock()
		ghAuthOKByBinary = origAuth
		ghAuthScopesByBinary = origScopes
		ghAuthMu.Unlock()
	})

	fake := &fakeCommandExecutor{
		out: []byte("Token scopes: 'repo', 'workflow'"),
	}
	if err := checkGHAuth(context.Background(), fake, "", "gh"); err != nil {
		t.Fatalf("expected auth success, got %v", err)
	}
	if fake.calls != 1 {
		t.Fatalf("expected one auth probe call, got %d", fake.calls)
	}
	// Cached path should bypass second invocation.
	fake.err = errors.New("should not be called")
	if err := checkGHAuth(context.Background(), fake, "", "gh"); err != nil {
		t.Fatalf("expected cached auth success, got %v", err)
	}
	if fake.calls != 1 {
		t.Fatalf("expected cached auth to skip command execution, got calls=%d", fake.calls)
	}
}

// === git/gh auto-routing via shell_command ===

func TestShellCommand_SubShellBlocked(t *testing.T) {
	r := NewRunner(nil, nil, nil)
	ctx := context.Background()
	shells := []string{"bash", "sh", "cmd", "powershell", "pwsh", "zsh", "fish", "csh"}
	for _, sh := range shells {
		item := planner.SuggestionItem{
			Name:   "blocked-shell-" + sh,
			Action: planner.ActionSpec{Type: "shell_command", Command: sh + " -c echo hello"},
		}
		result := r.ExecuteSuggestion(ctx, 1, item)
		if result.Success {
			t.Errorf("expected %q to be blocked as sub-shell", sh)
		}
		if !findSubstring(result.Stderr, "sub-shell") {
			t.Errorf("expected 'sub-shell' in error for %q, got %q", sh, result.Stderr)
		}
	}
}

func TestShellCommand_LegitFlagNotBlocked(t *testing.T) {
	r := NewRunner(nil, nil, nil)
	ctx := context.Background()
	item := planner.SuggestionItem{
		Name:   "legit-flag",
		Action: planner.ActionSpec{Type: "shell_command", Command: "echo -c test"},
	}
	result := r.ExecuteSuggestion(ctx, 1, item)
	if !result.Success {
		if findSubstring(result.Stderr, "blocked") {
			t.Errorf("flag -c should not be blocked anymore, got: %q", result.Stderr)
		}
	}
}

func TestExecuteSuggestion_CommandPlaceholderBlockedByPreflight(t *testing.T) {
	r := NewRunner(nil, nil, nil)
	ctx := context.Background()
	item := planner.SuggestionItem{
		Name:   "placeholder-command",
		Action: planner.ActionSpec{Type: "shell_command", Command: "mkdir .github/..."},
	}
	result := r.ExecuteSuggestion(ctx, 1, item)
	if result.Success {
		t.Fatal("expected placeholder command to fail preflight")
	}
	if !findSubstring(result.Stderr, "preflight validation failed") {
		t.Fatalf("expected preflight validation error, got %q", result.Stderr)
	}
}

// === file_read tool tests ===

func TestExecuteSuggestion_FileReadSuccess(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "read-test.txt")
	content := "hello file_read"
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	r := NewRunner(nil, nil, nil)
	ctx := context.Background()
	item := planner.SuggestionItem{
		Name:   "read-file",
		Action: planner.ActionSpec{Type: "file_read", FilePath: filePath},
	}
	result := r.ExecuteSuggestion(ctx, 1, item)
	if !result.Success {
		t.Fatalf("expected Success=true, got Stderr=%q", result.Stderr)
	}
	if result.Stdout != content {
		t.Errorf("expected Stdout=%q, got %q", content, result.Stdout)
	}
}

func TestExecuteSuggestion_FileReadMissing(t *testing.T) {
	r := NewRunner(nil, nil, nil)
	ctx := context.Background()
	item := planner.SuggestionItem{
		Name:   "read-missing",
		Action: planner.ActionSpec{Type: "file_read", FilePath: "/nonexistent/path/file.txt"},
	}
	result := r.ExecuteSuggestion(ctx, 1, item)
	if result.Success {
		t.Error("expected failure for nonexistent file")
	}
}

func TestExecuteSuggestion_FileReadEmptyPath(t *testing.T) {
	r := NewRunner(nil, nil, nil)
	ctx := context.Background()
	item := planner.SuggestionItem{
		Name:   "read-empty-path",
		Action: planner.ActionSpec{Type: "file_read", FilePath: ""},
	}
	result := r.ExecuteSuggestion(ctx, 1, item)
	if result.Success {
		t.Error("expected failure for empty file_path")
	}
}

func TestExecuteSuggestion_FileReadPathEscape(t *testing.T) {
	dir := t.TempDir()
	mgr := dotgitdex.New(dir)
	if err := mgr.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	r := NewRunner(nil, mgr, nil)
	ctx := context.Background()
	item := planner.SuggestionItem{
		Name:   "read-escape",
		Action: planner.ActionSpec{Type: "file_read", FilePath: "../../etc/passwd"},
	}
	result := r.ExecuteSuggestion(ctx, 1, item)
	if result.Success {
		t.Error("expected failure for path traversal")
	}
	if !findSubstring(result.Stderr, "BLOCKED") {
		t.Errorf("expected BLOCKED in stderr, got %q", result.Stderr)
	}
}

func TestExecuteSuggestion_FileReadGitdexBlocked(t *testing.T) {
	r := NewRunner(nil, nil, nil)
	ctx := context.Background()
	item := planner.SuggestionItem{
		Name:   "read-gitdex",
		Action: planner.ActionSpec{Type: "file_read", FilePath: ".gitdex/maintain/output.txt"},
	}
	result := r.ExecuteSuggestion(ctx, 1, item)
	if result.Success {
		t.Error("expected file_read targeting .gitdex to be blocked")
	}
	if !findSubstring(result.Stderr, "BLOCKED") {
		t.Errorf("expected BLOCKED in stderr, got %q", result.Stderr)
	}
}

func TestExecutionLogger_NextRoundResetsState(t *testing.T) {
	dir := t.TempDir()
	mgr := dotgitdex.New(dir)
	if err := mgr.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	logger := NewExecutionLogger(mgr, "sess-1", "maintain")
	logger.LogStep(1, planner.SuggestionItem{Name: "x", Action: planner.ActionSpec{}}, &ExecutionResult{Success: false})
	if len(logger.steps) != 1 {
		t.Fatalf("expected 1 step before NextRound, got %d", len(logger.steps))
	}
	if !logger.hasError {
		t.Error("expected hasError=true after failed step")
	}
	initialRoundID := logger.roundID
	logger.NextRound()
	if len(logger.steps) != 0 {
		t.Errorf("expected 0 steps after NextRound, got %d", len(logger.steps))
	}
	if logger.hasError {
		t.Error("expected hasError=false after NextRound")
	}
	if logger.roundID != initialRoundID+1 {
		t.Errorf("expected roundID %d, got %d", initialRoundID+1, logger.roundID)
	}
}

func TestExecuteSuggestion_IdempotencySkipForDuplicateFileWrite(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "dup.txt")
	r := NewRunner(nil, nil, nil)
	ctx := context.Background()
	item := planner.SuggestionItem{
		Name: "create-file",
		Action: planner.ActionSpec{
			Type:        "file_write",
			FilePath:    filePath,
			FileContent: "hello",
			FileOp:      "create",
		},
	}

	first := r.ExecuteSuggestion(ctx, 1, item)
	if !first.Success {
		t.Fatalf("first execution should succeed, stderr=%q", first.Stderr)
	}
	second := r.ExecuteSuggestion(ctx, 2, item)
	if !second.Success {
		t.Fatalf("second execution should be idempotent success, stderr=%q", second.Stderr)
	}
	if !contains(second.Stdout, "idempotency preflight") {
		t.Fatalf("expected idempotency skip message, got %q", second.Stdout)
	}
}

func TestRecommendRecovery_PermissionDeniedNeedsManual(t *testing.T) {
	r := NewRunner(nil, nil, nil)
	got := r.recommendRecovery(&ExecutionResult{
		Success: false,
		Stderr:  "permission denied",
	})
	if got.Type != "manual" {
		t.Fatalf("expected manual recovery, got %q", got.Type)
	}
}
