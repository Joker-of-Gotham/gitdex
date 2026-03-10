package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Joker-of-Gotham/gitdex/internal/engine"
	"github.com/Joker-of-Gotham/gitdex/internal/engine/executor"
	"github.com/Joker-of-Gotham/gitdex/internal/git"
	gitcli "github.com/Joker-of-Gotham/gitdex/internal/git/cli"
	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
	"github.com/Joker-of-Gotham/gitdex/internal/llm"
	"github.com/Joker-of-Gotham/gitdex/internal/llm/prompt"
	"github.com/Joker-of-Gotham/gitdex/internal/llm/response"
	"github.com/Joker-of-Gotham/gitdex/internal/memory"
	"github.com/Joker-of-Gotham/gitdex/internal/tui/oplog"
)

type screenMode int

const (
	screenLoading screenMode = iota
	screenLanguageSelect
	screenModelSelect
	screenMain
	screenInput // text input mode for NeedsInput suggestions
	screenGoalInput
	screenWorkflowSelect
)

type modelSelectPhase int

const (
	selectPrimary modelSelectPhase = iota
	selectSecondary
)

const localFileMode = 0o600

type StartupInfo struct {
	GitVersion   string
	GitAvailable bool
	OllamaStatus string
	SystemLang   string
	FirstRun     bool
}

type Model struct {
	width       int
	height      int
	ready       bool
	screen      screenMode
	startupInfo StartupInfo

	watcher     *status.StatusWatcher
	pipeline    *engine.Pipeline
	executor    *executor.CommandExecutor
	gitCLI      *gitcli.CLIExecutor
	llmProvider llm.LLMProvider

	// model selection
	availModels       []llm.ModelInfo
	modelCursor       int
	modelSelectPhase  modelSelectPhase
	selectedPrimary   string
	selectedSecondary string
	secondaryEnabled  bool
	modelsFetched     bool

	// language selection
	languageCursor     int
	languageConfigured bool
	languageReturnTo   screenMode

	gitState          *status.GitState
	suggestions       []git.Suggestion
	suggExecState     []git.ExecState // parallel to suggestions; tracks per-item status
	suggExecMsg       []string        // parallel to suggestions; short result text
	suggIdx           int
	expanded          bool
	statusMsg         string
	mode              string
	llmReason         string
	llmAnalysis       string // LLM's analysis of repo state (shown in TUI)
	llmThinking       string // LLM's reasoning from <think> tags (shown when expanded)
	analysisSeq       int
	pendingAnalysisID int
	explainSeq        int
	pendingExplainID  int
	skipNextAnalysis  bool // when true, gitStateMsg will not trigger LLM analysis
	execSuggIdx       int  // index of the suggestion currently being executed (-1 = none)

	// Operation log panel state
	opLog           *oplog.Log
	logExpanded     bool
	logScrollOffset int

	// Text input state (for NeedsInput suggestions)
	inputFields   []git.InputField // fields to fill
	inputIdx      int              // which field is being edited
	inputValues   []string         // current values for each field
	inputCursorAt int              // cursor position within the current value
	inputSuggRef  *git.Suggestion  // the suggestion being parameterized

	// Goal input and workflow menu
	goalInput      string
	goalCursorAt   int
	workflowCursor int
	workflows      []workflowDefinition

	// Session memory
	session         SessionContext
	analysisHistory []string
	memoryStore     *memory.Store

	// Plan rendering
	llmPlanOverview string
	llmGoalStatus   string

	// Diagnostics (shown in analysis panel for transparency)
	llmDebugInfo  string
	analysisTrace engine.AnalysisTrace
	lastCommand   commandTrace
	obsTab        observabilityTab
	workflowStage workflowStage
	workflowAt    time.Time
}

func NewModel() Model {
	return Model{
		mode:             "zen",
		screen:           screenLoading,
		opLog:            oplog.New(oplog.DefaultMaxEntries),
		logExpanded:      false,
		execSuggIdx:      -1,
		modelSelectPhase: selectPrimary,
		languageReturnTo: screenMain,
		session: SessionContext{
			Preferences: make(map[string]string),
		},
		memoryStore: memory.NewStore(),
		obsTab:      observabilityWorkflow,
	}
}

func (m Model) SetStartupInfo(info StartupInfo) Model {
	m.startupInfo = info
	return m
}

func (m Model) SetWatcher(w *status.StatusWatcher) Model {
	m.watcher = w
	return m
}

func (m Model) SetPipeline(p *engine.Pipeline) Model {
	m.pipeline = p
	return m
}

func (m Model) SetGitCLI(cli *gitcli.CLIExecutor) Model {
	m.gitCLI = cli
	if cli != nil {
		m.executor = executor.NewCommandExecutor(cli)
	}
	return m
}

func (m Model) SetLLMProvider(p llm.LLMProvider) Model {
	m.llmProvider = p
	return m
}

type gitStateMsg struct {
	state        *status.GitState
	skipAnalysis bool
}
type fileWriteResultMsg struct {
	path       string
	backupPath string
	err        error
}
type llmResultMsg struct {
	requestID    int
	suggestions  []git.Suggestion
	analysis     string
	thinking     string
	planOverview string
	goalStatus   string
	debugInfo    string
	trace        engine.AnalysisTrace
	err          error
}
type commandResultMsg struct {
	result *git.ExecutionResult
	err    error
}
type ollamaModelsMsg struct {
	models []llm.ModelInfo
	err    error
}
type llmExplainMsg struct {
	requestID int
	text      string
	err       error
}

type ollamaCheckTickMsg struct{}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.refreshGitState, m.fetchOllamaModels)
}

func scheduleOllamaCheck() tea.Cmd {
	return func() tea.Msg {
		time.Sleep(15 * time.Second)
		return ollamaCheckTickMsg{}
	}
}

func (m Model) fetchOllamaModels() tea.Msg {
	if m.llmProvider == nil {
		return ollamaModelsMsg{}
	}
	if !m.llmProvider.IsAvailable(context.Background()) {
		return ollamaModelsMsg{}
	}
	models, err := m.llmProvider.ListModels(context.Background())
	return ollamaModelsMsg{models: models, err: err}
}

func (m Model) refreshGitState() tea.Msg {
	if m.watcher == nil {
		return nil
	}
	state, err := m.watcher.GetStatus(context.Background())
	if err != nil {
		return gitStateMsg{state: &status.GitState{}}
	}
	return gitStateMsg{state: state}
}

func (m Model) refreshGitStateOnly() tea.Msg {
	if m.watcher == nil {
		return nil
	}
	state, err := m.watcher.GetStatus(context.Background())
	if err != nil {
		return gitStateMsg{state: &status.GitState{}, skipAnalysis: true}
	}
	return gitStateMsg{state: state, skipAnalysis: true}
}

func (m Model) executeFileOp(fo *git.FileWriteInfo) tea.Cmd {
	return func() tea.Msg {
		op := strings.ToLower(fo.Operation)
		if op == "" {
			op = "create"
		}
		var err error
		var backupPath string

		absPath, pathErr := filepath.Abs(fo.Path)
		if pathErr != nil {
			return fileWriteResultMsg{path: fo.Path, err: pathErr}
		}

		switch op {
		case "create":
			dir := filepath.Dir(absPath)
			if dir != "." && dir != "" {
				_ = os.MkdirAll(dir, 0o755)
			}
			err = os.WriteFile(absPath, []byte(fo.Content), localFileMode)

		case "update":
			if fo.Backup {
				if data, readErr := os.ReadFile(absPath); readErr == nil {
					backupPath = absPath + ".bak"
					_ = os.WriteFile(backupPath, data, localFileMode)
				}
			}
			err = os.WriteFile(absPath, []byte(fo.Content), localFileMode)

		case "append":
			var existing []byte
			if data, readErr := os.ReadFile(absPath); readErr == nil {
				existing = data
			}
			dir := filepath.Dir(absPath)
			if dir != "." && dir != "" {
				_ = os.MkdirAll(dir, 0o755)
			}
			newContent := append(existing, []byte(fo.Content)...)
			err = os.WriteFile(absPath, newContent, localFileMode)

		case "delete":
			if fo.Backup {
				if data, readErr := os.ReadFile(absPath); readErr == nil {
					backupPath = absPath + ".bak"
					_ = os.WriteFile(backupPath, data, localFileMode)
				}
			}
			err = os.Remove(absPath)

		default:
			err = fmt.Errorf("unknown file operation: %s", op)
		}

		return fileWriteResultMsg{path: absPath, backupPath: backupPath, err: err}
	}
}

// runLLMAnalysis sends the full Git state to the LLM and returns its analysis.
func (m Model) runLLMAnalysis(requestID int, state *status.GitState) tea.Cmd {
	recentOps := m.collectRecentOps()
	session := m.session.ToPromptContext()

	var memCtx *prompt.MemoryContext
	if m.memoryStore != nil && state != nil {
		remoteURL := ""
		if len(state.RemoteInfos) > 0 {
			remoteURL = state.RemoteInfos[0].URL
		}
		fp := memory.RepoFingerprint(remoteURL, state.LocalBranch.Name)
		memCtx = m.memoryStore.ToPromptMemory(fp)

		if state.CommitSummaryInfo != nil && state.CommitSummaryInfo.UsesConventional {
			m.memoryStore.SetRepoCommitStyle(fp, "conventional")
			_ = m.memoryStore.Save()
		}
	}

	opts := engine.AnalyzeOptions{
		Session:         &session,
		AnalysisHistory: append([]string(nil), m.analysisHistory...),
		Memory:          memCtx,
	}
	return func() tea.Msg {
		if m.pipeline == nil || state == nil {
			return llmResultMsg{requestID: requestID, err: fmt.Errorf("not ready")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		result, err := m.pipeline.Analyze(ctx, state, recentOps, opts)
		if err != nil {
			return llmResultMsg{requestID: requestID, err: err}
		}

		d := result.DebugInfo
		debugStr := fmt.Sprintf("sys:%d usr:%d budget:%d parse:%s",
			d.SystemPromptTokens, d.UserPromptTokens, d.ContextBudget, d.ParseLog)
		if d.PartitionSummary != "" {
			debugStr += " | " + d.PartitionSummary
		}

		return llmResultMsg{
			requestID:    requestID,
			suggestions:  result.Suggestions,
			analysis:     result.Analysis,
			thinking:     result.Thinking,
			planOverview: result.PlanOverview,
			goalStatus:   result.GoalStatus,
			debugInfo:    debugStr,
			trace:        result.Trace,
		}
	}
}

// collectRecentOps extracts the last few command results from the opLog
// so the LLM knows what was already tried.
func (m Model) collectRecentOps() []prompt.OperationRecord {
	if m.opLog == nil {
		return nil
	}
	entries := m.opLog.Latest(50)
	var ops []prompt.OperationRecord
	for _, e := range entries {
		switch e.Type {
		case oplog.EntryCmdSuccess:
			summary := strings.TrimSpace(e.Summary)
			switch {
			case strings.HasPrefix(summary, "Command succeeded: "):
				ops = append(ops, prompt.OperationRecord{
					Type:    "executed",
					Command: strings.TrimPrefix(summary, "Command succeeded: "),
					Result:  "success",
				})
			case strings.HasPrefix(summary, "File operation succeeded: "):
				ops = append(ops, prompt.OperationRecord{
					Type:    "executed",
					Action:  "file_op:" + strings.TrimPrefix(summary, "File operation succeeded: "),
					Command: "file_op " + strings.TrimPrefix(summary, "File operation succeeded: "),
					Result:  "success",
				})
			}
		case oplog.EntryCmdFail:
			summary := strings.TrimSpace(e.Summary)
			switch {
			case strings.HasPrefix(summary, "Command failed: "):
				ops = append(ops, prompt.OperationRecord{
					Type:    "executed",
					Command: strings.TrimPrefix(summary, "Command failed: "),
					Result:  "failed: " + e.Detail,
				})
			case strings.HasPrefix(summary, "File write failed: "):
				ops = append(ops, prompt.OperationRecord{
					Type:    "executed",
					Action:  "file_op:" + strings.TrimPrefix(summary, "File write failed: "),
					Command: "file_op " + strings.TrimPrefix(summary, "File write failed: "),
					Result:  "failed: " + e.Detail,
				})
			}
		case oplog.EntryUserAction:
			summary := strings.TrimSpace(e.Summary)
			switch {
			case strings.HasPrefix(summary, "Viewed advisory: "):
				ops = append(ops, prompt.OperationRecord{
					Type:   "viewed",
					Action: strings.TrimPrefix(summary, "Viewed advisory: "),
					Result: "viewed",
				})
			case strings.HasPrefix(summary, "Skipped suggestion: "):
				ops = append(ops, prompt.OperationRecord{
					Type:   "skipped",
					Action: strings.TrimPrefix(summary, "Skipped suggestion: "),
					Result: "user skipped",
				})
			case strings.HasPrefix(summary, "Input cancelled"):
				ops = append(ops, prompt.OperationRecord{
					Type:   "cancelled",
					Action: "input",
					Result: "user cancelled input",
				})
			case strings.HasPrefix(summary, "Switched AI mode to "):
				mode := strings.TrimSpace(strings.TrimPrefix(summary, "Switched AI mode to "))
				ops = append(ops, prompt.OperationRecord{
					Type:   "mode_switch",
					Mode:   mode,
					Result: "mode switched",
				})
			}
		}
	}
	if len(ops) > 20 {
		ops = ops[len(ops)-20:]
	}
	return ops
}

func (m *Model) revalidatePendingSuggestions() {
	if m.gitState == nil || len(m.suggestions) == 0 {
		return
	}
	validated := engine.ValidateSuggestionsAgainstState(m.suggestions, m.gitState)
	if len(validated) != len(m.suggestions) {
		removed := len(m.suggestions) - len(validated)
		m.suggestions = validated
		m.suggExecState = make([]git.ExecState, len(validated))
		m.suggExecMsg = make([]string, len(validated))
		if m.suggIdx >= len(validated) {
			m.suggIdx = 0
		}
		*m = m.addLog(oplog.Entry{
			Type:    oplog.EntryStateRefresh,
			Summary: fmt.Sprintf("State change: removed %d stale suggestion(s)", removed),
		})
	}
}

func (m Model) executeCommand(args []string) tea.Cmd {
	return func() tea.Msg {
		if m.executor == nil || len(args) == 0 {
			return commandResultMsg{err: nil}
		}
		// If single-string command, use Execute which does shell-like parsing.
		// If multi-token command, use ExecuteTokenized to preserve arg boundaries.
		if len(args) == 1 {
			result, err := m.executor.Execute(context.Background(), args)
			return commandResultMsg{result: result, err: err}
		}
		result, err := m.executor.ExecuteTokenized(context.Background(), args)
		return commandResultMsg{result: result, err: err}
	}
}

func (m Model) llmExplainSuggestion(requestID int, s git.Suggestion, state *status.GitState) tea.Cmd {
	return func() tea.Msg {
		if m.llmProvider == nil {
			return llmExplainMsg{requestID: requestID, text: s.Reason, err: fmt.Errorf("LLM not available")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		builder := prompt.NewBuilder()
		system, user := builder.BuildExplain(s.Action, joinCmd(s.Command), s.Reason, state)
		resp, err := llm.GenerateText(ctx, m.llmProvider, llm.GenerateRequest{
			System:      system,
			Prompt:      user,
			Temperature: 0.3,
		})
		if err != nil {
			return llmExplainMsg{requestID: requestID, text: s.Reason, err: err}
		}
		cleaned := response.StripThinking(resp.Text)
		return llmExplainMsg{requestID: requestID, text: cleaned}
	}
}

func humanSize(b int64) string {
	const gb = 1024 * 1024 * 1024
	const mb = 1024 * 1024
	switch {
	case b >= gb:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(gb))
	case b >= mb:
		return fmt.Sprintf("%.0f MB", float64(b)/float64(mb))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

func riskLabel(r git.RiskLevel) string {
	switch r {
	case git.RiskSafe:
		return "safe"
	case git.RiskCaution:
		return "caution"
	case git.RiskDangerous:
		return "dangerous"
	}
	return "safe"
}

func joinCmd(parts []string) string {
	quoted := make([]string, 0, len(parts))
	for _, part := range parts {
		if strings.ContainsAny(part, " \t\"") {
			escaped := strings.ReplaceAll(part, `"`, `\"`)
			quoted = append(quoted, `"`+escaped+`"`)
			continue
		}
		quoted = append(quoted, part)
	}
	return strings.Join(quoted, " ")
}

func (m Model) addLog(entry oplog.Entry) Model {
	if m.opLog == nil {
		m.opLog = oplog.New(oplog.DefaultMaxEntries)
	}
	m.opLog.Add(entry)
	return m
}

func (m Model) buildAnalysisStartDetail() string {
	if m.gitState == nil {
		return ""
	}
	s := m.gitState
	var parts []string
	parts = append(parts, fmt.Sprintf("working:%d staged:%d commits:%d", len(s.WorkingTree), len(s.StagingArea), s.CommitCount))
	if s.FileInspect != nil {
		parts = append(parts, "file-inspect:yes")
	}
	if s.CommitSummaryInfo != nil {
		parts = append(parts, "commit-summary:yes")
	}
	if s.ConfigInfo != nil {
		parts = append(parts, "config:yes")
	}
	if goal := strings.TrimSpace(m.session.ActiveGoal); goal != "" {
		parts = append(parts, "goal:"+goal)
	}
	return strings.Join(parts, " | ")
}
