package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Joker-of-Gotham/gitdex/internal/config"
	"github.com/Joker-of-Gotham/gitdex/internal/engine"
	"github.com/Joker-of-Gotham/gitdex/internal/engine/executor"
	"github.com/Joker-of-Gotham/gitdex/internal/git"
	gitcli "github.com/Joker-of-Gotham/gitdex/internal/git/cli"
	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
	"github.com/Joker-of-Gotham/gitdex/internal/i18n"
	"github.com/Joker-of-Gotham/gitdex/internal/llm"
	"github.com/Joker-of-Gotham/gitdex/internal/llm/ollama"
	"github.com/Joker-of-Gotham/gitdex/internal/llm/prompt"
	"github.com/Joker-of-Gotham/gitdex/internal/llm/response"
	"github.com/Joker-of-Gotham/gitdex/internal/memory"
	"github.com/Joker-of-Gotham/gitdex/internal/platform"
	"github.com/Joker-of-Gotham/gitdex/internal/tui/oplog"
)

type screenMode int

const (
	screenLoading screenMode = iota
	screenLanguageSelect
	screenModelSelect
	screenProviderConfig
	screenAutomationConfig
	screenMain
	screenInput // text input mode for NeedsInput suggestions
	screenGoalInput
	screenWorkflowSelect
	screenPlatformEdit
	screenFileEdit
)

type modelSelectPhase int

const (
	selectPrimary modelSelectPhase = iota
	selectSecondary
)

type modelSelectMode int

const (
	modelSelectProviders modelSelectMode = iota
	modelSelectModels
)

const localFileMode = 0o600

type StartupInfo struct {
	GitVersion   string
	GitAvailable bool
	AIStatus     string
	SystemLang   string
	FirstRun     bool
}

type scrollPane int

const (
	scrollPaneWorkspace scrollPane = iota
	scrollPaneLog
	scrollPaneAreas
	scrollPaneObservability
)

type workspaceTab int

const (
	workspaceTabOverview workspaceTab = iota
	workspaceTabSuggestions
	workspaceTabResult
	workspaceTabAnalysis
)

type providerField int

const (
	providerFieldProvider providerField = iota
	providerFieldModel
	providerFieldEndpoint
	providerFieldAPIKey
)

type Model struct {
	width       int
	height      int
	ready       bool
	screen      screenMode
	startupInfo StartupInfo
	llmConfig   config.LLMConfig
	automation  config.AutomationConfig
	platformCfg config.PlatformConfig
	adapterCfg  config.AdapterConfig
	reportsCfg  config.ReportsConfig

	watcher     *status.StatusWatcher
	pipeline    *engine.Pipeline
	executor    *executor.CommandExecutor
	gitCLI      *gitcli.CLIExecutor
	llmProvider llm.LLMProvider

	// model selection
	availModels       []llm.ModelInfo
	availModelsSource string
	modelCursor       int
	modelSelectPhase  modelSelectPhase
	modelSelectMode   modelSelectMode
	modelListProvider string
	selectedPrimary   string
	selectedSecondary string
	secondaryEnabled  bool
	modelsFetched     bool
	primaryProvider   string
	secondaryProvider string
	providerRole      modelSelectPhase
	providerDraft     config.ModelConfig
	providerField     providerField
	providerCursorAt  int
	providerStoredKey string
	providerKeyDirty  bool

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
	goalInput       string
	goalCursorAt    int
	workflowCursor  int
	workflowScroll  int
	workflows       []workflowDefinition
	composerInput   string
	composerCursor  int
	slashCursor     int
	composerFocused bool
	platformEdit    string
	platformCursor  int
	platformScroll  int
	platformTitle   string
	fileEditReq     *git.FileWriteInfo
	fileEdit        string
	fileCursor      int
	fileScroll      int
	fileTitle       string

	// Session memory
	session         SessionContext
	analysisHistory []string
	memoryStore     *memory.Store
	workflowPlan    *prompt.WorkflowOrchestration
	workflowFlow    *workflowFlowState

	// Plan rendering
	llmPlanOverview string
	llmGoalStatus   string

	// Diagnostics (shown in analysis panel for transparency)
	llmDebugInfo            string
	analysisTrace           engine.AnalysisTrace
	lastCommand             commandTrace
	lastPlatformOp          *git.PlatformExecInfo
	lastPlatform            *platformActionState
	mutationLedger          []platform.MutationLedgerEntry
	obsTab                  observabilityTab
	workflowStage           workflowStage
	workflowAt              time.Time
	leftScroll              int
	areasScroll             int
	obsScroll               int
	scrollFocus             scrollPane
	autoSteps                  int
	consecutiveAnalysisFailures  int
	consecutiveEmptySuggestions int
	scheduleLastRun         map[string]time.Time
	automationLocks         map[string]string
	automationFailures      map[string]int
	automationObserveOnly   bool
	automationDraft         config.AutomationConfig
	automationField         automationField
	lastEscalation          time.Time
	lastRecovery            time.Time
	lastAnalysisFingerprint string
	cachedPlatformID        platform.Platform
	loadedCheckpointRepo    string
	pendingCheckpointGoal   string
	pendingCheckpointWf     *prompt.WorkflowOrchestration
	pendingCheckpointFlow   *workflowFlowState
	lastCheckpointHash      string
	lastReportExportHash    string
	lastReportExportAt      time.Time
	commandResponseTitle    string
	commandResponseBody     string
	workspaceTab            workspaceTab
	batchRunRequested       bool
	resolveAdminBundle      adminBundleResolver
	renderCache             *renderCache
}

func NewModel() Model {
	state := loadAutomationCheckpoint()
	lastRuns := state.ScheduleLastRun
	if lastRuns == nil {
		lastRuns = map[string]time.Time{}
	}
	locks := state.AutomationLocks
	if locks == nil {
		locks = map[string]string{}
	}
	failures := state.AutomationFailures
	if failures == nil {
		failures = map[string]int{}
	}
	m := Model{
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
		memoryStore:             memory.NewStore(),
		pendingCheckpointGoal:   strings.TrimSpace(state.ActiveGoal),
		pendingCheckpointWf:     state.Workflow,
		pendingCheckpointFlow:   state.Flow,
		mutationLedger:          append([]platform.MutationLedgerEntry(nil), state.Ledger...),
		obsTab:                observabilityWorkflow,
		scrollFocus:           scrollPaneWorkspace,
		scheduleLastRun:       lastRuns,
		automationLocks:       locks,
		automationFailures:    failures,
		automationObserveOnly: state.ObserveOnly,
		lastEscalation:        state.EscalatedAt,
		lastRecovery:          state.RecoveredAt,
		workspaceTab:          workspaceTabOverview,
		loadedCheckpointRepo:  strings.TrimSpace(state.RepoFingerprint),
		renderCache:           newRenderCache(),
	}
	return m.syncSessionLanguagePreference()
}

func (m Model) SetStartupInfo(info StartupInfo) Model {
	m.startupInfo = info
	return m
}

func (m Model) SetLLMConfig(cfg config.LLMConfig) Model {
	m.llmConfig = cfg
	primary := cfg.PrimaryRole()
	secondary := cfg.SecondaryRole()
	m.primaryProvider = config.RoleProvider(primary)
	m.secondaryProvider = config.RoleProvider(secondary)
	m.selectedPrimary = strings.TrimSpace(primary.Model)
	if cfg.Secondary.Enabled {
		m.selectedSecondary = strings.TrimSpace(secondary.Model)
	}
	m.secondaryEnabled = cfg.Secondary.Enabled && strings.TrimSpace(secondary.Model) != ""
	return m
}

func (m Model) SetAutomationConfig(cfg config.AutomationConfig) Model {
	config.ApplyAutomationMode(&cfg)
	m.automation = cfg
	return m
}

func (m Model) SetPlatformConfig(cfg config.PlatformConfig) Model {
	m.platformCfg = cfg
	return m
}

func (m Model) SetAdapterConfig(cfg config.AdapterConfig) Model {
	m.adapterCfg = cfg
	return m
}

func (m Model) SetReportsConfig(cfg config.ReportsConfig) Model {
	m.reportsCfg = cfg
	return m
}

func (m Model) openModelSetup(role modelSelectPhase) Model {
	m.modelSelectPhase = role
	m.modelSelectMode = modelSelectProviders
	m.modelListProvider = ""
	m.modelCursor = providerOptionIndex(m.roleProvider(role))
	m.screen = screenModelSelect
	m.statusMsg = i18n.T("model_select.opened")
	return m
}

func (m Model) openLocalModelSelection(role modelSelectPhase, provider string) Model {
	m.modelSelectPhase = role
	m.modelSelectMode = modelSelectModels
	if strings.TrimSpace(provider) == "" {
		provider = m.roleProvider(role)
	}
	if strings.TrimSpace(provider) == "" {
		provider = "ollama"
	}
	m.modelListProvider = provider
	target := m.roleModel(role)
	m.modelCursor = 0
	for i, model := range m.currentSelectableModels() {
		if strings.TrimSpace(model.Name) == target {
			m.modelCursor = i
			break
		}
	}
	m.screen = screenModelSelect
	return m
}

func (m Model) modelsForProvider(provider string) []llm.ModelInfo {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "" {
		return append([]llm.ModelInfo(nil), m.availModels...)
	}
	out := make([]llm.ModelInfo, 0, len(m.availModels))
	for _, model := range m.availModels {
		if strings.EqualFold(strings.TrimSpace(model.Provider), provider) {
			out = append(out, model)
		}
	}
	if len(out) > 0 {
		return out
	}
	if strings.EqualFold(m.availModelsSource, provider) {
		return append([]llm.ModelInfo(nil), m.availModels...)
	}
	return nil
}

func (m Model) currentSelectableModels() []llm.ModelInfo {
	provider := strings.TrimSpace(m.modelListProvider)
	if provider == "" {
		provider = m.roleProvider(m.modelSelectPhase)
	}
	models := m.modelsForProvider(provider)
	if len(models) > 0 {
		return models
	}
	if strings.TrimSpace(provider) != "" {
		return nil
	}
	return append([]llm.ModelInfo(nil), m.availModels...)
}

func (m Model) hasModelsForProvider(provider string) bool {
	return len(m.modelsForProvider(provider)) > 0
}

func (m Model) persistRoleModelSelection(role modelSelectPhase, provider, model string) (Model, error) {
	next := config.Get()
	if next == nil {
		next = config.DefaultConfig()
	}
	cfg := *next

	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "" {
		provider = "ollama"
	}
	model = strings.TrimSpace(model)
	spec := llm.ProviderSpecFor(provider)

	if role == selectSecondary {
		roleCfg := cfg.LLM.SecondaryRole()
		roleCfg.Provider = provider
		roleCfg.Model = model
		roleCfg.Enabled = model != ""
		if strings.TrimSpace(roleCfg.Endpoint) == "" || config.RoleProvider(roleCfg) != provider {
			roleCfg.Endpoint = spec.DefaultBaseURL
		}
		if provider == "ollama" {
			roleCfg.APIKey = ""
			roleCfg.APIKeyEnv = ""
		} else if strings.TrimSpace(roleCfg.APIKeyEnv) == "" {
			roleCfg.APIKeyEnv = spec.APIKeyEnv
		}
		cfg.LLM.Secondary = roleCfg
	} else {
		roleCfg := cfg.LLM.PrimaryRole()
		roleCfg.Provider = provider
		roleCfg.Model = model
		roleCfg.Enabled = model != ""
		if strings.TrimSpace(roleCfg.Endpoint) == "" || config.RoleProvider(roleCfg) != provider {
			roleCfg.Endpoint = spec.DefaultBaseURL
		}
		if provider == "ollama" {
			roleCfg.APIKey = ""
			roleCfg.APIKeyEnv = ""
		} else if strings.TrimSpace(roleCfg.APIKeyEnv) == "" {
			roleCfg.APIKeyEnv = spec.APIKeyEnv
		}
		cfg.LLM.Provider = provider
		cfg.LLM.Model = model
		cfg.LLM.Endpoint = roleCfg.Endpoint
		cfg.LLM.APIKey = roleCfg.APIKey
		cfg.LLM.APIKeyEnv = roleCfg.APIKeyEnv
		cfg.LLM.Primary = roleCfg
	}

	if err := config.SaveGlobal(&cfg); err != nil {
		return m, err
	}

	config.Set(&cfg)
	m = m.applyLLMConfigRuntime(cfg.LLM)
	return m, nil
}

func (m Model) roleProvider(role modelSelectPhase) string {
	if role == selectSecondary && strings.TrimSpace(m.secondaryProvider) != "" {
		return m.secondaryProvider
	}
	if strings.TrimSpace(m.primaryProvider) != "" {
		return m.primaryProvider
	}
	return "ollama"
}

func (m Model) roleModel(role modelSelectPhase) string {
	if role == selectSecondary {
		return strings.TrimSpace(m.selectedSecondary)
	}
	return strings.TrimSpace(m.selectedPrimary)
}

func (m Model) selectedProviderID() string {
	options := providerOptions()
	if len(options) == 0 {
		return "ollama"
	}
	if m.modelCursor < 0 || m.modelCursor >= len(options) {
		return m.roleProvider(m.modelSelectPhase)
	}
	return options[m.modelCursor]
}

func (m Model) selectedProviderSpec() llm.ProviderSpec {
	return llm.ProviderSpecFor(m.selectedProviderID())
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
	path          string
	backupPath    string
	operation     string
	beforeContent string
	afterContent  string
	err           error
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
type providerModelsMsg struct {
	models    []llm.ModelInfo
	err       error
	available bool
}
type setupProviderModelsMsg struct {
	provider string
	models   []llm.ModelInfo
	err      error
}
type llmExplainMsg struct {
	requestID int
	text      string
	err       error
}

type providerCheckTickMsg struct{}
type automationTickMsg struct{}
type autoRetryAnalysisMsg struct{}

type paneScrollMsg struct {
	pane  scrollPane
	delta int
}

type paneFocusMsg struct {
	pane scrollPane
}

type uiClickMsg struct {
	action string
	index  int
}

func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{m.refreshGitState, m.fetchProviderModels}
	if m.automation.Enabled && m.automation.MonitorInterval > 0 {
		cmds = append(cmds, scheduleAutomationTick(time.Duration(m.automation.MonitorInterval)*time.Second))
	}
	return tea.Batch(cmds...)
}

func scheduleProviderCheck() tea.Cmd {
	return func() tea.Msg {
		time.Sleep(15 * time.Second)
		return providerCheckTickMsg{}
	}
}

func scheduleAutomationTick(interval time.Duration) tea.Cmd {
	return func() tea.Msg {
		if interval <= 0 {
			interval = 300 * time.Second
		}
		time.Sleep(interval)
		return automationTickMsg{}
	}
}

func (m Model) fetchProviderModels() tea.Msg {
	if m.llmProvider == nil {
		return providerModelsMsg{}
	}
	if !m.llmProvider.IsAvailable(context.Background()) {
		return providerModelsMsg{available: false}
	}
	models, err := m.llmProvider.ListModels(context.Background())
	return providerModelsMsg{models: models, err: err, available: true}
}

func (m Model) fetchSetupProviderModels(provider string) tea.Msg {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider != "ollama" {
		return setupProviderModelsMsg{provider: provider}
	}

	endpoint := defaultProviderEndpoint(provider)
	role := m.llmConfig.PrimaryRole()
	if m.modelSelectPhase == selectSecondary {
		role = m.llmConfig.SecondaryRole()
	}
	if config.RoleProvider(role) == provider && strings.TrimSpace(role.Endpoint) != "" {
		endpoint = config.RoleEndpoint(role)
	}

	client := ollama.NewClient(endpoint, "")
	if !client.IsAvailable(context.Background()) {
		return setupProviderModelsMsg{provider: provider, err: fmt.Errorf("ollama is unavailable at %s", endpoint)}
	}

	models, err := client.ListModels(context.Background())
	return setupProviderModelsMsg{provider: provider, models: models, err: err}
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
		var beforeContent string
		var afterContent string

		absPath, pathErr := filepath.Abs(fo.Path)
		if pathErr != nil {
			return fileWriteResultMsg{path: fo.Path, err: pathErr}
		}
		if data, readErr := os.ReadFile(absPath); readErr == nil {
			beforeContent = string(data)
		}

		switch op {
		case "create":
			dir := filepath.Dir(absPath)
			if dir != "." && dir != "" {
				_ = os.MkdirAll(dir, 0o755)
			}
			err = os.WriteFile(absPath, []byte(fo.Content), localFileMode)
			afterContent = fo.Content

		case "update":
			if fo.Backup {
				if data, readErr := os.ReadFile(absPath); readErr == nil {
					backupPath = absPath + ".bak"
					_ = os.WriteFile(backupPath, data, localFileMode)
				}
			}
			err = os.WriteFile(absPath, []byte(fo.Content), localFileMode)
			afterContent = fo.Content

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
			afterContent = string(newContent)

		case "delete":
			if fo.Backup {
				if data, readErr := os.ReadFile(absPath); readErr == nil {
					backupPath = absPath + ".bak"
					_ = os.WriteFile(backupPath, data, localFileMode)
				}
			}
			err = os.Remove(absPath)
			afterContent = ""

		default:
			err = fmt.Errorf("unknown file operation: %s", op)
		}

		return fileWriteResultMsg{
			path:          absPath,
			backupPath:    backupPath,
			operation:     op,
			beforeContent: beforeContent,
			afterContent:  afterContent,
			err:           err,
		}
	}
}

// runLLMAnalysis sends the full Git state to the LLM and returns its analysis.
func (m Model) runLLMAnalysis(requestID int, state *status.GitState) tea.Cmd {
	recentOps := m.collectRecentOps()
	session := m.session.ToPromptContext()
	session.ActiveGoalStatus = m.currentGoalStatus()

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
		Workflow:        m.workflowPlan,
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
// so the LLM knows what was already tried and what happened.
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
					Output:  truncateOutput(e.Detail, 300),
				})
			case strings.HasPrefix(summary, "File operation succeeded: "):
				ops = append(ops, prompt.OperationRecord{
					Type:    "executed",
					Action:  "file_op:" + strings.TrimPrefix(summary, "File operation succeeded: "),
					Command: "file_op " + strings.TrimPrefix(summary, "File operation succeeded: "),
					Result:  "success",
					Output:  truncateOutput(e.Detail, 200),
				})
			case strings.HasPrefix(summary, "Platform action succeeded: "):
				action := strings.TrimPrefix(summary, "Platform action succeeded: ")
				if identity := extractPlatformIdentity(e.Detail); identity != "" {
					action = identity
				}
				ops = append(ops, prompt.OperationRecord{
					Type:    "executed",
					Action:  action,
					Command: action,
					Result:  "success",
					Output:  truncateOutput(e.Detail, 200),
				})
			}
		case oplog.EntryCmdFail:
			summary := strings.TrimSpace(e.Summary)
			switch {
			case strings.HasPrefix(summary, "Command failed: "):
				ops = append(ops, prompt.OperationRecord{
					Type:    "executed",
					Command: strings.TrimPrefix(summary, "Command failed: "),
					Result:  "failed",
					Output:  truncateOutput(e.Detail, 300),
				})
			case strings.HasPrefix(summary, "File write failed: "):
				ops = append(ops, prompt.OperationRecord{
					Type:    "executed",
					Action:  "file_op:" + strings.TrimPrefix(summary, "File write failed: "),
					Command: "file_op " + strings.TrimPrefix(summary, "File write failed: "),
					Result:  "failed",
					Output:  truncateOutput(e.Detail, 200),
				})
			case strings.HasPrefix(summary, "Platform action failed: "):
				action := strings.TrimPrefix(summary, "Platform action failed: ")
				if identity := extractPlatformIdentity(e.Detail); identity != "" {
					action = identity
				}
				ops = append(ops, prompt.OperationRecord{
					Type:    "executed",
					Action:  action,
					Command: action,
					Result:  "failed",
					Output:  truncateOutput(e.Detail, 200),
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

func truncateOutput(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if s == "" || maxLen <= 0 {
		return ""
	}
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "…(truncated)"
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
		if len(args) == 0 {
			return commandResultMsg{err: fmt.Errorf("suggestion has no executable command")}
		}
		if m.executor == nil {
			return commandResultMsg{err: fmt.Errorf("git executor unavailable")}
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

func extractPlatformIdentity(detail string) string {
	for _, line := range strings.Split(strings.ReplaceAll(detail, "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "identity=") {
			return strings.TrimSpace(strings.TrimPrefix(line, "identity="))
		}
	}
	return ""
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

func (m Model) recordAutomationTransitions(prev, next *status.GitState) Model {
	if !m.automation.Enabled || prev == nil || next == nil {
		return m
	}

	if prev.LocalBranch.Name != next.LocalBranch.Name {
		m = m.addLog(oplog.Entry{
			Type:    oplog.EntryStateRefresh,
			Summary: fmt.Sprintf("Automation observed branch change: %s -> %s", prev.LocalBranch.Name, next.LocalBranch.Name),
		})
		m.rememberOperationEvent("automation:branch:" + prev.LocalBranch.Name + "->" + next.LocalBranch.Name)
	}

	prevAhead, prevBehind := branchDivergence(prev)
	nextAhead, nextBehind := branchDivergence(next)
	if prevAhead != nextAhead || prevBehind != nextBehind {
		m = m.addLog(oplog.Entry{
			Type:    oplog.EntryStateRefresh,
			Summary: fmt.Sprintf("Automation observed sync delta: ahead %d->%d behind %d->%d", prevAhead, nextAhead, prevBehind, nextBehind),
		})
		m.rememberOperationEvent(fmt.Sprintf("automation:sync:ahead%d->%d:behind%d->%d", prevAhead, nextAhead, prevBehind, nextBehind))
	}

	if len(prev.WorkingTree) != len(next.WorkingTree) || len(prev.StagingArea) != len(next.StagingArea) {
		m = m.addLog(oplog.Entry{
			Type:    oplog.EntryStateRefresh,
			Summary: fmt.Sprintf("Automation observed local delta: working %d->%d staged %d->%d", len(prev.WorkingTree), len(next.WorkingTree), len(prev.StagingArea), len(next.StagingArea)),
		})
		m.rememberOperationEvent(fmt.Sprintf("automation:local:working%d->%d:staged%d->%d", len(prev.WorkingTree), len(next.WorkingTree), len(prev.StagingArea), len(next.StagingArea)))
	}

	return m
}

func branchDivergence(state *status.GitState) (ahead, behind int) {
	if state == nil {
		return 0, 0
	}
	ahead = state.LocalBranch.Ahead
	behind = state.LocalBranch.Behind
	if state.UpstreamState != nil {
		ahead = state.UpstreamState.Ahead
		behind = state.UpstreamState.Behind
	}
	return ahead, behind
}

func (m Model) shouldOpenModelSelection(models []llm.ModelInfo) bool {
	if m.primaryProvider != "ollama" {
		return strings.TrimSpace(m.selectedPrimary) == ""
	}
	if strings.TrimSpace(m.selectedPrimary) == "" {
		return true
	}
	for _, model := range models {
		if strings.TrimSpace(model.Name) == strings.TrimSpace(m.selectedPrimary) {
			return false
		}
	}
	return true
}

func (m Model) describeScrollFocus() string {
	switch m.scrollFocus {
	case scrollPaneLog:
		return "Scroll focus: Operation Log"
	case scrollPaneAreas:
		return "Scroll focus: Git Areas"
	case scrollPaneObservability:
		return "Scroll focus: Observability"
	default:
		return "Scroll focus: Main workspace"
	}
}

func (m Model) shouldAutoRefresh() bool {
	return m.shouldAutoAnalyzeOnTick()
}
