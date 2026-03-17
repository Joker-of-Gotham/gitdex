package tui

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Joker-of-Gotham/gitdex/internal/dotgitdex"
	"github.com/Joker-of-Gotham/gitdex/internal/flow"
	"github.com/Joker-of-Gotham/gitdex/internal/llm"
	"github.com/Joker-of-Gotham/gitdex/internal/planner"
	tuictx "github.com/Joker-of-Gotham/gitdex/internal/tui/context"
	"github.com/Joker-of-Gotham/gitdex/internal/tui/components/footer"
	"github.com/Joker-of-Gotham/gitdex/internal/tui/components/header"
	"github.com/Joker-of-Gotham/gitdex/internal/tui/components/sidebar"
	"github.com/Joker-of-Gotham/gitdex/internal/tui/components/tabs"
	"github.com/Joker-of-Gotham/gitdex/internal/tui/oplog"
	"github.com/Joker-of-Gotham/gitdex/internal/tui/theme"
)

// Page represents the current screen/page in the TUI.
type Page int

const (
	PageMain Page = iota
	PageConfig
	PageConfigModel
	PageConfigMode
	PageConfigLang
	PageConfigTheme
)

// FocusZone identifies which panel currently has keyboard/scroll focus.
type FocusZone int

const (
	FocusInput FocusZone = iota
	FocusLeft
	FocusGit
	FocusGoals
	FocusLog
)

type SuggestionStatus int

const (
	StatusPending SuggestionStatus = iota
	StatusExecuting
	StatusDone
	StatusFailed
	StatusSkipped
)

type SuggestionDisplay struct {
	Item   planner.SuggestionItem
	Status SuggestionStatus
	Output string
	Error  string
}

type CompressedRound struct {
	Commands []string
	Flow     string
}

type BranchSnap struct {
	Name      string
	Upstream  string
	Ahead     int
	Behind    int
	IsCurrent bool
	IsMerged  bool
	Last      string
}

type RemoteSnap struct {
	Name     string
	FetchURL string
	PushURL  string
}

// GitSnapshot holds rich git state parsed from git-content.txt.
type GitSnapshot struct {
	Branch        string
	Detached      bool
	HeadRef       string
	DefaultBranch string
	CommitCount   int
	IsInitial     bool

	LocalBranches  []BranchSnap
	RemoteBranches []string
	MergedBranches []string
	Remotes        []RemoteSnap

	Ahead, Behind int
	AheadCommits  []string
	BehindCommits []string

	WorkingDirty int
	StagingDirty int
	WorkingFiles []string
	StagingFiles []string

	MergeInProgress  bool
	RebaseInProgress bool
	CherryInProgress bool
	BisectInProgress bool
	HasConflicts     bool

	Stash        int
	StashEntries []string
	Tags         []string
	Submodules   []string
	RecentReflog []string

	UserName   string
	UserEmail  string
	LastCommit string
	CommitFreq string
}

// LLMRoleSnapshot holds config for one LLM role (helper or planner).
type LLMRoleSnapshot struct {
	Provider  string
	Model     string
	Endpoint  string
	APIKeyEnv string
}

// ConfigSnapshot holds runtime config for display in the TUI.
type ConfigSnapshot struct {
	Helper         LLMRoleSnapshot
	Planner        LLMRoleSnapshot
	Language       string
	Theme          string
	RepoRoot       string
	CruiseInterval int // cruise patrol interval in seconds
}

// Backward-compatible accessors
func (c ConfigSnapshot) Provider() string { return c.Helper.Provider }
func (c ConfigSnapshot) Model() string    { return c.Helper.Model }
func (c ConfigSnapshot) Endpoint() string { return c.Helper.Endpoint }

// ProviderMeta stores static metadata about a supported LLM provider.
type ProviderMeta struct {
	ID                string
	Label             string
	Kind              string
	DefaultBaseURL    string
	APIKeyEnv         string
	RecommendedModels []string
}

var providerMetas = []ProviderMeta{
	{
		ID: "ollama", Label: "Ollama", Kind: "Local model server",
		DefaultBaseURL:    llm.DefaultOllamaURL,
		RecommendedModels: []string{"qwen2.5:3b", "llama3.1:8b", "gemma2:9b", "deepseek-r1:7b"},
	},
	{
		ID: "openai", Label: "OpenAI", Kind: "Cloud API",
		DefaultBaseURL: llm.DefaultOpenAIURL, APIKeyEnv: "OPENAI_API_KEY",
		RecommendedModels: []string{"gpt-4.1-mini", "gpt-4.1", "gpt-4o"},
	},
	{
		ID: "deepseek", Label: "DeepSeek", Kind: "Cloud API",
		DefaultBaseURL: llm.DefaultDeepSeekURL, APIKeyEnv: "DEEPSEEK_API_KEY",
		RecommendedModels: []string{"deepseek-chat", "deepseek-coder"},
	},
}

var draftProviders = []string{"ollama", "openai", "deepseek"}

func providerMetaFor(id string) ProviderMeta {
	for _, m := range providerMetas {
		if m.ID == id {
			return m
		}
	}
	return providerMetas[0]
}

// OllamaModelInfo holds metadata for a locally available ollama model.
type OllamaModelInfo struct {
	Name      string
	ParamSize string
	Family    string
	Quant     string
	Size      int64
}

// LLMRole identifies which LLM role is being configured.
type LLMRole int

const (
	RoleHelper  LLMRole = 0
	RolePlanner LLMRole = 1
)

// ConfigDraft holds in-progress edits on the model config page.
type ConfigDraft struct {
	Role        LLMRole // which role is being configured
	ProviderIdx int     // 0=ollama, 1=openai, 2=deepseek
	Model       string
	Endpoint    string
	APIKeyEnv   string
	FieldIdx    int // 0=role-switch, 1=provider, 2=model, 3=endpoint, 4=apikey
	CursorAt    int // cursor position within the active text field

	PerProviderModel [3]string // saved model per provider to avoid overwriting
}

type Model struct {
	width, height int
	ready         bool

	mode string // manual, auto, cruise
	page Page

	programCtx tuictx.ProgramContext
	headerComp header.Model
	footerComp footer.Model
	sidebarComp sidebar.Model
	tabsComp   tabs.Model

	orchestrator *flow.Orchestrator
	store        *dotgitdex.Manager

	activeFlow   string // maintain, goal, creative, idle
	currentRound *flow.FlowRound
	roundHistory []CompressedRound

	suggestions []SuggestionDisplay
	suggIdx     int

	goals      []dotgitdex.Goal
	activeGoal string

	gitInfo       GitSnapshot
	configInfo    ConfigSnapshot
	lastTokenUsed int // last round's estimated input tokens
	lastTokenMax  int // last round's max token budget

	composerText       string
	composerFocus      bool
	showHelpOverlay    bool
	showCommandPalette bool
	paletteQuery       string
	paletteIdx         int
	logCursor          int
	detailPaneOpen     bool

	// Focus & scroll
	focusZone    FocusZone
	panelScrolls [5]int // indexed by FocusZone

	// Config page navigation
	configMenuIdx   int
	configModeIdx   int
	configLangIdx   int
	configThemeIdx  int
	cruiseIntervalS int  // cruise interval in seconds (editable)
	editingInterval bool // true when editing the interval field
	intervalBuf     string

	// Interactive model config editing
	configDraft   ConfigDraft
	configEditing bool // true when editing a text field

	// Ollama local model list
	ollamaModels     []OllamaModelInfo
	ollamaModelIdx   int
	ollamaFetching   bool
	ollamaFetchError string

	opLog *oplog.Log

	analyzing            bool
	executing            bool
	runAllMode           bool // when /run all is used in manual mode, chain all suggestions
	cruiseCycleActive    bool // true while a cruise cycle (goal→maintain) is in progress
	creativeRanThisSlice bool // true once creative has run in the current time slice; reset on next tick
	cruiseTimer          *time.Ticker
	consecutiveReplans   int      // counts consecutive replan-after-failure, resets on success
	lastSuggestionSigs   []string // signatures of the last round's suggestions for no-progress detection

	language string

	helperLLM  llm.LLMProvider
	plannerLLM llm.LLMProvider
}

func NewModel(orch *flow.Orchestrator, store *dotgitdex.Manager, mode, language string, cfg ConfigSnapshot, providers ...llm.LLMProvider) Model {
	interval := cfg.CruiseInterval
	if interval <= 0 {
		interval = 900
	}

	th := theme.Current
	styles := tuictx.InitStyles(th)

	m := Model{
		orchestrator:    orch,
		store:           store,
		mode:            mode,
		page:            PageMain,
		activeFlow:      "idle",
		opLog:           oplog.New(200),
		language:        language,
		composerFocus:   true,
		focusZone:       FocusInput,
		configInfo:      cfg,
		cruiseIntervalS: interval,
		headerComp:      header.New(),
		footerComp:      footer.New(),
		sidebarComp:     sidebar.New(),
		tabsComp:        tabs.New([]string{"Maintain", "Goal", "Creative"}),
	}
	m.programCtx = tuictx.ProgramContext{
		Theme:  th,
		Styles: styles,
	}
	m.headerComp.UpdateProgramContext(&m.programCtx)
	m.footerComp.UpdateProgramContext(&m.programCtx)
	m.sidebarComp.UpdateProgramContext(&m.programCtx)
	m.tabsComp.UpdateProgramContext(&m.programCtx)

	if len(providers) > 0 && providers[0] != nil {
		m.helperLLM = providers[0]
	}
	if len(providers) > 1 && providers[1] != nil {
		m.plannerLLM = providers[1]
	}
	return m
}

func (m Model) Init() tea.Cmd { return func() tea.Msg { return initMsg{} } }
