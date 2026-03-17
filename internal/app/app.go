package app

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Joker-of-Gotham/gitdex/internal/collector"
	"github.com/Joker-of-Gotham/gitdex/internal/config"
	"github.com/Joker-of-Gotham/gitdex/internal/dotgitdex"
	"github.com/Joker-of-Gotham/gitdex/internal/executor"
	"github.com/Joker-of-Gotham/gitdex/internal/flow"
	gitcli "github.com/Joker-of-Gotham/gitdex/internal/git/cli"
	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
	"github.com/Joker-of-Gotham/gitdex/internal/helper"
	"github.com/Joker-of-Gotham/gitdex/internal/i18n"
	"github.com/Joker-of-Gotham/gitdex/internal/knowledge"
	"github.com/Joker-of-Gotham/gitdex/internal/llm"
	"github.com/Joker-of-Gotham/gitdex/internal/llm/ollama"
	"github.com/Joker-of-Gotham/gitdex/internal/llmfactory"
	"github.com/Joker-of-Gotham/gitdex/internal/observability"
	"github.com/Joker-of-Gotham/gitdex/internal/planner"
	"github.com/Joker-of-Gotham/gitdex/internal/tui"
	"github.com/Joker-of-Gotham/gitdex/internal/tui/theme"
)

type Config struct {
	Version string
}

type Application struct {
	config Config
}

func New(cfg Config) *Application {
	return &Application{config: cfg}
}

func (a *Application) Run() error {
	cfg, err := config.Load()
	if err != nil {
		slog.Warn("config load failed, using defaults", "err", err)
		cfg = config.DefaultConfig()
		config.Set(cfg)
	}

	if err := i18n.Init(cfg.I18n.Language); err != nil {
		slog.Warn("i18n init failed", "err", err)
	}

	theme.Init(cfg.Theme.Name)
	theme.InitIcons()

	gitBinary := resolveGitBinary(cfg)
	repoRoot := detectRepoRoot(gitBinary)
	store := dotgitdex.New(repoRoot)
	if err := store.Init(); err != nil {
		slog.Warn("failed to init .gitdex directory", "err", err)
	}

	if err := knowledge.Extract(store); err != nil {
		slog.Warn("knowledge extraction failed", "err", err)
	}

	gitCLI, err := gitcli.NewCLIExecutorWithBinary(gitBinary)
	if err != nil {
		slog.Error("git not available", "err", err)
		return err
	}

	watcher := status.NewStatusWatcher(gitCLI)
	watcher.SetAutoFetchInterval(time.Duration(cfg.Sync.AutoFetchInterval) * time.Second)

	autoSelectOllamaModel(cfg)
	autoDetectContextLength(cfg)
	helperLLM, plannerLLM := buildDualLLM(cfg)

	mode := resolveMode(cfg)
	language := cfg.I18n.Language

	sessionID := time.Now().Format("20060102-150405")
	logger := executor.NewExecutionLogger(store, sessionID, mode)
	runner := executor.NewRunner(gitCLI, store, logger)

	knReader := knowledge.NewReader(store.KnowledgeDir())
	gitCollector := collector.NewGitCollector(watcher, store)
	ghCollector := collector.NewGitHubCollector()

	ks := &helper.KnowledgeSelector{LLM: helperLLM, Store: store, Language: language}
	gm := &helper.GoalMaintainer{LLM: helperLLM, Store: store, Language: language}
	pr := &helper.ProposalReviewer{LLM: helperLLM, Store: store, Language: language}

	mp := &planner.MaintenancePlanner{LLM: plannerLLM, Language: language}
	gp := &planner.GoalPlanner{LLM: plannerLLM, Language: language}
	cp := &planner.CreativePlanner{LLM: plannerLLM, Language: language}

	interval := 2400 * time.Second
	if cfg.Automation.MonitorInterval > 0 {
		interval = time.Duration(cfg.Automation.MonitorInterval) * time.Second
	}

	ctxLimit := cfg.LLM.ContextLength
	if ctxLimit <= 0 {
		ctxLimit = 32768
	}

	orch := &flow.Orchestrator{
		Maintain: &flow.MaintainFlow{
			Collector:    gitCollector,
			Helper:       ks,
			Planner:      mp,
			Store:        store,
			KnReader:     knReader,
			ContextLimit: ctxLimit,
		},
		Goal: &flow.GoalFlow{
			Collector:    gitCollector,
			Helper:       ks,
			GoalHelper:   gm,
			Planner:      gp,
			Store:        store,
			KnReader:     knReader,
			ContextLimit: ctxLimit,
		},
		Creative: &flow.CreativeFlow{
			GitCollector: gitCollector,
			GHCollector:  ghCollector,
			Planner:      cp,
			Reviewer:     pr,
			Store:        store,
			ContextLimit: ctxLimit,
		},
		Runner:   runner,
		Logger:   logger,
		Mode:     mode,
		Interval: interval,
	}

	// Planner uses RolePrimary (cfg.LLM.Primary), Helper uses RoleSecondary (cfg.LLM.Secondary).
	plannerProv := cfg.LLM.Primary.Provider
	if plannerProv == "" {
		plannerProv = cfg.LLM.Provider
	}
	plannerModel := cfg.LLM.Primary.Model
	if plannerModel == "" {
		plannerModel = cfg.LLM.Model
	}
	plannerEP := cfg.LLM.Primary.Endpoint
	if plannerEP == "" {
		plannerEP = cfg.LLM.Endpoint
	}

	helperProv := cfg.LLM.Secondary.Provider
	if helperProv == "" {
		helperProv = plannerProv
	}
	helperModel := cfg.LLM.Secondary.Model
	helperEP := cfg.LLM.Secondary.Endpoint
	if helperEP == "" {
		helperEP = plannerEP
	}

	cfgSnap := tui.ConfigSnapshot{
		Planner: tui.LLMRoleSnapshot{
			Provider: plannerProv, Model: plannerModel, Endpoint: plannerEP,
			APIKeyEnv: cfg.LLM.Primary.APIKeyEnv,
		},
		Helper: tui.LLMRoleSnapshot{
			Provider: helperProv, Model: helperModel, Endpoint: helperEP,
			APIKeyEnv: cfg.LLM.Secondary.APIKeyEnv,
		},
		Language:       language,
		Theme:          cfg.Theme.Name,
		RepoRoot:       repoRoot,
		CruiseInterval: cfg.Automation.MonitorInterval,
	}

	model := tui.NewModel(orch, store, mode, language, cfgSnap, helperLLM, plannerLLM)
	prog := tea.NewProgram(model)
	_, err = prog.Run()
	return err
}

func buildDualLLM(cfg *config.Config) (helperLLM, plannerLLM llm.LLMProvider) {
	router, _, diag := llmfactory.BuildWithDiagnostics(cfg.LLM)
	if router == nil {
		observability.SetProviderAvailability(false)
		primary := cfg.LLM.PrimaryRole()
		secondary := cfg.LLM.SecondaryRole()
		slog.Warn("no LLM provider available — using NopProvider (configure via /config)",
			"primary_provider", config.RoleProvider(primary),
			"primary_model", primary.Model,
			"primary_key_present", config.ResolveRoleAPIKey(primary) != "",
			"primary_api_key_env", primary.APIKeyEnv,
			"secondary_enabled", cfg.LLM.Secondary.Enabled,
			"secondary_provider", config.RoleProvider(secondary),
			"secondary_model", secondary.Model,
			"secondary_key_present", config.ResolveRoleAPIKey(secondary) != "",
			"secondary_api_key_env", secondary.APIKeyEnv,
			"diag_primary_health", diag.Primary.Health,
			"diag_primary_code", diag.Primary.Code,
			"diag_primary_reason", diag.Primary.Reason,
			"diag_secondary_health", diag.Secondary.Health,
			"diag_secondary_code", diag.Secondary.Code,
			"diag_secondary_reason", diag.Secondary.Reason,
			"diag_fallback_promoted", diag.FallbackPromoted,
		)
		nop := llm.NopProvider{}
		return nop, nop
	}
	observability.SetProviderAvailability(true)
	slog.Info("LLM control-plane ready",
		"primary_health", diag.Primary.Health,
		"primary_code", diag.Primary.Code,
		"secondary_health", diag.Secondary.Health,
		"secondary_code", diag.Secondary.Code,
		"fallback_promoted", diag.FallbackPromoted,
	)
	return router, router
}

// autoSelectOllamaModel scans locally available Ollama models at startup.
// If the configured model doesn't exist locally, it auto-selects the first
// available model and persists the change.
func autoSelectOllamaModel(cfg *config.Config) {
	provider := strings.ToLower(strings.TrimSpace(cfg.LLM.Primary.Provider))
	if provider == "" {
		provider = strings.ToLower(strings.TrimSpace(cfg.LLM.Provider))
	}
	if provider != "ollama" {
		return
	}

	endpoint := strings.TrimSpace(cfg.LLM.Primary.Endpoint)
	if endpoint == "" {
		endpoint = strings.TrimSpace(cfg.LLM.Endpoint)
	}
	if endpoint == "" {
		endpoint = config.DefaultOllamaEndpoint
	}

	client := ollama.NewClient(endpoint, "")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	models, err := client.ListModels(ctx)
	if err != nil {
		slog.Warn("ollama: cannot list local models", "err", err)
		return
	}
	if len(models) == 0 {
		slog.Warn("ollama: no local models found, run 'ollama pull <model>' to download one")
		return
	}

	configuredModel := strings.TrimSpace(cfg.LLM.Primary.Model)
	if configuredModel == "" {
		configuredModel = strings.TrimSpace(cfg.LLM.Model)
	}

	found := false
	for _, m := range models {
		if m.Name == configuredModel {
			found = true
			break
		}
	}

	if found {
		return
	}

	selected := models[0].Name
	slog.Warn("ollama: configured model not found locally, auto-selecting",
		"configured", configuredModel, "selected", selected,
		"available", modelNames(models))

	cfg.LLM.Model = selected
	cfg.LLM.Primary.Model = selected
	config.Set(cfg)
	if err := config.SaveGlobal(cfg); err != nil {
		slog.Warn("ollama: failed to save auto-selected model", "err", err)
	}
}

func modelNames(models []llm.ModelInfo) []string {
	names := make([]string, len(models))
	for i, m := range models {
		names[i] = m.Name
	}
	return names
}

// autoDetectContextLength probes the Ollama model metadata to determine the
// actual context window, then stores it in config so the flow layer can budget.
func autoDetectContextLength(cfg *config.Config) {
	if cfg.LLM.ContextLength > 0 {
		return
	}
	provider := strings.ToLower(strings.TrimSpace(cfg.LLM.Primary.Provider))
	if provider == "" {
		provider = strings.ToLower(strings.TrimSpace(cfg.LLM.Provider))
	}
	if provider != "ollama" {
		return
	}
	endpoint := strings.TrimSpace(cfg.LLM.Primary.Endpoint)
	if endpoint == "" {
		endpoint = strings.TrimSpace(cfg.LLM.Endpoint)
	}
	if endpoint == "" {
		endpoint = config.DefaultOllamaEndpoint
	}
	model := strings.TrimSpace(cfg.LLM.Primary.Model)
	if model == "" {
		model = strings.TrimSpace(cfg.LLM.Model)
	}
	if model == "" {
		return
	}

	client := ollama.NewClient(endpoint, model)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	detected := client.DetectModelContext(ctx, model)
	if detected > 0 {
		cfg.LLM.ContextLength = detected
		slog.Info("ollama: auto-detected context length", "model", model, "context_length", detected)
	}
}

// detectRepoRoot uses git rev-parse --show-toplevel to find the repository root.
// Falls back to os.Getwd() if git is not available.
func detectRepoRoot(gitBinary string) string {
	if strings.TrimSpace(gitBinary) == "" {
		gitBinary = "git"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, gitBinary, "rev-parse", "--show-toplevel").Output()
	if err == nil {
		root := strings.TrimSpace(string(out))
		if root != "" {
			return root
		}
	}
	cwd, _ := os.Getwd()
	return cwd
}

func resolveGitBinary(cfg *config.Config) string {
	if cfg != nil && strings.TrimSpace(cfg.Adapters.Git.Binary) != "" {
		return strings.TrimSpace(cfg.Adapters.Git.Binary)
	}
	return "git"
}

func resolveMode(cfg *config.Config) string {
	mode := cfg.Automation.Mode
	switch mode {
	case "manual", "auto", "cruise":
		return mode
	default:
		return "manual"
	}
}
