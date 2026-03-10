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

const analyzeSystem = `You are a Git decision engine in a terminal UI. Output all text in Simplified Chinese.
Reason ONLY from provided context. Do not invent facts.

PRIORITY RULES (in order):
1. If ActiveGoal exists, ALL suggestions MUST directly advance that goal. Nothing else.
2. Never repeat commands the user has already skipped or that already succeeded.
3. Prefer executable actions ("auto"/"needs_input"/"file_write") over "info".

MODE: "zen" -> 1-3 suggestions. "full" -> up to 6 suggestions.

COMMAND FORMAT:
- "argv": JSON string array, e.g. ["git","add","."]
- "needs_input": when argv has user-specific values. Use "<placeholder>" syntax.
- Never emit fake values like "your-username" in auto commands.

FILE OPERATIONS (interaction "file_write"):
- Set "file_path", "file_content", "file_operation" ("create"|"update"|"delete"|"append").
- For delete: set "file_operation":"delete", "argv" can be empty [].
- For update: read current content from file_context, output COMPLETE new content.
- Prefer file_write over git commands when goal involves file changes.

RISK: "safe" | "caution" | "dangerous".

OUTPUT: Strict JSON only. No markdown fences. No prose outside JSON.
{
  "analysis": "2-4 sentences in Chinese",
  "goal_status": "in_progress|completed|blocked (if goal set)",
  "suggestions": [
    {
      "action": "short title",
      "argv": ["git","..."],
      "reason": "why now",
      "risk": "safe",
      "interaction": "auto|needs_input|info|file_write",
      "file_path": "optional",
      "file_content": "optional",
      "file_operation": "optional"
    }
  ]
}`

// MemoryContext holds long-term memory data to inject into prompts.
type MemoryContext struct {
	UserPreferences map[string]string `json:"user_preferences,omitempty"`
	RepoPatterns    []string          `json:"repo_patterns,omitempty"`
	ResolvedIssues  []string          `json:"resolved_issues,omitempty"`
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
	Mode    string `json:"mode,omitempty"`
}

type GoalRecord struct {
	Goal      string `json:"goal"`
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

type SessionContext struct {
	ActiveGoal     string            `json:"active_goal,omitempty"`
	GoalHistory    []GoalRecord      `json:"goal_history,omitempty"`
	SkippedActions []string          `json:"skipped_actions,omitempty"`
	Preferences    map[string]string `json:"preferences,omitempty"`
}

type PRSummary struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	URL    string `json:"url"`
}

type PlatformState struct {
	Detected      string      `json:"detected"`
	DefaultBranch string      `json:"default_branch,omitempty"`
	CIStatus      string      `json:"ci_status,omitempty"`
	OpenPRs       []PRSummary `json:"open_prs,omitempty"`
	LastError     string      `json:"last_error,omitempty"`
}

// AnalyzeInput holds all data sources for analysis prompt construction.
type AnalyzeInput struct {
	State           *status.GitState
	Mode            string
	RecentOps       []OperationRecord
	Session         *SessionContext
	AnalysisHistory []string
	PlatformState   *PlatformState
	Memory          *MemoryContext
	Knowledge       []KnowledgeFragment
	FileContext     *FileContext // file contents for LLM to read/modify
}

// FileContext holds important file contents.
type FileContext struct {
	Files map[string]string // path -> content
}

// BuildAnalyzeRich constructs prompts with full data sources and budget management.
func (b *PromptBuilder) BuildAnalyzeRich(input AnalyzeInput) (system, user string) {
	systemPrompt := analyzeSystem

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

	if goal := strings.TrimSpace(session.ActiveGoal); goal != "" {
		sb.WriteString("ACTIVE GOAL: " + goal + "\n")
		sb.WriteString("ALL suggestions must directly advance this goal. Do NOT suggest unrelated actions.\n")
	}

	if len(session.SkippedActions) > 0 {
		sb.WriteString("SKIPPED (do NOT repeat): ")
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
	type promptContext struct {
		Mode        string `json:"mode"`
		Environment struct {
			SSHKeysAvailable      bool   `json:"ssh_keys_available"`
			PreferredRemoteScheme string `json:"preferred_remote_scheme"`
		} `json:"environment"`
		Repository struct {
			IsInitial        bool     `json:"is_initial"`
			HeadRef          string   `json:"head_ref,omitempty"`
			Branch           string   `json:"branch"`
			DetachedHead     bool     `json:"detached_head"`
			CommitCount      int      `json:"commit_count"`
			WorkingCount     int      `json:"working_count"`
			StagedCount      int      `json:"staged_count"`
			LocalBranches    []string `json:"local_branches,omitempty"`
			Tags             []string `json:"tags,omitempty"`
			HasGitIgnore     bool     `json:"has_gitignore"`
			MergeInProgress  bool     `json:"merge_in_progress"`
			RebaseInProgress bool     `json:"rebase_in_progress"`
			CherryInProgress bool     `json:"cherry_pick_in_progress"`
			BisectInProgress bool     `json:"bisect_in_progress"`
			StashCount       int      `json:"stash_count"`
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
	ctx.Repository.LocalBranches = append(ctx.Repository.LocalBranches, state.LocalBranches...)
	ctx.Repository.Tags = append(ctx.Repository.Tags, state.Tags...)
	ctx.Repository.HasGitIgnore = state.HasGitIgnore
	ctx.Repository.MergeInProgress = state.MergeInProgress
	ctx.Repository.RebaseInProgress = state.RebaseInProgress
	ctx.Repository.CherryInProgress = state.CherryInProgress
	ctx.Repository.BisectInProgress = state.BisectInProgress
	ctx.Repository.StashCount = len(state.StashStack)

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
	if len(parts) == 0 {
		return ""
	}
	return "Long-term memory context:\n" + strings.Join(parts, "\n")
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
