package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Joker-of-Gotham/gitdex/internal/config"
	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
	"github.com/Joker-of-Gotham/gitdex/internal/i18n"
	"github.com/Joker-of-Gotham/gitdex/internal/llm"
	"github.com/Joker-of-Gotham/gitdex/internal/llm/ollama"
	"github.com/Joker-of-Gotham/gitdex/internal/platform"
	"github.com/Joker-of-Gotham/gitdex/internal/tui/oplog"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyPressMsg:
		switch m.screen {
		case screenLanguageSelect:
			return m.updateLanguageSelect(msg)
		case screenModelSelect:
			return m.updateModelSelect(msg)
		case screenProviderConfig:
			return m.updateProviderConfig(msg)
		case screenAutomationConfig:
			return m.updateAutomationConfig(msg)
		case screenInput:
			return m.updateInput(msg)
		case screenGoalInput:
			return m.updateGoalInput(msg)
		case screenWorkflowSelect:
			return m.updateWorkflowSelect(msg)
		case screenPlatformEdit:
			return m.updatePlatformEdit(msg)
		case screenFileEdit:
			return m.updateFileEdit(msg)
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
		if m.screen == screenPlatformEdit {
			return m.handlePlatformEditPaste(msg.Content)
		}
		if m.screen == screenFileEdit {
			return m.handleFileEditPaste(msg.Content)
		}
		if m.screen == screenProviderConfig {
			return m.handleProviderPaste(msg.Content)
		}
		if m.screen == screenMain {
			m.composerFocused = true
			return m.handleComposerPaste(msg.Content)
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		return m, nil

	case providerModelsMsg:
		m.modelsFetched = true
		if msg.available && msg.err == nil && len(msg.models) > 0 {
			m.availModels = msg.models
			m.availModelsSource = inferModelProvider(msg.models, m.primaryProvider)
			if m.shouldShowFirstRunLanguageSelection() {
				m = m.openLanguageSelection(screenLoading)
			} else if m.shouldOpenModelSelection(msg.models) {
				if m.primaryProvider == "ollama" {
					m = m.openLocalModelSelection(selectPrimary, "ollama")
				} else {
					m = m.openModelSetup(selectPrimary)
				}
			} else {
				m.screen = screenMain
			}
			m = m.addLog(oplog.Entry{
				Type:    oplog.EntryStateRefresh,
				Summary: fmt.Sprintf("Provider models loaded: %d", len(msg.models)),
			})
		} else {
			m.availModels = nil
			m.availModelsSource = ""
			m.setWorkflowStage(workflowConfirm)
			if m.shouldShowFirstRunLanguageSelection() {
				m = m.openLanguageSelection(screenLoading)
			} else if m.shouldRequireProviderSetupOnStartup() {
				m = m.openModelSetup(selectPrimary)
			} else {
				m.screen = screenMain
			}
			m.llmAnalysis = i18n.T("analysis.no_ollama")
			detail := "provider unavailable"
			if msg.err != nil {
				detail = msg.err.Error()
			} else if msg.available {
				detail = "no models discovered"
			}
			m = m.addLog(oplog.Entry{
				Type:    oplog.EntryLLMError,
				Summary: "AI setup required during startup",
				Detail:  detail,
			})
			return m, nil
		}
		return m, nil

	case setupProviderModelsMsg:
		if msg.provider != "ollama" {
			return m, nil
		}
		if msg.err != nil || len(msg.models) == 0 {
			m = m.openProviderConfig(m.modelSelectPhase)
			m.providerDraft.Provider = msg.provider
			m.providerDraft.Endpoint = defaultProviderEndpoint(msg.provider)
			m.providerDraft.APIKeyEnv = defaultProviderAPIKeyEnv(msg.provider)
			m.providerField = providerFieldEndpoint
			m.providerCursorAt = runeLen(m.providerDraft.Endpoint)
			if msg.err != nil {
				m.statusMsg = msg.err.Error()
			} else {
				m.statusMsg = i18n.T("provider_config.empty_models")
			}
			return m, nil
		}

		m.availModels = msg.models
		m.availModelsSource = msg.provider
		m = m.openLocalModelSelection(m.modelSelectPhase, msg.provider)
		m.statusMsg = fmt.Sprintf(i18n.T("model_select.found"), len(msg.models))
		return m, nil

	case providerCheckTickMsg:
		if m.llmProvider != nil && m.llmProvider.IsAvailable(context.Background()) {
			return m, tea.Cmd(m.fetchProviderModels)
		}
		if m.primaryProvider == "ollama" {
			return m, scheduleProviderCheck()
		}
		return m, nil

	case automationTickMsg:
		if !m.automation.Enabled || m.automation.MonitorInterval <= 0 {
			return m, nil
		}
		now := time.Now()
		if m.promoteDueWorkflowRetries(now) {
			m.persistAutomationCheckpoint()
		}
		nextTick := scheduleAutomationTick(time.Duration(m.automation.MonitorInterval) * time.Second)
		if !m.shouldAutoRefresh() {
			if nextModel, cmd, ok := m.applyCruiseGoalIfNeeded(); ok {
				return nextModel, tea.Batch(cmd, nextTick)
			}
			return m, tea.Batch(m.refreshGitStateOnly, nextTick)
		}
		m.autoSteps = 0
		if nextModel, cmd, ok := m.applyCruiseGoalIfNeeded(); ok {
			return nextModel, tea.Batch(cmd, nextTick)
		}
		if nextModel, ok := m.applyDueScheduledAutomation(now); ok {
			return nextModel, tea.Batch(nextModel.refreshGitState, nextTick)
		}
		m.statusMsg = localizedText("Automation scan running", "自动巡检进行中", "Automation scan running")
		m = m.addLog(oplog.Entry{
			Type:    oplog.EntryStateRefresh,
			Summary: localizedText("Automation monitor tick", "自动巡检触发", "Automation monitor tick"),
		})
		return m, tea.Batch(m.refreshGitState, nextTick)

	case paneScrollMsg:
		m = m.scrollPaneBy(msg.pane, msg.delta)
		return m, nil

	case paneFocusMsg:
		m.scrollFocus = msg.pane
		return m, nil

	case uiClickMsg:
		return m.handleUIClick(msg)

	case autoRetryAnalysisMsg:
		if m.llmProvider != nil && m.gitState != nil && config.AutomationModeIsAutoLoop(m.automationMode()) {
			m.lastAnalysisFingerprint = ""
			return m, m.refreshGitState
		}
		return m, nil

	case gitStateMsg:
		prevState := m.gitState
		m.gitState = msg.state
		m.refreshCachedPlatform()
		if msg.state != nil {
			m.reconcileRepoScopedState()
		}
		m.setWorkflowStage(workflowPerceive)
		m = m.addLog(oplog.Entry{
			Type:    oplog.EntryStateRefresh,
			Summary: "State refreshed: " + summarizeGitState(msg.state),
		})
		m = m.recordAutomationTransitions(prevState, msg.state)
		if msg.state != nil && len(msg.state.WorkingTree) == 0 && len(msg.state.StagingArea) == 0 {
			m.analysisHistory = nil
		}
		if m.screen == screenMain || m.screen == screenLoading {
			if m.screen == screenLoading {
				if m.shouldShowFirstRunLanguageSelection() {
					m = m.openLanguageSelection(screenLoading)
					return m, nil
				}
				if m.primaryProvider == "ollama" && !m.modelsFetched {
					return m, nil
				}
				if m.shouldRequireProviderSetupOnStartup() {
					m = m.openModelSetup(selectPrimary)
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
				fingerprint := analysisStateFingerprint(msg.state, m.session.ActiveGoal, m.mode)
				if fingerprint != "" && fingerprint == m.lastAnalysisFingerprint && strings.TrimSpace(m.llmAnalysis) != "" {
					m.revalidatePendingSuggestions()
					if strings.TrimSpace(m.statusMsg) == "" {
						m.statusMsg = "State refreshed; analysis unchanged"
					}
					return m, nil
				}
				m.lastAnalysisFingerprint = fingerprint
				m.analysisSeq++
				m.pendingAnalysisID = m.analysisSeq
				m.setWorkflowStage(workflowAnalyze)
				m.llmAnalysis = i18n.T("analysis.analyzing")
				m.statusMsg = i18n.T("analysis.in_progress_status")
				m.logScrollOffset = 0
				m.leftScroll = 0

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
			m.consecutiveAnalysisFailures++
			m.setWorkflowStage(workflowSuggest)
			m.llmAnalysis = fmt.Sprintf(i18n.T("analysis.error_prefix"), msg.err.Error())
			m.llmThinking = ""
			m.llmPlanOverview = ""
			m.llmGoalStatus = ""
			m.suggestions = nil
			m.suggExecState = nil
			m.suggExecMsg = nil
			m.statusMsg = ""
			m = m.addLog(oplog.Entry{
				Type:    oplog.EntryLLMError,
				Summary: "LLM analysis failed",
				Detail:  msg.err.Error(),
			})
			if config.AutomationModeIsAutoLoop(m.automationMode()) && m.shouldRunUnattended() && m.consecutiveAnalysisFailures <= 3 {
				delay := time.Duration(m.consecutiveAnalysisFailures) * 3 * time.Second
				m.lastAnalysisFingerprint = ""
				m = m.addLog(oplog.Entry{
					Type:    oplog.EntryStateRefresh,
					Summary: fmt.Sprintf("Auto-retry analysis in %s (attempt %d/3)", delay, m.consecutiveAnalysisFailures),
				})
				return m, tea.Tick(delay, func(time.Time) tea.Msg { return autoRetryAnalysisMsg{} })
			}
		} else {
			m.setWorkflowStage(workflowConfirm)
			m.llmAnalysis = msg.analysis
			m.llmThinking = msg.thinking
			m.llmPlanOverview = strings.TrimSpace(msg.planOverview)
			m.llmGoalStatus = strings.TrimSpace(msg.goalStatus)
			preparedSuggestions, preparedNotes, dropped := m.prepareSuggestionsForDisplay(msg.suggestions)
			m.suggestions = preparedSuggestions
			m.suggExecState = make([]git.ExecState, len(preparedSuggestions))
			m.suggExecMsg = make([]string, len(preparedSuggestions))
			copy(m.suggExecMsg, preparedNotes)
			m.reconcileWorkflowFlowSuggestions()
			m.syncTaskMemory()
			m.persistAutomationCheckpoint()
			m.suggIdx = 0
			m.expanded = false
			m.statusMsg = ""

			isParseFailure := len(preparedSuggestions) == 0 && strings.Contains(msg.analysis, "could not be parsed")

			if len(m.suggestions) > 0 {
				m.consecutiveAnalysisFailures = 0
				m.consecutiveEmptySuggestions = 0
				m.showSelectedSuggestionGuidance()
				m.workspaceTab = workspaceTabSuggestions
			} else {
				m.workspaceTab = workspaceTabAnalysis
			}
			m = m.addLog(oplog.Entry{
				Type:    oplog.EntryLLMOutput,
				Summary: fmt.Sprintf("LLM output: %d suggestion(s)", len(preparedSuggestions)),
				Detail:  oneLine(msg.analysis),
			})
			if dropped > 0 {
				m = m.addLog(oplog.Entry{
					Type:    oplog.EntryLLMOutput,
					Summary: fmt.Sprintf("Filtered %d suggestion(s) that did not match this repository", dropped),
				})
			}
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
					m.workflowPlan = nil
					m.workflowFlow = nil
					m.analysisHistory = nil
					m.syncTaskMemory()
					m.persistAutomationCheckpoint()
				}
			}

			if isParseFailure && config.AutomationModeIsAutoLoop(m.automationMode()) && m.shouldRunUnattended() {
				m.consecutiveAnalysisFailures++
				if m.consecutiveAnalysisFailures <= 3 {
					delay := time.Duration(m.consecutiveAnalysisFailures) * 3 * time.Second
					m.lastAnalysisFingerprint = ""
					m = m.addLog(oplog.Entry{
						Type:    oplog.EntryStateRefresh,
						Summary: fmt.Sprintf("Parse failure; auto-retry in %s (attempt %d/3)", delay, m.consecutiveAnalysisFailures),
					})
					return m, tea.Tick(delay, func(time.Time) tea.Msg { return autoRetryAnalysisMsg{} })
				}
			}

			m.autoSteps = 0

		if config.AutomationModeIsAutoLoop(m.automationMode()) && m.shouldRunUnattended() {
			if next, cmd, ok := m.autoExecuteNextSafeSuggestion(true); ok {
				next.batchRunRequested = false
				next.consecutiveEmptySuggestions = 0
				return next, cmd
			}
			if len(m.suggestions) == 0 && !isParseFailure {
				goalStatus := strings.ToLower(strings.TrimSpace(msg.goalStatus))
				hasGoal := strings.TrimSpace(m.session.ActiveGoal) != ""
				if goalStatus == "completed" || goalStatus == "blocked" || !hasGoal {
					m.consecutiveEmptySuggestions = 0
					break
				}
				m.consecutiveEmptySuggestions++
				if m.consecutiveEmptySuggestions > 5 {
					m.consecutiveEmptySuggestions = 0
					m = m.addLog(oplog.Entry{
						Type:    oplog.EntryStateRefresh,
						Summary: "Stopped auto-retry: 5 consecutive empty analyses with active goal",
					})
					break
				}
				m.lastAnalysisFingerprint = ""
				delay := time.Duration(m.consecutiveEmptySuggestions) * 5 * time.Second
				m = m.addLog(oplog.Entry{
					Type:    oplog.EntryStateRefresh,
					Summary: fmt.Sprintf("No actionable suggestions; will re-analyze in %s (attempt %d/5)", delay, m.consecutiveEmptySuggestions),
				})
				return m, tea.Tick(delay, func(time.Time) tea.Msg { return autoRetryAnalysisMsg{} })
			}
		} else if m.batchRunRequested {
				if next, cmd, ok := m.autoExecuteNextSafeSuggestion(true); ok {
					next.batchRunRequested = true
					return next, cmd
				}
				m.batchRunRequested = false
			}
		}

	case commandResultMsg:
		m.pendingExplainID = 0
		idx := m.execSuggIdx
		m.execSuggIdx = -1
		isAutoRun := m.batchRunRequested || m.shouldRunUnattended()
		errDetail := bestErrorDetail(msg.err, msg.result)
		outcomeKey := cmdSummaryFromResult(msg.result)
		if strings.TrimSpace(outcomeKey) == "" && idx >= 0 && idx < len(m.suggestions) {
			outcomeKey = m.suggestions[idx].Action
		}
		var resultCommand []string
		if msg.result != nil {
			resultCommand = append([]string(nil), msg.result.Command...)
		}
		m.setWorkflowStage(workflowExecute)
		if errDetail != "" {
			m.statusMsg = "Failed: " + errDetail
			m.lastCommand = commandTrace{
				Title:      cmdSummaryFromResult(msg.result),
				Status:     "failed",
				Output:     errDetail,
				At:         time.Now(),
				Command:    resultCommand,
				ResultKind: detectResultKind(resultCommand),
			}
			if !isAutoRun {
				m.workspaceTab = workspaceTabResult
			}
			m = m.addLog(oplog.Entry{
				Type:    oplog.EntryCmdFail,
				Summary: "Command failed: " + cmdSummaryFromResult(msg.result),
				Detail:  errDetail,
			})
			m.rememberOperationEvent("command failed: " + cmdSummaryFromResult(msg.result))
			m = m.markSuggExec(idx, git.ExecFailed, errDetail)
			m.recordAutomationOutcome(strings.TrimSpace(outcomeKey), false, platform.FailureExecutor)
		} else if msg.result != nil && msg.result.Success {
			m.statusMsg = "OK " + joinCmd(msg.result.Command)
			output := strings.TrimSpace(msg.result.Stdout)
			if output == "" {
				output = strings.TrimSpace(msg.result.Stderr)
			}
			m.lastCommand = commandTrace{
				Title:      joinCmd(msg.result.Command),
				Status:     "success",
				Output:     output,
				At:         time.Now(),
				Command:    resultCommand,
				ResultKind: detectResultKind(resultCommand),
			}
			if !isAutoRun {
				m.workspaceTab = workspaceTabResult
			}
			m = m.addLog(oplog.Entry{
				Type:    oplog.EntryCmdSuccess,
				Summary: "Command succeeded: " + joinCmd(msg.result.Command),
				Detail:  strings.TrimSpace(msg.result.Stdout),
			})
			m.rememberOperationEvent("command succeeded: " + joinCmd(msg.result.Command))
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
			m.recordAutomationOutcome(strings.TrimSpace(outcomeKey), true, "")
		}
		return m.handlePostExecution(errDetail != "")

	case fileWriteResultMsg:
		idx := m.execSuggIdx
		m.execSuggIdx = -1
		isAutoRun := m.batchRunRequested || m.shouldRunUnattended()
		m.setWorkflowStage(workflowExecute)
		outcomeKey := strings.TrimSpace(msg.path)
		if msg.err != nil {
			m.statusMsg = "File operation failed: " + msg.err.Error()
			m.lastCommand = commandTrace{
				Title:         msg.path,
				Status:        "file failed",
				Output:        msg.err.Error(),
				At:            time.Now(),
				ResultKind:    resultKindFileWrite,
				FilePath:      msg.path,
				FileOperation: msg.operation,
				BeforeContent: msg.beforeContent,
				AfterContent:  msg.afterContent,
			}
			if !isAutoRun {
				m.workspaceTab = workspaceTabResult
			}
			m = m.addLog(oplog.Entry{
				Type:    oplog.EntryCmdFail,
				Summary: "File write failed: " + msg.path,
				Detail:  msg.err.Error(),
			})
			m = m.markSuggExec(idx, git.ExecFailed, msg.err.Error())
			m.recordAutomationOutcome(outcomeKey, false, platform.FailureExecutor)
		} else {
			opMsg := "File operation succeeded: " + msg.path
			if msg.backupPath != "" {
				opMsg += " (backup: " + msg.backupPath + ")"
			}
			m.statusMsg = opMsg
			m.lastCommand = commandTrace{
				Title:         msg.path,
				Status:        "file success",
				Output:        strings.TrimSpace(msg.backupPath),
				At:            time.Now(),
				ResultKind:    resultKindFileWrite,
				FilePath:      msg.path,
				FileOperation: msg.operation,
				BeforeContent: msg.beforeContent,
				AfterContent:  msg.afterContent,
			}
			if !isAutoRun {
				m.workspaceTab = workspaceTabResult
			}
			m = m.addLog(oplog.Entry{
				Type:    oplog.EntryCmdSuccess,
				Summary: "File operation succeeded: " + msg.path,
				Detail:  msg.backupPath,
			})
			m.rememberArtifactNote(fmt.Sprintf("%s %s", msg.operation, msg.path))
			m.rememberOperationEvent("file " + msg.operation + ": " + msg.path)
			m = m.markSuggExec(idx, git.ExecDone, "done")
			m.recordAutomationOutcome(outcomeKey, true, "")
		}
		return m.handlePostExecution(msg.err != nil)

	case platformExecResultMsg:
		idx := m.execSuggIdx
		m.execSuggIdx = -1
		isAutoRun := m.batchRunRequested || m.shouldRunUnattended()
		m.setWorkflowStage(workflowExecute)
		m.lastPlatformOp = clonePlatformExecInfo(msg.Request.Op)
		lockKey := m.workflowConcurrencyKey(msg.Request.Op)
		lockOwner := strings.TrimSpace(platformActionTitle(msg.Request.Op))
		m.releaseAutomationLock(lockKey, lockOwner)
		stepID := ""
		if step := m.findWorkflowFlowStep(msg.Request.Op); step != nil {
			stepID = step.Identity
		}
		ledgerEntry := buildPlatformLedgerEntry(msg.Platform, msg.Request, msg, stepID)
		title := platformActionTitle(msg.Request.Op)
		hadError := false
		if msg.Err != nil {
			hadError = true
			m.statusMsg = "Platform action failed: " + msg.Err.Error()
			m.lastCommand = commandTrace{
				Title:              title,
				Status:             "platform failed",
				Output:             msg.Err.Error(),
				At:                 time.Now(),
				ResultKind:         resultKindPlatformAdmin,
				PlatformCapability: strings.TrimSpace(msg.Request.Op.CapabilityID),
				PlatformFlow:       strings.TrimSpace(msg.Request.Op.Flow),
				PlatformOperation:  strings.TrimSpace(msg.Request.Op.Operation),
				PlatformResourceID: strings.TrimSpace(msg.Request.Op.ResourceID),
				PlatformAdapter:    string(ledgerEntry.ExecMeta.Adapter),
				PlatformRollback:   string(ledgerEntry.ExecMeta.Rollback),
				PlatformBoundary:   ledgerEntry.ExecMeta.BoundaryReason,
				PlatformLedgerID:   ledgerEntry.ID,
				PlatformApproval:   ledgerEntry.ExecMeta.ApprovalRequired,
			}
			if !isAutoRun {
				m.workspaceTab = workspaceTabResult
			}
			m = m.addLog(oplog.Entry{
				Type:    oplog.EntryCmdFail,
				Summary: "Platform action failed: " + title,
				Detail:  msg.Err.Error(),
			})
			m.rememberOperationEvent("platform failed: " + title)
			m = m.appendMutationLedger(ledgerEntry)
			if strings.EqualFold(strings.TrimSpace(msg.Request.Op.Flow), "validate") {
				m.completeWorkflowFlowValidation(msg.Request.Op, msg.Err.Error(), true)
			} else {
				m.markWorkflowFlowResult(msg.Request.Op, true, msg.Err.Error())
			}
			m.recordWorkflowFlowLedger(msg.Request.Op, ledgerEntry.ID)
			m.recordAutomationOutcome(strings.TrimSpace(msg.Request.Op.CapabilityID), false, ledgerEntry.Failure)
			m.applyDeadLetterPolicy()
			m.persistAutomationCheckpoint()
			m = m.markSuggExec(idx, git.ExecFailed, msg.Err.Error())
		} else {
			trace := platformTraceFromResult(msg)
			trace.PlatformLedgerID = strings.TrimSpace(firstNonEmpty(trace.PlatformLedgerID, ledgerEntry.ID))
			m.lastCommand = trace
			if !isAutoRun {
				m.workspaceTab = workspaceTabResult
			}
			m.statusMsg = trace.Status + ": " + title
			detail := strings.TrimSpace(trace.Output)
			if identity := git.PlatformExecIdentity(msg.Request.Op); identity != "" {
				if detail != "" {
					detail += "\n"
				}
				detail += "identity=" + identity
			}
			m = m.addLog(oplog.Entry{
				Type:    oplog.EntryCmdSuccess,
				Summary: "Platform action succeeded: " + title,
				Detail:  detail,
			})
			m.rememberOperationEvent("platform success: " + git.PlatformExecIdentity(msg.Request.Op))
			m = m.appendMutationLedger(ledgerEntry)
			if strings.EqualFold(strings.TrimSpace(msg.Request.Op.Flow), "validate") {
				m.completeWorkflowFlowValidation(msg.Request.Op, trace.Output, false)
			} else {
				m.markWorkflowFlowResult(msg.Request.Op, false, trace.Output)
			}
			m.recordWorkflowFlowLedger(msg.Request.Op, ledgerEntry.ID)
			m.persistAutomationCheckpoint()
			if msg.Mutation != nil {
				ledgerChain := []string{strings.TrimSpace(firstNonEmpty(msg.Mutation.LedgerID, ledgerEntry.ID))}
				if m.lastPlatform != nil && strings.EqualFold(m.lastPlatform.CapabilityID, msg.Mutation.CapabilityID) {
					ledgerChain = append([]string(nil), m.lastPlatform.LedgerChain...)
					ledgerChain = append(ledgerChain, strings.TrimSpace(firstNonEmpty(msg.Mutation.LedgerID, ledgerEntry.ID)))
				}
				m.lastPlatform = &platformActionState{
					CapabilityID:    msg.Mutation.CapabilityID,
					Scope:           cloneStringMap(msg.Request.Op.Scope),
					Mutation:        cloneMutation(msg.Mutation),
					ValidatePayload: cloneRaw(msg.Request.Op.ValidatePayload),
					RollbackPayload: cloneRaw(msg.Request.Op.RollbackPayload),
					ExecMeta:        msg.Mutation.ExecMeta,
					LedgerID:        strings.TrimSpace(firstNonEmpty(msg.Mutation.LedgerID, ledgerEntry.ID)),
					RequestRevision: maxInt(1, msg.Request.Revision),
					LedgerChain:     compactStringList(ledgerChain, 8),
				}
			}
			m = m.markSuggExec(idx, git.ExecDone, "done")
			m.recordAutomationOutcome(strings.TrimSpace(msg.Request.Op.CapabilityID), true, "")
		}
		return m.handlePostExecution(hadError)

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

func (m Model) autoExecuteNextSafeSuggestion(force bool) (Model, tea.Cmd, bool) {
	if !force && !m.shouldRunUnattended() {
		return m, nil, false
	}
	limit := m.automation.MaxAutoSteps
	if limit <= 0 {
		limit = 8
	}
	if m.autoSteps >= limit {
		m.statusMsg = "Automation paused: max auto steps reached"
		m = m.addLog(oplog.Entry{
			Type:    oplog.EntryUserAction,
			Summary: "Automation paused after max auto steps",
		})
		return m, nil, false
	}

	autoLoop := config.AutomationModeIsAutoLoop(m.automationMode())
	for idx, suggestion := range m.suggestions {
		candidate := m.batchRunCandidateForSuggestion(idx, suggestion, force)
		if idx < len(m.suggExecState) && m.suggExecState[idx] != git.ExecPending {
			continue
		}
		if !candidate.Runnable {
			if autoLoop && (suggestion.Interaction == git.NeedsInput || suggestion.Interaction == git.CommitMessage || suggestion.Interaction == git.ConflictGuide || suggestion.Interaction == git.RecoveryGuide) {
				m = m.markSuggExec(idx, git.ExecSkipped, "auto-skipped: "+candidate.Reason)
				m = m.addLog(oplog.Entry{
					Type:    oplog.EntryStateRefresh,
					Summary: "Auto mode skipped manual suggestion: " + suggestion.Action,
					Detail:  candidate.Reason,
				})
				m.rememberOperationEvent("auto-skipped: " + suggestion.Action)
				continue
			}
			if strings.TrimSpace(candidate.Reason) != "" && candidate.Reason != localizedText("already processed", "已处理", "already processed") {
				summary := "Automation stopped at suggestion: " + suggestion.Action
				if force {
					summary = "Batch run stopped at suggestion: " + suggestion.Action
				}
				m = m.addLog(oplog.Entry{
					Type:    oplog.EntryStateRefresh,
					Summary: summary,
					Detail:  candidate.Reason,
				})
			}
			return m, nil, false
		}
		m.suggIdx = idx
		m.autoSteps++
		switch suggestion.Interaction {
		case git.InfoOnly:
			m.statusMsg = fmt.Sprintf(i18n.T("messages.info_prefix"), suggestion.Reason)
			m.expanded = true
			m.llmReason = suggestion.Reason
			m.lastCommand = commandTrace{
				Title:  suggestion.Action,
				Status: "advisory viewed",
				Output: suggestion.Reason,
				At:     time.Now(),
			}
			m = m.addLog(oplog.Entry{
				Type:    oplog.EntryUserAction,
				Summary: "Automation viewed advisory: " + suggestion.Action,
				Detail:  suggestion.Reason,
			})
			m.rememberOperationEvent("automation advisory viewed: " + suggestion.Action)
			m = m.markSuggExec(idx, git.ExecDone, i18n.T("suggestions.viewed"))
			m = m.advanceToNextPending()
			continue
		case git.FileWrite:
			if suggestion.FileOp == nil {
				m = m.markSuggExec(idx, git.ExecFailed, "missing file metadata")
				continue
			}
			m.execSuggIdx = idx
			op := strings.TrimSpace(suggestion.FileOp.Operation)
			if op == "" {
				op = "create"
			}
			m.statusMsg = fmt.Sprintf("%s: %s", cases.Title(language.Und).String(op), suggestion.FileOp.Path)
			m = m.addLog(oplog.Entry{
				Type:    oplog.EntryUserAction,
				Summary: "Automation accepted file suggestion: " + suggestion.Action,
			})
			m = m.addLog(oplog.Entry{
				Type:    oplog.EntryCmdExec,
				Summary: fmt.Sprintf("File %s: %s", op, suggestion.FileOp.Path),
			})
			m.setWorkflowStage(workflowExecute)
			m = m.markSuggExec(idx, git.ExecRunning, op+"ing...")
			return m, m.executeFileOp(suggestion.FileOp), true
		case git.AutoExec:
			command := suggestionCommandForExecution(suggestion)
			m.execSuggIdx = idx
			m.statusMsg = fmt.Sprintf(i18n.T("messages.executing"), joinCmd(command))
			m = m.addLog(oplog.Entry{
				Type:    oplog.EntryUserAction,
				Summary: "Automation accepted suggestion: " + suggestion.Action,
			})
			m = m.addLog(oplog.Entry{
				Type:    oplog.EntryCmdExec,
				Summary: "Executing: " + joinCmd(command),
			})
			m.setWorkflowStage(workflowExecute)
			m = m.markSuggExec(idx, git.ExecRunning, "running...")
			return m, m.executeCommand(command), true
		case git.PlatformExec:
			m.execSuggIdx = idx
			m.statusMsg = "Executing platform action: " + platformActionTitle(suggestion.PlatformOp)
			m = m.addLog(oplog.Entry{
				Type:    oplog.EntryUserAction,
				Summary: "Automation accepted platform suggestion: " + suggestion.Action,
			})
			m.setWorkflowStage(workflowExecute)
			m = m.markSuggExec(idx, git.ExecRunning, "running...")
			model, cmd := m.beginPlatformExecution(platformExecRequest{Op: clonePlatformExecInfo(suggestion.PlatformOp)}, "Executing platform action: "+platformActionTitle(suggestion.PlatformOp))
			if cmd == nil {
				m = model
				continue
			}
			return model, cmd, true
		}
	}
	if force {
		m.statusMsg = localizedText("Batch run paused: no more executable suggestions.", "批量执行已暂停：没有更多可执行建议。", "Batch run paused: no more executable suggestions.")
	}
	return m, nil, false
}

func (m Model) handlePostExecution(hadError bool) (tea.Model, tea.Cmd) {
	m = m.advanceToNextPending()
	m.syncTaskMemory()

	if hadError && config.AutomationModeIsAutoLoop(m.automationMode()) {
		m.batchRunRequested = false
		m.lastAnalysisFingerprint = ""
		m = m.addLog(oplog.Entry{
			Type:    oplog.EntryStateRefresh,
			Summary: localizedText(
				"Command failed; triggering re-analysis and re-planning",
				"命令失败，触发重新分析和重新规划",
				"Command failed; triggering re-analysis and re-planning",
			),
		})
		return m, m.refreshGitState
	}

	if m.allSuggestionsDone() {
		m.batchRunRequested = false
		m.lastAnalysisFingerprint = ""
		goalStatus := strings.ToLower(strings.TrimSpace(m.llmGoalStatus))
		goalDone := goalStatus == "completed" || goalStatus == "blocked"
		hasGoal := strings.TrimSpace(m.session.ActiveGoal) != ""

		if goalDone || !hasGoal {
			m = m.addLog(oplog.Entry{
				Type:    oplog.EntryStateRefresh,
				Summary: localizedText(
					"All suggestions executed; goal done or no goal — refreshing state only",
					"所有建议已执行；目标已完成或无目标——仅刷新状态",
					"All suggestions executed; goal done or no goal — refreshing state only",
				),
			})
			m.skipNextAnalysis = true
			return m, tea.Cmd(m.refreshGitStateOnly)
		}

		m = m.addLog(oplog.Entry{
			Type:    oplog.EntryStateRefresh,
			Summary: localizedText(
				"All suggestions executed; re-analyzing to continue",
				"所有建议已执行；重新分析以继续",
				"All suggestions executed; re-analyzing to continue",
			),
		})
		return m, m.refreshGitState
	}

	if next, cmd, ok := m.continueBatchRunAfterExecution(); ok {
		return next, cmd
	}

	if m.shouldRunUnattended() {
		if next, cmd, ok := m.autoExecuteNextSafeSuggestion(false); ok {
			return next, cmd
		}
		m.lastAnalysisFingerprint = ""
		m = m.addLog(oplog.Entry{
			Type:    oplog.EntryStateRefresh,
			Summary: "No more auto-executable suggestions; refreshing state for re-analysis",
		})
		return m, m.refreshGitState
	}

	m.skipNextAnalysis = true
	return m, tea.Cmd(m.refreshGitStateOnly)
}

func (m Model) continueBatchRunAfterExecution() (Model, tea.Cmd, bool) {
	if !m.batchRunRequested {
		return m, nil, false
	}
	if next, cmd, ok := m.autoExecuteNextSafeSuggestion(true); ok {
		next.batchRunRequested = true
		return next, cmd, true
	}
	m.batchRunRequested = false
	m.setCommandResponse(localizedCommandsTitle(), m.batchRunSummary(true))
	return m, nil, false
}

func isAutomationSafeSuggestion(s git.Suggestion, trusted bool) bool {
	if trusted {
		return true
	}
	if s.RiskLevel == git.RiskDangerous {
		return false
	}
	switch s.Interaction {
	case git.InfoOnly:
		return true
	case git.AutoExec:
		return isAutomationSafeCommand(s.Command)
	case git.PlatformExec:
		if s.PlatformOp == nil {
			return false
		}
		flow := strings.ToLower(strings.TrimSpace(s.PlatformOp.Flow))
		return flow == "inspect" || flow == "validate"
	default:
		return false
	}
}

func isAutomationSafeCommand(argv []string) bool {
	if len(argv) < 2 || !strings.EqualFold(argv[0], "git") {
		return false
	}
	sub := strings.ToLower(strings.TrimSpace(argv[1]))
	switch sub {
	case "status", "diff", "log", "show", "branch", "check-ignore",
		"remote", "rev-parse", "ls-files", "stash", "tag",
		"add", "commit", "switch", "checkout", "merge", "rebase",
		"push", "pull", "fetch", "reset", "clean", "rm", "mv":
		return true
	default:
		return false
	}
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

	switch {
	case key == "escape" || key == "esc" || msg.Key().Code == tea.KeyEscape:
		m.screen = screenMain
		m.inputSuggRef = nil
		m.statusMsg = i18n.T("input.cancelled")
		m = m.addLog(oplog.Entry{
			Type:    oplog.EntryUserAction,
			Summary: "Input cancelled",
		})
		return m, nil

	case key == "enter":
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

	case key == "tab":
		if m.inputIdx < len(m.inputFields)-1 {
			m.inputIdx++
			m.inputCursorAt = runeLen(m.inputValues[m.inputIdx])
		}
		return m, nil

	case key == "shift+tab":
		if m.inputIdx > 0 {
			m.inputIdx--
			m.inputCursorAt = runeLen(m.inputValues[m.inputIdx])
		}
		return m, nil

	case key == "backspace":
		v, nextCursor := deleteRuneBefore(m.inputValues[m.inputIdx], m.inputCursorAt)
		m.inputValues[m.inputIdx] = v
		m.inputCursorAt = nextCursor
		return m, nil

	case key == "delete":
		v, nextCursor := deleteRuneAt(m.inputValues[m.inputIdx], m.inputCursorAt)
		m.inputValues[m.inputIdx] = v
		m.inputCursorAt = nextCursor
		return m, nil

	case key == "left":
		if m.inputCursorAt > 0 {
			m.inputCursorAt--
		}
		return m, nil

	case key == "right":
		if m.inputCursorAt < runeLen(m.inputValues[m.inputIdx]) {
			m.inputCursorAt++
		}
		return m, nil

	case key == "home" || key == "ctrl+a":
		m.inputCursorAt = 0
		return m, nil

	case key == "end" || key == "ctrl+e":
		m.inputCursorAt = runeLen(m.inputValues[m.inputIdx])
		return m, nil

	case key == "ctrl+c":
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
	updated, nextCursor := insertAtRune(m.inputValues[m.inputIdx], m.inputCursorAt, text)
	m.inputValues[m.inputIdx] = updated
	m.inputCursorAt = nextCursor
	return m, nil
}

func isMainShortcutKey(key string) bool {
	switch key {
	case "q", "ctrl+c", "l", "o", "O", "]", "[", "L", "S", "shift+s", "up", "down", "pgup", "pgdown", "y", "n", "tab", "shift+tab", "w", "z", "t", "r", "g", "m", "f", "v", "b", "e", "u", "U", "P", "p", "shift+p", "R", "shift+r", "X", "x", "shift+x", "A", "a", "shift+a", "K", "k", "shift+k", "C", "c", "shift+c", "H", "shift+h", "Y", "shift+y", ".", ",", ">", "<", "shift+.", "shift+,":
		return true
	default:
		return false
	}
}

func (m Model) handleComposerPaste(text string) (tea.Model, tea.Cmd) {
	if text == "" {
		return m, nil
	}
	updated, nextCursor := insertAtRune(m.composerInput, m.composerCursor, text)
	m.composerInput = updated
	m.composerCursor = nextCursor
	m.slashCursor = 0
	m.clampSlashCursor()
	return m, nil
}

func (m Model) submitInlineGoal() (tea.Model, tea.Cmd) {
	goal := strings.TrimSpace(m.composerInput)
	m.composerInput = ""
	m.composerCursor = 0
	m.slashCursor = 0
	m.composerFocused = true
	if strings.HasPrefix(goal, "/") {
		return m.runSlashCommand(goal)
	}
	return m.applyActiveGoal(goal)
}

func (m Model) applyActiveGoal(goal string) (tea.Model, tea.Cmd) {
	m.session.ActiveGoal = goal
	m.llmGoalStatus = "in_progress"
	m.workflowPlan = nil
	m.workflowFlow = nil
	m.batchRunRequested = false
	m.lastAnalysisFingerprint = ""
	m.clearCommandResponse()
	m.syncTaskMemory()
	m.persistAutomationCheckpoint()
	m.statusMsg = fmt.Sprintf(i18n.T("goal.set"), goal)
	m = m.addLog(oplog.Entry{
		Type:    oplog.EntryUserAction,
		Summary: "Set active goal: " + goal,
	})
	m.rememberOperationEvent("goal:" + goal)
	if m.llmProvider != nil && m.gitState != nil {
		m.analysisSeq++
		m.pendingAnalysisID = m.analysisSeq
		m.pendingExplainID = 0
		m.expanded = false
		m.llmReason = ""
		m.llmThinking = ""
		m.llmPlanOverview = ""
		m.llmGoalStatus = ""
		m.suggestions = nil
		m.suggExecState = nil
		m.suggExecMsg = nil
		m.lastPlatformOp = nil
		m.workspaceTab = workspaceTabAnalysis
		m.leftScroll = 0
		m.obsScroll = 0
		m.setWorkflowStage(workflowAnalyze)
		m.llmAnalysis = i18n.T("analysis.analyzing_repo")
		m = m.addLog(oplog.Entry{
			Type:    oplog.EntryLLMStart,
			Summary: fmt.Sprintf("Goal analysis started (%s mode)", m.mode),
			Detail:  m.buildAnalysisStartDetail(),
		})
		return m, m.runLLMAnalysis(m.pendingAnalysisID, m.gitState)
	}
	return m, nil
}

// executeInputCommand substitutes user input into the command template and executes.
func (m Model) executeInputCommand() (tea.Model, tea.Cmd) {
	if m.inputSuggRef == nil {
		m.screen = screenMain
		return m, nil
	}

	m.execSuggIdx = m.suggIdx
	m.screen = screenMain
	m = m.addLog(oplog.Entry{
		Type:    oplog.EntryUserAction,
		Summary: "Accepted input suggestion: " + m.inputSuggRef.Action,
	})
	m.setWorkflowStage(workflowExecute)
	m = m.markSuggExec(m.suggIdx, git.ExecRunning, "running...")
	suggestion := *m.inputSuggRef
	m.inputSuggRef = nil
	if suggestion.Interaction == git.PlatformExec {
		op := applyPlatformInputs(suggestion.PlatformOp, m.inputFields, m.inputValues)
		return m.beginPlatformExecution(platformExecRequest{Op: op}, "Executing platform action: "+platformActionTitle(op))
	}

	cmd := make([]string, len(suggestion.Command))
	copy(cmd, suggestion.Command)
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
	m.statusMsg = fmt.Sprintf(i18n.T("messages.executing"), joinCmd(cmd))
	m = m.addLog(oplog.Entry{
		Type:    oplog.EntryCmdExec,
		Summary: "Executing: " + joinCmd(cmd),
	})
	return m, m.executeCommand(cmd)
}

func (m Model) updateModelSelect(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if m.modelSelectMode == modelSelectProviders {
		options := providerOptions()
		switch key {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "escape", "esc":
			m.screen = screenMain
			m.statusMsg = ""
			return m, nil
		case "up", "k":
			if m.modelCursor > 0 {
				m.modelCursor--
			}
			return m, nil
		case "down", "j":
			if m.modelCursor < len(options)-1 {
				m.modelCursor++
			}
			return m, nil
		case "enter":
			spec := m.selectedProviderSpec()
			switch spec.Kind {
			case llm.ProviderUILocalModels:
				if m.hasModelsForProvider(spec.ID) {
					return m.openLocalModelSelection(m.modelSelectPhase, spec.ID), nil
				}
				return m, func() tea.Msg {
					return m.fetchSetupProviderModels(spec.ID)
				}
			default:
				m = m.openProviderConfig(m.modelSelectPhase)
				if m.providerDraft.Provider != spec.ID {
					m.providerDraft.Provider = spec.ID
					m.providerDraft.Endpoint = defaultProviderEndpoint(spec.ID)
					m.providerDraft.APIKeyEnv = defaultProviderAPIKeyEnv(spec.ID)
					m.providerDraft.Model = firstRecommendedModel(spec.ID)
				}
				m.providerField = providerFieldModel
				m.providerCursorAt = runeLen(m.providerDraft.Model)
				return m, nil
			}
		case "c":
			spec := m.selectedProviderSpec()
			m = m.openProviderConfig(m.modelSelectPhase)
			if m.providerDraft.Provider != spec.ID {
				m.providerDraft.Provider = spec.ID
				m.providerDraft.Endpoint = defaultProviderEndpoint(spec.ID)
				m.providerDraft.APIKeyEnv = defaultProviderAPIKeyEnv(spec.ID)
				m.providerDraft.Model = firstRecommendedModel(spec.ID)
			}
			if spec.Kind == llm.ProviderUILocalModels {
				m.providerField = providerFieldEndpoint
				m.providerCursorAt = runeLen(m.providerDraft.Endpoint)
			} else {
				m.providerField = providerFieldModel
				m.providerCursorAt = runeLen(m.providerDraft.Model)
			}
			return m, nil
		case "tab":
			if m.modelSelectPhase == selectPrimary {
				m = m.openModelSetup(selectSecondary)
			} else {
				m = m.openModelSetup(selectPrimary)
			}
			return m, nil
		}
		return m, nil
	}

	switch key {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "escape", "esc":
		return m.openModelSetup(m.modelSelectPhase), nil

	case "up", "k":
		if m.modelCursor > 0 {
			m.modelCursor--
		}

	case "down", "j":
		if m.modelCursor < len(m.currentSelectableModels())-1 {
			m.modelCursor++
		}

	case "enter":
		models := m.currentSelectableModels()
		if len(models) == 0 {
			return m.openProviderConfig(m.modelSelectPhase), nil
		}
		if m.modelCursor < 0 {
			m.modelCursor = 0
		}
		if m.modelCursor >= len(models) {
			m.modelCursor = len(models) - 1
		}
		selected := models[m.modelCursor]
		updated, err := m.persistRoleModelSelection(m.modelSelectPhase, m.modelListProvider, selected.Name)
		if err != nil {
			m.statusMsg = err.Error()
			return m, nil
		}
		m = updated
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
			m = m.openModelSetup(selectSecondary)
			m.statusMsg = fmt.Sprintf(i18n.T("model_select.primary_selected"), selected.Name)
			m = m.addLog(oplog.Entry{
				Type:    oplog.EntryUserAction,
				Summary: "Selected primary model: " + m.primaryProvider + "/" + selected.Name,
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
			Summary: "Selected secondary model: " + m.secondaryProvider + "/" + selected.Name,
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

	case "p":
		return m.openProviderConfig(m.modelSelectPhase), nil

	case "tab":
		if m.modelSelectPhase == selectPrimary {
			return m.openModelSetup(selectSecondary), nil
		}
		return m.openModelSetup(selectPrimary), nil
	}
	return m, nil
}

func (m Model) updateMain(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	text := msg.Key().Text
	if key == "" {
		key = strings.TrimSpace(text)
	}

	if m.composerFocused {
		switch key {
		case "ctrl+c":
			return m, tea.Quit
		case "left":
			if m.composerCursor > 0 {
				m.composerCursor--
			}
			return m, nil
		case "right":
			if m.composerCursor < runeLen(m.composerInput) {
				m.composerCursor++
			}
			return m, nil
		case "home", "ctrl+a":
			m.composerCursor = 0
			return m, nil
		case "end", "ctrl+e":
			m.composerCursor = runeLen(m.composerInput)
			return m, nil
		case "backspace":
			if m.composerInput != "" {
				m.composerInput, m.composerCursor = deleteRuneBefore(m.composerInput, m.composerCursor)
				m.slashCursor = 0
				m.clampSlashCursor()
			}
			return m, nil
		case "delete":
			if m.composerInput != "" {
				m.composerInput, m.composerCursor = deleteRuneAt(m.composerInput, m.composerCursor)
				m.slashCursor = 0
				m.clampSlashCursor()
			}
			return m, nil
		case "up":
			if strings.HasPrefix(strings.TrimSpace(m.composerInput), "/") && m.moveSlashCursor(-1) {
				return m, nil
			}
			return m, nil
		case "down":
			if strings.HasPrefix(strings.TrimSpace(m.composerInput), "/") && m.moveSlashCursor(1) {
				return m, nil
			}
			return m, nil
		case "tab":
			if strings.HasPrefix(strings.TrimSpace(m.composerInput), "/") {
				if m.applySlashSuggestion(m.slashCursor) {
					return m, nil
				}
			}
			return m, nil
		case "enter":
			if strings.TrimSpace(m.composerInput) != "" {
				if strings.HasPrefix(strings.TrimSpace(m.composerInput), "/") {
					if selected, ok := m.selectedSlashCommand(); ok {
						query := normalizeSlashQuery(m.composerInput)
						commandName := strings.ToLower(strings.TrimSpace(selected.Command))
						templateName := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(selected.Template, "/")))
						if query != commandName && query != templateName {
							if m.applySlashSuggestion(m.slashCursor) {
								return m, nil
							}
						}
					}
				}
				return m.submitInlineGoal()
			}
			return m, nil
		case "escape", "esc":
			m.composerFocused = false
			m.statusMsg = "Prompt unfocused"
			return m, nil
		default:
			if text != "" {
				return m.handleComposerPaste(text)
			}
		}
	}

	switch key {
	case "/":
		m.composerFocused = true
		m.statusMsg = "Prompt focused"
		return m, nil
	}

	if text != "" && !isMainShortcutKey(key) {
		m.composerFocused = true
		return m.handleComposerPaste(text)
	}

	switch key {
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
		m.obsScroll = 0
		m.statusMsg = fmt.Sprintf(i18n.T("observability.inspector_status"), m.obsTab.label())
		return m, nil

	case "O":
		m.obsTab = m.obsTab.prev()
		m.obsScroll = 0
		m.statusMsg = fmt.Sprintf(i18n.T("observability.inspector_status"), m.obsTab.label())
		return m, nil

	case "]":
		m = m.cycleScrollPane(true)
		m.statusMsg = m.describeScrollFocus()
		return m, nil

	case "[":
		m = m.cycleScrollPane(false)
		m.statusMsg = m.describeScrollFocus()
		return m, nil

	case ">", ".", "shift+.":
		title, ok := m.moveWorkflowStepSelection(1)
		if !ok {
			m.statusMsg = "No workflow step available to select"
			return m, nil
		}
		m.statusMsg = "Selected workflow step: " + title
		return m, nil

	case "<", ",", "shift+,":
		title, ok := m.moveWorkflowStepSelection(-1)
		if !ok {
			m.statusMsg = "No workflow step available to select"
			return m, nil
		}
		m.statusMsg = "Selected workflow step: " + title
		return m, nil

	case "L":
		m = m.openLanguageSelection(screenMain)
		return m, nil

	case "P", "p", "shift+p":
		if m.workflowFlow == nil {
			m.statusMsg = "No active workflow flow to pause"
			return m, nil
		}
		target := ""
		if step := m.selectedWorkflowStep(); step != nil {
			target = strings.TrimSpace(firstNonEmpty(step.Step.Title, step.Identity))
		}
		m.pauseWorkflowFlow("operator pause")
		m.persistAutomationCheckpoint()
		if target != "" {
			m.statusMsg = "Workflow step paused: " + target
		} else {
			m.statusMsg = "Workflow flow paused"
		}
		m = m.addLog(oplog.Entry{
			Type:    oplog.EntryUserAction,
			Summary: "Workflow step paused",
			Detail:  target,
		})
		return m, nil

	case "R", "shift+r":
		if !m.workflowHasPausedSteps() {
			m.statusMsg = "No paused workflow flow to resume"
			return m, nil
		}
		target := ""
		if step := m.selectedWorkflowStep(); step != nil {
			target = strings.TrimSpace(firstNonEmpty(step.Step.Title, step.Identity))
		}
		m.resumeWorkflowFlow()
		m.persistAutomationCheckpoint()
		if target != "" {
			m.statusMsg = "Workflow step resumed: " + target
		} else {
			m.statusMsg = "Workflow flow resumed"
		}
		m = m.addLog(oplog.Entry{
			Type:    oplog.EntryUserAction,
			Summary: "Workflow step resumed",
			Detail:  target,
		})
		return m, nil

	case "X", "x", "shift+x":
		title, ok := m.retryDeadLetterWorkflowStep()
		if !ok {
			m.statusMsg = "No dead-letter workflow step available to retry"
			return m, nil
		}
		m.persistAutomationCheckpoint()
		m.statusMsg = "Queued retry for dead-letter step: " + title
		m = m.addLog(oplog.Entry{
			Type:    oplog.EntryUserAction,
			Summary: "Workflow dead-letter retry queued",
			Detail:  title,
		})
		return m, nil

	case "A", "a", "shift+a":
		title, ok := m.ackDeadLetterWorkflowStep()
		if !ok {
			m.statusMsg = "No dead-letter workflow step available to acknowledge"
			return m, nil
		}
		m.persistAutomationCheckpoint()
		m.statusMsg = "Acknowledged dead-letter step: " + title
		m = m.addLog(oplog.Entry{
			Type:    oplog.EntryUserAction,
			Summary: "Workflow dead-letter acknowledged",
			Detail:  title,
		})
		return m, nil

	case "K", "k", "shift+k":
		title, ok := m.skipDeadLetterWorkflowStep("operator skipped dead-letter step")
		if !ok {
			m.statusMsg = "No dead-letter workflow step available to skip"
			return m, nil
		}
		m.persistAutomationCheckpoint()
		m.statusMsg = "Skipped dead-letter step: " + title
		m = m.addLog(oplog.Entry{
			Type:    oplog.EntryUserAction,
			Summary: "Workflow dead-letter skipped",
			Detail:  title,
		})
		return m, nil

	case "C", "c", "shift+c":
		step := m.compensableWorkflowStep()
		req, ok := m.compensationRequestForStep(step)
		if !ok {
			m.statusMsg = "No compensating rollback is available for the current workflow flow"
			return m, nil
		}
		m = m.addLog(oplog.Entry{
			Type:    oplog.EntryUserAction,
			Summary: "Workflow compensation requested",
			Detail:  step.Step.Title,
		})
		return m.beginPlatformExecution(req, "Running compensating rollback for workflow step")

	case "u", "U":
		key, ok := m.clearSelectedAutomationLock()
		if !ok {
			m.statusMsg = "No automation lock available to clear"
			return m, nil
		}
		m.persistAutomationCheckpoint()
		m.statusMsg = "Cleared automation lock: " + key
		m = m.addLog(oplog.Entry{
			Type:    oplog.EntryUserAction,
			Summary: "Automation lock cleared",
			Detail:  key,
		})
		return m, nil

	case "H", "shift+h":
		if !m.recoverAutomationEscalation("operator recovered unattended execution path") {
			m.statusMsg = "Automation is not in observe-only recovery state"
			return m, nil
		}
		m.persistAutomationCheckpoint()
		return m, nil

	case "Y", "shift+y":
		step := m.selectedWorkflowStep()
		if !m.approveWorkflowStep(step, "operator approved selected step") {
			m.statusMsg = "No approval-required workflow step available to approve"
			return m, nil
		}
		target := ""
		if step != nil {
			target = strings.TrimSpace(firstNonEmpty(step.Step.Title, step.Identity))
		}
		m.persistAutomationCheckpoint()
		m.statusMsg = "Workflow step approved: " + target
		m = m.addLog(oplog.Entry{
			Type:    oplog.EntryUserAction,
			Summary: "Workflow step approved",
			Detail:  target,
		})
		return m, nil

	case "up":
		return m.scrollPaneBy(m.scrollFocus, -1), nil

	case "down":
		return m.scrollPaneBy(m.scrollFocus, 1), nil

	case "pgup":
		return m.scrollPaneBy(m.scrollFocus, -6), nil

	case "pgdown":
		return m.scrollPaneBy(m.scrollFocus, 6), nil

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
				m.rememberOperationEvent("advisory viewed: " + s.Action)
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
				m.statusMsg = fmt.Sprintf("%s: %s", cases.Title(language.Und).String(op), s.FileOp.Path)
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
				m.inputCursorAt = runeLen(m.inputValues[0])
				m.inputSuggRef = &s
				m.statusMsg = ""
				m = m.addLog(oplog.Entry{
					Type:    oplog.EntryUserAction,
					Summary: "Preparing input for: " + s.Action,
					Detail:  "Command template: " + joinCmd(s.Command),
				})
				return m, nil

			case git.PlatformExec:
				if s.PlatformOp == nil {
					m.statusMsg = "Platform suggestion is missing executor metadata"
					m = m.markSuggExec(m.suggIdx, git.ExecFailed, "missing platform metadata")
					return m, nil
				}
				if len(s.Inputs) > 0 {
					m.screen = screenInput
					m.inputFields = s.Inputs
					m.inputIdx = 0
					m.inputValues = make([]string, len(s.Inputs))
					for i := range s.Inputs {
						m.inputValues[i] = s.Inputs[i].DefaultValue
					}
					m.inputCursorAt = runeLen(m.inputValues[0])
					m.inputSuggRef = &s
					m.statusMsg = ""
					m = m.addLog(oplog.Entry{
						Type:    oplog.EntryUserAction,
						Summary: "Preparing platform input: " + s.Action,
						Detail:  platformActionTitle(s.PlatformOp),
					})
					return m, nil
				}
				m.execSuggIdx = m.suggIdx
				m = m.addLog(oplog.Entry{
					Type:    oplog.EntryUserAction,
					Summary: "Accepted platform suggestion: " + s.Action,
				})
				m.setWorkflowStage(workflowExecute)
				m = m.markSuggExec(m.suggIdx, git.ExecRunning, "running...")
				return m.beginPlatformExecution(platformExecRequest{Op: clonePlatformExecInfo(s.PlatformOp)}, "Executing platform action: "+platformActionTitle(s.PlatformOp))

			default: // AutoExec
				command := suggestionCommandForExecution(s)
				m.execSuggIdx = m.suggIdx
				m.statusMsg = fmt.Sprintf(i18n.T("messages.executing"), joinCmd(command))
				m = m.addLog(oplog.Entry{
					Type:    oplog.EntryUserAction,
					Summary: "Accepted suggestion: " + s.Action,
				})
				m = m.addLog(oplog.Entry{
					Type:    oplog.EntryCmdExec,
					Summary: "Executing: " + joinCmd(command),
				})
				m.setWorkflowStage(workflowExecute)
				m = m.markSuggExec(m.suggIdx, git.ExecRunning, "running...")
				return m, m.executeCommand(command)
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
			m.showSelectedSuggestionGuidance()
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
			m.showSelectedSuggestionGuidance()
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
		m.lastAnalysisFingerprint = ""
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
		m.lastAnalysisFingerprint = ""
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
		m.leftScroll = 0
		m.areasScroll = 0
		m.obsScroll = 0
		return m, m.refreshGitState

	case "g":
		m.composerFocused = false
		m.screen = screenGoalInput
		if strings.TrimSpace(m.composerInput) != "" {
			m.goalInput = strings.TrimSpace(m.composerInput)
		} else {
			m.goalInput = strings.TrimSpace(m.session.ActiveGoal)
		}
		m.goalCursorAt = runeLen(m.goalInput)
		m.statusMsg = i18n.T("goal.prompt")
		return m, nil

	case "m":
		m.composerFocused = false
		m = m.openModelSetup(selectPrimary)
		return m, nil

	case "S", "shift+s":
		m.composerFocused = false
		m = m.openAutomationConfig()
		return m, nil

	case "v":
		req, ok := m.lastPlatformRequest("validate")
		if !ok {
			m.statusMsg = "No platform mutation available to validate"
			return m, nil
		}
		m = m.addLog(oplog.Entry{
			Type:    oplog.EntryUserAction,
			Summary: "Validate latest platform mutation",
			Detail:  platformActionTitle(req.Op),
		})
		return m.beginPlatformExecution(req, "Validating latest platform mutation")

	case "b":
		req, ok := m.lastPlatformRequest("rollback")
		if !ok {
			m.statusMsg = "No platform mutation available to roll back"
			return m, nil
		}
		m = m.addLog(oplog.Entry{
			Type:    oplog.EntryUserAction,
			Summary: "Rollback latest platform mutation",
			Detail:  platformActionTitle(req.Op),
		})
		return m.beginPlatformExecution(req, "Rolling back latest platform mutation")

	case "e":
		if req := m.editableFileRequest(); req != nil {
			m = m.openFileEdit(req)
			return m, nil
		}
		req := m.editablePlatformRequest()
		if req == nil {
			m.statusMsg = "No editable result is available"
			return m, nil
		}
		m = m.openPlatformEdit(req)
		return m, nil

	case "f":
		m.composerFocused = false
		if len(m.workflows) == 0 {
			m.workflows = loadWorkflowDefinitions()
		}
		if len(m.workflows) == 0 {
			m.statusMsg = i18n.T("workflow_menu.no_workflows")
			return m, nil
		}
		m.workflowCursor = 0
		m.workflowScroll = 0
		m.screen = screenWorkflowSelect
		m.statusMsg = i18n.T("workflow_menu.prompt")
		return m, nil
	}
	return m, nil
}

func (m Model) updateGoalInput(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	text := msg.Key().Text

	switch {
	case key == "escape" || key == "esc" || msg.Key().Code == tea.KeyEscape:
		m.screen = screenMain
		m.statusMsg = i18n.T("goal.cancelled")
		return m, nil
	case key == "enter":
		goal := strings.TrimSpace(m.goalInput)
		m.session.ActiveGoal = goal
		m.workflowPlan = nil
		m.workflowFlow = nil
		m.lastAnalysisFingerprint = ""
		m.syncTaskMemory()
		m.persistAutomationCheckpoint()
		m.composerInput = ""
		m.composerCursor = 0
		m.composerFocused = false
		m.screen = screenMain
		if goal == "" {
			m.statusMsg = i18n.T("goal.cleared")
			m = m.addLog(oplog.Entry{
				Type:    oplog.EntryUserAction,
				Summary: "Cleared active goal",
			})
			m.rememberOperationEvent("goal:cleared")
		} else {
			m.statusMsg = fmt.Sprintf(i18n.T("goal.active"), goal)
			m = m.addLog(oplog.Entry{
				Type:    oplog.EntryUserAction,
				Summary: "Set active goal: " + goal,
			})
			m.rememberOperationEvent("goal:" + goal)
		}
		if m.llmProvider != nil && m.gitState != nil {
			m.analysisSeq++
			m.pendingAnalysisID = m.analysisSeq
			m.pendingExplainID = 0
			m.expanded = false
			m.llmReason = ""
			m.llmThinking = ""
			m.llmPlanOverview = ""
			m.llmGoalStatus = ""
			m.suggestions = nil
			m.suggExecState = nil
			m.suggExecMsg = nil
			m.lastPlatformOp = nil
			m.leftScroll = 0
			m.obsScroll = 0
			m.setWorkflowStage(workflowAnalyze)
			m.llmAnalysis = i18n.T("analysis.analyzing_repo")
			m = m.addLog(oplog.Entry{
				Type:    oplog.EntryLLMStart,
				Summary: fmt.Sprintf("Goal analysis started (%s mode)", m.mode),
				Detail:  m.buildAnalysisStartDetail(),
			})
			return m, m.runLLMAnalysis(m.pendingAnalysisID, m.gitState)
		}
		return m, nil
	case key == "backspace":
		m.goalInput, m.goalCursorAt = deleteRuneBefore(m.goalInput, m.goalCursorAt)
		return m, nil
	case key == "delete":
		m.goalInput, m.goalCursorAt = deleteRuneAt(m.goalInput, m.goalCursorAt)
		return m, nil
	case key == "left":
		if m.goalCursorAt > 0 {
			m.goalCursorAt--
		}
		return m, nil
	case key == "right":
		if m.goalCursorAt < runeLen(m.goalInput) {
			m.goalCursorAt++
		}
		return m, nil
	case key == "home" || key == "ctrl+a":
		m.goalCursorAt = 0
		return m, nil
	case key == "end" || key == "ctrl+e":
		m.goalCursorAt = runeLen(m.goalInput)
		return m, nil
	case key == "ctrl+c":
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
	m.goalInput, m.goalCursorAt = insertAtRune(m.goalInput, m.goalCursorAt, text)
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
		if len(m.workflows) == 0 {
			return m, nil
		}
		if m.workflowCursor > 0 {
			m.workflowCursor--
		} else {
			m.workflowCursor = len(m.workflows) - 1
		}
		if m.workflowCursor < m.workflowScroll {
			m.workflowScroll = m.workflowCursor
		}
		return m, nil
	case key == "down" || key == "j":
		if len(m.workflows) == 0 {
			return m, nil
		}
		if m.workflowCursor < len(m.workflows)-1 {
			m.workflowCursor++
		} else {
			m.workflowCursor = 0
			m.workflowScroll = 0
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
		m.composerInput = ""
		m.composerCursor = 0
		m.composerFocused = false
		m.lastAnalysisFingerprint = ""
		m.rememberRepoPattern("workflow:" + wf.ID)
		m.rememberOperationEvent("workflow:" + wf.ID)
		m.screen = screenMain
		m.workflowPlan = buildWorkflowOrchestration(wf, m.gitState)
		m.syncWorkflowFlowFromPlan()
		m.syncTaskMemory()
		m.persistAutomationCheckpoint()
		m.statusMsg = fmt.Sprintf(i18n.T("workflow_menu.selected"), wf.Label)
		m = m.addLog(oplog.Entry{
			Type:    oplog.EntryUserAction,
			Summary: "Workflow selected: " + wf.ID,
			Detail:  wf.Goal,
		})
		if m.llmProvider != nil && m.gitState != nil {
			m.analysisSeq++
			m.pendingAnalysisID = m.analysisSeq
			m.pendingExplainID = 0
			m.expanded = false
			m.llmReason = ""
			m.llmThinking = ""
			m.llmPlanOverview = ""
			m.llmGoalStatus = ""
			m.suggestions = nil
			m.suggExecState = nil
			m.suggExecMsg = nil
			m.lastPlatformOp = nil
			m.leftScroll = 0
			m.obsScroll = 0
			m.setWorkflowStage(workflowAnalyze)
			if m.workflowPlan != nil && len(m.workflowPlan.Steps) > 0 {
				m.llmAnalysis = fmt.Sprintf("Workflow %s selected. Sending %d schema-backed platform steps to the LLM for final orchestration.", wf.Label, len(m.workflowPlan.Steps))
				m.llmPlanOverview = fmt.Sprintf("workflow seed: %d platform step(s)", len(m.workflowPlan.Steps))
			} else {
				m.llmAnalysis = i18n.T("analysis.analyzing_repo")
			}
			m = m.addLog(oplog.Entry{
				Type:    oplog.EntryLLMStart,
				Summary: fmt.Sprintf("Workflow analysis started (%s mode)", m.mode),
				Detail:  m.buildAnalysisStartDetail(),
			})
			return m, m.runLLMAnalysis(m.pendingAnalysisID, m.gitState)
		}
		if m.workflowPlan != nil && len(m.workflowPlan.Steps) > 0 {
			m.llmAnalysis = fmt.Sprintf("Workflow %s prepared %d platform orchestration hints, but AI setup is required to materialize final suggestions.", wf.Label, len(m.workflowPlan.Steps))
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

func (m Model) needsProviderConfiguration() bool {
	if m.llmProvider == nil {
		return true
	}
	if strings.TrimSpace(m.selectedPrimary) == "" {
		return true
	}
	role := m.llmConfig.PrimaryRole()
	switch config.RoleProvider(role) {
	case "openai", "deepseek":
		return config.ResolveRoleAPIKey(role) == ""
	default:
		return false
	}
}

func (m Model) shouldRequireProviderSetupOnStartup() bool {
	if m.needsProviderConfiguration() {
		return true
	}
	if m.primaryProvider == "ollama" && m.modelsFetched && len(m.availModels) == 0 {
		return true
	}
	return false
}

func inferModelProvider(models []llm.ModelInfo, fallback string) string {
	if len(models) == 0 {
		return strings.ToLower(strings.TrimSpace(fallback))
	}
	provider := strings.ToLower(strings.TrimSpace(models[0].Provider))
	if provider != "" {
		return provider
	}
	return strings.ToLower(strings.TrimSpace(fallback))
}

func oneLine(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	lines := strings.Split(text, "\n")
	return strings.TrimSpace(lines[0])
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
