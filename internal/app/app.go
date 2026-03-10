package app

import (
	"log/slog"

	tea "charm.land/bubbletea/v2"

	"github.com/Joker-of-Gotham/gitdex/internal/config"
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

	theme.Init(cfg.Theme.Name)
	theme.InitIcons()

	checks := a.runStartupChecks()

	model := tui.NewModel()
	model = model.SetStartupInfo(tui.StartupInfo{
		GitVersion:   checks.GitVersion,
		GitAvailable: checks.GitAvailable,
		OllamaStatus: checks.OllamaStatus,
		SystemLang:   checks.SystemLang,
		FirstRun:     checks.FirstRun,
	})

	deps, err := Wire(cfg)
	if err != nil {
		slog.Warn("dependency wiring failed, running in view-only mode", "err", err)
	} else {
		model = model.SetWatcher(deps.StatusWatcher)
		model = model.SetPipeline(deps.Pipeline)
		model = model.SetGitCLI(deps.GitCLI)
		if deps.LLMProvider != nil {
			model = model.SetLLMProvider(deps.LLMProvider)
		}
	}

	p := tea.NewProgram(model)
	_, err = p.Run()
	return err
}
