package app

import (
	"log/slog"
	"time"

	"github.com/Joker-of-Gotham/gitdex/internal/config"
	"github.com/Joker-of-Gotham/gitdex/internal/engine"
	gitcli "github.com/Joker-of-Gotham/gitdex/internal/git/cli"
	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
	"github.com/Joker-of-Gotham/gitdex/internal/i18n"
	"github.com/Joker-of-Gotham/gitdex/internal/llm/ollama"
	"github.com/Joker-of-Gotham/gitdex/internal/platform"
)

type Dependencies struct {
	Config        *config.Config
	GitCLI        *gitcli.CLIExecutor
	StatusWatcher *status.StatusWatcher
	Pipeline      *engine.Pipeline
	LLMProvider   *ollama.OllamaClient
}

func Wire(cfg *config.Config) (*Dependencies, error) {
	if err := i18n.Init(cfg.I18n.Language); err != nil {
		slog.Warn("i18n init failed, using keys as fallback", "err", err)
	}

	git, err := gitcli.NewCLIExecutor()
	if err != nil {
		return nil, err
	}

	watcher := status.NewStatusWatcher(git)
	watcher.SetAutoFetchInterval(time.Duration(cfg.Sync.AutoFetchInterval) * time.Second)

	var llmProvider *ollama.OllamaClient
	if cfg.LLM.Provider == "ollama" {
		model := cfg.LLM.Primary.Model
		if model == "" {
			model = cfg.LLM.Model
		}
		llmProvider = ollama.NewClient(cfg.LLM.Endpoint, model, cfg.LLM.ContextLength)
	}

	pipeline := engine.NewPipeline(cfg.Suggestion.Mode)
	if llmProvider != nil {
		pipeline = engine.NewPipelineWithLLM(cfg.Suggestion.Mode, llmProvider, cfg.LLM)
	}
	if cfg.LLM.ContextLength == 0 {
		pipeline.SetContextBudget(32768)
	}
	pipeline.SetPlatformCollector(platform.NewCollector(
		cfg.Platform.GitHubToken,
		cfg.Platform.GitLabToken,
		cfg.Platform.BitbucketToken,
	))

	return &Dependencies{
		Config:        cfg,
		GitCLI:        git,
		StatusWatcher: watcher,
		Pipeline:      pipeline,
		LLMProvider:   llmProvider,
	}, nil
}
