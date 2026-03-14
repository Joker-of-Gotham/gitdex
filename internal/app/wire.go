package app

import (
	"log/slog"
	"time"

	"github.com/Joker-of-Gotham/gitdex/internal/config"
	"github.com/Joker-of-Gotham/gitdex/internal/engine"
	gitcli "github.com/Joker-of-Gotham/gitdex/internal/git/cli"
	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
	"github.com/Joker-of-Gotham/gitdex/internal/i18n"
	"github.com/Joker-of-Gotham/gitdex/internal/llm"
	"github.com/Joker-of-Gotham/gitdex/internal/llmfactory"
	"github.com/Joker-of-Gotham/gitdex/internal/platform"
)

type Dependencies struct {
	Config        *config.Config
	GitCLI        *gitcli.CLIExecutor
	StatusWatcher *status.StatusWatcher
	Pipeline      *engine.Pipeline
	LLMProvider   llm.LLMProvider
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

	llmProvider, effectiveLLM := llmfactory.Build(cfg.LLM)

	pipeline := engine.NewPipeline(cfg.Suggestion.Mode)
	if llmProvider != nil {
		pipeline = engine.NewPipelineWithLLM(cfg.Suggestion.Mode, llmProvider, effectiveLLM)
	}
	if effectiveLLM.ContextLength == 0 {
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
