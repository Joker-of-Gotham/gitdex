package tui

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Joker-of-Gotham/gitdex/internal/config"
	"github.com/Joker-of-Gotham/gitdex/internal/contract"
	"github.com/Joker-of-Gotham/gitdex/internal/dotgitdex"
	"github.com/Joker-of-Gotham/gitdex/internal/llm"
	"github.com/Joker-of-Gotham/gitdex/internal/llmfactory"
	"github.com/Joker-of-Gotham/gitdex/internal/observability"
	"github.com/Joker-of-Gotham/gitdex/internal/tui/components/table"
	"github.com/Joker-of-Gotham/gitdex/internal/tui/oplog"
	"github.com/Joker-of-Gotham/gitdex/internal/tui/theme"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		m.programCtx.UpdateDimensions(msg.Width, msg.Height)
		m.headerComp.SetDimensions(msg.Width)
		m.footerComp.SetWidth(msg.Width)
		m.tabsComp.SetWidth(msg.Width)
		m.sidebarComp.SetDimensions(msg.Width/3, msg.Height-4)
		return m, nil

	case initMsg:
		goals, _ := m.store.ReadGoalList()
		m.goals = goals
		if len(goals) > 0 {
			pending := dotgitdex.PendingGoals(goals)
			if len(pending) > 0 {
				m.activeGoal = pending[0].Title
			}
		}
		m.opLog.Add(oplog.Entry{Type: oplog.EntryStateRefresh, Summary: "Gitdex started"})

		var startCmd tea.Cmd
		if m.mode == "cruise" {
			m.cruiseCycleActive = true
			m.creativeRanThisSlice = false
			startCmd = m.startAnalysis()
		} else {
			startCmd = m.startAnalysis()
		}

		cmds := []tea.Cmd{m.refreshGitInfo(), startCmd}
		if m.helperLLM != nil {
			cmds = append(cmds, m.runConnectivityTest("helper", m.helperLLM))
		}
		if m.plannerLLM != nil && m.plannerLLM != m.helperLLM {
			cmds = append(cmds, m.runConnectivityTest("planner", m.plannerLLM))
		}
		return m, tea.Batch(cmds...)

	case gitRefreshMsg:
		m.gitInfo = msg.info
		return m, nil

	case goalTriageMsg:
		if msg.err != nil {
			m.opLog.Add(oplog.Entry{Type: oplog.EntryLLMError,
				Summary: "Goal triage failed: " + msg.err.Error()})
			// On triage failure, fall back to adding as gitdex goal
			goals, _ := m.store.ReadGoalList()
			goals = append(goals, dotgitdex.Goal{Title: msg.goalTitle})
			_ = m.store.WriteGoalList(goals)
			m.goals = goals
			m.activeGoal = msg.goalTitle
			return m, m.decomposeGoal(msg.goalTitle)
		}

		switch msg.result.Category {
		case "gitdex":
			m.opLog.Add(oplog.Entry{
				Type:    oplog.EntryCmdSuccess,
				Summary: fmt.Sprintf("Goal accepted (gitdex): %s", msg.goalTitle),
				Detail:  msg.result.Reason,
			})
			m.activeGoal = msg.goalTitle
			goals, _ := m.store.ReadGoalList()
			newGoal := dotgitdex.Goal{Title: msg.goalTitle}
			if len(msg.result.Todos) > 0 {
				newGoal.Todos = msg.result.Todos
			}
			goals = append(goals, newGoal)
			_ = m.store.WriteGoalList(goals)
			m.goals = goals
			if len(msg.result.Todos) > 0 {
				todoStrs := make([]string, len(msg.result.Todos))
				for j, t := range msg.result.Todos {
					todoStrs[j] = t.Title
				}
				m.opLog.Add(oplog.Entry{
					Type:    oplog.EntryLLMOutput,
					Summary: fmt.Sprintf("Goal decomposed: %d sub-tasks", len(msg.result.Todos)),
					Detail:  strings.Join(todoStrs, "\n"),
				})
			}
			return m, m.startAnalysis()

		case "creative":
			m.opLog.Add(oplog.Entry{
				Type:    oplog.EntryLLMOutput,
				Summary: fmt.Sprintf("Goal classified as creative proposal: %s", msg.goalTitle),
				Detail:  fmt.Sprintf("Reason: %s\nSaved to creative-proposal.md", msg.result.Reason),
			})
			_ = m.store.AppendCreativeProposal([]string{msg.goalTitle})
			return m, nil

		case "discard":
			m.opLog.Add(oplog.Entry{
				Type:    oplog.EntryLLMOutput,
				Summary: fmt.Sprintf("Goal discarded: %s", msg.goalTitle),
				Detail:  fmt.Sprintf("Reason: %s\nSaved to discarded-proposal.md", msg.result.Reason),
			})
			_ = m.store.AppendDiscardedProposal([]string{msg.goalTitle + " — " + msg.result.Reason})
			return m, nil

		default:
			// Unknown category, treat as gitdex
			m.activeGoal = msg.goalTitle
			goals, _ := m.store.ReadGoalList()
			goals = append(goals, dotgitdex.Goal{Title: msg.goalTitle})
			_ = m.store.WriteGoalList(goals)
			m.goals = goals
			return m, m.decomposeGoal(msg.goalTitle)
		}

	case goalDecomposedMsg:
		if msg.err != nil {
			m.opLog.Add(oplog.Entry{Type: oplog.EntryLLMError,
				Summary: "Goal decomposition failed: " + msg.err.Error()})
		} else if len(msg.todos) > 0 {
			goals, _ := m.store.ReadGoalList()
			for i := range goals {
				if goals[i].Title == msg.goalTitle && len(goals[i].Todos) == 0 {
					goals[i].Todos = msg.todos
					break
				}
			}
			_ = m.store.WriteGoalList(goals)
			m.goals = goals
			todoStrs := make([]string, len(msg.todos))
			for j, t := range msg.todos {
				todoStrs[j] = t.Title
			}
			m.opLog.Add(oplog.Entry{
				Type:    oplog.EntryLLMOutput,
				Summary: fmt.Sprintf("Goal decomposed: %d sub-tasks", len(msg.todos)),
				Detail:  strings.Join(todoStrs, "\n"),
			})
		}
		return m, m.startAnalysis()

	case goalProgressUpdatedMsg:
		if msg.err != nil {
			m.opLog.Add(oplog.Entry{Type: oplog.EntryLLMError,
				Summary: "Goal progress update failed: " + msg.err.Error()})
		}
		goals, _ := m.store.ReadGoalList()
		m.goals = goals

		// Log progress for the active goal
		for _, g := range goals {
			if g.Title == m.activeGoal && len(g.Todos) > 0 {
				done, total := dotgitdex.GoalProgress(g)
				if done > 0 {
					m.opLog.Add(oplog.Entry{Type: oplog.EntryStateRefresh,
						Summary: fmt.Sprintf("Goal progress: %d/%d sub-tasks done", done, total)})
				}
				break
			}
		}

		// Advance activeGoal if current one was completed
		pending := dotgitdex.PendingGoals(goals)
		if len(pending) > 0 {
			currentDone := true
			for _, g := range pending {
				if g.Title == m.activeGoal {
					currentDone = false
					break
				}
			}
			if currentDone {
				m.opLog.Add(oplog.Entry{Type: oplog.EntryCmdSuccess,
					Summary: fmt.Sprintf("Goal completed: %s", m.activeGoal)})
				m.activeGoal = pending[0].Title
			}
		} else if m.activeGoal != "" {
			m.opLog.Add(oplog.Entry{Type: oplog.EntryCmdSuccess,
				Summary: fmt.Sprintf("Goal completed: %s", m.activeGoal)})
			m.activeGoal = ""
		}

		// If this was triggered from a failure path, replan instead of continuing normally
		if msg.replan {
			delay := replanBackoffDelay(m.consecutiveReplans)
			m.opLog.Add(oplog.Entry{
				Type:    oplog.EntryStateRefresh,
				Summary: fmt.Sprintf("Replan backoff: waiting %s before next analysis", delay),
			})
			return m, tea.Tick(delay, func(time.Time) tea.Msg { return flowRetryMsg{} })
		}
		return m, m.continueFlowLoop()

	case flowRoundMsg:
		m.analyzing = false
		if msg.err != nil {
			m.opLog.Add(oplog.Entry{Type: oplog.EntryLLMError, Summary: msg.err.Error()})
			if m.mode == "auto" || m.mode == "cruise" {
				return m, tea.Tick(5*time.Second, func(time.Time) tea.Msg { return flowRetryMsg{} })
			}
			return m, nil
		}
		m.currentRound = msg.round
		m.activeFlow = msg.flow

		if msg.round != nil {
			m.lastTokenUsed = msg.round.TokensUsed
			m.lastTokenMax = msg.round.TokensBudget
			m.programCtx.ContextUsed = msg.round.TokensUsed
			m.programCtx.ContextMax = msg.round.TokensBudget
		}

		m.suggestions = nil
		m.suggIdx = 0
		if msg.round != nil {
			for _, item := range msg.round.Suggestions {
				m.suggestions = append(m.suggestions, SuggestionDisplay{
					Item:   item,
					Status: StatusPending,
				})
			}
		}

		if len(m.suggestions) > 0 {
			newSigs := suggestionSignatures(m.suggestions)
			if signaturesEqual(newSigs, m.lastSuggestionSigs) {
				m.consecutiveReplans++
				if m.consecutiveReplans >= maxConsecutiveReplans {
					m.opLog.Add(oplog.Entry{
						Type:    oplog.EntryCmdFail,
						Summary: "No-progress detected: LLM keeps suggesting identical actions — halting. Use /run to retry.",
					})
					m.consecutiveReplans = 0
					m.lastSuggestionSigs = nil
					(&m).syncAgentTable()
					return m, m.refreshGitInfo()
				}
			}
			m.lastSuggestionSigs = newSigs
		}

		(&m).syncAgentTable()

		if len(m.suggestions) == 0 {
			m.opLog.Add(oplog.Entry{Type: oplog.EntryLLMOutput,
				Summary: fmt.Sprintf("[%s] No actions needed", msg.flow)})

			if msg.flow == "maintain" {
				return m, m.startGoalAnalysis()
			}

			// Goal flow returned 0 suggestions: check if goal is actually done
			goals, _ := m.store.ReadGoalList()
			pending := dotgitdex.PendingGoals(goals)
			if len(pending) > 0 {
				// Goals still pending but planner sees no actions → try maintain first
				if m.mode == "auto" || m.mode == "cruise" {
					m.opLog.Add(oplog.Entry{Type: oplog.EntryStateRefresh,
						Summary: fmt.Sprintf("Goals still pending (%d), running maintenance check...", len(pending))})
					return m, m.startMaintainAnalysis()
				}
			} else if m.mode == "cruise" {
				return m, func() tea.Msg { return cruiseCycleCompleteMsg{} }
			}
			return m, nil
		}

		analysis := ""
		if msg.round != nil {
			analysis = msg.round.Analysis
			usage := formatTokenSectionUsage(msg.round.TokenSections)
			if usage != "" {
				if analysis != "" {
					analysis += "\n\n"
				}
				analysis += fmt.Sprintf("context_usage: %d/%d\n%s",
					msg.round.TokensUsed, msg.round.TokensBudget, usage)
			}
		}
		m.opLog.Add(oplog.Entry{
			Type:    oplog.EntryLLMOutput,
			Summary: fmt.Sprintf("[%s] %d suggestions", msg.flow, len(m.suggestions)),
			Detail:  analysis,
		})

		if m.mode == "auto" || m.mode == "cruise" {
			return m, tea.Batch(m.refreshGitInfo(), m.executeNext())
		}
		return m, m.refreshGitInfo()

	// Note: consecutiveReplans is reset in executionResultMsg when a full round
	// completes successfully (m.suggIdx >= len(m.suggestions)).

	case executionResultMsg:
		m.executing = false
		if msg.index < 0 || msg.index >= len(m.suggestions) {
			return m, nil
		}

		s := &m.suggestions[msg.index]
		cmdStr := ""
		if s.Item.Action.Command != "" {
			cmdStr = s.Item.Action.Command
		} else if s.Item.Action.FilePath != "" {
			cmdStr = s.Item.Action.FileOp + " " + s.Item.Action.FilePath
		}

		if msg.result != nil && msg.result.Success {
			s.Status = StatusDone
			s.Output = msg.result.Stdout
			detail := "$ " + cmdStr
			if msg.result.Trace.TraceID != "" {
				detail = "[trace=" + msg.result.Trace.TraceID + "]\n" + detail
			}
			if msg.result.Stdout != "" {
				detail += "\n" + msg.result.Stdout
			}
			m.opLog.Add(oplog.Entry{
				Type:    oplog.EntryCmdSuccess,
				Summary: s.Item.Name,
				Detail:  detail,
			})
		} else {
			s.Status = StatusFailed
			errText := ""
			if msg.result != nil {
				errText = msg.result.Stderr
			}
			if msg.err != nil {
				errText = msg.err.Error()
			}
			s.Error = errText
			detail := "$ " + cmdStr
			if msg.result != nil && msg.result.Trace.TraceID != "" {
				detail = "[trace=" + msg.result.Trace.TraceID + "]\n" + detail
			}
			if errText != "" {
				detail += "\n" + errText
			}
			m.opLog.Add(oplog.Entry{
				Type:    oplog.EntryCmdFail,
				Summary: s.Item.Name,
				Detail:  detail,
			})
		}

		m.suggIdx++
		(&m).syncAgentTable()

		if s.Status == StatusFailed {
			recovery := contract.RecoveryAbort
			if msg.result != nil {
				recovery = msg.result.RecoverBy.Type
			}

			switch recovery {
			case contract.RecoverySkip:
				// Non-fatal (already exists, nothing to commit): continue to next suggestion.
				m.opLog.Add(oplog.Entry{
					Type:    oplog.EntryStateRefresh,
					Summary: fmt.Sprintf("Non-fatal failure (skip): %s", s.Error),
				})
				if m.suggIdx < len(m.suggestions) && (m.mode == "auto" || m.mode == "cruise" || m.runAllMode) {
					return m, m.executeNext()
				}
				return m, m.refreshGitInfo()

			case contract.RecoveryManual:
				// Auth/permission failure: halt and notify user. Do not auto-replan.
				for i := m.suggIdx; i < len(m.suggestions); i++ {
					m.suggestions[i].Status = StatusSkipped
				}
				m.runAllMode = false
				_ = m.orchestrator.FlushLog()
				m.opLog.Add(oplog.Entry{
					Type:    oplog.EntryCmdFail,
					Summary: "Manual intervention required: " + s.Error,
				})
				return m, m.refreshGitInfo()

			default:
				// RecoveryAbort or RecoveryRetry: skip remaining, trigger replan.
				for i := m.suggIdx; i < len(m.suggestions); i++ {
					m.suggestions[i].Status = StatusSkipped
				}
				m.runAllMode = false
				m.consecutiveReplans++
				observability.RecordReplanAttempt()
				_ = m.orchestrator.FlushLog()

				if m.consecutiveReplans >= maxConsecutiveReplans {
					m.opLog.Add(oplog.Entry{
						Type:    oplog.EntryCmdFail,
						Summary: fmt.Sprintf("Circuit breaker: %d consecutive replans reached — halting automatic execution. Use /run to retry manually.", maxConsecutiveReplans),
					})
					m.consecutiveReplans = 0
					return m, m.refreshGitInfo()
				}

				m.opLog.Add(oplog.Entry{
					Type:    oplog.EntryStateRefresh,
					Summary: fmt.Sprintf("Replanning after failure (attempt %d/%d)...", m.consecutiveReplans, maxConsecutiveReplans),
				})
				return m, tea.Batch(m.refreshGitInfo(), m.updateGoalProgressWithReplan(true))
			}
		}

		if m.suggIdx >= len(m.suggestions) {
			m.compressCurrentRound()
			m.runAllMode = false
			m.consecutiveReplans = 0 // reset on successful round completion
			_ = m.orchestrator.FlushLog()

			return m, tea.Batch(m.refreshGitInfo(), m.updateGoalProgress())
		}

		if m.mode == "auto" || m.mode == "cruise" || m.runAllMode {
			return m, m.executeNext()
		}
		return m, nil

	case goalProgressMsg:
		m.goals = msg.goals
		return m, nil

	case cruiseCycleCompleteMsg:
		m.cruiseCycleActive = false

		// Three-condition gate for creative module:
		// 1. creative has not run in this time slice
		// 2. all goals are complete
		// 3. repository is completely clean
		goals, _ := m.store.ReadGoalList()
		allGoalsDone := len(dotgitdex.PendingGoals(goals)) == 0
		repoClean := m.orchestrator != nil && m.orchestrator.IsMaintainClean(context.Background())

		if !m.creativeRanThisSlice && allGoalsDone && repoClean {
			m.creativeRanThisSlice = true
			m.cruiseCycleActive = true
			m.opLog.Add(oplog.Entry{Type: oplog.EntryStateRefresh,
				Summary: "All goals done & repo clean → starting creative flow..."})
			return m, m.startCreativeFlow()
		}

		m.opLog.Add(oplog.Entry{Type: oplog.EntryCmdSuccess,
			Summary: fmt.Sprintf("Cruise cycle complete. Next patrol in %s",
				formatDuration(m.cruiseIntervalS))})
		return m, tea.Tick(time.Duration(m.cruiseIntervalS)*time.Second,
			func(time.Time) tea.Msg { return cruiseTickMsg{} })

	case flowRetryMsg:
		if m.mode == "auto" || m.mode == "cruise" {
			m.opLog.Add(oplog.Entry{Type: oplog.EntryStateRefresh,
				Summary: "Retrying analysis after error..."})
			return m, m.startAnalysis()
		}
		return m, nil

	case cruiseTickMsg:
		if m.mode != "cruise" {
			return m, nil
		}
		// New time slice: reset the creative-ran flag
		m.creativeRanThisSlice = false

		if m.cruiseCycleActive {
			m.opLog.Add(oplog.Entry{Type: oplog.EntryStateRefresh,
				Summary: "Cruise tick: previous cycle still in progress, continuing goal/maintain..."})
			return m, nil
		}
		m.cruiseCycleActive = true
		m.opLog.Add(oplog.Entry{Type: oplog.EntryStateRefresh,
			Summary: "Cruise patrol triggered, starting goal/maintain cycle..."})
		return m, m.startAnalysis()

	case creativeResultMsg:
		m.analyzing = false
		if msg.err != nil {
			m.opLog.Add(oplog.Entry{Type: oplog.EntryLLMError,
				Summary: "Creative flow error: " + msg.err.Error()})
		} else if msg.result != nil {
			newGoalCount := len(msg.result.NewGitdexGoals)
			newCreativeCount := len(msg.result.NewCreative)
			discardedCount := len(msg.result.Discarded)

			detail := ""
			if newGoalCount > 0 {
				detail += "New Gitdex goals:\n"
				for _, g := range msg.result.NewGitdexGoals {
					detail += "  + " + g + "\n"
				}
			}
			if newCreativeCount > 0 {
				detail += "Creative proposals:\n"
				for _, c := range msg.result.NewCreative {
					detail += "  ◆ " + c + "\n"
				}
			}
			if discardedCount > 0 {
				detail += fmt.Sprintf("Discarded: %d\n", discardedCount)
			}

			m.opLog.Add(oplog.Entry{
				Type: oplog.EntryLLMOutput,
				Summary: fmt.Sprintf("[creative] +%d goals, +%d proposals, -%d discarded",
					newGoalCount, newCreativeCount, discardedCount),
				Detail: detail,
			})

			// Refresh goals from store after creative flow may have added new ones
			goals, _ := m.store.ReadGoalList()
			m.goals = goals
			if len(dotgitdex.PendingGoals(goals)) > 0 && m.activeGoal == "" {
				m.activeGoal = dotgitdex.PendingGoals(goals)[0].Title
			}

			// Decompose any new goals that have no todos
			for _, g := range msg.result.NewGitdexGoals {
				for _, goal := range goals {
					if goal.Title == g && len(goal.Todos) == 0 {
						return m, m.decomposeGoal(g)
					}
				}
			}
		}

		// After creative flow: if new goals were created, re-enter goal/maintain cycle
		goals2, _ := m.store.ReadGoalList()
		if len(dotgitdex.PendingGoals(goals2)) > 0 {
			m.opLog.Add(oplog.Entry{Type: oplog.EntryStateRefresh,
				Summary: "Creative flow added new goals, re-entering goal/maintain cycle..."})
			return m, m.startAnalysis()
		}

		// No new goals, creative is done for this slice → end the cruise cycle
		return m, func() tea.Msg { return cruiseCycleCompleteMsg{} }

	case llmConnectivityMsg:
		label := strings.ToUpper(msg.role[:1]) + msg.role[1:]
		if msg.ok {
			m.opLog.Add(oplog.Entry{
				Type:    oplog.EntryCmdSuccess,
				Summary: fmt.Sprintf("%s LLM OK (%s/%s, %dms)", label, msg.provider, msg.model, msg.latencyMs),
			})
		} else {
			m.opLog.Add(oplog.Entry{
				Type:    oplog.EntryLLMError,
				Summary: fmt.Sprintf("%s LLM FAILED (%s)", label, msg.provider),
				Detail:  msg.err,
			})
		}
		return m, nil

	case ollamaModelsMsg:
		m.ollamaFetching = false
		if msg.err != nil {
			m.ollamaFetchError = msg.err.Error()
		} else {
			m.ollamaModels = msg.models
			m.ollamaFetchError = ""
			if len(msg.models) > 0 {
				for i, om := range msg.models {
					if om.Name == m.configDraft.Model {
						m.ollamaModelIdx = i
						break
					}
				}
			}
		}
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKeyPress(msg)

	case tea.PasteMsg:
		return m.handlePaste(msg.Content)

	case tea.MouseClickMsg:
		return m.handleMouseClick(msg)

	case tea.MouseWheelMsg:
		return m.handleMouseWheel(msg)
	}

	if m.page == PageMain {
		(&m).syncSidebar()
	}

	return m, nil
}

// ---------- Paste handling ----------

func (m Model) handlePaste(text string) (tea.Model, tea.Cmd) {
	if text == "" {
		return m, nil
	}
	if m.page == PageConfigModel && m.configEditing {
		fi := m.configDraft.FieldIdx
		if fi >= 2 {
			val := m.configDraftFieldValue(fi)
			newVal, newCur := insertAtRune(val, m.configDraft.CursorAt, text)
			m.setConfigDraftFieldValue(fi, newVal)
			m.configDraft.CursorAt = newCur
		}
		return m, nil
	}
	if m.composerFocus || m.page == PageMain {
		m.composerFocus = true
		m.composerText += text
		return m, nil
	}
	return m, nil
}

// ---------- Mouse handling ----------

func (m Model) handleMouseClick(msg tea.MouseClickMsg) (tea.Model, tea.Cmd) {
	if msg.Button != tea.MouseLeft {
		return m, nil
	}

	if m.page != PageMain {
		return m, nil
	}

	zone := m.zoneFromXY(msg.X, msg.Y)
	m.focusZone = zone
	m.composerFocus = (zone == FocusInput)
	return m, nil
}

func (m Model) handleMouseWheel(msg tea.MouseWheelMsg) (tea.Model, tea.Cmd) {
	if m.page != PageMain {
		return m, nil
	}

	zone := m.zoneFromXY(msg.X, msg.Y)
	if zone == FocusInput {
		return m, nil
	}

	switch msg.Button {
	case tea.MouseWheelUp:
		m.applyScrollDelta(zone, -scrollStepWheel)
	case tea.MouseWheelDown:
		m.applyScrollDelta(zone, scrollStepWheel)
	}

	return m, nil
}

// ---------- Keyboard handling ----------

func (m Model) handleKeyPress(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Normalize special keys from KeyCode for reliable detection across terminals
	switch msg.Code {
	case tea.KeyEscape:
		key = "esc"
	case tea.KeyPgUp:
		key = "pgup"
	case tea.KeyPgDown:
		key = "pgdown"
	case tea.KeyHome:
		key = "home"
	case tea.KeyEnd:
		key = "end"
	case tea.KeyUp:
		key = "up"
	case tea.KeyDown:
		key = "down"
	case tea.KeyTab:
		key = "tab"
	}

	if key == "ctrl+c" {
		return m, tea.Quit
	}

	if key == "ctrl+p" {
		m.showCommandPalette = !m.showCommandPalette
		if m.showCommandPalette {
			m.paletteQuery = ""
			m.paletteIdx = 0
			m.showHelpOverlay = false
			m.composerFocus = false
		}
		return m, nil
	}

	if m.showCommandPalette {
		return m.handleCommandPaletteKeys(key, msg)
	}

	if key == "?" || key == "f1" {
		m.showHelpOverlay = !m.showHelpOverlay
		return m, nil
	}
	if m.showHelpOverlay && isEscKey(key) {
		m.showHelpOverlay = false
		return m, nil
	}

	if m.page != PageMain {
		return m.handleConfigPageKeys(key, msg)
	}

	if key == "q" && !m.composerFocus {
		return m, tea.Quit
	}

	if key == "tab" && !m.composerFocus {
		m.tabsComp.Next()
		return m, nil
	}
	if key == "shift+tab" && !m.composerFocus {
		m.tabsComp.Prev()
		return m, nil
	}

	if isEscKey(key) && m.composerFocus {
		m.composerFocus = false
		m.focusZone = FocusLeft
		return m, nil
	}

	// Allow pgup/pgdown even when composer is focused (they control panel scroll)
	if (key == "pgup" || key == "pgdown") && m.composerFocus {
		return m.handleNavigation(key)
	}

	if m.composerFocus {
		return m.handleComposerInput(key, msg)
	}

	return m.handleNavigation(key)
}

func (m Model) handleComposerInput(key string, msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch key {
	case "enter":
		return m.handleComposerSubmit()
	case "backspace":
		runes := []rune(m.composerText)
		if len(runes) > 0 {
			m.composerText = string(runes[:len(runes)-1])
		}
	case "escape", "esc":
		m.composerFocus = false
		m.focusZone = FocusLeft
	default:
		text := msg.Text
		if text != "" {
			m.composerText += text
		} else if key == "space" {
			m.composerText += " "
		}
	}
	return m, nil
}

func (m Model) handleNavigation(key string) (tea.Model, tea.Cmd) {
	zone := m.focusZone
	pageSize := m.height / 2
	if pageSize < 5 {
		pageSize = 10
	}
	if zone == FocusLog && m.detailPaneOpen {
		max := len(m.opLog.Entries())
		if max > 30 {
			max = 30
		}
		if max <= 0 {
			return m, nil
		}
		switch key {
		case "up", "k":
			if m.logCursor > 0 {
				m.logCursor--
			}
			return m, nil
		case "down", "j":
			if m.logCursor < max-1 {
				m.logCursor++
			}
			return m, nil
		case "pgup":
			m.logCursor -= pageSize
			if m.logCursor < 0 {
				m.logCursor = 0
			}
			return m, nil
		case "pgdown":
			m.logCursor += pageSize
			if m.logCursor > max-1 {
				m.logCursor = max - 1
			}
			return m, nil
		case "home", "g":
			m.logCursor = 0
			return m, nil
		case "end", "G":
			if max > 0 {
				m.logCursor = max - 1
			}
			return m, nil
		}
	}
	switch key {
	case "up", "k":
		if zone == FocusLeft {
			m.agentTable.PrevItem()
			(&m).syncSidebar()
		} else if m.detailPaneOpen && (zone == FocusGit || zone == FocusGoals || zone == FocusLog) {
			m.sidebarComp.ScrollUp(1)
		}
		m.applyScrollDelta(zone, -scrollStepLine)
	case "down", "j":
		if zone == FocusLeft {
			m.agentTable.NextItem()
			(&m).syncSidebar()
		} else if m.detailPaneOpen && (zone == FocusGit || zone == FocusGoals || zone == FocusLog) {
			m.sidebarComp.ScrollDown(1)
		}
		m.applyScrollDelta(zone, scrollStepLine)
	case "pgup":
		if zone == FocusLeft {
			m.agentTable.PageUp()
			(&m).syncSidebar()
		} else if m.detailPaneOpen && (zone == FocusGit || zone == FocusGoals || zone == FocusLog) {
			m.sidebarComp.PageUp()
		}
		m.applyScrollDelta(zone, -pageSize)
	case "pgdown":
		if zone == FocusLeft {
			m.agentTable.PageDown()
			(&m).syncSidebar()
		} else if m.detailPaneOpen && (zone == FocusGit || zone == FocusGoals || zone == FocusLog) {
			m.sidebarComp.PageDown()
		}
		m.applyScrollDelta(zone, pageSize)
	case "ctrl+u":
		m.applyScrollDelta(zone, -(pageSize / 2))
	case "ctrl+d":
		m.applyScrollDelta(zone, pageSize/2)
	case "home", "g":
		m.panelScrolls[zone] = 0
	case "end", "G":
		m.panelScrolls[zone] = 9999
	case "enter":
		if zone == FocusLog {
			m.detailPaneOpen = !m.detailPaneOpen
			if m.detailPaneOpen {
				entries := m.opLog.Entries()
				if len(entries) > 0 {
					max := len(entries)
					if max > 30 {
						max = 30
					}
					if m.logCursor < 0 || m.logCursor >= max {
						m.logCursor = max - 1
					}
				}
			}
		}
	case "r":
		return m, m.startAnalysis()
	case "p":
		m.detailPaneOpen = !m.detailPaneOpen
		m.programCtx.SidebarOpen = m.detailPaneOpen
	case "left", "h":
		m.applyScrollDelta(zone, -1)
	case "right", "l":
		m.applyScrollDelta(zone, 1)
	}
	return m, nil
}

func (m Model) handleCommandPaletteKeys(key string, msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	items := m.filteredCommandPaletteItems()
	clamp := func() {
		if len(items) == 0 {
			m.paletteIdx = 0
			return
		}
		if m.paletteIdx < 0 {
			m.paletteIdx = 0
		}
		if m.paletteIdx >= len(items) {
			m.paletteIdx = len(items) - 1
		}
	}
	clamp()

	switch key {
	case "esc", "escape":
		m.showCommandPalette = false
		return m, nil
	case "up", "k":
		m.paletteIdx--
		clamp()
		return m, nil
	case "down", "j":
		m.paletteIdx++
		clamp()
		return m, nil
	case "backspace":
		r := []rune(m.paletteQuery)
		if len(r) > 0 {
			m.paletteQuery = string(r[:len(r)-1])
			m.paletteIdx = 0
		}
		return m, nil
	case "tab":
		if len(items) > 0 {
			m.composerText = paletteHead(items[m.paletteIdx])
			m.composerFocus = true
			m.focusZone = FocusInput
			m.showCommandPalette = false
		}
		return m, nil
	case "enter":
		if len(items) == 0 {
			m.showCommandPalette = false
			return m, nil
		}
		m.composerText = paletteHead(items[m.paletteIdx])
		m.composerFocus = true
		m.focusZone = FocusInput
		m.showCommandPalette = false
		m.opLog.Add(oplog.Entry{Type: oplog.EntryUserAction, Summary: "Command selected: " + m.composerText})
		return m, nil
	}

	text := msg.Text
	if text == "" && key == "space" {
		text = " "
	}
	if text != "" {
		m.paletteQuery += text
		m.paletteIdx = 0
	}
	return m, nil
}

func (m Model) filteredCommandPaletteItems() []string {
	items := CommandPaletteItems()
	query := strings.ToLower(strings.TrimSpace(m.paletteQuery))
	if query == "" {
		return items
	}
	var out []string
	for _, it := range items {
		if strings.Contains(strings.ToLower(it), query) {
			out = append(out, it)
		}
	}
	return out
}

func paletteHead(item string) string {
	fields := strings.Fields(item)
	if len(fields) == 0 {
		return item
	}
	return fields[0]
}

func formatTokenSectionUsage(m map[string]int) string {
	if len(m) == 0 {
		return ""
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var lines []string
	for _, k := range keys {
		lines = append(lines, fmt.Sprintf("- %s: %d", k, m[k]))
	}
	return strings.Join(lines, "\n")
}

// ---------- Config page keys ----------

func isEscKey(key string) bool {
	return key == "escape" || key == "esc"
}

func (m Model) handleConfigPageKeys(key string, msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.page == PageConfigModel && m.configEditing {
		return m.handleConfigModelEditKeys(key, msg)
	}

	if isEscKey(key) {
		switch m.page {
		case PageConfigModel, PageConfigMode, PageConfigLang, PageConfigTheme:
			m.page = PageConfig
			m.panelScrolls[FocusLeft] = 0
			m.configEditing = false
		default:
			m.page = PageMain
			m.panelScrolls[FocusLeft] = 0
		}
		return m, nil
	}

	switch m.page {
	case PageConfig:
		return m.handleConfigMenuKeys(key)
	case PageConfigMode:
		return m.handleConfigModeKeys(key)
	case PageConfigLang:
		return m.handleConfigLangKeys(key)
	case PageConfigTheme:
		return m.handleConfigThemeKeys(key)
	case PageConfigModel:
		return m.handleConfigModelKeys(key)
	}
	return m, nil
}

func (m Model) handleConfigModelKeys(key string) (tea.Model, tea.Cmd) {
	provID := draftProviders[m.configDraft.ProviderIdx]
	fi := m.configDraft.FieldIdx
	maxField := 4
	if provID == "ollama" {
		maxField = 3 // no apikey field
	}

	// Ollama model list navigation (fi == 2, provider == ollama)
	if fi == 2 && provID == "ollama" && len(m.ollamaModels) > 0 && !m.configEditing {
		switch key {
		case "up", "k":
			if m.ollamaModelIdx > 0 {
				m.ollamaModelIdx--
			}
			return m, nil
		case "down", "j":
			if m.ollamaModelIdx < len(m.ollamaModels)-1 {
				m.ollamaModelIdx++
			}
			return m, nil
		case "enter":
			m.configDraft.Model = m.ollamaModels[m.ollamaModelIdx].Name
			m.configDraft.PerProviderModel[m.configDraft.ProviderIdx] = m.configDraft.Model
			m.applyConfigDraft()
			if !m.persistConfigOrLog() {
				return m, nil
			}
			if !m.applyLLMConfigRuntime() {
				return m, nil
			}
			roleName := "helper"
			if m.configDraft.Role == RolePlanner {
				roleName = "planner"
			}
			m.opLog.Add(oplog.Entry{Type: oplog.EntryUserAction,
				Summary: fmt.Sprintf("Config saved (%s): %s / %s", roleName, provID, m.configDraft.Model)})
			m.page = PageConfig
			return m, nil
		case "tab":
			m.configDraft.FieldIdx = 3
			return m, nil
		}
		if isEscKey(key) || key == "q" {
			m.page = PageConfig
			m.panelScrolls[FocusLeft] = 0
			return m, nil
		}
		return m, nil
	}

	switch key {
	case "tab", "down", "j":
		next := fi + 1
		if next > maxField {
			next = 0
		}
		m.configDraft.FieldIdx = next
		m.configDraft.CursorAt = runeCount(m.configDraftFieldValue(next))
	case "shift+tab", "up", "k":
		next := fi - 1
		if next < 0 {
			next = maxField
		}
		m.configDraft.FieldIdx = next
		m.configDraft.CursorAt = runeCount(m.configDraftFieldValue(next))
	case "enter":
		if fi <= 1 {
			return m, nil
		}
		m.configEditing = true
		m.configDraft.CursorAt = runeCount(m.configDraftFieldValue(fi))
	case "left", "h":
		if fi == 0 {
			if m.configDraft.Role == RolePlanner {
				m.configDraft.Role = RoleHelper
				m.loadRoleIntoDraft()
			}
		} else if fi == 1 && m.configDraft.ProviderIdx > 0 {
			m.configDraft.PerProviderModel[m.configDraft.ProviderIdx] = m.configDraft.Model
			m.configDraft.ProviderIdx--
			cmd := m.applyProviderDefaults()
			return m, cmd
		}
	case "right", "l":
		if fi == 0 {
			if m.configDraft.Role == RoleHelper {
				m.configDraft.Role = RolePlanner
				m.loadRoleIntoDraft()
			}
		} else if fi == 1 && m.configDraft.ProviderIdx < 2 {
			m.configDraft.PerProviderModel[m.configDraft.ProviderIdx] = m.configDraft.Model
			m.configDraft.ProviderIdx++
			cmd := m.applyProviderDefaults()
			return m, cmd
		}
	case " ":
		if fi == 0 {
			if m.configDraft.Role == RoleHelper {
				m.configDraft.Role = RolePlanner
			} else {
				m.configDraft.Role = RoleHelper
			}
			m.loadRoleIntoDraft()
		} else if fi == 1 {
			m.configDraft.PerProviderModel[m.configDraft.ProviderIdx] = m.configDraft.Model
			m.configDraft.ProviderIdx = (m.configDraft.ProviderIdx + 1) % 3
			cmd := m.applyProviderDefaults()
			return m, cmd
		}
	case "q":
		m.page = PageConfig
		m.panelScrolls[FocusLeft] = 0
		m.configEditing = false
	}
	return m, nil
}

func (m Model) handleConfigModelEditKeys(key string, msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	fi := m.configDraft.FieldIdx

	switch key {
	case "enter":
		m.configEditing = false
		m.configDraft.PerProviderModel[m.configDraft.ProviderIdx] = m.configDraft.Model
		m.applyConfigDraft()
		if !m.persistConfigOrLog() {
			return m, nil
		}
		if !m.applyLLMConfigRuntime() {
			return m, nil
		}
		roleName := "helper"
		if m.configDraft.Role == RolePlanner {
			roleName = "planner"
		}
		m.opLog.Add(oplog.Entry{Type: oplog.EntryUserAction,
			Summary: fmt.Sprintf("Config saved (%s): %s / %s", roleName, draftProviders[m.configDraft.ProviderIdx], m.configDraft.Model)})
		m.page = PageConfig
		return m, nil
	case "tab":
		m.configEditing = false
		maxField := 4
		if draftProviders[m.configDraft.ProviderIdx] == "ollama" {
			maxField = 3
		}
		next := fi + 1
		if next > maxField {
			next = 0
		}
		m.configDraft.FieldIdx = next
		if next >= 2 {
			m.configEditing = true
			m.configDraft.CursorAt = runeCount(m.configDraftFieldValue(next))
		}
	case "left":
		if m.configDraft.CursorAt > 0 {
			m.configDraft.CursorAt--
		}
	case "right":
		if m.configDraft.CursorAt < runeCount(m.configDraftFieldValue(fi)) {
			m.configDraft.CursorAt++
		}
	case "home", "ctrl+a":
		m.configDraft.CursorAt = 0
	case "end", "ctrl+e":
		m.configDraft.CursorAt = runeCount(m.configDraftFieldValue(fi))
	case "backspace":
		val := m.configDraftFieldValue(fi)
		newVal, newCur := deleteRuneBefore(val, m.configDraft.CursorAt)
		m.setConfigDraftFieldValue(fi, newVal)
		m.configDraft.CursorAt = newCur
	case "delete":
		val := m.configDraftFieldValue(fi)
		newVal, newCur := deleteRuneAt(val, m.configDraft.CursorAt)
		m.setConfigDraftFieldValue(fi, newVal)
		m.configDraft.CursorAt = newCur
	default:
		if isEscKey(key) {
			m.configEditing = false
			return m, nil
		}
		text := msg.Text
		if text == "" && key == "space" {
			text = " "
		}
		if text != "" {
			val := m.configDraftFieldValue(fi)
			newVal, newCur := insertAtRune(val, m.configDraft.CursorAt, text)
			m.setConfigDraftFieldValue(fi, newVal)
			m.configDraft.CursorAt = newCur
		}
	}
	return m, nil
}

func (m Model) configDraftFieldValue(fi int) string {
	switch fi {
	case 2:
		return m.configDraft.Model
	case 3:
		return m.configDraft.Endpoint
	case 4:
		return m.configDraft.APIKeyEnv
	default:
		return ""
	}
}

func (m *Model) setConfigDraftFieldValue(fi int, v string) {
	switch fi {
	case 2:
		m.configDraft.Model = v
	case 3:
		m.configDraft.Endpoint = v
	case 4:
		m.configDraft.APIKeyEnv = v
	}
}

func (m *Model) applyProviderDefaults() tea.Cmd {
	idx := m.configDraft.ProviderIdx
	meta := providerMetaFor(draftProviders[idx])
	m.configDraft.Endpoint = meta.DefaultBaseURL
	m.configDraft.APIKeyEnv = meta.APIKeyEnv

	saved := m.configDraft.PerProviderModel[idx]
	if saved != "" {
		m.configDraft.Model = saved
	} else if len(meta.RecommendedModels) > 0 {
		m.configDraft.Model = meta.RecommendedModels[0]
		m.configDraft.PerProviderModel[idx] = m.configDraft.Model
	}
	m.configDraft.CursorAt = runeCount(m.configDraft.Model)

	m.ollamaModels = nil
	m.ollamaFetchError = ""
	m.ollamaModelIdx = 0
	if meta.ID == "ollama" {
		m.ollamaFetching = true
		return fetchOllamaModels(meta.DefaultBaseURL)
	}
	return nil
}

func (m *Model) applyConfigDraft() {
	role := &m.configInfo.Helper
	if m.configDraft.Role == RolePlanner {
		role = &m.configInfo.Planner
	}
	role.Provider = draftProviders[m.configDraft.ProviderIdx]
	role.Model = m.configDraft.Model
	role.Endpoint = m.configDraft.Endpoint
	role.APIKeyEnv = m.configDraft.APIKeyEnv
}

func (m *Model) loadRoleIntoDraft() {
	role := m.configInfo.Helper
	if m.configDraft.Role == RolePlanner {
		role = m.configInfo.Planner
	}
	provIdx := 0
	for i, p := range draftProviders {
		if p == role.Provider {
			provIdx = i
			break
		}
	}
	m.configDraft.ProviderIdx = provIdx
	m.configDraft.Model = role.Model
	m.configDraft.Endpoint = role.Endpoint
	m.configDraft.APIKeyEnv = role.APIKeyEnv
	m.configDraft.PerProviderModel = [3]string{}
	m.configDraft.PerProviderModel[provIdx] = role.Model
	m.configDraft.CursorAt = runeCount(role.Model)
}

func (m *Model) persistConfigOrLog() bool {
	if err := m.persistConfig(); err != nil {
		m.opLog.Add(oplog.Entry{Type: oplog.EntryLLMError, Summary: "Config save error: " + err.Error()})
		return false
	}
	return true
}

func (m Model) persistConfig() error {
	cfg := config.Get()
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	next := *cfg
	// Planner → Primary (RolePrimary), Helper → Secondary (RoleSecondary).
	p := m.configInfo.Planner
	next.LLM.Provider = p.Provider
	next.LLM.Primary.Provider = p.Provider
	next.LLM.Primary.Model = p.Model
	next.LLM.Primary.Endpoint = p.Endpoint
	next.LLM.Primary.APIKeyEnv = p.APIKeyEnv
	next.LLM.Primary.Enabled = true
	next.LLM.Endpoint = p.Endpoint
	next.LLM.Model = p.Model

	h := m.configInfo.Helper
	next.LLM.Secondary.Provider = h.Provider
	next.LLM.Secondary.Model = h.Model
	next.LLM.Secondary.Endpoint = h.Endpoint
	next.LLM.Secondary.APIKeyEnv = h.APIKeyEnv
	next.LLM.Secondary.Enabled = h.Model != ""

	next.Automation.Mode = m.mode
	if m.cruiseIntervalS > 0 {
		next.Automation.MonitorInterval = m.cruiseIntervalS
	}
	next.I18n.Language = m.language
	next.Suggestion.Language = m.language
	next.Theme.Name = m.configInfo.Theme
	if err := config.SaveGlobal(&next); err != nil {
		return err
	}
	// Reload effective config (defaults/global/project/env) so runtime switch
	// follows the exact same precedence used at startup.
	if _, err := config.Load(); err != nil {
		return fmt.Errorf("reload effective config: %w", err)
	}
	return nil
}

// applyLLMConfigRuntime rebuilds the LLM providers from the current config
// and propagates them to the orchestrator's flows.
func (m *Model) applyLLMConfigRuntime() bool {
	cfg := config.Get()
	if cfg == nil {
		return false
	}
	router, _, diag := llmfactory.BuildWithDiagnostics(cfg.LLM)
	if router == nil {
		observability.SetProviderAvailability(false)
		primary := cfg.LLM.PrimaryRole()
		m.opLog.Add(oplog.Entry{
			Type:    oplog.EntryLLMError,
			Summary: "LLM rebuild failed: no provider available",
			Detail: fmt.Sprintf("primary=%s/%s key_present=%t api_key_env=%q",
				config.RoleProvider(primary),
				primary.Model,
				config.ResolveRoleAPIKey(primary) != "",
				primary.APIKeyEnv,
			),
		})
		m.opLog.Add(oplog.Entry{
			Type:    oplog.EntryLLMError,
			Summary: "LLM diagnostics",
			Detail: fmt.Sprintf("primary=%s/%s (%s) secondary=%s/%s (%s) fallback=%t",
				diag.Primary.Health, diag.Primary.Code, diag.Primary.Reason,
				diag.Secondary.Health, diag.Secondary.Code, diag.Secondary.Reason,
				diag.FallbackPromoted,
			),
		})
		// Keep current runtime providers untouched on failed rebuild.
		return false
	}
	observability.SetProviderAvailability(true)
	// Transactional runtime reload: only after successful build do we switch runtime providers.
	m.helperLLM = router
	m.plannerLLM = router
	if m.orchestrator != nil {
		m.orchestrator.SetProviders(router, router)
	}
	m.opLog.Add(oplog.Entry{
		Type: oplog.EntryCmdSuccess,
		Summary: fmt.Sprintf("LLM runtime updated: planner=%s/%s helper=%s/%s",
			m.configInfo.Planner.Provider, m.configInfo.Planner.Model,
			m.configInfo.Helper.Provider, m.configInfo.Helper.Model),
		Detail: fmt.Sprintf("primary=%s/%s secondary=%s/%s fallback=%t",
			diag.Primary.Health, diag.Primary.Code,
			diag.Secondary.Health, diag.Secondary.Code,
			diag.FallbackPromoted,
		),
	})
	return true
}

func fetchOllamaModels(endpoint string) tea.Cmd {
	return func() tea.Msg {
		url := strings.TrimRight(endpoint, "/") + "/api/tags"
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Get(url)
		if err != nil {
			return ollamaModelsMsg{err: fmt.Errorf("connect to ollama: %w", err)}
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return ollamaModelsMsg{err: fmt.Errorf("ollama returned status %d", resp.StatusCode)}
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		if err != nil {
			return ollamaModelsMsg{err: fmt.Errorf("read response: %w", err)}
		}
		var out struct {
			Models []struct {
				Name    string `json:"name"`
				Size    int64  `json:"size"`
				Details struct {
					Family            string `json:"family"`
					ParameterSize     string `json:"parameter_size"`
					QuantizationLevel string `json:"quantization_level"`
				} `json:"details"`
			} `json:"models"`
		}
		if err := json.Unmarshal(body, &out); err != nil {
			return ollamaModelsMsg{err: fmt.Errorf("decode response: %w", err)}
		}
		models := make([]OllamaModelInfo, 0, len(out.Models))
		for _, m := range out.Models {
			models = append(models, OllamaModelInfo{
				Name:      m.Name,
				ParamSize: m.Details.ParameterSize,
				Family:    m.Details.Family,
				Quant:     m.Details.QuantizationLevel,
				Size:      m.Size,
			})
		}
		return ollamaModelsMsg{models: models}
	}
}

// ── Text editing helpers (ported from old TUI) ────────────────────────

func runeCount(s string) int {
	return len([]rune(s))
}

func clampRuneIdx(s string, pos int) int {
	n := runeCount(s)
	if pos < 0 {
		return 0
	}
	if pos > n {
		return n
	}
	return pos
}

func insertAtRune(text string, pos int, insert string) (string, int) {
	pos = clampRuneIdx(text, pos)
	runes := []rune(text)
	before := string(runes[:pos])
	after := string(runes[pos:])
	return before + insert + after, pos + runeCount(insert)
}

func deleteRuneBefore(text string, pos int) (string, int) {
	pos = clampRuneIdx(text, pos)
	if pos == 0 {
		return text, 0
	}
	runes := []rune(text)
	return string(append(runes[:pos-1], runes[pos:]...)), pos - 1
}

func deleteRuneAt(text string, pos int) (string, int) {
	pos = clampRuneIdx(text, pos)
	runes := []rune(text)
	if pos >= len(runes) {
		return text, pos
	}
	return string(append(runes[:pos], runes[pos+1:]...)), pos
}

func (m Model) handleConfigMenuKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "up", "k":
		if m.configMenuIdx > 0 {
			m.configMenuIdx--
		}
	case "down", "j":
		if m.configMenuIdx < 3 {
			m.configMenuIdx++
		}
	case "enter":
		switch m.configMenuIdx {
		case 0:
			m.page = PageConfigModel
			m.configEditing = false
			m.configDraft = ConfigDraft{Role: RoleHelper}
			m.loadRoleIntoDraft()
			m.configDraft.FieldIdx = 0
			provID := draftProviders[m.configDraft.ProviderIdx]
			m.ollamaModels = nil
			m.ollamaFetchError = ""
			m.ollamaModelIdx = 0
			if provID == "ollama" {
				ep := m.configDraft.Endpoint
				if ep == "" {
					ep = llm.DefaultOllamaURL
				}
				m.ollamaFetching = true
				return m, fetchOllamaModels(ep)
			}
		case 1:
			m.page = PageConfigMode
			m.configModeIdx = modeToIdx(m.mode)
		case 2:
			m.page = PageConfigLang
			m.configLangIdx = langToIdx(m.language)
		case 3:
			m.page = PageConfigTheme
			m.configThemeIdx = themeToIdx(m.configInfo.Theme)
		}
	}
	return m, nil
}

func (m Model) handleConfigModeKeys(key string) (tea.Model, tea.Cmd) {
	if m.editingInterval {
		return m.handleIntervalEditKeys(key)
	}

	switch key {
	case "up", "k":
		if m.configModeIdx > 0 {
			m.configModeIdx--
		}
	case "down", "j":
		if m.configModeIdx < 3 {
			m.configModeIdx++
		}
	case "enter":
		if m.configModeIdx <= 2 {
			modes := []string{"manual", "auto", "cruise"}
			m.mode = modes[m.configModeIdx]
			m.cruiseCycleActive = false
			m.persistConfigOrLog()
			m.opLog.Add(oplog.Entry{Type: oplog.EntryUserAction, Summary: "Mode changed to " + m.mode})
			m.page = PageConfig
			if m.mode == "auto" || m.mode == "cruise" {
				if m.findNextPending() >= 0 {
					return m, m.executeNext()
				}
				return m, m.startAnalysis()
			}
		} else {
			m.editingInterval = true
			m.intervalBuf = strconv.Itoa(m.cruiseIntervalS)
		}
	}
	return m, nil
}

func (m Model) handleIntervalEditKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "enter":
		m.editingInterval = false
		val, err := strconv.Atoi(m.intervalBuf)
		if err != nil || val < 60 {
			val = 60
		}
		m.cruiseIntervalS = val
		m.configInfo.CruiseInterval = val
		if m.orchestrator != nil {
			m.orchestrator.Interval = time.Duration(val) * time.Second
		}
		m.persistConfigOrLog()
		m.opLog.Add(oplog.Entry{Type: oplog.EntryUserAction,
			Summary: fmt.Sprintf("Cruise interval set to %ds", val)})
		return m, nil
	case "backspace":
		if len(m.intervalBuf) > 0 {
			m.intervalBuf = m.intervalBuf[:len(m.intervalBuf)-1]
		}
	default:
		if isEscKey(key) {
			m.editingInterval = false
			return m, nil
		}
		if len(key) == 1 && key[0] >= '0' && key[0] <= '9' {
			if len(m.intervalBuf) < 6 {
				m.intervalBuf += key
			}
		}
	}
	return m, nil
}

func (m Model) handleConfigLangKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "up", "k":
		if m.configLangIdx > 0 {
			m.configLangIdx--
		}
	case "down", "j":
		if m.configLangIdx < 2 {
			m.configLangIdx++
		}
	case "enter":
		langs := []string{"en", "zh", "ja"}
		m.language = langs[m.configLangIdx]
		m.configInfo.Language = m.language
		m.persistConfigOrLog()
		m.opLog.Add(oplog.Entry{Type: oplog.EntryUserAction, Summary: "Language changed to " + m.language})
		m.page = PageConfig
	}
	return m, nil
}

func (m Model) handleConfigThemeKeys(key string) (tea.Model, tea.Cmd) {
	allThemes := theme.Names()
	maxIdx := len(allThemes) - 1
	switch key {
	case "up", "k":
		if m.configThemeIdx > 0 {
			m.configThemeIdx--
		}
	case "down", "j":
		if m.configThemeIdx < maxIdx {
			m.configThemeIdx++
		}
	case "enter":
		if m.configThemeIdx >= 0 && m.configThemeIdx < len(allThemes) {
			m.configInfo.Theme = allThemes[m.configThemeIdx]
			theme.Init(m.configInfo.Theme)
			m.persistConfigOrLog()
			m.opLog.Add(oplog.Entry{Type: oplog.EntryUserAction, Summary: "Theme changed to " + m.configInfo.Theme})
			m.page = PageConfig
		}
	}
	return m, nil
}

func modeToIdx(mode string) int {
	switch mode {
	case "manual":
		return 0
	case "auto":
		return 1
	case "cruise":
		return 2
	}
	return 0
}

func langToIdx(lang string) int {
	switch lang {
	case "en":
		return 0
	case "zh":
		return 1
	case "ja":
		return 2
	}
	return 0
}

func themeToIdx(t string) int {
	for i, name := range theme.Names() {
		if name == t {
			return i
		}
	}
	return 0
}

const maxConsecutiveReplans = 8

func suggestionSignatures(suggs []SuggestionDisplay) []string {
	sigs := make([]string, len(suggs))
	for i, s := range suggs {
		sigs[i] = s.Item.Action.Type + "|" + s.Item.Action.Command + "|" + s.Item.Action.FilePath + "|" + s.Item.Action.FileOp
	}
	return sigs
}

func signaturesEqual(a, b []string) bool {
	if len(a) != len(b) || len(a) == 0 {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func replanBackoffDelay(attempt int) time.Duration {
	if attempt <= 0 {
		return 1 * time.Second
	}
	// Exponential backoff capped at 60s: 1,2,4,8,16,32,60...
	secs := 1 << (attempt - 1)
	if secs > 60 {
		secs = 60
	}
	return time.Duration(secs) * time.Second
}

// ---------- Composer submit ----------

func (m Model) handleComposerSubmit() (tea.Model, tea.Cmd) {
	text := strings.TrimSpace(m.composerText)
	m.composerText = ""

	if text == "" {
		return m, nil
	}

	if !strings.HasPrefix(text, "/") {
		m.opLog.Add(oplog.Entry{
			Type:    oplog.EntryUserAction,
			Summary: "Unknown input. All commands start with /. Try /help",
		})
		return m, nil
	}

	oa, err := ParseObjectActionFromSlash(text)
	if err != nil {
		m.opLog.Add(oplog.Entry{
			Type:    oplog.EntryUserAction,
			Summary: fmt.Sprintf("Unknown command: %s. Try /help", text),
		})
		return m, nil
	}
	arg := oa.Arg

	switch oa.Key() {
	case "goal.create":
		if arg == "" {
			m.opLog.Add(oplog.Entry{Type: oplog.EntryUserAction, Summary: "Usage: /goal <description>"})
			return m, nil
		}
		m.opLog.Add(oplog.Entry{Type: oplog.EntryUserAction, Summary: "Goal submitted: " + arg})
		return m, m.triageGoal(arg)

	case "suggestion.execute":
		switch arg {
		case "accept":
			if m.mode != "manual" {
				m.opLog.Add(oplog.Entry{Type: oplog.EntryUserAction, Summary: "/run accept is only for manual mode"})
				return m, nil
			}
			return m, m.executeNext()
		case "all":
			if m.mode != "manual" {
				m.opLog.Add(oplog.Entry{Type: oplog.EntryUserAction, Summary: "/run all is only for manual mode"})
				return m, nil
			}
			m.runAllMode = true
			return m, m.executeNext()
		default:
			m.opLog.Add(oplog.Entry{Type: oplog.EntryUserAction, Summary: "Usage: /run accept | /run all"})
			return m, nil
		}

	case "mode.set":
		switch arg {
		case "manual":
			m.mode = "manual"
			m.cruiseCycleActive = false
			m.persistConfigOrLog()
			m.opLog.Add(oplog.Entry{Type: oplog.EntryUserAction, Summary: "Switched to manual mode"})
			return m, nil
		case "auto":
			m.mode = "auto"
			m.cruiseCycleActive = false
			m.persistConfigOrLog()
			m.opLog.Add(oplog.Entry{Type: oplog.EntryUserAction, Summary: "Switched to auto mode"})
			if m.findNextPending() >= 0 {
				return m, m.executeNext()
			}
			return m, m.startAnalysis()
		case "cruise":
			m.mode = "cruise"
			m.cruiseCycleActive = false
			m.persistConfigOrLog()
			m.opLog.Add(oplog.Entry{Type: oplog.EntryUserAction, Summary: "Switched to cruise mode"})
			if m.findNextPending() >= 0 {
				return m, m.executeNext()
			}
			return m, m.startAnalysis()
		default:
			m.opLog.Add(oplog.Entry{Type: oplog.EntryUserAction, Summary: "Usage: /mode manual | auto | cruise"})
			return m, nil
		}

	case "config.set":
		if arg == "" {
			m.page = PageConfig
			m.configMenuIdx = 0
			return m, nil
		}
		return m.handleInlineConfigSet(arg)

	case "creative.run":
		m.opLog.Add(oplog.Entry{Type: oplog.EntryUserAction,
			Summary: "Manual creative flow triggered"})
		return m, m.startCreativeFlow()

	case "flow.analyze":
		return m, m.startAnalysis()

	case "cruise.set_interval":
		if arg == "" {
			m.opLog.Add(oplog.Entry{Type: oplog.EntryUserAction,
				Summary: fmt.Sprintf("Current cruise interval: %ds (%s). Usage: /interval <seconds>",
					m.cruiseIntervalS, formatDuration(m.cruiseIntervalS))})
			return m, nil
		}
		val, err := strconv.Atoi(arg)
		if err != nil || val < 60 {
			m.opLog.Add(oplog.Entry{Type: oplog.EntryUserAction,
				Summary: "Cruise interval must be >= 60 seconds"})
			return m, nil
		}
		m.cruiseIntervalS = val
		m.configInfo.CruiseInterval = val
		if m.orchestrator != nil {
			m.orchestrator.Interval = time.Duration(val) * time.Second
		}
		m.persistConfigOrLog()
		m.opLog.Add(oplog.Entry{Type: oplog.EntryUserAction,
			Summary: fmt.Sprintf("Cruise interval set to %ds (%s)", val, formatDuration(val))})
		return m, nil

	case "ui.help":
		m.showHelpOverlay = true
		m.opLog.Add(oplog.Entry{
			Type:    oplog.EntryUserAction,
			Summary: "Help overlay opened",
		})
		return m, nil

	case "ui.command_palette":
		m.showCommandPalette = true
		m.showHelpOverlay = false
		m.paletteQuery = ""
		m.paletteIdx = 0
		return m, nil

	case "llm.probe":
		var cmds []tea.Cmd
		if m.helperLLM != nil {
			cmds = append(cmds, m.runConnectivityTest("helper", m.helperLLM))
		}
		if m.plannerLLM != nil && m.plannerLLM != m.helperLLM {
			cmds = append(cmds, m.runConnectivityTest("planner", m.plannerLLM))
		}
		if len(cmds) == 0 {
			m.opLog.Add(oplog.Entry{Type: oplog.EntryLLMError, Summary: "No LLM provider configured"})
			return m, nil
		}
		m.opLog.Add(oplog.Entry{Type: oplog.EntryStateRefresh, Summary: "Running LLM connectivity tests..."})
		return m, tea.Batch(cmds...)

	case "ui.clear_log":
		m.opLog = oplog.New(200)
		m.roundHistory = nil
		return m, nil

	case "metrics.failure_dashboard":
		d := oplog.BuildFailureDashboard(m.opLog.Entries())
		m.opLog.Add(oplog.Entry{
			Type:    oplog.EntryStateRefresh,
			Summary: "Failure taxonomy dashboard",
			Detail:  d.Render(),
		})
		return m, nil

	case "execution.replay":
		ol := dotgitdex.NewOutputLog(m.store)
		script, err := ol.BuildReplayScript(3)
		if err != nil {
			m.opLog.Add(oplog.Entry{
				Type:    oplog.EntryLLMError,
				Summary: "Replay script generation failed",
				Detail:  err.Error(),
			})
			return m, nil
		}
		if strings.TrimSpace(script) == "" {
			m.opLog.Add(oplog.Entry{
				Type:    oplog.EntryStateRefresh,
				Summary: "Replay script: no recent rounds to replay",
			})
			return m, nil
		}
		m.opLog.Add(oplog.Entry{
			Type:    oplog.EntryStateRefresh,
			Summary: "Replay script generated (last 3 rounds)",
			Detail:  script,
		})
		return m, nil

	default:
		m.opLog.Add(oplog.Entry{
			Type:    oplog.EntryUserAction,
			Summary: fmt.Sprintf("Unknown command: %s. Try /help", oa.Raw),
		})
		return m, nil
	}
}

func (m Model) handleInlineConfigSet(arg string) (tea.Model, tea.Cmd) {
	parts := strings.SplitN(arg, " ", 2)
	key := strings.ToLower(parts[0])
	val := ""
	if len(parts) > 1 {
		val = strings.TrimSpace(parts[1])
	}

	if val == "" {
		m.opLog.Add(oplog.Entry{Type: oplog.EntryUserAction, Summary: fmt.Sprintf("Usage: /config %s <value>", key)})
		return m, nil
	}

	switch key {
	case "mode":
		switch val {
		case "manual", "auto", "cruise":
			m.mode = val
			m.cruiseCycleActive = false
			m.persistConfigOrLog()
			m.opLog.Add(oplog.Entry{Type: oplog.EntryUserAction, Summary: "Mode changed to " + val})
			if val == "auto" || val == "cruise" {
				if m.findNextPending() >= 0 {
					return m, m.executeNext()
				}
				return m, m.startAnalysis()
			}
			return m, nil
		default:
			m.opLog.Add(oplog.Entry{Type: oplog.EntryUserAction, Summary: "Valid modes: manual, auto, cruise"})
			return m, nil
		}
	case "language", "lang":
		m.language = val
		m.configInfo.Language = val
		m.persistConfigOrLog()
		m.opLog.Add(oplog.Entry{Type: oplog.EntryUserAction, Summary: "Language changed to " + val})
		return m, nil
	default:
		m.opLog.Add(oplog.Entry{
			Type:    oplog.EntryUserAction,
			Summary: fmt.Sprintf("Unknown config key: %s. Editable: mode, language", key),
		})
		return m, nil
	}
}

// ---------- Flow commands ----------

// beginAnalysis sets m.analyzing = true on the live model and returns a Cmd.
// Because Update() passes Model by value, callers must use the returned Model:
//   m, cmd = m.beginAnalysis()  — but since it returns (Model, tea.Cmd), use via:
//   return m, m.startAnalysis()   where startAnalysis is called after m.analyzing is set.
//
// Since Bubbletea model is a value type in Update(), we use a pattern where
// these methods are called in the same scope where m is assigned back.

func (m *Model) startAnalysis() tea.Cmd {
	if m.analyzing {
		return nil
	}
	m.analyzing = true
	goals, _ := m.store.ReadGoalList()
	hasPending := len(dotgitdex.PendingGoals(goals)) > 0
	orch := m.orchestrator

	return func() tea.Msg {
		ctx := context.Background()
		if hasPending {
			round, err := orch.RunGoalRound(ctx)
			return flowRoundMsg{flow: "goal", round: round, err: err}
		}
		round, err := orch.RunMaintainRound(ctx)
		return flowRoundMsg{flow: "maintain", round: round, err: err}
	}
}

func (m *Model) startGoalAnalysis() tea.Cmd {
	goals, _ := m.store.ReadGoalList()
	hasPending := len(dotgitdex.PendingGoals(goals)) > 0
	if !hasPending {
		if m.mode == "cruise" {
			return func() tea.Msg { return cruiseCycleCompleteMsg{} }
		}
		return nil
	}
	m.analyzing = true
	orch := m.orchestrator
	return func() tea.Msg {
		ctx := context.Background()
		round, err := orch.RunGoalRound(ctx)
		return flowRoundMsg{flow: "goal", round: round, err: err}
	}
}

func (m *Model) startCreativeFlow() tea.Cmd {
	if m.analyzing {
		return nil
	}
	m.analyzing = true
	m.opLog.Add(oplog.Entry{Type: oplog.EntryStateRefresh,
		Summary: "Running creative flow (Planner + Prompt E)..."})
	orch := m.orchestrator
	return func() tea.Msg {
		ctx := context.Background()
		result, err := orch.RunCreativeRound(ctx)
		return creativeResultMsg{result: result, err: err}
	}
}

func (m *Model) startMaintainAnalysis() tea.Cmd {
	if m.analyzing {
		return nil
	}
	m.analyzing = true
	orch := m.orchestrator
	return func() tea.Msg {
		ctx := context.Background()
		round, err := orch.RunMaintainRound(ctx)
		return flowRoundMsg{flow: "maintain", round: round, err: err}
	}
}

func (m Model) triageGoal(goalTitle string) tea.Cmd {
	m.opLog.Add(oplog.Entry{Type: oplog.EntryStateRefresh,
		Summary: "Triaging goal: classifying and decomposing..."})
	return func() tea.Msg {
		ctx := context.Background()
		gitContent, _ := m.store.ReadGitContent()
		if m.orchestrator == nil || m.orchestrator.Goal == nil || m.orchestrator.Goal.GoalHelper == nil {
			return goalTriageMsg{goalTitle: goalTitle, err: fmt.Errorf("helper LLM not configured")}
		}
		result, err := m.orchestrator.Goal.GoalHelper.TriageAndDecomposeGoal(ctx, goalTitle, gitContent)
		return goalTriageMsg{goalTitle: goalTitle, result: result, err: err}
	}
}

func (m Model) decomposeGoal(goalTitle string) tea.Cmd {
	m.opLog.Add(oplog.Entry{Type: oplog.EntryStateRefresh,
		Summary: "Decomposing goal into To Do list..."})
	return func() tea.Msg {
		ctx := context.Background()
		gitContent, _ := m.store.ReadGitContent()
		if m.orchestrator.Goal == nil || m.orchestrator.Goal.GoalHelper == nil {
			return goalDecomposedMsg{goalTitle: goalTitle, err: fmt.Errorf("helper LLM not configured")}
		}
		todos, err := m.orchestrator.Goal.GoalHelper.DecomposeGoal(ctx, goalTitle, gitContent)
		return goalDecomposedMsg{goalTitle: goalTitle, todos: todos, err: err}
	}
}

func (m Model) updateGoalProgress() tea.Cmd {
	return m.updateGoalProgressWithReplan(false)
}

func (m Model) updateGoalProgressWithReplan(replan bool) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := m.orchestrator.UpdateGoalProgress(ctx)
		return goalProgressUpdatedMsg{err: err, replan: replan}
	}
}

// continueFlowLoop decides the next action after a round completes and
// goal progress has been updated. This implements the closed-loop design:
// maintain → goal → check done → loop or stop.
// In cruise mode, when the cycle finishes it sends cruiseCycleCompleteMsg
// instead of directly scheduling the next cruiseTickMsg, so the handler
// can clear the cruiseCycleActive flag first.
func (m Model) continueFlowLoop() tea.Cmd {
	goals, _ := m.store.ReadGoalList()
	hasPending := len(dotgitdex.PendingGoals(goals)) > 0

	if m.activeFlow == "maintain" {
		if hasPending {
			return m.startGoalAnalysis()
		}
		if m.mode == "auto" || m.mode == "cruise" {
			if m.orchestrator.IsMaintainClean(context.Background()) {
				m.opLog.Add(oplog.Entry{Type: oplog.EntryCmdSuccess,
					Summary: "All goals completed and repository is clean"})
				if m.mode == "cruise" {
					return func() tea.Msg { return cruiseCycleCompleteMsg{} }
				}
				return nil
			}
			return m.startAnalysis()
		}
		return nil
	}

	// Goal flow completed a round
	if hasPending {
		if m.mode == "auto" || m.mode == "cruise" {
			return m.startAnalysis()
		}
		return nil
	}

	// All goals done: check repo cleanliness
	if m.mode == "auto" || m.mode == "cruise" {
		if m.orchestrator.IsMaintainClean(context.Background()) {
			m.opLog.Add(oplog.Entry{Type: oplog.EntryCmdSuccess,
				Summary: "All goals completed and repository is clean"})
			if m.mode == "cruise" {
				return func() tea.Msg { return cruiseCycleCompleteMsg{} }
			}
			return nil
		}
		return m.startMaintainAnalysis()
	}
	return nil
}

func (m *Model) executeNext() tea.Cmd {
	idx := m.findNextPending()
	if idx < 0 {
		return nil
	}
	m.suggestions[idx].Status = StatusExecuting
	m.executing = true
	item := m.suggestions[idx].Item
	orch := m.orchestrator

	return func() tea.Msg {
		trace := contract.TraceMetadata{TraceID: observability.NewTraceID()}
		if m.currentRound != nil {
			trace.RoundID = m.currentRound.RoundID
			trace.AttemptID = m.currentRound.AttemptID
			trace.SliceID = m.currentRound.SliceID
		}
		ctx := observability.WithTrace(context.Background(), trace)
		result := orch.ExecuteSingleSuggestion(ctx, idx+1, item)
		return executionResultMsg{index: idx, result: result}
	}
}

func (m *Model) syncAgentTable() {
	var rows []table.Row
	for _, s := range m.suggestions {
		status := ""
		switch s.Status {
		case StatusPending:
			status = "WAIT"
		case StatusExecuting:
			status = "RUN"
		case StatusDone:
			status = "OK"
		case StatusFailed:
			status = "ERR"
		case StatusSkipped:
			status = "SKIP"
		}
		
		cmdStr := ""
		if len(s.Item.Action.Command) > 0 {
			cmdStr = s.Item.Action.Command
		} else if s.Item.Action.FilePath != "" {
			cmdStr = s.Item.Action.FileOp + " " + s.Item.Action.FilePath
		}
		
		rows = append(rows, table.Row{status, s.Item.Action.Type, cmdStr})
	}
	m.agentTable.SetRows(rows)
}

func (m *Model) syncSidebar() {
	if len(m.suggestions) == 0 {
		geo := m.calcLayout()
		m.sidebarComp.SetContent(m.renderRightPanel(geo.rightW, geo.contentH, geo))
		return
	}
	idx := m.agentTable.CurrIdx()
	if idx < 0 || idx >= len(m.suggestions) {
		return
	}
	s := m.suggestions[idx]
	
	var sb strings.Builder
	sb.WriteString(titleStyle().Render("◆ " + s.Item.Name) + "\n\n")
	sb.WriteString(infoStyle().Render("Reason: ") + s.Item.Reason + "\n\n")
	
	cmdStr := ""
	if len(s.Item.Action.Command) > 0 {
		cmdStr = s.Item.Action.Command
	} else if s.Item.Action.FilePath != "" {
		cmdStr = s.Item.Action.FileOp + " " + s.Item.Action.FilePath
	}
	sb.WriteString(commandStyle().Render("$ " + cmdStr) + "\n\n")
	
	if s.Output != "" {
		sb.WriteString(successStyle().Render("Output:") + "\n" + s.Output + "\n\n")
	}
	if s.Error != "" {
		sb.WriteString(dangerStyle().Render("Error:") + "\n" + s.Error + "\n\n")
	}
	
	m.sidebarComp.SetContent(sb.String())
}

func (m Model) findNextPending() int {
	for i, s := range m.suggestions {
		if s.Status == StatusPending {
			return i
		}
	}
	return -1
}

func (m *Model) compressCurrentRound() {
	var cmds []string
	for _, s := range m.suggestions {
		if s.Status == StatusDone && s.Item.Action.Command != "" {
			cmds = append(cmds, s.Item.Action.Command)
		}
	}
	flowName := "maintain"
	if m.currentRound != nil {
		flowName = m.currentRound.Flow
	}
	m.roundHistory = append(m.roundHistory, CompressedRound{
		Commands: cmds,
		Flow:     flowName,
	})
}

// ---------- Git content refresh ----------

func (m Model) refreshGitInfo() tea.Cmd {
	return func() tea.Msg {
		content, err := m.store.ReadGitContent()
		if err != nil || content == "" {
			return gitRefreshMsg{}
		}
		return gitRefreshMsg{info: parseGitSnapshot(content)}
	}
}

func (m Model) runConnectivityTest(role string, provider llm.LLMProvider) tea.Cmd {
	if provider == nil {
		return nil
	}
	// Planner uses RolePrimary, Helper uses RoleSecondary.
	llmRole := llm.RoleSecondary
	if role == "planner" {
		llmRole = llm.RolePrimary
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		start := time.Now()
		provName := provider.Name()

		if !provider.IsAvailable(ctx) {
			return llmConnectivityMsg{
				role:     role,
				provider: provName,
				ok:       false,
				err:      "provider not available (missing API key or server unreachable)",
			}
		}

		info, err := provider.ModelInfo(ctx)
		modelName := ""
		if err != nil {
			return llmConnectivityMsg{
				role:      role,
				provider:  provName,
				ok:        false,
				err:       "model info failed: " + err.Error(),
				latencyMs: time.Since(start).Milliseconds(),
			}
		}
		if info != nil {
			modelName = info.Name
		}

		resp, err := provider.Generate(ctx, llm.GenerateRequest{
			System: "You are a connectivity test. Reply with exactly one word: OK",
			Prompt: "ping",
			Role:   llmRole,
		})
		elapsed := time.Since(start).Milliseconds()

		if err != nil {
			return llmConnectivityMsg{
				role:      role,
				provider:  provName,
				model:     modelName,
				ok:        false,
				err:       err.Error(),
				latencyMs: elapsed,
			}
		}
		if resp == nil || strings.TrimSpace(resp.Text) == "" {
			return llmConnectivityMsg{
				role:      role,
				provider:  provName,
				model:     modelName,
				ok:        false,
				err:       "empty response from provider",
				latencyMs: elapsed,
			}
		}
		return llmConnectivityMsg{
			role:      role,
			provider:  provName,
			model:     modelName,
			ok:        true,
			latencyMs: elapsed,
		}
	}
}

// parseGitSnapshot is a section-aware state machine that parses
// all sections of the git-content.txt file into a GitSnapshot.
func parseGitSnapshot(content string) GitSnapshot {
	info := GitSnapshot{}
	scanner := bufio.NewScanner(strings.NewReader(content))

	section := ""
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Detect section headers
		if strings.HasPrefix(trimmed, "## ") {
			section = strings.TrimPrefix(trimmed, "## ")
			continue
		}

		// Skip comments and blanks
		if strings.HasPrefix(trimmed, "#") || trimmed == "" {
			continue
		}

		// Top-level scalars (before any ## section)
		if section == "" {
			parseScalar(&info, trimmed)
			continue
		}

		switch section {
		case "Local Branches":
			if b, ok := parseBranchLine(trimmed); ok {
				info.LocalBranches = append(info.LocalBranches, b)
			}
		case "Merged Branches":
			info.MergedBranches = append(info.MergedBranches, trimmed)
		case "Remote Branches":
			info.RemoteBranches = append(info.RemoteBranches, trimmed)
		case "Remotes":
			if r, ok := parseRemoteLine(trimmed); ok {
				info.Remotes = append(info.Remotes, r)
			}
		case "Upstream":
			parseUpstream(&info, trimmed)
		case "Working Tree Changes":
			info.WorkingFiles = append(info.WorkingFiles, trimmed)
			info.WorkingDirty++
		case "Staging Area":
			info.StagingFiles = append(info.StagingFiles, trimmed)
			info.StagingDirty++
		case "Repository State":
			parseRepoState(&info, trimmed)
		case "Stash":
			info.StashEntries = append(info.StashEntries, trimmed)
			info.Stash++
		case "Tags":
			info.Tags = append(info.Tags, trimmed)
		case "Submodules":
			info.Submodules = append(info.Submodules, trimmed)
		case "Recent Reflog":
			info.RecentReflog = append(info.RecentReflog, trimmed)
		case "Ahead Commits":
			info.AheadCommits = append(info.AheadCommits, trimmed)
		case "Behind Commits":
			info.BehindCommits = append(info.BehindCommits, trimmed)
		case "Config":
			parseConfigLine(&info, trimmed)
		case "Commit Summary":
			parseCommitSummary(&info, trimmed)
		case "Summary":
			parseSummaryLine(&info, trimmed)
		}
	}

	// Derive ahead/behind from commits if not set by Summary
	if info.Ahead == 0 && len(info.AheadCommits) > 0 {
		info.Ahead = len(info.AheadCommits)
	}
	if info.Behind == 0 && len(info.BehindCommits) > 0 {
		info.Behind = len(info.BehindCommits)
	}

	// Mark merged branches
	mergedSet := make(map[string]bool)
	for _, b := range info.MergedBranches {
		mergedSet[b] = true
	}
	for i := range info.LocalBranches {
		if mergedSet[info.LocalBranches[i].Name] {
			info.LocalBranches[i].IsMerged = true
		}
	}

	return info
}

func parseScalar(info *GitSnapshot, line string) {
	if k, v, ok := splitKV(line); ok {
		switch k {
		case "current_branch":
			info.Branch = v
		case "detached_head":
			info.Detached = v == "true"
		case "head_ref":
			info.HeadRef = v
		case "is_initial":
			info.IsInitial = v == "true"
		case "commit_count":
			info.CommitCount, _ = strconv.Atoi(v)
		case "default_branch":
			info.DefaultBranch = v
		}
	}
}

func parseBranchLine(line string) (BranchSnap, bool) {
	b := BranchSnap{}
	if strings.HasPrefix(line, "* ") {
		b.IsCurrent = true
		line = strings.TrimPrefix(line, "* ")
	} else {
		line = strings.TrimSpace(line)
	}

	// Format: name [-> upstream] [ahead N, behind N] [(merged)] [| last_commit]
	if idx := strings.Index(line, " | "); idx >= 0 {
		b.Last = strings.TrimSpace(line[idx+3:])
		line = line[:idx]
	}

	if strings.HasSuffix(line, "(merged)") {
		b.IsMerged = true
		line = strings.TrimSuffix(line, "(merged)")
		line = strings.TrimSpace(line)
	}

	// Parse [ahead N, behind N]
	if bidx := strings.Index(line, "[ahead "); bidx >= 0 {
		end := strings.Index(line[bidx:], "]")
		if end >= 0 {
			bracket := line[bidx+1 : bidx+end]
			for _, part := range strings.Split(bracket, ",") {
				part = strings.TrimSpace(part)
				if strings.HasPrefix(part, "ahead ") {
					b.Ahead, _ = strconv.Atoi(strings.TrimPrefix(part, "ahead "))
				} else if strings.HasPrefix(part, "behind ") {
					b.Behind, _ = strconv.Atoi(strings.TrimPrefix(part, "behind "))
				}
			}
			line = strings.TrimSpace(line[:bidx])
		}
	}

	// Parse -> upstream
	if idx := strings.Index(line, " -> "); idx >= 0 {
		b.Upstream = strings.TrimSpace(line[idx+4:])
		line = line[:idx]
	}

	b.Name = strings.TrimSpace(line)
	if b.Name == "" {
		return b, false
	}
	return b, true
}

func parseRemoteLine(line string) (RemoteSnap, bool) {
	// Format: name  fetch=URL  push=URL
	r := RemoteSnap{}
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return r, false
	}
	r.Name = parts[0]
	for _, p := range parts[1:] {
		if strings.HasPrefix(p, "fetch=") {
			r.FetchURL = strings.TrimPrefix(p, "fetch=")
		} else if strings.HasPrefix(p, "push=") {
			r.PushURL = strings.TrimPrefix(p, "push=")
		}
	}
	return r, true
}

func parseUpstream(info *GitSnapshot, line string) {
	// Format: name  ahead=N  behind=N
	parts := strings.Fields(line)
	for _, p := range parts {
		if strings.HasPrefix(p, "ahead=") {
			info.Ahead, _ = strconv.Atoi(strings.TrimPrefix(p, "ahead="))
		} else if strings.HasPrefix(p, "behind=") {
			info.Behind, _ = strconv.Atoi(strings.TrimPrefix(p, "behind="))
		}
	}
}

func parseRepoState(info *GitSnapshot, line string) {
	k, v, ok := splitKV(line)
	if !ok {
		return
	}
	if v != "true" {
		return
	}
	switch k {
	case "merge_in_progress":
		info.MergeInProgress = true
	case "rebase_in_progress":
		info.RebaseInProgress = true
	case "cherry_pick_in_progress":
		info.CherryInProgress = true
	case "bisect_in_progress":
		info.BisectInProgress = true
	}
}

func parseConfigLine(info *GitSnapshot, line string) {
	k, v, ok := splitKV(line)
	if !ok {
		return
	}
	switch k {
	case "user.name":
		info.UserName = v
	case "user.email":
		info.UserEmail = v
	}
}

func parseCommitSummary(info *GitSnapshot, line string) {
	k, v, ok := splitKV(line)
	if !ok {
		return
	}
	switch k {
	case "commit_frequency":
		info.CommitFreq = v
	case "last_commit":
		info.LastCommit = v
	}
}

func parseSummaryLine(info *GitSnapshot, line string) {
	k, v, ok := splitKV(line)
	if !ok {
		return
	}
	n, _ := strconv.Atoi(v)
	switch k {
	case "working_tree_dirty":
		if info.WorkingDirty == 0 {
			info.WorkingDirty = n
		}
	case "staging_area_dirty":
		if info.StagingDirty == 0 {
			info.StagingDirty = n
		}
	}
}

func splitKV(line string) (string, string, bool) {
	idx := strings.Index(line, ":")
	if idx < 0 {
		return "", "", false
	}
	return strings.TrimSpace(line[:idx]), strings.TrimSpace(line[idx+1:]), true
}
