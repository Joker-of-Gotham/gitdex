package engine

import (
	"context"
	"fmt"
	"strings"

	"github.com/Joker-of-Gotham/gitdex/internal/config"
	enginectx "github.com/Joker-of-Gotham/gitdex/internal/engine/context"
	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
	"github.com/Joker-of-Gotham/gitdex/internal/llm"
	llmctx "github.com/Joker-of-Gotham/gitdex/internal/llm/context"
	"github.com/Joker-of-Gotham/gitdex/internal/llm/prompt"
	"github.com/Joker-of-Gotham/gitdex/internal/llm/response"
	"github.com/Joker-of-Gotham/gitdex/internal/platform"
)

type AnalyzeResult struct {
	Suggestions      []git.Suggestion
	Analysis         string
	Thinking         string
	PlanOverview     string
	GoalStatus       string
	KnowledgeRequest []string
	DebugInfo        AnalyzeDebugInfo
	Trace            AnalysisTrace
}

// AnalyzeDebugInfo provides transparency about the analysis pipeline.
type AnalyzeDebugInfo struct {
	SystemPromptTokens int
	UserPromptTokens   int
	ContextBudget      int
	PartitionSummary   string // e.g. "repo:420 goal:85 ops:210 files:300"
	ParseLog           string // e.g. "direct:fail -> repair:ok"
}

type AnalyzeOptions struct {
	Session         *prompt.SessionContext
	Workflow        *prompt.WorkflowOrchestration
	AnalysisHistory []string
	PlatformState   *prompt.PlatformState
	Memory          *prompt.MemoryContext
	Knowledge       []prompt.KnowledgeFragment
}

type Pipeline struct {
	llmProvider       llm.LLMProvider
	mode              string
	primaryModel      string
	secondaryModel    string
	secondaryOn       bool
	contextBudget     int
	platformCollector *platform.Collector
	retriever         *enginectx.Retriever
}

func NewPipeline(mode string) *Pipeline {
	return &Pipeline{mode: strings.ToLower(strings.TrimSpace(mode))}
}

func NewPipelineWithLLM(mode string, llmProvider llm.LLMProvider, opts interface{}) *Pipeline {
	p := &Pipeline{
		llmProvider: llmProvider,
		mode:        strings.ToLower(strings.TrimSpace(mode)),
		retriever:   enginectx.NewRetriever(),
	}
	switch v := opts.(type) {
	case config.LLMConfig:
		p.applyLLMConfig(v)
	case *config.LLMConfig:
		if v != nil {
			p.applyLLMConfig(*v)
		}
	}
	return p
}

func (p *Pipeline) Analyze(ctx context.Context, state *status.GitState, recentOps []prompt.OperationRecord, opts AnalyzeOptions) (*AnalyzeResult, error) {
	return p.analyzeInternal(ctx, state, recentOps, opts, false)
}

func (p *Pipeline) analyzeInternal(ctx context.Context, state *status.GitState, recentOps []prompt.OperationRecord, opts AnalyzeOptions, isKnowledgeRetry bool) (*AnalyzeResult, error) {
	if state == nil {
		return nil, fmt.Errorf("git state is required for analysis")
	}
	if p.llmProvider == nil || !p.llmProvider.IsAvailable(ctx) {
		return nil, fmt.Errorf("LLM not available; configure a supported provider and model")
	}

	optsLocal := opts
	if p.platformCollector != nil && optsLocal.PlatformState == nil {
		if platformState := p.platformCollector.Collect(ctx, state); platformState != nil {
			if optsLocal.Session != nil {
				platform.ApplyGoalRecommendations(platformState, optsLocal.Session.ActiveGoal, 5)
			}
			optsLocal.PlatformState = toPromptPlatformState(platformState)
		}
	}
	if len(optsLocal.Knowledge) == 0 && p.retriever != nil {
		activeGoal := ""
		if optsLocal.Session != nil {
			activeGoal = optsLocal.Session.ActiveGoal
		}
		optsLocal.Knowledge = p.retriever.RetrieveWithGoal(state, activeGoal)
	}
	fileCtx := CollectFileContext(ctx, state)

	var knowledgeCatalog []prompt.KnowledgeCatalogEntry
	if p.retriever != nil {
		for _, entry := range p.retriever.Catalog() {
			knowledgeCatalog = append(knowledgeCatalog, prompt.KnowledgeCatalogEntry{
				ID:       entry.ID,
				Source:   entry.Source,
				Summary:  entry.Summary,
				Triggers: entry.Triggers,
			})
		}
	}

	builder := prompt.NewBuilderWithBudget(p.contextBudget)
	input := prompt.AnalyzeInput{
		State:            state,
		Mode:             p.mode,
		RecentOps:        recentOps,
		Session:          optsLocal.Session,
		Workflow:         optsLocal.Workflow,
		AnalysisHistory:  optsLocal.AnalysisHistory,
		PlatformState:    optsLocal.PlatformState,
		Memory:           optsLocal.Memory,
		Knowledge:        optsLocal.Knowledge,
		KnowledgeCatalog: knowledgeCatalog,
		FileContext:      fileCtx,
	}
	system, user := builder.BuildAnalyzeRich(input)
	buildTrace := builder.LastBuildTrace()

	sysTokens := llmctx.EstimateTokens(system)
	userTokens := llmctx.EstimateTokens(user)

	debug := AnalyzeDebugInfo{
		SystemPromptTokens: sysTokens,
		UserPromptTokens:   userTokens,
		ContextBudget:      buildTrace.Budget,
		PartitionSummary:   builder.LastPartitionSummary(),
	}
	trace := AnalysisTrace{
		Mode:           p.mode,
		PrimaryModel:   p.primaryModel,
		SecondaryModel: p.secondaryModel,
		Budget:         buildTrace.Budget,
		Reserved:       buildTrace.Reserved,
		Available:      buildTrace.Available,
		SystemPrompt:   buildTrace.SystemPrompt,
		UserPrompt:     buildTrace.UserPrompt,
		Partitions:     append([]prompt.PartitionTrace(nil), buildTrace.Partitions...),
		RecentOps:      append([]prompt.OperationRecord(nil), recentOps...),
		Knowledge:      append([]prompt.KnowledgeFragment(nil), optsLocal.Knowledge...),
		Memory:         optsLocal.Memory,
		PlatformState:  optsLocal.PlatformState,
		Workflow:       optsLocal.Workflow,
	}

	resp, err := llm.GenerateText(ctx, p.llmProvider, llm.GenerateRequest{
		Model:       p.primaryModel,
		Role:        llm.RolePrimary,
		System:      system,
		Prompt:      user,
		Temperature: builder.Temperature(),
	})
	if err != nil {
		return nil, fmt.Errorf("LLM: %w", err)
	}
	trace.RawResponse = strings.TrimSpace(resp.Raw)
	if trace.RawResponse == "" {
		trace.RawResponse = resp.Text
	}

	tagThinking, cleaned := response.ExtractThinking(resp.Text)
	thinking := strings.TrimSpace(resp.Thinking)
	if thinking == "" {
		thinking = tagThinking
	} else if tagThinking != "" && !strings.Contains(thinking, tagThinking) {
		thinking = thinking + "\n\n" + tagThinking
	}
	trace.CleanedResponse = cleaned
	parsed, parseErr := parseLLMResponse(state, cleaned)
	if parseErr != nil {
		debug.ParseLog = "parse failed: " + parseErr.Error()
		raw := normalizedRawResponseForDisplay(trace.RawResponse, cleaned)
		return &AnalyzeResult{
			Analysis:  fmt.Sprintf("AI output could not be parsed into the expected JSON suggestion format.\n\nRaw response:\n%s", truncateForDisplay(raw, 800)),
			Thinking:  thinking,
			DebugInfo: debug,
			Trace:     trace,
		}, nil
	}
	debug.ParseLog = fmt.Sprintf("ok (%d suggestions)", len(parsed.suggestions))
	trace.Rejected = append([]string(nil), parsed.rejected...)

	parsed.suggestions = suppressRepeatedSuccessfulSuggestions(parsed.suggestions, recentOps)
	parsed.suggestions = suppressSemanticDuplicates(parsed.suggestions, recentOps)
	validSuggestions, validationIssues := ValidateSuggestionsWithIssues(parsed.suggestions, state)
	parsed.suggestions = validSuggestions
	for _, issue := range validationIssues {
		trace.Rejected = append(trace.Rejected, issue.Reason)
	}
	if len(validationIssues) > 0 {
		repaired, repairIssues, repairErr := p.repairRejectedSuggestions(ctx, state, validationIssues)
		if repairErr == nil && len(repaired) > 0 {
			parsed.suggestions = append(parsed.suggestions, repaired...)
			debug.ParseLog += fmt.Sprintf(" -> repaired:%d", len(repaired))
			if strings.TrimSpace(parsed.analysis) != "" {
				parsed.analysis = strings.TrimSpace(parsed.analysis + fmt.Sprintf("\n\nRepaired %d invalid suggestion(s) to match the current repository state.", len(repaired)))
			}
		} else if repairErr != nil {
			debug.ParseLog += " -> repair_failed"
			trace.Rejected = append(trace.Rejected, "repair failed: "+repairErr.Error())
		}
		for _, issue := range repairIssues {
			trace.Rejected = append(trace.Rejected, "repair rejected: "+issue.Reason)
		}
	}
	if p.secondaryOn && strings.TrimSpace(p.secondaryModel) != "" {
		verified, fixes, verifyErr := newVerifier(p.llmProvider, p.secondaryModel).Verify(ctx, state, parsed.suggestions)
		if verifyErr == nil {
			parsed.suggestions = verified
			if fixes > 0 {
				parsed.analysis = strings.TrimSpace(parsed.analysis + fmt.Sprintf("\n\nVerifier model corrected %d suggestion(s).", fixes))
			}
		}
	}

	if !isKnowledgeRetry && len(parsed.knowledgeRequest) > 0 && len(parsed.suggestions) == 0 && p.retriever != nil {
		extra := p.retriever.FetchByIDs(parsed.knowledgeRequest)
		if len(extra) > 0 {
			optsLocal.Knowledge = append(optsLocal.Knowledge, extra...)
			return p.analyzeInternal(ctx, state, recentOps, optsLocal, true)
		}
	}

	return &AnalyzeResult{
		Suggestions:      parsed.suggestions,
		Analysis:         parsed.analysis,
		Thinking:         thinking,
		PlanOverview:     parsed.planOverview,
		GoalStatus:       parsed.goalStatus,
		KnowledgeRequest: parsed.knowledgeRequest,
		DebugInfo:        debug,
		Trace:            trace,
	}, nil
}

func (p *Pipeline) SetMode(mode string) {
	p.mode = strings.ToLower(strings.TrimSpace(mode))
}

func (p *Pipeline) SetPrimaryModel(model string) {
	p.primaryModel = strings.TrimSpace(model)
	if p.llmProvider != nil && p.primaryModel != "" {
		p.llmProvider.SetModelForRole(llm.RolePrimary, p.primaryModel)
	}
}

func (p *Pipeline) SetSecondaryModel(model string, enabled bool) {
	p.secondaryModel = strings.TrimSpace(model)
	p.secondaryOn = enabled && p.secondaryModel != ""
	if p.llmProvider != nil && p.secondaryModel != "" {
		p.llmProvider.SetModelForRole(llm.RoleSecondary, p.secondaryModel)
	}
}

func (p *Pipeline) SetContextBudget(tokens int) {
	p.contextBudget = tokens
}

func (p *Pipeline) SetLLMProvider(provider llm.LLMProvider, cfg config.LLMConfig) {
	p.llmProvider = provider
	p.applyLLMConfig(cfg)
}

func (p *Pipeline) applyLLMConfig(cfg config.LLMConfig) {
	primary := strings.TrimSpace(cfg.Primary.Model)
	if primary == "" {
		primary = strings.TrimSpace(cfg.Model)
	}
	if primary != "" {
		p.primaryModel = primary
	}
	secondary := strings.TrimSpace(cfg.Secondary.Model)
	p.secondaryModel = secondary
	p.secondaryOn = cfg.Secondary.Enabled && secondary != ""
	p.contextBudget = cfg.ContextLength
	if p.llmProvider != nil {
		if p.primaryModel != "" {
			p.llmProvider.SetModelForRole(llm.RolePrimary, p.primaryModel)
		}
		if p.secondaryModel != "" {
			p.llmProvider.SetModelForRole(llm.RoleSecondary, p.secondaryModel)
		}
	}
}

func (p *Pipeline) SetPlatformCollector(c *platform.Collector) {
	p.platformCollector = c
}

func toPromptPlatformState(in *platform.PlatformState) *prompt.PlatformState {
	if in == nil {
		return nil
	}
	out := &prompt.PlatformState{
		Detected:      in.Detected,
		DefaultBranch: in.DefaultBranch,
		CIStatus:      in.CIStatus,
		Capabilities:  append([]string(nil), in.Capabilities...),
		AdminSummary:  append([]string(nil), in.AdminSummary...),
		SurfaceStates: append([]string(nil), in.SurfaceStates...),
		LastError:     in.LastError,
	}
	out.OpenPRs = make([]prompt.PRSummary, 0, len(in.OpenPRs))
	out.Playbooks = make([]prompt.CapabilityPlaybook, 0, len(in.Playbooks))
	for _, pr := range in.OpenPRs {
		out.OpenPRs = append(out.OpenPRs, prompt.PRSummary{
			Number: pr.Number,
			Title:  pr.Title,
			URL:    pr.URL,
		})
	}
	for _, playbook := range in.Playbooks {
		out.Playbooks = append(out.Playbooks, prompt.CapabilityPlaybook{
			ID:       playbook.ID,
			Label:    playbook.Label,
			Category: playbook.Category,
			DocsURL:  playbook.DocsURL,
			Inspect:  append([]string(nil), playbook.Inspect...),
			Apply:    append([]string(nil), playbook.Apply...),
			Verify:   append([]string(nil), playbook.Verify...),
			Score:    playbook.Score,
		})
	}
	return out
}
