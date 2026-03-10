package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
	"github.com/Joker-of-Gotham/gitdex/internal/i18n"
	"github.com/Joker-of-Gotham/gitdex/internal/llm"
	"github.com/Joker-of-Gotham/gitdex/internal/llm/ollama"
	"github.com/Joker-of-Gotham/gitdex/internal/tui/oplog"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyPressMsg:
		switch m.screen {
		case screenLanguageSelect:
			return m.updateLanguageSelect(msg)
		case screenModelSelect:
			return m.updateModelSelect(msg)
		case screenInput:
			return m.updateInput(msg)
		case screenGoalInput:
			return m.updateGoalInput(msg)
		case screenWorkflowSelect:
			return m.updateWorkflowSelect(msg)
		default:
			return m.updateMain(msg)
		}

	case tea.PasteMsg:
		if m.screen == screenInput {
			return m.handleInputPaste(msg.Content)
		}
		if m.screen == screenGoalInput {
			return m.handleGoalPaste(msg.Content)
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		return m, nil

	case ollamaModelsMsg:
		m.modelsFetched = true
		if msg.err == nil && len(msg.models) > 0 {
			m.availModels = msg.models
			m.modelCursor = 0
			m.modelSelectPhase = selectPrimary
			if m.shouldShowFirstRunLanguageSelection() {
				m = m.openLanguageSelection(screenLoading)
			} else if strings.TrimSpace(m.selectedPrimary) == "" {
				m.screen = screenModelSelect
			}
			m = m.addLog(oplog.Entry{
				Type:    oplog.EntryStateRefresh,
				Summary: fmt.Sprintf("Ollama models loaded: %d", len(msg.models)),
			})
		} else {
			m.setWorkflowStage(workflowConfirm)
			if m.shouldShowFirstRunLanguageSelection() {
				m = m.openLanguageSelection(screenLoading)
			} else {
				m.screen = screenMain
			}
			m.llmAnalysis = i18n.T("analysis.no_ollama")
			m = m.addLog(oplog.Entry{
				Type:    oplog.EntryLLMError,
				Summary: "Ollama not detected; background check scheduled",
			})
			return m, scheduleOllamaCheck()
		}
		return m, nil

	case ollamaCheckTickMsg:
		if m.llmProvider != nil && m.llmProvider.IsAvailable(context.Background()) {
			return m, tea.Cmd(m.fetchOllamaModels)
		}
		return m, scheduleOllamaCheck()

	case gitStateMsg:
		m.gitState = msg.state
		m.setWorkflowStage(workflowPerceive)
		m = m.addLog(oplog.Entry{
			Type:    oplog.EntryStateRefresh,
			Summary: "State refreshed: " + summarizeGitState(msg.state),
		})
		if msg.state != nil && len(msg.state.WorkingTree) == 0 && len(msg.state.StagingArea) == 0 {
			m.analysisHistory = nil
		}
		if m.screen == screenMain || m.screen == screenLoading {
			if m.screen == screenLoading {
				if m.shouldShowFirstRunLanguageSelection() {
					m = m.openLanguageSelection(screenLoading)
					return m, nil
				}
				m.screen = screenMain
			}
			if msg.skipAnalysis || m.skipNextAnalysis {
				m.skipNextAnalysis = false
				m.revalidatePendingSuggestions()
				return m, nil
			}
			if m.llmProvider != nil {
				m.analysisSeq++
				m.pendingAnalysisID = m.analysisSeq
				m.setWorkflowStage(workflowAnalyze)
				m.llmAnalysis = i18n.T("analysis.analyzing")
				m.statusMsg = i18n.T("analysis.in_progress_status")
				m.logScrollOffset = 0

				detail := m.buildAnalysisStartDetail()
				m = m.addLog(oplog.Entry{
					Type:    oplog.EntryLLMStart,
					Summary: fmt.Sprintf("LLM analysis started (%s mode)", m.mode),
					Detail:  detail,
				})
				return m, m.runLLMAnalysis(m.pendingAnalysisID, m.gitState)
			}
			m.llmAnalysis = i18n.T("analysis.no_ollama")
		}
		return m, nil

	case llmResultMsg:
		if msg.requestID != m.pendingAnalysisID {
			return m, nil
		}
		m.llmDebugInfo = msg.debugInfo
		m.analysisTrace = msg.trace
		if msg.err != nil {
			m.setWorkflowStage(workflowSuggest)
			m.llmAnalysis = fmt.Sprintf(i18n.T("analysis.error_prefix"), msg.err.Error())
			m.llmPlanOverview = ""
			m.llmGoalStatus = ""
			m.statusMsg = ""
			m = m.addLog(oplog.Entry{
				Type:    oplog.EntryLLMError,
				Summary: "LLM analysis failed",
				Detail:  msg.err.Error(),
			})
		} else {
			m.setWorkflowStage(workflowConfirm)
			m.llmAnalysis = msg.analysis
			m.llmThinking = msg.thinking
			m.llmPlanOverview = strings.TrimSpace(msg.planOverview)
			m.llmGoalStatus = strings.TrimSpace(msg.goalStatus)
			m.suggestions = msg.suggestions
			m.suggExecState = make([]git.ExecState, len(msg.suggestions))
			m.suggExecMsg = make([]string, len(msg.suggestions))
			m.suggIdx = 0
			m.expanded = false
			m.statusMsg = ""
			m = m.addLog(oplog.Entry{
				Type:    oplog.EntryLLMOutput,
				Summary: fmt.Sprintf("LLM output: %d suggestion(s)", len(msg.suggestions)),
				Detail:  oneLine(msg.analysis),
			})
			if msg.debugInfo != "" {
				m = m.addLog(oplog.Entry{
					Type:    oplog.EntryLLMOutput,
					Summary: "Context budget: " + msg.debugInfo,
				})
			}
			if summary := oneLine(msg.analysis); summary != "" {
				m.analysisHistory = append(m.analysisHistory, summary)
				if len(m.analysisHistory) > 3 {
					m.analysisHistory = m.analysisHistory[len(m.analysisHistory)-3:]
				}
			}
			if status := strings.ToLower(strings.TrimSpace(msg.goalStatus)); status != "" {
				if status == "completed" || status == "blocked" {
					if status == "completed" {
						m.rememberResolvedGoal(m.session.ActiveGoal)
					}
					m.session.markGoalStatus(status)
					m.analysisHistory = nil
				}
			}
		}

	case commandResultMsg:
		m.pendingExplainID = 0
		idx := m.execSuggIdx
		m.execSuggIdx = -1
		errDetail := bestErrorDetail(msg.err, msg.result)
		m.setWorkflowStage(workflowExecute)
		if errDetail != "" {
			if len(errDetail) > 300 {
				errDetail = errDetail[:300] + "..."
			}
			m.statusMsg = "Failed: " + errDetail
			m.lastCommand = commandTrace{
				Title:  cmdSummaryFromResult(msg.result),
				Status: "failed",
				Output: errDetail,
				At:     time.Now(),
			}
			m = m.addLog(oplog.Entry{
				Type:    oplog.EntryCmdFail,
				Summary: "Command failed: " + cmdSummaryFromResult(msg.result),
				Detail:  errDetail,
			})
			m = m.markSuggExec(idx, git.ExecFailed, errDetail)
		} else if msg.result != nil && msg.result.Success {
			m.statusMsg = "OK " + joinCmd(msg.result.Command)
			output := strings.TrimSpace(msg.result.Stdout)
			if output == "" {
				output = strings.TrimSpace(msg.result.Stderr)
			}
			m.lastCommand = commandTrace{
				Title:  joinCmd(msg.result.Command),
				Status: "success",
				Output: output,
				At:     time.Now(),
			}
			m = m.addLog(oplog.Entry{
				Type:    oplog.EntryCmdSuccess,
				Summary: "Command succeeded: " + joinCmd(msg.result.Command),
				Detail:  strings.TrimSpace(msg.result.Stdout),
			})
			cmdText := strings.ToLower(joinCmd(msg.result.Command))
			if strings.Contains(cmdText, "git@") {
				if m.session.Preferences == nil {
					m.session.Preferences = make(map[string]string)
				}
				m.session.Preferences["remote_protocol"] = "ssh"
				m.rememberPreference("remote_protocol", "ssh")
			} else if strings.Contains(cmdText, "https://") || strings.Contains(cmdText, "http://") {
				if m.session.Preferences == nil {
					m.session.Preferences = make(map[string]string)
				}
				m.session.Preferences["remote_protocol"] = "https"
				m.rememberPreference("remote_protocol", "https")
			}
			m = m.markSuggExec(idx, git.ExecDone, "success")
		}
		m = m.advanceToNextPending()
		if m.allSuggestionsDone() {
			return m, m.refreshGitState
		}
		m.skipNextAnalysis = true
		return m, tea.Cmd(m.refreshGitStateOnly)

	case fileWriteResultMsg:
		idx := m.execSuggIdx
		m.execSuggIdx = -1
		m.setWorkflowStage(workflowExecute)
		if msg.err != nil {
			m.statusMsg = "File operation failed: " + msg.err.Error()
			m.lastCommand = commandTrace{
				Title:  msg.path,
				Status: "file failed",
				Output: msg.err.Error(),
				At:     time.Now(),
			}
			m = m.addLog(oplog.Entry{
				Type:    oplog.EntryCmdFail,
				Summary: "File write failed: " + msg.path,
				Detail:  msg.err.Error(),
			})
			m = m.markSuggExec(idx, git.ExecFailed, msg.err.Error())
		} else {
			opMsg := "File operation succeeded: " + msg.path
			if msg.backupPath != "" {
				opMsg += " (backup: " + msg.backupPath + ")"
			}
			m.statusMsg = opMsg
			m.lastCommand = commandTrace{
				Title:  msg.path,
				Status: "file success",
				Output: strings.TrimSpace(msg.backupPath),
				At:     time.Now(),
			}
			m = m.addLog(oplog.Entry{
				Type:    oplog.EntryCmdSuccess,
				Summary: "File operation succeeded: " + msg.path,
				Detail:  msg.backupPath,
			})
			m = m.markSuggExec(idx, git.ExecDone, "done")
		}
		m = m.advanceToNextPending()
		if m.allSuggestionsDone() {
			return m, m.refreshGitState
		}
		m.skipNextAnalysis = true
		return m, tea.Cmd(m.refreshGitStateOnly)

	case llmExplainMsg:
		if msg.requestID != m.pendingExplainID {
			return m, nil
		}
		m.pendingExplainID = 0
		m.statusMsg = ""
		if msg.err != nil {
			m.llmReason = m.currentReason()
			m.statusMsg = i18n.T("analysis.explain_failed")
			m = m.addLog(oplog.Entry{
				Type:    oplog.EntryLLMError,
				Summary: "AI explanation failed",
				Detail:  msg.err.Error(),
			})
		} else {
			m.llmReason = msg.text
			m = m.addLog(oplog.Entry{
				Type:    oplog.EntryLLMOutput,
				Summary: "AI explanation received",
				Detail:  oneLine(msg.text),
			})
		}
		m.expanded = true
	}

	return m, nil
}

func (m Model) currentReason() string {
	if len(m.suggestions) > 0 && m.suggIdx < len(m.suggestions) {
		return m.suggestions[m.suggIdx].Reason
	}
	return ""
}

// updateInput handles text input mode for NeedsInput suggestions.
func (m Model) updateInput(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	text := msg.Key().Text
	if len(m.inputFields) == 0 || len(m.inputValues) == 0 {
		m.screen = screenMain
		m.statusMsg = i18n.T("input.missing_fields")
		return m, nil
	}

	switch key {
	case "escape":
		m.screen = screenMain
		m.inputSuggRef = nil
		m.statusMsg = i18n.T("input.cancelled")
		m = m.addLog(oplog.Entry{
			Type:    oplog.EntryUserAction,
			Summary: "Input cancelled",
		})
		return m, nil

	case "enter":
		val := strings.TrimSpace(m.inputValues[m.inputIdx])
		if val == "" {
			m.statusMsg = i18n.T("input.empty")
			return m, nil
		}
		// Move to next field or execute
		if m.inputIdx < len(m.inputFields)-1 {
			m.inputIdx++
			m.inputCursorAt = 0
			return m, nil
		}
		// All fields filled; substitute placeholders and execute.
		return m.executeInputCommand()

	case "tab":
		if m.inputIdx < len(m.inputFields)-1 {
			m.inputIdx++
			m.inputCursorAt = len(m.inputValues[m.inputIdx])
		}
		return m, nil

	case "shift+tab":
		if m.inputIdx > 0 {
			m.inputIdx--
			m.inputCursorAt = len(m.inputValues[m.inputIdx])
		}
		return m, nil

	case "backspace":
		v := m.inputValues[m.inputIdx]
		if m.inputCursorAt > 0 && len(v) > 0 {
			// Remove char before cursor
			before := v[:m.inputCursorAt-1]
			after := v[m.inputCursorAt:]
			m.inputValues[m.inputIdx] = before + after
			m.inputCursorAt--
		}
		return m, nil

	case "delete":
		v := m.inputValues[m.inputIdx]
		if m.inputCursorAt < len(v) {
			before := v[:m.inputCursorAt]
			after := v[m.inputCursorAt+1:]
			m.inputValues[m.inputIdx] = before + after
		}
		return m, nil

	case "left":
		if m.inputCursorAt > 0 {
			m.inputCursorAt--
		}
		return m, nil

	case "right":
		if m.inputCursorAt < len(m.inputValues[m.inputIdx]) {
			m.inputCursorAt++
		}
		return m, nil

	case "home", "ctrl+a":
		m.inputCursorAt = 0
		return m, nil

	case "end", "ctrl+e":
		m.inputCursorAt = len(m.inputValues[m.inputIdx])
		return m, nil

	case "ctrl+c":
		return m, tea.Quit

	default:
		if text != "" {
			return m.handleInputPaste(text)
		}
		return m, nil
	}
}

func (m Model) handleInputPaste(text string) (tea.Model, tea.Cmd) {
	if len(m.inputValues) == 0 || m.inputIdx >= len(m.inputValues) {
		return m, nil
	}
	if text == "" {
		return m, nil
	}
	v := m.inputValues[m.inputIdx]
	before := v[:m.inputCursorAt]
	after := v[m.inputCursorAt:]
	m.inputValues[m.inputIdx] = before + text + after
	m.inputCursorAt += len(text)
	return m, nil
}

// executeInputCommand substitutes user input into the command template and executes.
func (m Model) executeInputCommand() (tea.Model, tea.Cmd) {
	if m.inputSuggRef == nil {
		m.screen = screenMain
		return m, nil
	}

	cmd := make([]string, len(m.inputSuggRef.Command))
	copy(cmd, m.inputSuggRef.Command)

	for i, field := range m.inputFields {
		val := strings.TrimSpace(m.inputValues[i])
		if field.ArgIndex >= 0 && field.ArgIndex < len(cmd) {
			cmd[field.ArgIndex] = val
			continue
		}
		if field.Key != "" {
			for j := range cmd {
				cmd[j] = strings.ReplaceAll(cmd[j], field.Key, val)
			}
		}
	}

	m.execSuggIdx = m.suggIdx
	m.screen = screenMain
	m.statusMsg = fmt.Sprintf(i18n.T("messages.executing"), joinCmd(cmd))
	m = m.addLog(oplog.Entry{
		Type:    oplog.EntryUserAction,
		Summary: "Accepted input suggestion: " + m.inputSuggRef.Action,
	})
	m = m.addLog(oplog.Entry{
		Type:    oplog.EntryCmdExec,
		Summary: "Executing: " + joinCmd(cmd),
	})
	m.setWorkflowStage(workflowExecute)
	m = m.markSuggExec(m.suggIdx, git.ExecRunning, "running...")
	m.inputSuggRef = nil
	return m, m.executeCommand(cmd)
}

func (m Model) updateModelSelect(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "up", "k":
		if m.modelCursor > 0 {
			m.modelCursor--
		}

	case "down", "j":
		if m.modelCursor < len(m.availModels)-1 {
			m.modelCursor++
		}

	case "enter":
		if len(m.availModels) == 0 {
			return m, nil
		}
		selected := m.availModels[m.modelCursor]
		if m.modelSelectPhase == selectPrimary {
			m.selectedPrimary = selected.Name
			if m.llmProvider != nil {
				m.llmProvider.SetModelForRole(llm.RolePrimary, selected.Name)
				m.llmProvider.SetModel(selected.Name)
				if oc, ok := m.llmProvider.(*ollama.OllamaClient); ok {
					if detected := oc.DetectModelContext(context.Background(), selected.Name); detected > 0 {
						oc.SetContextLength(detected)
						if m.pipeline != nil {
							m.pipeline.SetContextBudget(detected)
						}
					}
				}
			}
			if m.pipeline != nil {
				m.pipeline.SetPrimaryModel(selected.Name)
			}
			m.modelSelectPhase = selectSecondary
			m.modelCursor = 0
			m.statusMsg = fmt.Sprintf(i18n.T("model_select.primary_selected"), selected.Name)
			m = m.addLog(oplog.Entry{
				Type:    oplog.EntryUserAction,
				Summary: "Selected primary model: " + selected.Name,
			})
			return m, nil
		}

		m.selectedSecondary = selected.Name
		m.secondaryEnabled = true
		if m.llmProvider != nil {
			m.llmProvider.SetModelForRole(llm.RoleSecondary, selected.Name)
		}
		if m.pipeline != nil {
			m.pipeline.SetSecondaryModel(selected.Name, true)
		}
		m.statusMsg = fmt.Sprintf(i18n.T("model_select.summary_on"), m.selectedPrimary, selected.Name)
		m = m.addLog(oplog.Entry{
			Type:    oplog.EntryUserAction,
			Summary: "Selected secondary model: " + selected.Name,
		})
		m.screen = screenMain
		return m, m.refreshGitState

	case "s":
		if m.modelSelectPhase == selectPrimary {
			m.screen = screenMain
			m.llmProvider = nil
			m.statusMsg = i18n.T("model_select.skipped")
			m = m.addLog(oplog.Entry{
				Type:    oplog.EntryUserAction,
				Summary: "Skipped model selection",
			})
			return m, m.refreshGitState
		}
		m.secondaryEnabled = false
		m.selectedSecondary = ""
		if m.pipeline != nil {
			m.pipeline.SetSecondaryModel("", false)
		}
		m.statusMsg = fmt.Sprintf(i18n.T("model_select.summary_off"), m.selectedPrimary)
		m = m.addLog(oplog.Entry{
			Type:    oplog.EntryUserAction,
			Summary: "Skipped secondary model selection",
		})
		m.screen = screenMain
		return m, m.refreshGitState
	}
	return m, nil
}

func (m Model) updateMain(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "l":
		m.logExpanded = !m.logExpanded
		if !m.logExpanded {
			m.logScrollOffset = 0
		}
		if m.logExpanded {
			m.statusMsg = i18n.T("oplog.title") + " (" + i18n.T("oplog.expanded") + ")"
		} else {
			m.statusMsg = i18n.T("oplog.title") + " (" + i18n.T("oplog.collapsed") + ")"
		}
		m = m.addLog(oplog.Entry{
			Type:    oplog.EntryUserAction,
			Summary: m.statusMsg,
		})
		return m, nil

	case "o":
		m.obsTab = m.obsTab.next()
		m.statusMsg = fmt.Sprintf(i18n.T("observability.inspector_status"), m.obsTab.label())
		return m, nil

	case "O":
		m.obsTab = m.obsTab.prev()
		m.statusMsg = fmt.Sprintf(i18n.T("observability.inspector_status"), m.obsTab.label())
		return m, nil

	case "L":
		m = m.openLanguageSelection(screenMain)
		return m, nil

	case "up":
		if m.logExpanded {
			m.logScrollOffset++
			return m.clampLogOffset(), nil
		}

	case "down":
		if m.logExpanded {
			m.logScrollOffset--
			return m.clampLogOffset(), nil
		}

	case "pgup":
		if m.logExpanded {
			m.logScrollOffset += 5
			return m.clampLogOffset(), nil
		}

	case "pgdown":
		if m.logExpanded {
			m.logScrollOffset -= 5
			return m.clampLogOffset(), nil
		}

	case "y":
		if len(m.suggestions) > 0 && m.suggIdx < len(m.suggestions) {
			if m.suggIdx < len(m.suggExecState) && m.suggExecState[m.suggIdx] != git.ExecPending {
				m.statusMsg = i18n.T("suggestions.already_processed")
				return m, nil
			}
			s := m.suggestions[m.suggIdx]

			switch s.Interaction {
			case git.InfoOnly:
				m.statusMsg = fmt.Sprintf(i18n.T("messages.info_prefix"), s.Reason)
				m.expanded = true
				m.llmReason = s.Reason
				m.lastCommand = commandTrace{
					Title:  s.Action,
					Status: "advisory viewed",
					Output: s.Reason,
					At:     time.Now(),
				}
				m = m.addLog(oplog.Entry{
					Type:    oplog.EntryUserAction,
					Summary: "Viewed advisory: " + s.Action,
					Detail:  s.Reason,
				})
				m = m.markSuggExec(m.suggIdx, git.ExecDone, i18n.T("suggestions.viewed"))
				m = m.advanceToNextPending()
				if m.allSuggestionsDone() {
					return m, m.refreshGitState
				}
				m.skipNextAnalysis = true
				return m, tea.Cmd(m.refreshGitStateOnly)

			case git.FileWrite:
				if s.FileOp == nil {
					m.statusMsg = "File write suggestion has no file info"
					m = m.markSuggExec(m.suggIdx, git.ExecFailed, "no file info")
					return m, nil
				}
				m.execSuggIdx = m.suggIdx
				op := s.FileOp.Operation
				if op == "" {
					op = "create"
				}
				m.statusMsg = fmt.Sprintf("%s: %s", strings.Title(op), s.FileOp.Path)
				m = m.addLog(oplog.Entry{
					Type:    oplog.EntryUserAction,
					Summary: fmt.Sprintf("Accepted file %s: %s", op, s.FileOp.Path),
				})
				m = m.addLog(oplog.Entry{
					Type:    oplog.EntryCmdExec,
					Summary: fmt.Sprintf("File %s: %s", op, s.FileOp.Path),
				})
				m.setWorkflowStage(workflowExecute)
				m = m.markSuggExec(m.suggIdx, git.ExecRunning, op+"ing...")
				return m, m.executeFileOp(s.FileOp)

			case git.NeedsInput:
				if len(s.Inputs) == 0 {
					m.statusMsg = "AI suggestion is missing required input metadata"
					m = m.addLog(oplog.Entry{
						Type:    oplog.EntryCmdFail,
						Summary: "Suggestion missing input metadata: " + s.Action,
					})
					m = m.markSuggExec(m.suggIdx, git.ExecFailed, "missing input metadata")
					return m, nil
				}
				m.screen = screenInput
				m.inputFields = s.Inputs
				m.inputIdx = 0
				m.inputValues = make([]string, len(s.Inputs))
				for i := range s.Inputs {
					m.inputValues[i] = s.Inputs[i].DefaultValue
				}
				m.inputCursorAt = len(m.inputValues[0])
				m.inputSuggRef = &s
				m.statusMsg = ""
				m = m.addLog(oplog.Entry{
					Type:    oplog.EntryUserAction,
					Summary: "Preparing input for: " + s.Action,
					Detail:  "Command template: " + joinCmd(s.Command),
				})
				return m, nil

			default: // AutoExec
				m.execSuggIdx = m.suggIdx
				m.statusMsg = fmt.Sprintf(i18n.T("messages.executing"), joinCmd(s.Command))
				m = m.addLog(oplog.Entry{
					Type:    oplog.EntryUserAction,
					Summary: "Accepted suggestion: " + s.Action,
				})
				m = m.addLog(oplog.Entry{
					Type:    oplog.EntryCmdExec,
					Summary: "Executing: " + joinCmd(s.Command),
				})
				m.setWorkflowStage(workflowExecute)
				m = m.markSuggExec(m.suggIdx, git.ExecRunning, "running...")
				return m, m.executeCommand(s.Command)
			}
		}

	case "n":
		if len(m.suggestions) > 0 && m.suggIdx < len(m.suggestions) {
			if m.session.SkippedActions == nil {
				m.session.SkippedActions = []string{}
			}
			action := strings.TrimSpace(m.suggestions[m.suggIdx].Action)
			if action != "" {
				m.session.SkippedActions = append(m.session.SkippedActions, action)
				if len(m.session.SkippedActions) > 12 {
					m.session.SkippedActions = m.session.SkippedActions[len(m.session.SkippedActions)-12:]
				}
			}
			m = m.addLog(oplog.Entry{
				Type:    oplog.EntryUserAction,
				Summary: "Skipped suggestion: " + m.suggestions[m.suggIdx].Action,
			})
			m = m.markSuggExec(m.suggIdx, git.ExecDone, i18n.T("suggestions.skipped"))
			m.pendingExplainID = 0
			m.expanded = false
			m.llmReason = ""
			m = m.advanceToNextPending()
			if m.allSuggestionsDone() {
				return m, m.refreshGitState
			}
		}

	case "tab":
		if len(m.suggestions) > 0 {
			m.pendingExplainID = 0
			m.suggIdx = (m.suggIdx + 1) % len(m.suggestions)
			m.expanded = false
			m.llmReason = ""
			if strings.Contains(strings.ToLower(m.statusMsg), "explanation") {
				m.statusMsg = ""
			}
		}

	case "shift+tab":
		if len(m.suggestions) > 0 {
			m.pendingExplainID = 0
			m.suggIdx = (m.suggIdx - 1 + len(m.suggestions)) % len(m.suggestions)
			m.expanded = false
			m.llmReason = ""
			if strings.Contains(strings.ToLower(m.statusMsg), "explanation") {
				m.statusMsg = ""
			}
		}

	case "w":
		if m.expanded {
			m.pendingExplainID = 0
			m.expanded = false
			m.llmReason = ""
			if strings.Contains(strings.ToLower(m.statusMsg), "explanation") {
				m.statusMsg = ""
			}
		} else if len(m.suggestions) > 0 && m.suggIdx < len(m.suggestions) {
			s := m.suggestions[m.suggIdx]
			if m.llmProvider != nil {
				m.explainSeq++
				m.pendingExplainID = m.explainSeq
				m.statusMsg = i18n.T("analysis.asking_explanation")
				m.expanded = true
				m.llmReason = fmt.Sprintf(i18n.T("analysis.thinking_pending"), s.Reason)
				m = m.addLog(oplog.Entry{
					Type:    oplog.EntryUserAction,
					Summary: "Requested AI explanation: " + s.Action,
				})
				m = m.addLog(oplog.Entry{
					Type:    oplog.EntryLLMStart,
					Summary: "LLM explanation started",
					Detail:  s.Action,
				})
				return m, m.llmExplainSuggestion(m.pendingExplainID, s, m.gitState)
			}
			m.expanded = true
			m.llmReason = s.Reason
		}

	case "z":
		if m.mode == "zen" {
			m.mode = "full"
		} else {
			m.mode = "zen"
		}
		m.statusMsg = "AI mode: " + m.mode
		m = m.addLog(oplog.Entry{
			Type:    oplog.EntryUserAction,
			Summary: "Switched AI mode to " + m.mode,
		})
		if m.llmProvider != nil && m.gitState != nil {
			m.analysisSeq++
			m.pendingAnalysisID = m.analysisSeq
			m.llmAnalysis = i18n.T("analysis.reanalyzing")
			m.logScrollOffset = 0
			m = m.addLog(oplog.Entry{
				Type:    oplog.EntryLLMStart,
				Summary: fmt.Sprintf("LLM analysis started (%s mode)", m.mode),
			})
			return m, m.runLLMAnalysis(m.pendingAnalysisID, m.gitState)
		}
		return m, nil

	case "t":
		if m.llmThinking != "" {
			m.expanded = !m.expanded
		}

	case "r":
		m.statusMsg = i18n.T("ui.refreshing")
		m = m.addLog(oplog.Entry{
			Type:    oplog.EntryUserAction,
			Summary: "Manual refresh requested",
		})
		m.analysisSeq++
		m.pendingAnalysisID = m.analysisSeq
		m.pendingExplainID = 0
		m.suggestions = nil
		m.llmAnalysis = ""
		m.llmThinking = ""
		m.llmReason = ""
		m.llmPlanOverview = ""
		m.llmGoalStatus = ""
		return m, m.refreshGitState

	case "g":
		m.screen = screenGoalInput
		m.goalInput = strings.TrimSpace(m.session.ActiveGoal)
		m.goalCursorAt = len(m.goalInput)
		m.statusMsg = i18n.T("goal.prompt")
		return m, nil

	case "f":
		if len(m.workflows) == 0 {
			m.workflows = loadWorkflowDefinitions()
		}
		if len(m.workflows) == 0 {
			m.statusMsg = i18n.T("workflow_menu.no_workflows")
			return m, nil
		}
		m.workflowCursor = 0
		m.screen = screenWorkflowSelect
		m.statusMsg = i18n.T("workflow_menu.prompt")
		return m, nil
	}
	return m, nil
}

func (m Model) updateGoalInput(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	text := msg.Key().Text

	switch key {
	case "escape":
		m.screen = screenMain
		m.statusMsg = i18n.T("goal.cancelled")
		return m, nil
	case "enter":
		goal := strings.TrimSpace(m.goalInput)
		m.session.ActiveGoal = goal
		m.screen = screenMain
		if goal == "" {
			m.statusMsg = i18n.T("goal.cleared")
			m = m.addLog(oplog.Entry{
				Type:    oplog.EntryUserAction,
				Summary: "Cleared active goal",
			})
		} else {
			m.statusMsg = fmt.Sprintf(i18n.T("goal.active"), goal)
			m = m.addLog(oplog.Entry{
				Type:    oplog.EntryUserAction,
				Summary: "Set active goal: " + goal,
			})
		}
		if m.llmProvider != nil && m.gitState != nil {
			m.analysisSeq++
			m.pendingAnalysisID = m.analysisSeq
			m.llmAnalysis = i18n.T("analysis.analyzing_repo")
			return m, m.runLLMAnalysis(m.pendingAnalysisID, m.gitState)
		}
		return m, nil
	case "backspace":
		if m.goalCursorAt > 0 && len(m.goalInput) > 0 {
			m.goalInput = m.goalInput[:m.goalCursorAt-1] + m.goalInput[m.goalCursorAt:]
			m.goalCursorAt--
		}
		return m, nil
	case "delete":
		if m.goalCursorAt < len(m.goalInput) {
			m.goalInput = m.goalInput[:m.goalCursorAt] + m.goalInput[m.goalCursorAt+1:]
		}
		return m, nil
	case "left":
		if m.goalCursorAt > 0 {
			m.goalCursorAt--
		}
		return m, nil
	case "right":
		if m.goalCursorAt < len(m.goalInput) {
			m.goalCursorAt++
		}
		return m, nil
	case "home", "ctrl+a":
		m.goalCursorAt = 0
		return m, nil
	case "end", "ctrl+e":
		m.goalCursorAt = len(m.goalInput)
		return m, nil
	case "ctrl+c":
		return m, tea.Quit
	default:
		if text != "" {
			return m.handleGoalPaste(text)
		}
	}
	return m, nil
}

func (m Model) handleGoalPaste(text string) (tea.Model, tea.Cmd) {
	if text == "" {
		return m, nil
	}
	before := m.goalInput[:m.goalCursorAt]
	after := m.goalInput[m.goalCursorAt:]
	m.goalInput = before + text + after
	m.goalCursorAt += len(text)
	return m, nil
}

func (m Model) updateWorkflowSelect(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch {
	case key == "q" || key == "ctrl+c":
		return m, tea.Quit
	case key == "escape" || key == "esc" || msg.Key().Code == tea.KeyEscape:
		m.screen = screenMain
		m.statusMsg = ""
		return m, nil
	case key == "up" || key == "k":
		if m.workflowCursor > 0 {
			m.workflowCursor--
		}
		return m, nil
	case key == "down" || key == "j":
		if m.workflowCursor < len(m.workflows)-1 {
			m.workflowCursor++
		}
		return m, nil
	case key == "enter":
		if len(m.workflows) == 0 || m.workflowCursor >= len(m.workflows) {
			m.screen = screenMain
			return m, nil
		}
		wf := m.workflows[m.workflowCursor]
		ok, failed := checkWorkflowPrerequisites(m.gitState, wf)
		if !ok {
			m.statusMsg = fmt.Sprintf(i18n.T("workflow_menu.blocked"), failed)
			m = m.addLog(oplog.Entry{
				Type:    oplog.EntryUserAction,
				Summary: "Workflow blocked: " + wf.ID,
				Detail:  failed,
			})
			m.screen = screenMain
			return m, nil
		}

		m.session.ActiveGoal = strings.TrimSpace(wf.Goal)
		m.rememberRepoPattern("workflow:" + wf.ID)
		m.screen = screenMain
		m.statusMsg = fmt.Sprintf(i18n.T("workflow_menu.selected"), wf.Label)
		m = m.addLog(oplog.Entry{
			Type:    oplog.EntryUserAction,
			Summary: "Workflow selected: " + wf.ID,
			Detail:  wf.Goal,
		})
		if m.llmProvider != nil && m.gitState != nil {
			m.analysisSeq++
			m.pendingAnalysisID = m.analysisSeq
			m.llmAnalysis = i18n.T("analysis.analyzing_repo")
			return m, m.runLLMAnalysis(m.pendingAnalysisID, m.gitState)
		}
		return m, nil
	}
	return m, nil
}

func summarizeGitState(state *status.GitState) string {
	if state == nil {
		return "unknown"
	}
	parts := []string{fmt.Sprintf("branch=%s", state.LocalBranch.Name)}
	if len(state.WorkingTree) > 0 {
		parts = append(parts, fmt.Sprintf("working=%d", len(state.WorkingTree)))
	}
	if len(state.StagingArea) > 0 {
		parts = append(parts, fmt.Sprintf("staged=%d", len(state.StagingArea)))
	}
	if len(state.RemoteInfos) > 0 {
		parts = append(parts, fmt.Sprintf("remotes=%d", len(state.RemoteInfos)))
	} else {
		parts = append(parts, "remotes=0")
	}
	if state.LocalBranch.Upstream != "" {
		parts = append(parts, "upstream="+state.LocalBranch.Upstream)
	}
	return strings.Join(parts, " ")
}

func oneLine(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	lines := strings.Split(text, "\n")
	s := strings.TrimSpace(lines[0])
	if len(s) > 180 {
		return s[:180] + "..."
	}
	return s
}

// bestErrorDetail extracts the most useful error message from a commandResultMsg,
// combining msg.err and msg.result.Stderr into a single human-readable string.
func bestErrorDetail(err error, result *git.ExecutionResult) string {
	if err == nil && (result == nil || result.Success) {
		return ""
	}
	if result != nil && !result.Success {
		stderr := strings.TrimSpace(result.Stderr)
		if stderr != "" {
			return stderr
		}
		stdout := strings.TrimSpace(result.Stdout)
		if stdout != "" {
			return stdout
		}
		if result.ExitCode != 0 {
			return fmt.Sprintf("exit code %d", result.ExitCode)
		}
	}
	if err != nil {
		return err.Error()
	}
	return "unknown error"
}

func cmdSummaryFromResult(result *git.ExecutionResult) string {
	if result != nil && len(result.Command) > 0 {
		return joinCmd(result.Command)
	}
	return "(unknown command)"
}

func (m Model) markSuggExec(idx int, state git.ExecState, msg string) Model {
	if idx >= 0 && idx < len(m.suggExecState) {
		m.suggExecState[idx] = state
		m.suggExecMsg[idx] = msg
	}
	return m
}

func (m Model) advanceToNextPending() Model {
	if len(m.suggestions) == 0 {
		return m
	}
	start := m.suggIdx
	for i := 1; i <= len(m.suggestions); i++ {
		next := (start + i) % len(m.suggestions)
		if next < len(m.suggExecState) && m.suggExecState[next] == git.ExecPending {
			m.suggIdx = next
			m.expanded = false
			m.llmReason = ""
			return m
		}
	}
	return m
}

func (m Model) allSuggestionsDone() bool {
	if len(m.suggestions) == 0 {
		return true
	}
	for _, s := range m.suggExecState {
		if s == git.ExecPending || s == git.ExecRunning {
			return false
		}
	}
	return true
}

func (m Model) clampLogOffset() Model {
	if m.logScrollOffset < 0 {
		m.logScrollOffset = 0
	}
	max := 0
	if m.opLog != nil {
		if entries := m.opLog.Entries(); len(entries) > 0 {
			max = len(entries) - 1
		}
	}
	if m.logScrollOffset > max {
		m.logScrollOffset = max
	}
	return m
}
