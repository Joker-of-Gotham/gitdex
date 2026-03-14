package prompt

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
	llmctx "github.com/Joker-of-Gotham/gitdex/internal/llm/context"
	gitplatform "github.com/Joker-of-Gotham/gitdex/internal/platform"
)

func analyzeSystem(languageCode string) string {
	languageName := "English"
	analysisFieldLanguage := "English"
	switch strings.ToLower(strings.TrimSpace(languageCode)) {
	case "zh":
		languageName = "Simplified Chinese"
		analysisFieldLanguage = "Chinese"
	case "ja":
		languageName = "Japanese"
		analysisFieldLanguage = "Japanese"
	}

	return fmt.Sprintf(`You are a Git operations engine. Analyze the provided context and suggest actions.
Output all text in %s. Reason from provided context.

OUTPUT FORMAT (strict JSON, no markdown fences, no prose outside JSON):
{
  "analysis": "situational analysis in %s",
  "goal_status": "in_progress|completed|blocked (when a goal is set)",
  "knowledge_request": ["source#id", "..."] (optional, request knowledge by ID from catalog),
  "suggestions": [
    {
      "action": "short title",
      "argv": ["git","..."],
      "reason": "why",
      "risk": "safe|caution|dangerous",
      "interaction": "auto|needs_input|info|file_write|platform_exec",
      "file_path": "(for file_write)",
      "file_content": "(for file_write)",
      "file_operation": "create|update|delete|append (for file_write)",
      "capability_id": "(for platform_exec)",
      "flow": "inspect|mutate|validate|rollback (for platform_exec)",
      "operation": "create|update|delete|build|ping (for platform_exec mutate)",
      "resource_id": "(for platform_exec, existing resource)",
      "scope": {},
      "query": {},
      "payload": {},
      "validate_payload": {},
      "rollback_payload": {}
    }
  ]
}

INTERACTION TYPES:
- "auto": executable immediately, argv is a JSON string array
- "needs_input": requires user input, use <placeholder> in argv
- "info": informational observation
- "file_write": file operation (set file_path, file_content, file_operation)
- "platform_exec": platform admin operation (set capability_id, flow, and optional scope/query/payload)

RISK LEVELS: "safe" | "caution" | "dangerous"`, languageName, analysisFieldLanguage)
}

// MemoryContext holds long-term memory data to inject into prompts.
type MemoryContext struct {
	UserPreferences map[string]string `json:"user_preferences,omitempty"`
	RepoPatterns    []string          `json:"repo_patterns,omitempty"`
	ResolvedIssues  []string          `json:"resolved_issues,omitempty"`
	RecentEvents    []string          `json:"recent_events,omitempty"`
	ArtifactNotes   []string          `json:"artifact_notes,omitempty"`
	Episodes        []MemoryEpisode   `json:"episodes,omitempty"`
	SemanticFacts   []SemanticFact    `json:"semantic_facts,omitempty"`
	TaskState       *TaskMemory       `json:"task_state,omitempty"`
}

type EvidenceRef struct {
	Kind  string `json:"kind,omitempty"`
	Ref   string `json:"ref,omitempty"`
	Label string `json:"label,omitempty"`
}

type MemoryEpisode struct {
	ID           string        `json:"id,omitempty"`
	At           time.Time     `json:"at,omitempty"`
	Kind         string        `json:"kind,omitempty"`
	Surface      string        `json:"surface,omitempty"`
	Action       string        `json:"action,omitempty"`
	Summary      string        `json:"summary,omitempty"`
	Result       string        `json:"result,omitempty"`
	WorkflowID   string        `json:"workflow_id,omitempty"`
	CapabilityID string        `json:"capability_id,omitempty"`
	Flow         string        `json:"flow,omitempty"`
	Operation    string        `json:"operation,omitempty"`
	Confidence   float64       `json:"confidence,omitempty"`
	Evidence     []string      `json:"evidence,omitempty"`
	EvidenceRefs []EvidenceRef `json:"evidence_refs,omitempty"`
	LedgerID     string        `json:"ledger_id,omitempty"`
}

type SemanticFact struct {
	Fact          string        `json:"fact,omitempty"`
	Confidence    float64       `json:"confidence,omitempty"`
	Evidence      []string      `json:"evidence,omitempty"`
	EvidenceRefs  []EvidenceRef `json:"evidence_refs,omitempty"`
	LastValidated time.Time     `json:"last_validated,omitempty"`
	Decay         float64       `json:"decay,omitempty"`
	CurrentScore  float64       `json:"current_score,omitempty"`
	Stale         bool          `json:"stale,omitempty"`
}

type TaskMemory struct {
	Goal        string    `json:"goal,omitempty"`
	WorkflowID  string    `json:"workflow_id,omitempty"`
	Status      string    `json:"status,omitempty"`
	Constraints []string  `json:"constraints,omitempty"`
	Pending     []string  `json:"pending,omitempty"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
}

// KnowledgeFragment is a retrieved piece of Git knowledge.
type KnowledgeFragment struct {
	ScenarioID string `json:"scenario_id"`
	Content    string `json:"content"`
}

// PartitionTrace describes a single context partition in the final prompt build.
type PartitionTrace struct {
	Name      string `json:"name"`
	Priority  int    `json:"priority"`
	Tokens    int    `json:"tokens"`
	Required  bool   `json:"required"`
	Included  bool   `json:"included"`
	Truncated bool   `json:"truncated"`
}

// BuildTrace captures the final prompt assembly for observability.
type BuildTrace struct {
	Budget       int              `json:"budget"`
	Reserved     int              `json:"reserved"`
	Available    int              `json:"available"`
	SystemPrompt string           `json:"system_prompt"`
	UserPrompt   string           `json:"user_prompt"`
	Partitions   []PartitionTrace `json:"partitions"`
}

// PromptBuilder constructs prompts from GitState for LLM analysis.
type PromptBuilder struct {
	ContextBudget    int // total context tokens; 0 = no budget management
	lastPartitionLog string
	lastBuildTrace   BuildTrace
}

func NewBuilder() *PromptBuilder { return &PromptBuilder{} }

func NewBuilderWithBudget(contextTokens int) *PromptBuilder {
	return &PromptBuilder{ContextBudget: contextTokens}
}

// LastPartitionSummary returns a human-readable summary of partitions
// included in the last BuildAnalyzeRich call.
func (b *PromptBuilder) LastPartitionSummary() string {
	return b.lastPartitionLog
}

// LastBuildTrace returns the latest prompt-build trace.
func (b *PromptBuilder) LastBuildTrace() BuildTrace {
	return b.lastBuildTrace
}

// OperationRecord summarizes a recently executed operation for LLM context.
type OperationRecord struct {
	Type    string `json:"type,omitempty"` // executed|skipped|cancelled|mode_switch
	Command string `json:"command,omitempty"`
	Action  string `json:"action,omitempty"`
	Result  string `json:"result"`
	Output  string `json:"output,omitempty"`
	Mode    string `json:"mode,omitempty"`
}

type GoalRecord struct {
	Goal      string `json:"goal"`
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

type SessionContext struct {
	ActiveGoal       string            `json:"active_goal,omitempty"`
	ActiveGoalStatus string            `json:"active_goal_status,omitempty"`
	GoalHistory      []GoalRecord      `json:"goal_history,omitempty"`
	SkippedActions   []string          `json:"skipped_actions,omitempty"`
	Preferences      map[string]string `json:"preferences,omitempty"`
}

type WorkflowOrchestrationStep struct {
	Title      string            `json:"title"`
	Rationale  string            `json:"rationale,omitempty"`
	Capability string            `json:"capability_id,omitempty"`
	Flow       string            `json:"flow,omitempty"`
	Operation  string            `json:"operation,omitempty"`
	ResourceID string            `json:"resource_id,omitempty"`
	Scope      map[string]string `json:"scope,omitempty"`
	Query      map[string]string `json:"query,omitempty"`
	Payload    json.RawMessage   `json:"payload,omitempty"`
	Validate   json.RawMessage   `json:"validate_payload,omitempty"`
	Rollback   json.RawMessage   `json:"rollback_payload,omitempty"`
}

type WorkflowOrchestration struct {
	WorkflowID    string                      `json:"workflow_id"`
	WorkflowLabel string                      `json:"workflow_label"`
	Goal          string                      `json:"goal,omitempty"`
	Capabilities  []string                    `json:"capabilities,omitempty"`
	Steps         []WorkflowOrchestrationStep `json:"steps,omitempty"`
}

type PRSummary struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	URL    string `json:"url"`
}

type CapabilityPlaybook struct {
	ID       string   `json:"id"`
	Label    string   `json:"label"`
	Category string   `json:"category"`
	DocsURL  string   `json:"docs_url,omitempty"`
	Inspect  []string `json:"inspect,omitempty"`
	Apply    []string `json:"apply,omitempty"`
	Verify   []string `json:"verify,omitempty"`
	Score    int      `json:"score,omitempty"`
}

type PlatformState struct {
	Detected      string               `json:"detected"`
	DefaultBranch string               `json:"default_branch,omitempty"`
	CIStatus      string               `json:"ci_status,omitempty"`
	OpenPRs       []PRSummary          `json:"open_prs,omitempty"`
	Capabilities  []string             `json:"capabilities,omitempty"`
	AdminSummary  []string             `json:"admin_summary,omitempty"`
	SurfaceStates []string             `json:"surface_states,omitempty"`
	Playbooks     []CapabilityPlaybook `json:"playbooks,omitempty"`
	LastError     string               `json:"last_error,omitempty"`
}

// KnowledgeCatalogEntry is a lightweight index entry for the knowledge catalog partition.
type KnowledgeCatalogEntry struct {
	ID       string   `json:"id"`
	Source   string   `json:"source"`
	Summary  string   `json:"summary"`
	Triggers []string `json:"triggers"`
}

// AnalyzeInput holds all data sources for analysis prompt construction.
type AnalyzeInput struct {
	State            *status.GitState
	Mode             string
	RecentOps        []OperationRecord
	Session          *SessionContext
	Workflow         *WorkflowOrchestration
	AnalysisHistory  []string
	PlatformState    *PlatformState
	Memory           *MemoryContext
	Knowledge        []KnowledgeFragment
	KnowledgeCatalog []KnowledgeCatalogEntry
	FileContext      *FileContext
}

// FileContext holds important file contents.
type FileContext struct {
	Files map[string]string // path -> content
}

// BuildAnalyzeRich constructs prompts with full data sources and budget management.
func (b *PromptBuilder) BuildAnalyzeRich(input AnalyzeInput) (system, user string) {
	systemPrompt := analyzeSystem(preferredOutputLanguage(input.Session))

	budget := b.ContextBudget
	if budget <= 0 {
		budget = 32768
	}

	reserved := 2048
	mgr := llmctx.NewBudgetManager(budget, reserved)

	var partitions []llmctx.Partition

	// Critical state (always included)
	partitions = append(partitions, llmctx.Partition{
		Name:     "repository_context",
		Priority: llmctx.PrioCriticalState,
		Content:  b.formatFullContext(input.State, input.Mode),
		Required: true,
	})

	// User goal
	if input.Session != nil && strings.TrimSpace(input.Session.ActiveGoal) != "" {
		partitions = append(partitions, llmctx.Partition{
			Name:     "user_goal",
			Priority: llmctx.PrioUserGoal,
			Content:  b.formatSessionContext(*input.Session),
			Required: true,
		})
	} else if input.Session != nil {
		partitions = append(partitions, llmctx.Partition{
			Name:     "session_context",
			Priority: llmctx.PrioSessionHistory,
			Content:  b.formatSessionContext(*input.Session),
		})
	}

	if input.Workflow != nil && len(input.Workflow.Steps) > 0 {
		partitions = append(partitions, llmctx.Partition{
			Name:     "workflow_orchestration",
			Priority: llmctx.PrioExtendedState,
			Content:  b.formatWorkflowOrchestration(*input.Workflow),
		})
	}

	// Knowledge fragments
	if len(input.Knowledge) > 0 {
		var kbParts []string
		for _, k := range input.Knowledge {
			kbParts = append(kbParts, fmt.Sprintf("[%s]\n%s", k.ScenarioID, k.Content))
		}
		partitions = append(partitions, llmctx.Partition{
			Name:     "knowledge",
			Priority: llmctx.PrioKnowledge,
			Content:  "Relevant Git SOP/knowledge for current state:\n" + strings.Join(kbParts, "\n\n"),
		})
	}

	// Knowledge catalog (lightweight index of all available scenarios)
	if len(input.KnowledgeCatalog) > 0 {
		catalogJSON, _ := json.Marshal(input.KnowledgeCatalog)
		partitions = append(partitions, llmctx.Partition{
			Name:     "knowledge_catalog",
			Priority: llmctx.PrioKnowledgeCatalog,
			Content:  "Available knowledge base scenarios (request by ID via \"knowledge_request\" field if you need detailed SOP):\n" + string(catalogJSON),
		})
	}

	// Recent operations
	if len(input.RecentOps) > 0 {
		partitions = append(partitions, llmctx.Partition{
			Name:     "recent_ops",
			Priority: llmctx.PrioRecentOps,
			Content:  b.formatRecentOps(input.RecentOps),
		})
	}

	// File inspection
	if input.State.FileInspect != nil {
		partitions = append(partitions, llmctx.Partition{
			Name:     "file_inspection",
			Priority: llmctx.PrioFileInspect,
			Content:  b.formatFileInspection(input.State.FileInspect),
		})
	}

	// Commit summary
	if input.State.CommitSummaryInfo != nil {
		partitions = append(partitions, llmctx.Partition{
			Name:     "commit_summary",
			Priority: llmctx.PrioCommitSummary,
			Content:  b.formatCommitSummary(input.State.CommitSummaryInfo),
		})
	}

	// Config state
	if input.State.ConfigInfo != nil {
		partitions = append(partitions, llmctx.Partition{
			Name:     "config_state",
			Priority: llmctx.PrioConfigState,
			Content:  b.formatConfigState(input.State.ConfigInfo),
		})
	}

	// Analysis history
	if len(input.AnalysisHistory) > 0 {
		partitions = append(partitions, llmctx.Partition{
			Name:     "analysis_history",
			Priority: llmctx.PrioSessionHistory,
			Content:  b.formatAnalysisHistory(input.AnalysisHistory),
		})
	}

	// Platform state
	if input.PlatformState != nil {
		partitions = append(partitions, llmctx.Partition{
			Name:     "platform_state",
			Priority: llmctx.PrioPlatformState,
			Content:  b.formatPlatformState(*input.PlatformState),
		})
		if guidance := b.formatPlatformExecGuidance(input.PlatformState, input.Session); guidance != "" {
			partitions = append(partitions, llmctx.Partition{
				Name:     "platform_exec_schema",
				Priority: llmctx.PrioPlatformState,
				Content:  guidance,
			})
		}
	}

	// Long-term memory
	if input.Memory != nil {
		partitions = append(partitions, llmctx.Partition{
			Name:     "long_term_memory",
			Priority: llmctx.PrioLongTermMemory,
			Content:  b.formatMemoryContext(*input.Memory),
		})
	}

	// File context (important files for LLM to read/modify)
	if input.FileContext != nil && len(input.FileContext.Files) > 0 {
		partitions = append(partitions, llmctx.Partition{
			Name:     "file_context",
			Priority: llmctx.PrioFileInspect,
			Content:  b.formatFileContext(input.FileContext),
		})
	}

	system, user, usage := mgr.AssembleDetailed(systemPrompt, partitions)

	trace := BuildTrace{
		Budget:       budget,
		Reserved:     reserved,
		Available:    mgr.AvailableTokens(),
		SystemPrompt: system,
		UserPrompt:   user,
		Partitions:   make([]PartitionTrace, 0, len(usage)),
	}

	var logParts []string
	for _, entry := range usage {
		traceEntry := PartitionTrace{
			Name:      entry.Name,
			Priority:  int(entry.Priority),
			Tokens:    entry.Tokens,
			Required:  entry.Required,
			Included:  entry.Included,
			Truncated: entry.Truncated,
		}
		trace.Partitions = append(trace.Partitions, traceEntry)
		if entry.Tokens == 0 {
			continue
		}
		prefix := "-"
		if entry.Included {
			prefix = "+"
		}
		if entry.Truncated {
			prefix = "~"
		}
		logParts = append(logParts, fmt.Sprintf("%s%s:%d", prefix, entry.Name, entry.Tokens))
	}
	b.lastBuildTrace = trace
	b.lastPartitionLog = strings.Join(logParts, " ")

	return system, user
}

func preferredOutputLanguage(session *SessionContext) string {
	if session == nil || len(session.Preferences) == 0 {
		return "en"
	}
	lang := strings.ToLower(strings.TrimSpace(session.Preferences["language"]))
	switch lang {
	case "zh", "ja", "en":
		return lang
	default:
		return "en"
	}
}

func (b *PromptBuilder) formatRecentOps(ops []OperationRecord) string {
	data, err := json.Marshal(ops)
	if err != nil {
		return ""
	}
	return "Recently executed operations and user decisions:\n" +
		"- \"executed\" entries show commands that were run and their results\n" +
		"- \"skipped\" entries show suggestions the user explicitly rejected; avoid repeating them\n" +
		"- \"cancelled\" entries show input flows the user abandoned; consider alternatives\n" +
		"- \"mode_switch\" entries indicate user preference changes\n" +
		string(data)
}

func (b *PromptBuilder) formatSessionContext(session SessionContext) string {
	var sb strings.Builder

	if len(session.Preferences) > 0 {
		if lang := strings.TrimSpace(session.Preferences["language"]); lang != "" {
			sb.WriteString("PREFERRED RESPONSE LANGUAGE: " + lang + "\n")
		}
	}

	if goal := strings.TrimSpace(session.ActiveGoal); goal != "" {
		sb.WriteString("ACTIVE GOAL: " + goal + "\n")
		if status := strings.TrimSpace(session.ActiveGoalStatus); status != "" {
			sb.WriteString("GOAL STATUS: " + status + "\n")
		}
	}

	if len(session.SkippedActions) > 0 {
		sb.WriteString("Previously skipped actions: ")
		sb.WriteString(strings.Join(session.SkippedActions, "; "))
		sb.WriteString("\n")
	}

	if len(session.GoalHistory) > 0 {
		sb.WriteString("Previous goals: ")
		var goals []string
		for _, g := range session.GoalHistory {
			goals = append(goals, g.Goal+"("+g.Status+")")
		}
		sb.WriteString(strings.Join(goals, ", "))
		sb.WriteString("\n")
	}

	return sb.String()
}

func (b *PromptBuilder) formatWorkflowOrchestration(flow WorkflowOrchestration) string {
	var sb strings.Builder
	sb.WriteString("Available platform operations:\n")
	if id := strings.TrimSpace(flow.WorkflowID); id != "" {
		sb.WriteString("workflow: " + id + "\n")
	}
	if label := strings.TrimSpace(flow.WorkflowLabel); label != "" {
		sb.WriteString("label: " + label + "\n")
	}
	if goal := strings.TrimSpace(flow.Goal); goal != "" {
		sb.WriteString("goal: " + goal + "\n")
	}
	if len(flow.Capabilities) > 0 {
		sb.WriteString("capabilities: " + strings.Join(flow.Capabilities, ", ") + "\n")
	}
	for idx, step := range flow.Steps {
		sb.WriteString(fmt.Sprintf("step_%d: %s\n", idx+1, step.Title))
		if strings.TrimSpace(step.Rationale) != "" {
			sb.WriteString("  rationale: " + step.Rationale + "\n")
		}
		payload, _ := json.MarshalIndent(step, "", "  ")
		sb.WriteString(string(payload))
		sb.WriteString("\n")
	}
	return strings.TrimSpace(sb.String())
}

func (b *PromptBuilder) formatAnalysisHistory(history []string) string {
	clean := make([]string, 0, len(history))
	for _, item := range history {
		item = strings.TrimSpace(item)
		if item != "" {
			clean = append(clean, item)
		}
	}
	if len(clean) == 0 {
		return ""
	}
	data, err := json.Marshal(clean)
	if err != nil {
		return ""
	}
	return "Recent analysis summaries (for reasoning continuity):\n" + string(data)
}

func (b *PromptBuilder) formatPlatformState(state PlatformState) string {
	data, err := json.Marshal(state)
	if err != nil {
		return ""
	}
	return "Platform API state:\n" + string(data)
}

func (b *PromptBuilder) formatPlatformExecGuidance(state *PlatformState, session *SessionContext) string {
	if state == nil {
		return ""
	}
	goal := ""
	if session != nil {
		goal = session.ActiveGoal
	}
	preferred := make([]string, 0, len(state.Playbooks))
	for _, playbook := range state.Playbooks {
		if id := strings.TrimSpace(playbook.ID); id != "" {
			preferred = append(preferred, id)
		}
	}
	hints := gitplatform.RecommendedExecutorSchemas(gitplatform.ParsePlatform(state.Detected), goal, preferred, 5)
	boundaryIDs := append([]string(nil), preferred...)
	for _, hint := range hints {
		if id := strings.TrimSpace(hint.CapabilityID); id != "" {
			boundaryIDs = append(boundaryIDs, id)
		}
	}
	boundaries := gitplatform.RelevantCapabilityBoundaries(gitplatform.ParsePlatform(state.Detected), boundaryIDs)
	if len(hints) == 0 && len(boundaries) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("Platform executor schema hints:\n")
	for _, hint := range hints {
		sb.WriteString("[" + hint.CapabilityID + "] " + hint.Label + "\n")
		if hint.Summary != "" {
			sb.WriteString("summary: " + hint.Summary + "\n")
		}
		if len(hint.InspectViews) > 0 {
			sb.WriteString("inspect views: " + strings.Join(hint.InspectViews, " | ") + "\n")
		}
		if len(hint.MutateOps) > 0 {
			sb.WriteString("mutate ops: " + strings.Join(hint.MutateOps, " | ") + "\n")
		}
		if len(hint.ScopeKeys) > 0 {
			sb.WriteString("scope keys: " + strings.Join(hint.ScopeKeys, ", ") + "\n")
		}
		if len(hint.QueryKeys) > 0 {
			sb.WriteString("query keys: " + strings.Join(hint.QueryKeys, ", ") + "\n")
		}
		for _, rule := range hint.FieldRules {
			sb.WriteString("- " + rule + "\n")
		}
		for _, note := range hint.Notes {
			sb.WriteString("note: " + note + "\n")
		}
		if strings.TrimSpace(hint.Example) != "" {
			sb.WriteString("example: " + hint.Example + "\n")
		}
		sb.WriteString("\n")
	}
	if len(boundaries) > 0 {
		sb.WriteString("Platform API boundaries:\n")
		for _, boundary := range boundaries {
			sb.WriteString("[" + boundary.CapabilityID + "] mode=" + boundary.Mode + "\n")
			sb.WriteString("reason: " + boundary.Reason + "\n")
			if len(boundary.Supported) > 0 {
				sb.WriteString("supported: " + strings.Join(boundary.Supported, " | ") + "\n")
			}
			if len(boundary.Missing) > 0 {
				sb.WriteString("missing: " + strings.Join(boundary.Missing, " | ") + "\n")
			}
			sb.WriteString("\n")
		}
	}
	return strings.TrimSpace(sb.String())
}

// BuildCommitMessage constructs a prompt specifically for commit message generation.
func (b *PromptBuilder) BuildCommitMessage(staged []git.FileStatus, diff string) (system, user string) {
	system = "You are a Git commit message generator. Output ONLY the commit message text (one line, max 72 chars, imperative mood). No quotes, no prefix, no explanation."
	var files []string
	for _, f := range staged {
		files = append(files, fmt.Sprintf("  %c %s", byte(f.StagingCode), f.Path))
	}
	user = fmt.Sprintf("Staged files:\n%s\n\nDiff summary:\n%s", strings.Join(files, "\n"), diff)
	return system, user
}

// BuildExplain constructs a prompt for explaining a specific suggestion.
func (b *PromptBuilder) BuildExplain(action, command, reason string, state *status.GitState) (system, user string) {
	system = "You are a Git coach. Explain in 2-3 concise sentences why the suggested action fits the current repository state."
	user = fmt.Sprintf("Repository context JSON:\n%s\n\nSuggested action: %s\nCommand: %s\nInitial reason: %s\n\nExplain why this is appropriate right now.",
		b.formatFullContext(state, "zen"), action, command, reason)
	return system, user
}

func (b *PromptBuilder) Temperature() float64 { return 0.1 }

func (b *PromptBuilder) formatFullContext(state *status.GitState, mode string) string {
	type fileSummary struct {
		Path     string `json:"path"`
		Staged   string `json:"staged,omitempty"`
		Worktree string `json:"worktree,omitempty"`
	}
	type remoteSummary struct {
		Name         string `json:"name"`
		FetchURL     string `json:"fetch_url,omitempty"`
		PushURL      string `json:"push_url,omitempty"`
		URLStatus    string `json:"url_status"`
		RemoteStatus string `json:"remote_status"`
		LastError    string `json:"last_error,omitempty"`
	}
	type branchSummary struct {
		Name       string `json:"name"`
		Upstream   string `json:"upstream,omitempty"`
		Ahead      int    `json:"ahead,omitempty"`
		Behind     int    `json:"behind,omitempty"`
		LastCommit string `json:"last_commit,omitempty"`
		IsMerged   bool   `json:"is_merged"`
		IsCurrent  bool   `json:"is_current"`
	}
	type stashSummary struct {
		Index   int    `json:"index"`
		Message string `json:"message"`
	}
	type submoduleSummary struct {
		Name   string `json:"name"`
		Path   string `json:"path"`
		URL    string `json:"url,omitempty"`
		Commit string `json:"commit,omitempty"`
		Status string `json:"status,omitempty"`
	}
	type promptContext struct {
		Mode        string `json:"mode"`
		Environment struct {
			SSHKeysAvailable      bool   `json:"ssh_keys_available"`
			PreferredRemoteScheme string `json:"preferred_remote_scheme"`
		} `json:"environment"`
		Repository struct {
			IsInitial        bool               `json:"is_initial"`
			HeadRef          string             `json:"head_ref,omitempty"`
			Branch           string             `json:"branch"`
			DetachedHead     bool               `json:"detached_head"`
			CommitCount      int                `json:"commit_count"`
			WorkingCount     int                `json:"working_count"`
			StagedCount      int                `json:"staged_count"`
			Branches         []branchSummary    `json:"branches,omitempty"`
			MergedBranches   []string           `json:"merged_branches,omitempty"`
			Tags             []string           `json:"tags,omitempty"`
			HasGitIgnore     bool               `json:"has_gitignore"`
			HasGitAttributes bool               `json:"has_gitattributes"`
			MergeInProgress  bool               `json:"merge_in_progress"`
			RebaseInProgress bool               `json:"rebase_in_progress"`
			CherryInProgress bool               `json:"cherry_pick_in_progress"`
			BisectInProgress bool               `json:"bisect_in_progress"`
			Stashes          []stashSummary     `json:"stashes,omitempty"`
			Submodules       []submoduleSummary `json:"submodules,omitempty"`
			RecentReflog     []string           `json:"recent_reflog,omitempty"`
			DescribeTag      string             `json:"describe_tag,omitempty"`
			Worktrees        []string           `json:"worktrees,omitempty"`
			DefaultBranch    string             `json:"default_branch,omitempty"`
		} `json:"repository"`
		Sync struct {
			Upstream             string   `json:"upstream,omitempty"`
			UpstreamRemote       string   `json:"upstream_remote,omitempty"`
			BranchHasUpstream    bool     `json:"branch_has_upstream"`
			Ahead                int      `json:"ahead"`
			Behind               int      `json:"behind"`
			AheadCommits         []string `json:"ahead_commits,omitempty"`
			BehindCommits        []string `json:"behind_commits,omitempty"`
			RemoteBranches       []string `json:"remote_branches,omitempty"`
			LastFetchSecondsAgo  int64    `json:"last_fetch_seconds_ago,omitempty"`
			HasUsableFetchRemote bool     `json:"has_usable_fetch_remote"`
			HasUsablePushRemote  bool     `json:"has_usable_push_remote"`
		} `json:"sync"`
		Platform struct {
			Detected       string `json:"detected"`
			RemoteProtocol string `json:"remote_protocol"`
		} `json:"platform"`
		Remotes     []remoteSummary `json:"remotes"`
		WorkingTree []fileSummary   `json:"working_tree"`
		StagingArea []fileSummary   `json:"staging_area"`
	}

	var ctx promptContext
	ctx.Mode = strings.ToLower(strings.TrimSpace(mode))
	if ctx.Mode == "" {
		ctx.Mode = "zen"
	}
	ctx.Environment.SSHKeysAvailable = gitplatform.HasSSHKeys()
	if ctx.Environment.SSHKeysAvailable {
		ctx.Environment.PreferredRemoteScheme = "ssh"
	} else {
		ctx.Environment.PreferredRemoteScheme = "https"
	}

	ctx.Repository.IsInitial = state.IsInitial
	ctx.Repository.HeadRef = state.HeadRef
	ctx.Repository.Branch = state.LocalBranch.Name
	ctx.Repository.DetachedHead = state.LocalBranch.IsDetached
	ctx.Repository.CommitCount = state.CommitCount
	ctx.Repository.WorkingCount = len(state.WorkingTree)
	ctx.Repository.StagedCount = len(state.StagingArea)

	const maxBranches = 30
	const maxMergedBranches = 20
	const maxTags = 15

	if len(state.BranchDetails) > 0 {
		for i, bd := range state.BranchDetails {
			if i >= maxBranches {
				break
			}
			ctx.Repository.Branches = append(ctx.Repository.Branches, branchSummary{
				Name:       bd.Name,
				Upstream:   bd.Upstream,
				Ahead:      bd.Ahead,
				Behind:     bd.Behind,
				LastCommit: bd.LastCommit,
				IsMerged:   bd.IsMerged,
				IsCurrent:  bd.IsCurrent,
			})
		}
	} else if len(state.LocalBranches) > 0 {
		for i, name := range state.LocalBranches {
			if i >= maxBranches {
				break
			}
			ctx.Repository.Branches = append(ctx.Repository.Branches, branchSummary{
				Name:      name,
				IsCurrent: name == state.LocalBranch.Name,
			})
		}
	}

	merged := state.MergedBranches
	if len(merged) > maxMergedBranches {
		merged = merged[:maxMergedBranches]
	}
	ctx.Repository.MergedBranches = append(ctx.Repository.MergedBranches, merged...)
	tags := state.Tags
	if len(tags) > maxTags {
		tags = tags[:maxTags]
	}
	ctx.Repository.Tags = append(ctx.Repository.Tags, tags...)
	ctx.Repository.HasGitIgnore = state.HasGitIgnore
	ctx.Repository.HasGitAttributes = state.HasGitAttributes
	ctx.Repository.MergeInProgress = state.MergeInProgress
	ctx.Repository.RebaseInProgress = state.RebaseInProgress
	ctx.Repository.CherryInProgress = state.CherryInProgress
	ctx.Repository.BisectInProgress = state.BisectInProgress

	for _, entry := range state.StashStack {
		ctx.Repository.Stashes = append(ctx.Repository.Stashes, stashSummary{
			Index:   entry.Index,
			Message: entry.Message,
		})
	}

	for _, sub := range state.Submodules {
		ctx.Repository.Submodules = append(ctx.Repository.Submodules, submoduleSummary{
			Name:   sub.Name,
			Path:   sub.Path,
			URL:    sub.URL,
			Commit: sub.Commit,
			Status: sub.Status,
		})
	}

	ctx.Repository.RecentReflog = append(ctx.Repository.RecentReflog, state.RecentReflog...)
	ctx.Repository.DescribeTag = state.DescribeTag
	ctx.Repository.Worktrees = append(ctx.Repository.Worktrees, state.Worktrees...)
	ctx.Repository.DefaultBranch = state.RepoConfig.DefaultBranch

	ahead, behind := state.LocalBranch.Ahead, state.LocalBranch.Behind
	if state.UpstreamState != nil {
		ahead = state.UpstreamState.Ahead
		behind = state.UpstreamState.Behind
	}
	ctx.Sync.Upstream = state.LocalBranch.Upstream
	ctx.Sync.UpstreamRemote = upstreamRemoteName(state.LocalBranch.Upstream)
	ctx.Sync.BranchHasUpstream = strings.TrimSpace(state.LocalBranch.Upstream) != ""
	ctx.Sync.Ahead = ahead
	ctx.Sync.Behind = behind
	ctx.Sync.AheadCommits = append(ctx.Sync.AheadCommits, state.AheadCommits...)
	ctx.Sync.BehindCommits = append(ctx.Sync.BehindCommits, state.BehindCommits...)
	ctx.Sync.RemoteBranches = append(ctx.Sync.RemoteBranches, state.RemoteBranches...)
	if !state.LastFetchTime.IsZero() {
		age := int64(time.Since(state.LastFetchTime).Seconds())
		if age < 0 {
			age = 0
		}
		ctx.Sync.LastFetchSecondsAgo = age
	}

	for _, remote := range state.RemoteInfos {
		urlStatus := "valid"
		if !remote.FetchURLValid && !remote.PushURLValid {
			urlStatus = "invalid_or_placeholder"
		} else if !remote.FetchURLValid || !remote.PushURLValid {
			urlStatus = "partially_valid"
		}
		remoteStatus := "available"
		if urlStatus == "invalid_or_placeholder" {
			remoteStatus = "not_usable"
		} else if remote.ReachabilityChecked && !remote.Reachable {
			remoteStatus = "unreachable"
		}
		ctx.Remotes = append(ctx.Remotes, remoteSummary{
			Name:         remote.Name,
			FetchURL:     remote.FetchURL,
			PushURL:      remote.PushURL,
			URLStatus:    urlStatus,
			RemoteStatus: remoteStatus,
			LastError:    remote.LastError,
		})
		usable := urlStatus != "invalid_or_placeholder" && remoteStatus != "unreachable"
		if usable && remote.FetchURLValid {
			ctx.Sync.HasUsableFetchRemote = true
		}
		if usable && remote.PushURLValid {
			ctx.Sync.HasUsablePushRemote = true
		}
	}
	platformURL := ""
	if remoteName := ctx.Sync.UpstreamRemote; remoteName != "" {
		for _, remote := range state.RemoteInfos {
			if remote.Name == remoteName {
				platformURL = remote.PushURL
				if strings.TrimSpace(platformURL) == "" {
					platformURL = remote.FetchURL
				}
				break
			}
		}
	}
	if strings.TrimSpace(platformURL) == "" && len(state.RemoteInfos) > 0 {
		platformURL = state.RemoteInfos[0].PushURL
		if strings.TrimSpace(platformURL) == "" {
			platformURL = state.RemoteInfos[0].FetchURL
		}
	}
	ctx.Platform.Detected = platformName(gitplatform.DetectPlatform(platformURL))
	ctx.Platform.RemoteProtocol = gitplatform.DetectRemoteProtocol(platformURL)

	for _, f := range state.WorkingTree {
		ctx.WorkingTree = append(ctx.WorkingTree, fileSummary{
			Path:     f.Path,
			Worktree: string(f.WorktreeCode),
		})
	}
	for _, f := range state.StagingArea {
		ctx.StagingArea = append(ctx.StagingArea, fileSummary{
			Path:   f.Path,
			Staged: string(f.StagingCode),
		})
	}

	data, err := json.MarshalIndent(ctx, "", "  ")
	if err != nil {
		return fmt.Sprintf("mode=%s\nbranch=%s\nupstream=%s\n", ctx.Mode, state.LocalBranch.Name, state.LocalBranch.Upstream)
	}
	return "Repository context JSON:\n" + string(data)
}

func upstreamRemoteName(upstream string) string {
	upstream = strings.TrimSpace(upstream)
	if upstream == "" {
		return ""
	}
	if idx := strings.Index(upstream, "/"); idx > 0 {
		return upstream[:idx]
	}
	return ""
}

func platformName(p gitplatform.Platform) string {
	switch p {
	case gitplatform.PlatformGitHub:
		return "github"
	case gitplatform.PlatformGitLab:
		return "gitlab"
	case gitplatform.PlatformBitbucket:
		return "bitbucket"
	default:
		return "unknown"
	}
}

func (b *PromptBuilder) formatFileInspection(fi *status.FileInspection) string {
	if fi == nil {
		return ""
	}
	type fileInspectJSON struct {
		TotalTracked   int                   `json:"total_tracked_files"`
		ImportantFiles []string              `json:"important_files,omitempty"`
		RecentModified []string              `json:"recent_modified,omitempty"`
		WorkingDiff    []status.FileDiffStat `json:"working_diff_stats,omitempty"`
		StagedDiff     []status.FileDiffStat `json:"staged_diff_stats,omitempty"`
	}
	obj := fileInspectJSON{
		TotalTracked:   fi.TotalFiles,
		ImportantFiles: fi.ImportantFiles,
		RecentModified: fi.RecentModified,
	}
	if len(fi.DiffStats) <= 15 {
		obj.WorkingDiff = fi.DiffStats
	} else {
		obj.WorkingDiff = fi.DiffStats[:15]
	}
	if len(fi.StagedStats) <= 15 {
		obj.StagedDiff = fi.StagedStats
	} else {
		obj.StagedDiff = fi.StagedStats[:15]
	}
	data, err := json.Marshal(obj)
	if err != nil {
		return ""
	}
	return "File inspection:\n" + string(data)
}

func (b *PromptBuilder) formatCommitSummary(cs *status.CommitSummary) string {
	if cs == nil {
		return ""
	}
	data, err := json.Marshal(cs)
	if err != nil {
		return ""
	}
	return "Commit history summary:\n" + string(data)
}

func (b *PromptBuilder) formatConfigState(cs *status.ConfigState) string {
	if cs == nil {
		return ""
	}
	data, err := json.Marshal(cs)
	if err != nil {
		return ""
	}
	return "Git configuration state:\n" + string(data)
}

func (b *PromptBuilder) formatMemoryContext(mc MemoryContext) string {
	var parts []string
	if len(mc.UserPreferences) > 0 {
		data, _ := json.Marshal(mc.UserPreferences)
		parts = append(parts, "User preferences: "+string(data))
	}
	if len(mc.RepoPatterns) > 0 {
		parts = append(parts, "Known repo patterns: "+strings.Join(mc.RepoPatterns, "; "))
	}
	if len(mc.ResolvedIssues) > 0 {
		recent := mc.ResolvedIssues
		if len(recent) > 5 {
			recent = recent[len(recent)-5:]
		}
		parts = append(parts, "Recently resolved issues: "+strings.Join(recent, "; "))
	}
	if len(mc.RecentEvents) > 0 {
		recent := mc.RecentEvents
		if len(recent) > 8 {
			recent = recent[len(recent)-8:]
		}
		parts = append(parts, "Recent episodic events: "+strings.Join(recent, "; "))
	}
	if len(mc.ArtifactNotes) > 0 {
		notes := mc.ArtifactNotes
		if len(notes) > 8 {
			notes = notes[len(notes)-8:]
		}
		parts = append(parts, "Artifact notes: "+strings.Join(notes, "; "))
	}
	if len(mc.Episodes) > 0 {
		episodes := mc.Episodes
		if len(episodes) > 6 {
			episodes = episodes[:6]
		}
		lines := make([]string, 0, len(episodes))
		for _, episode := range episodes {
			label := strings.TrimSpace(firstNonEmptyString(episode.Action, episode.Kind, episode.Surface))
			line := strings.TrimSpace(label + ": " + episode.Summary)
			if strings.TrimSpace(episode.CapabilityID) != "" {
				line += " | capability=" + strings.TrimSpace(episode.CapabilityID)
			}
			if strings.TrimSpace(episode.Flow) != "" {
				line += " | flow=" + strings.TrimSpace(episode.Flow)
			}
			if strings.TrimSpace(episode.Operation) != "" {
				line += " | op=" + strings.TrimSpace(episode.Operation)
			}
			if strings.TrimSpace(episode.Result) != "" {
				line += " [" + strings.TrimSpace(episode.Result) + "]"
			}
			lines = append(lines, line)
		}
		parts = append(parts, "Episodic memory: "+strings.Join(lines, "; "))
	}
	if len(mc.SemanticFacts) > 0 {
		facts := mc.SemanticFacts
		if len(facts) > 6 {
			facts = facts[:6]
		}
		lines := make([]string, 0, len(facts))
		for _, fact := range facts {
			line := fmt.Sprintf("%s (confidence %.2f, score %.2f)", fact.Fact, fact.Confidence, fact.CurrentScore)
			if fact.Stale {
				line += " [stale]"
			}
			lines = append(lines, line)
		}
		parts = append(parts, "Semantic memory: "+strings.Join(lines, "; "))
	}
	if mc.TaskState != nil {
		task := strings.TrimSpace(mc.TaskState.Goal)
		if strings.TrimSpace(mc.TaskState.WorkflowID) != "" {
			task += " | workflow=" + strings.TrimSpace(mc.TaskState.WorkflowID)
		}
		if strings.TrimSpace(mc.TaskState.Status) != "" {
			task += " | status=" + strings.TrimSpace(mc.TaskState.Status)
		}
		if len(mc.TaskState.Pending) > 0 {
			task += " | pending=" + strings.Join(mc.TaskState.Pending, ", ")
		}
		if task != "" {
			parts = append(parts, "Task memory: "+task)
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return "Long-term memory context:\n" + strings.Join(parts, "\n")
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func (b *PromptBuilder) formatFileContext(fc *FileContext) string {
	if fc == nil || len(fc.Files) == 0 {
		return ""
	}
	var parts []string
	parts = append(parts, "Important file contents (you can read and modify these):")
	for path, content := range fc.Files {
		parts = append(parts, fmt.Sprintf("\n--- %s ---\n%s", path, content))
	}
	return strings.Join(parts, "\n")
}
