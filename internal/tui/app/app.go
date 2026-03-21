package app

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"net/http"

	gh "github.com/google/go-github/v84/github"
	"github.com/your-org/gitdex/internal/app/autonomyexec"
	"github.com/your-org/gitdex/internal/app/bootstrap"
	"github.com/your-org/gitdex/internal/autonomy"
	"github.com/your-org/gitdex/internal/gitops"
	"github.com/your-org/gitdex/internal/llm/adapter"
	"github.com/your-org/gitdex/internal/llm/chat"
	"github.com/your-org/gitdex/internal/platform/config"
	ghclient "github.com/your-org/gitdex/internal/platform/github"
	"github.com/your-org/gitdex/internal/platform/identity"
	"github.com/your-org/gitdex/internal/state/repo"
	"github.com/your-org/gitdex/internal/storage"
	"github.com/your-org/gitdex/internal/tui/components"
	"github.com/your-org/gitdex/internal/tui/keymap"
	"github.com/your-org/gitdex/internal/tui/layout"
	"github.com/your-org/gitdex/internal/tui/panes"
	"github.com/your-org/gitdex/internal/tui/theme"
	"github.com/your-org/gitdex/internal/tui/views"
)

type FocusArea int

const (
	FocusNav FocusArea = iota
	FocusContent
	FocusComposer
	FocusInspector
	FocusPalette
)

type CommandHandler func(args string) string

type Model struct {
	theme      *theme.Theme
	styles     theme.Styles
	dims       layout.Dimensions
	globalKeys keymap.GlobalKeys
	showHelp   bool

	header        *components.Header
	router        *views.Router
	composer      *components.Composer
	statusBar     *components.StatusBar
	cmdPalette    *components.CmdPalette
	navPane       *panes.NavPane
	inspectorPane *panes.InspectorPane

	dashboardView *views.DashboardView
	chatView      *views.ChatView
	explorerView  *views.ExplorerView
	workspaceView *views.WorkspaceView
	settingsView  *views.SettingsView
	reflogView    *views.ReflogView

	focus FocusArea
	ready bool

	paletteIdx   int
	paletteName  string
	cmdHandlers  map[string]CommandHandler
	postCommand  tea.Cmd
	llmProvider  adapter.Provider
	chatSession  *chat.Session
	streamCancel context.CancelFunc
	runtimeApp   bootstrap.App
	activeRepo   *repo.RepoContext
	ghClient     *ghclient.Client
	localIndex   *gitops.LocalIndex
	cloneRemote  func(ctx context.Context, url, dir string, opts gitops.CloneOptions) error
}

func switchViewCmd(id views.ID) func() tea.Cmd {
	return func() tea.Cmd {
		return func() tea.Msg { return views.SwitchViewMsg{Target: id} }
	}
}

func New() Model {
	theme.SetNerdFont(theme.DetectNerdFont())
	th := theme.NewTheme(true)
	t := &th
	s := theme.NewStyles(th)

	dashboardView := views.NewDashboardView(t)
	chatView := views.NewChatView(t)
	explorerView := views.NewExplorerView(t)
	workspaceView := views.NewWorkspaceView(t)
	settingsView := views.NewSettingsView(t)
	reflogView := views.NewReflogView(t)

	router := views.NewRouter(views.ViewDashboard, dashboardView, chatView, explorerView, workspaceView, settingsView, reflogView)

	navItems := []panes.NavItem{
		{Label: "Dashboard", Path: "dashboard"},
		{Label: "Chat", Path: "chat"},
		{Label: "Explorer", Path: "explorer"},
		{Label: "Workspace", Path: "workspace"},
		{Label: "Settings", Path: "settings"},
		{Label: "Reflog", Path: "reflog"},
	}
	navPane := panes.NewNavPane(t, s, navItems)

	statusBar := components.NewStatusBar(t)
	statusBar.SetMode("INSERT")
	statusBar.SetThemeName("default")

	cmdPalette := components.NewCmdPalette(t)
	cmdPalette.AddItem(components.PaletteItem{Category: "Views", Label: "Dashboard", Description: "Overview & health", Shortcut: "F1", Action: switchViewCmd(views.ViewDashboard)})
	cmdPalette.AddItem(components.PaletteItem{Category: "Views", Label: "Chat", Description: "AI conversation", Shortcut: "F2", Action: switchViewCmd(views.ViewChat)})
	cmdPalette.AddItem(components.PaletteItem{Category: "Views", Label: "Explorer", Description: "PRs, issues, files", Shortcut: "F3", Action: switchViewCmd(views.ViewExplorer)})
	cmdPalette.AddItem(components.PaletteItem{Category: "Views", Label: "Workspace", Description: "Plans, tasks, evidence, cruise, approvals", Shortcut: "F4", Action: switchViewCmd(views.ViewWorkspace)})
	cmdPalette.AddItem(components.PaletteItem{Category: "Views", Label: "Settings", Description: "Configure Gitdex", Shortcut: "F5", Action: switchViewCmd(views.ViewSettings)})
	cmdPalette.AddItem(components.PaletteItem{Category: "Views", Label: "Reflog", Description: "Recovery and reflog history", Shortcut: "F6", Action: switchViewCmd(views.ViewReflog)})

	m := Model{
		theme:         t,
		styles:        s,
		globalKeys:    keymap.DefaultGlobalKeys(),
		header:        components.NewHeader(t),
		router:        router,
		composer:      components.NewComposer(t),
		statusBar:     statusBar,
		cmdPalette:    cmdPalette,
		navPane:       navPane,
		inspectorPane: panes.NewInspectorPane(t, s),
		dashboardView: dashboardView,
		chatView:      chatView,
		explorerView:  explorerView,
		workspaceView: workspaceView,
		settingsView:  settingsView,
		reflogView:    reflogView,
		focus:         FocusComposer,
		paletteIdx:    defaultPaletteIdx(),
		paletteName:   "default",
		cmdHandlers:   make(map[string]CommandHandler),
	}

	m.composer.SetFocused(true)
	m.chatSession = chat.NewSession("You are Gitdex, a repository operations assistant. Help operators understand repository state, execute Git operations, and manage GitHub collaboration objects. Respond in Chinese.")
	m.localIndex = gitops.NewLocalIndex(gitops.NewGitExecutor())
	m.cloneRemote = func(ctx context.Context, url, dir string, opts gitops.CloneOptions) error {
		rm := gitops.NewRemoteManager(gitops.NewGitExecutor())
		return rm.Clone(ctx, url, dir, opts)
	}
	m.tryLoadLLMProvider()
	m.tryLoadGitHubClient()
	m.tryLoadSettingsFromConfig()
	m.registerBuiltinCommands()
	m.registerRepoCommands()
	m.registerHelpUpdate()
	m.syncChrome()

	return m
}

func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{
		tea.RequestBackgroundColor,
		m.router.Init(),
	}
	if m.ghClient != nil {
		cmds = append(cmds, m.fetchRepos())
	}
	if root, err := config.ResolveRepositoryRoot(""); err == nil && root != "" {
		m.activeRepo = &repo.RepoContext{
			Name:       inferRepoName(root),
			FullName:   inferRepoName(root),
			LocalPaths: []string{root},
			IsLocal:    true,
		}
		m.explorerView.Files().SetEditable(true)
		m.explorerView.Files().SetRepository(views.RepoListItem{
			Name:       inferRepoName(root),
			FullName:   inferRepoName(root),
			LocalPaths: []string{root},
			IsLocal:    true,
		})
		m.explorerView.Commits().SetEditable(true)
		m.explorerView.Branches().SetEditable(true)
		cmds = append(cmds, m.loadFileTree(), m.loadCommitLog(), m.loadCommitGraph(), m.loadBranchTree())
	}
	if m.currentBootstrapApp().StorageProvider != nil {
		cmds = append(cmds, m.loadWorkspaceFromStores())
	}
	return tea.Batch(cmds...)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.dims = layout.Classify(msg.Width, msg.Height)
		m.resizeAll()
		m.ready = true

	case tea.BackgroundColorMsg:
		palette := theme.DefaultPalette()
		if fn, ok := theme.BuiltinPalettes[m.paletteName]; ok {
			palette = fn()
		}
		*m.theme = theme.NewTheme(msg.IsDark(), palette)
		m.styles = theme.NewStyles(*m.theme)
		m.refreshStyles()
		m.syncChrome()

	case tea.KeyPressMsg:
		if m.cmdPalette.IsVisible() {
			cmd := m.cmdPalette.Update(msg)
			return m, cmd
		}

		if m.chatView.IsStreaming() && (msg.String() == "esc" || msg.String() == "ctrl+c") {
			if m.streamCancel != nil {
				m.streamCancel()
				m.streamCancel = nil
			}
			m.chatView.EndStream()
			m.chatView.AppendMessage(views.Message{
				Role: views.RoleInfo, Content: "Interrupted.", Timestamp: time.Now(),
			})
			return m, nil
		}

		// Always-active keys (work regardless of focus area)
		switch {
		case msg.String() == "ctrl+c":
			return m, tea.Quit
		case msg.String() == "ctrl+r":
			return m, m.refreshCurrentViewCmd()
		case key.Matches(msg, m.globalKeys.CmdPalette):
			m.cmdPalette.Show()
			return m, nil
		case key.Matches(msg, m.globalKeys.CycleTheme):
			m.cycleTheme()
			return m, nil
		case key.Matches(msg, m.globalKeys.ToggleInspector):
			m.inspectorPane.Toggle()
			return m, nil
		case key.Matches(msg, m.globalKeys.SwitchDashboard):
			m.switchView(views.ViewDashboard)
			m.setFocusArea(FocusContent)
			m.statusBar.SetMode("NORMAL")
			if m.activeRepo != nil && m.ghClient != nil && m.activeRepo.Owner != "" {
				return m, m.buildRepoSummary(m.activeRepo.Owner, m.activeRepo.Name)
			}
			return m, nil
		case key.Matches(msg, m.globalKeys.SwitchChat):
			m.switchView(views.ViewChat)
			m.setFocusArea(FocusComposer)
			m.statusBar.SetMode("INSERT")
			return m, nil
		case key.Matches(msg, m.globalKeys.SwitchExplorer):
			m.switchView(views.ViewExplorer)
			m.setFocusArea(FocusContent)
			m.statusBar.SetMode("NORMAL")
			var exploreCmds []tea.Cmd
			if m.activeRepo != nil && m.activeRepo.IsLocal {
				exploreCmds = append(exploreCmds, m.loadFileTree())
			} else if m.activeRepo != nil && m.ghClient != nil && m.activeRepo.Owner != "" {
				exploreCmds = append(exploreCmds, m.loadRemoteFileTree(m.activeRepo.Owner, m.activeRepo.Name, m.activeRepo.DefaultBranch))
			}
			if m.activeRepo != nil {
				exploreCmds = append(exploreCmds,
					m.loadCommitLog(),
					m.loadCommitGraph(),
					m.loadBranchTree(),
				)
			}
			if m.activeRepo != nil && m.ghClient != nil && m.activeRepo.Owner != "" {
				exploreCmds = append(exploreCmds,
					m.fetchRepoPRs(m.activeRepo.Owner, m.activeRepo.Name),
					m.fetchRepoIssues(m.activeRepo.Owner, m.activeRepo.Name),
					m.loadWorkflowRuns(),
					m.loadDeployments(),
					m.loadReleases(),
				)
			}
			return m, tea.Batch(exploreCmds...)
		case key.Matches(msg, m.globalKeys.SwitchWorkspace):
			m.switchView(views.ViewWorkspace)
			m.setFocusArea(FocusContent)
			m.statusBar.SetMode("NORMAL")
			var wsCmd tea.Cmd
			if m.currentBootstrapApp().StorageProvider != nil {
				wsCmd = m.loadWorkspaceFromStores()
			}
			return m, wsCmd
		case key.Matches(msg, m.globalKeys.SwitchSettings):
			m.switchView(views.ViewSettings)
			m.setFocusArea(FocusContent)
			m.statusBar.SetMode("NORMAL")
			return m, nil
		case key.Matches(msg, m.globalKeys.SwitchReflog):
			m.switchView(views.ViewReflog)
			m.setFocusArea(FocusContent)
			m.statusBar.SetMode("NORMAL")
			return m, nil
		case key.Matches(msg, m.globalKeys.FocusNav):
			m.setFocusArea(FocusNav)
			m.statusBar.SetMode("NAV")
			return m, nil
		case key.Matches(msg, m.globalKeys.FocusMain):
			m.setFocusArea(FocusContent)
			m.statusBar.SetMode("NORMAL")
			return m, nil
		case key.Matches(msg, m.globalKeys.FocusInspector):
			m.setFocusArea(FocusInspector)
			m.statusBar.SetMode("INSPECT")
			return m, nil
		case msg.String() == "tab", msg.String() == "shift+tab":
			if !(m.focus == FocusContent && m.router.ActiveID() == views.ViewSettings) {
				m.toggleFocus()
				return m, nil
			}
		case msg.String() == "esc":
			if m.focus == FocusContent || m.focus == FocusInspector {
				m.setFocusArea(FocusComposer)
				m.statusBar.SetMode("INSERT")
				return m, nil
			}
		}

		if m.focus == FocusComposer && m.composer.IsEmpty() {
			switch msg.String() {
			case "up", "down", "pgup", "pgdown", "enter":
				m.setFocusArea(FocusContent)
				m.statusBar.SetMode("NORMAL")
				cmd := m.router.Update(msg)
				cmds = append(cmds, cmd)
				return m, tea.Batch(cmds...)
			}
		}

		// Keys that only work when NOT focused on content
		if m.focus != FocusContent {
			switch {
			case m.globalKeys.Quit.Matches(msg):
				return m, tea.Quit
			case m.globalKeys.Help.Matches(msg):
				m.showHelp = !m.showHelp
				return m, nil
			case m.globalKeys.Back.Matches(msg):
				if m.showHelp {
					m.showHelp = false
					return m, nil
				}
			}
		}

	case components.SubmitMsg:
		return m, m.handleSubmit(msg)

	case autonomyResultMsg:
		return m, m.handleAutonomyResult(msg)

	case views.AppendMessageMsg:
		cmd := m.router.Update(msg)
		cmds = append(cmds, cmd)
		m.header.SetTabs(m.router)
		return m, tea.Batch(cmds...)

	case views.SwitchViewMsg:
		m.switchView(msg.Target)
		return m, nil

	case panes.NavSelectMsg:
		if id := navPathToViewID(msg.Item.Path); id != "" {
			m.switchView(id)
		}
		return m, nil

	case views.StatusDataMsg:
		m.dashboardView.Update(msg)
		m.inspectorPane.UpdateRiskSummary(panes.StatusUpdateMsg{Summary: msg.Summary})
		var postCmd tea.Cmd
		if msg.Summary != nil {
			m.statusBar.SetRepoState(string(msg.Summary.OverallLabel))
			if msg.Summary.Local.Branch != "" {
				m.statusBar.SetBranch(msg.Summary.Local.Branch)
			}
			if m.currentBootstrapApp().StorageProvider != nil {
				postCmd = m.loadWorkspaceFromStores()
			} else {
				m.syncDerivedWorkspace(msg.Summary)
			}
		}
		m.syncChrome()
		return m, postCmd

	case views.WorkspaceStoresMsg:
		if msg.Err != nil {
			return m, nil
		}
		if m.workspaceView != nil {
			m.workspaceView.Plans().SetPlans(msg.Plans)
			m.workspaceView.Tasks().SetTasks(msg.Tasks)
			m.workspaceView.Evidence().SetEntries(msg.Evidence)
			m.setWorkspaceInspectorEvidence(msg.Evidence)
		}
		return m, nil

	case views.PullsDataMsg:
		m.explorerView.Update(msg)
		return m, nil

	case views.PRDetailMsg:
		m.explorerView.Update(msg)
		if msg.Err == nil && m.inspectorPane != nil {
			reviews := make([]string, 0, len(msg.Detail.Reviews))
			for _, review := range msg.Detail.Reviews {
				reviews = append(reviews, review.User+": "+review.State)
			}
			files := make([]string, 0, len(msg.Detail.Files))
			for _, file := range msg.Detail.Files {
				files = append(files, fmt.Sprintf("%s  +%d/-%d  %s", file.Filename, file.Additions, file.Deletions, file.Status))
			}
			comments := make([]string, 0, min(5, len(msg.Detail.Comments)))
			for i, comment := range msg.Detail.Comments {
				if i >= 5 {
					break
				}
				comments = append(comments, fmt.Sprintf("%s (%s): %s", comment.User, comment.Created, comment.Body))
			}
			m.inspectorPane.SetPRDetail(panes.InspectorPRData{
				Number:   msg.Detail.Number,
				Title:    msg.Detail.Title,
				State:    msg.Detail.State,
				Author:   msg.Detail.Author,
				Reviews:  strings.Join(reviews, " | "),
				Checks:   fmt.Sprintf("%d file(s), %d comment(s)", len(msg.Detail.Files), len(msg.Detail.Comments)),
				Labels:   strings.Join(msg.Detail.Labels, ", "),
				Body:     msg.Detail.Body,
				Files:    files,
				Comments: comments,
			})
		}
		return m, nil

	case views.PRActionResultMsg:
		m.explorerView.Update(msg)
		if strings.TrimSpace(msg.Message) != "" {
			role := views.RoleSystem
			if msg.Err != nil {
				role = views.RoleError
			}
			m.chatView.AppendMessage(views.Message{Role: role, Content: msg.Message, Timestamp: time.Now()})
		}
		return m, m.refreshPRAction(msg)

	case views.IssuesDataMsg:
		m.explorerView.Update(msg)
		return m, nil

	case views.IssueDetailMsg:
		m.explorerView.Update(msg)
		if msg.Err == nil && m.inspectorPane != nil {
			comments := make([]string, 0, min(5, len(msg.Detail.Comments)))
			for i, comment := range msg.Detail.Comments {
				if i >= 5 {
					break
				}
				comments = append(comments, fmt.Sprintf("%s (%s): %s", comment.User, comment.Created, comment.Body))
			}
			m.inspectorPane.SetIssueDetail(panes.InspectorIssueData{
				Number:    msg.Detail.Number,
				Title:     msg.Detail.Title,
				State:     msg.Detail.State,
				Labels:    strings.Join(msg.Detail.Labels, ", "),
				Assignees: strings.Join(msg.Detail.Assignees, ", "),
				Milestone: msg.Detail.Milestone,
				Body:      msg.Detail.Body,
				Comments:  comments,
			})
		}
		return m, nil

	case views.IssueActionResultMsg:
		m.explorerView.Update(msg)
		if strings.TrimSpace(msg.Message) != "" {
			role := views.RoleSystem
			if msg.Err != nil {
				role = views.RoleError
			}
			m.chatView.AppendMessage(views.Message{Role: role, Content: msg.Message, Timestamp: time.Now()})
		}
		return m, m.refreshIssueAction(msg)

	case views.FileTreeDataMsg:
		m.explorerView.Update(msg)
		return m, nil

	case views.FileContentMsg:
		m.explorerView.Update(msg)
		if m.inspectorPane != nil {
			preview := strings.TrimSpace(msg.Content)
			if len(preview) > 1800 {
				preview = preview[:1800] + "\n..."
			}
			mode := "remote-only"
			if m.activeRepo != nil && m.activeRepo.IsLocal {
				mode = "local writable"
			}
			size := "-"
			if msg.SizeBytes > 0 {
				size = fmt.Sprintf("%d bytes", msg.SizeBytes)
			}
			language := strings.TrimPrefix(strings.ToLower(filepath.Ext(msg.Path)), ".")
			if language == "" {
				language = "plain"
			}
			m.inspectorPane.SetFileDetail(panes.InspectorFileData{
				Path:     msg.Path,
				Size:     size,
				Language: language,
				Mode:     mode,
				Preview:  preview,
			})
		}
		return m, nil

	case views.FileDiffMsg:
		m.explorerView.Update(msg)
		return m, nil

	case views.FileEditMsg:
		_, cmd := m.explorerView.Update(msg)
		return m, cmd

	case views.FileSavedMsg:
		_, cmd := m.explorerView.Update(msg)
		return m, cmd

	case views.CommitLogDataMsg:
		m.explorerView.Update(msg)
		return m, nil

	case views.CommitGraphMsg:
		m.explorerView.Update(msg)
		return m, nil

	case views.ExplorerSubTabChangedMsg:
		m.syncInspectorForExplorerTab(msg.TabIdx)
		m.syncChrome()
		return m, nil

	case views.CommitDetailMsg:
		m.explorerView.Update(msg)
		if msg.Err == nil && m.inspectorPane != nil {
			firstLine := strings.SplitN(msg.Content, "\n", 2)[0]
			m.inspectorPane.SetCommitDetail(panes.InspectorCommitData{
				Hash:    msg.Hash,
				Message: firstLine,
				Stats:   fmt.Sprintf("%d lines", strings.Count(msg.Content, "\n")),
				Content: msg.Content,
			})
		}
		return m, nil

	case views.CommitSelectedMsg:
		if m.inspectorPane != nil {
			m.inspectorPane.SetCommitDetail(panes.InspectorCommitData{
				Hash:    msg.Commit.Hash,
				Author:  msg.Commit.Author,
				Date:    msg.Commit.Date,
				Message: msg.Commit.Message,
			})
		}
		return m, nil

	case views.CommitActionResultMsg:
		m.explorerView.Update(msg)
		if strings.TrimSpace(msg.Message) != "" {
			role := views.RoleSystem
			if msg.Err != nil {
				role = views.RoleError
			}
			m.chatView.AppendMessage(views.Message{Role: role, Content: msg.Message, Timestamp: time.Now()})
		}
		return m, m.refreshCommitAction(msg)

	case views.BranchTreeDataMsg:
		m.explorerView.Update(msg)
		return m, nil

	case views.BranchSelectedMsg:
		if m.inspectorPane != nil {
			ahead := "-"
			behind := "-"
			if strings.TrimSpace(msg.Branch.Upstream) != "" || !msg.Branch.IsRemote {
				ahead = fmt.Sprintf("%d", msg.Branch.Ahead)
				behind = fmt.Sprintf("%d", msg.Branch.Behind)
			}
			lastCommit := strings.TrimSpace(msg.Branch.LastCommit)
			if lastCommit == "" {
				lastCommit = msg.Branch.SHA
				if len(lastCommit) > 12 {
					lastCommit = lastCommit[:12]
				}
			}
			m.inspectorPane.SetBranchDetail(panes.InspectorBranchData{
				Name:       msg.Branch.Name,
				Upstream:   strings.TrimSpace(msg.Branch.Upstream),
				Ahead:      ahead,
				Behind:     behind,
				LastCommit: lastCommit,
			})
		}
		return m, nil

	case views.BranchCheckoutResultMsg:
		m.explorerView.Update(msg)
		if msg.Err == nil && msg.Name != "" {
			m.statusBar.SetBranch(msg.Name)
		}
		var cmds []tea.Cmd
		cmds = append(cmds, m.loadBranchTree())
		if m.activeRepo != nil && m.activeRepo.IsLocal && m.activeRepo.LocalPath() != "" && m.activeRepo.Owner != "" && m.activeRepo.Name != "" {
			cmds = append(cmds, m.buildRepoSummary(m.activeRepo.Owner, m.activeRepo.Name))
		}
		return m, tea.Batch(cmds...)

	case views.BranchActionResultMsg:
		m.explorerView.Update(msg)
		if strings.TrimSpace(msg.Message) != "" {
			role := views.RoleSystem
			if msg.Err != nil {
				role = views.RoleError
			}
			m.chatView.AppendMessage(views.Message{Role: role, Content: msg.Message, Timestamp: time.Now()})
		}
		return m, m.refreshBranchAction(msg)

	case views.WorkflowRunsDataMsg:
		m.explorerView.Update(msg)
		return m, nil

	case views.WorkflowSelectedMsg:
		if m.inspectorPane != nil {
			m.inspectorPane.SetWorkflowDetail(panes.InspectorWorkflowData{
				Name:       msg.Run.Name,
				RunID:      fmt.Sprintf("%d", msg.Run.RunID),
				WorkflowID: fmt.Sprintf("%d", msg.Run.WorkflowID),
				Status:     msg.Run.Status,
				Conclusion: msg.Run.Conclusion,
				Branch:     msg.Run.Branch,
				Event:      msg.Run.Event,
				CreatedAt:  msg.Run.CreatedAt,
				URL:        msg.Run.URL,
			})
		}
		return m, nil

	case views.WorkflowDispatchResultMsg:
		m.explorerView.Update(msg)
		if strings.TrimSpace(msg.Message) != "" {
			role := views.RoleSystem
			if msg.Err != nil {
				role = views.RoleError
			}
			m.chatView.AppendMessage(views.Message{Role: role, Content: msg.Message, Timestamp: time.Now()})
		}
		return m, m.refreshWorkflowDispatch(msg)

	case views.WorkflowActionResultMsg:
		m.explorerView.Update(msg)
		if strings.TrimSpace(msg.Message) != "" {
			role := views.RoleSystem
			if msg.Err != nil {
				role = views.RoleError
			}
			m.chatView.AppendMessage(views.Message{Role: role, Content: msg.Message, Timestamp: time.Now()})
		}
		return m, m.refreshWorkflowAction(msg)

	case views.DeploymentDataMsg:
		m.explorerView.Update(msg)
		return m, nil

	case views.DeploymentSelectedMsg:
		if m.inspectorPane != nil {
			m.inspectorPane.SetDeploymentDetail(panes.InspectorDeploymentData{
				ID:          fmt.Sprintf("%d", msg.Deployment.ID),
				Environment: msg.Deployment.Environment,
				State:       msg.Deployment.State,
				Ref:         msg.Deployment.Ref,
				CreatedAt:   msg.Deployment.CreatedAt,
				URL:         msg.Deployment.URL,
			})
		}
		return m, nil

	case views.ReleaseListMsg:
		m.explorerView.Update(msg)
		return m, nil

	case views.ReleaseSelectedMsg:
		if m.inspectorPane != nil {
			m.inspectorPane.SetReleaseDetail(panes.InspectorReleaseData{
				ID:          fmt.Sprintf("%d", msg.ID),
				TagName:     msg.TagName,
				Name:        msg.Name,
				Draft:       boolToYesNo(msg.Draft),
				Prerelease:  boolToYesNo(msg.Prerelease),
				CreatedAt:   msg.CreatedAt,
				PublishedAt: msg.PublishedAt,
				URL:         msg.URL,
				Body:        msg.Body,
			})
		}
		return m, nil

	case views.ReleaseOpResultMsg:
		m.explorerView.Update(msg)
		var cmd tea.Cmd
		if msg.Err == nil {
			cmd = m.loadReleases()
		}
		return m, cmd

	case views.BranchProtectionDataMsg:
		if m.inspectorPane != nil {
			m.inspectorPane.SetBranchProtection(msg.Branch, msg.Lines, msg.Err)
		}
		return m, nil

	case streamStartMsg:
		return m, m.handleStreamStart(msg)

	case streamNextMsg:
		m.chatView.Update(views.StreamChunkMsg{Content: msg.chunk.Content})
		if msg.chunk.FinishReason == "stop" {
			m.chatView.Update(views.StreamChunkMsg{Done: true})
			if n := len(m.chatView.Messages()); n > 0 {
				last := m.chatView.Messages()[n-1]
				if last.Role == views.RoleAssistant {
					m.chatSession.AddMessage("assistant", last.Content)
				}
			}
			m.streamCancel = nil
			return m, nil
		}
		return m, m.handleStreamNext(msg)

	case views.StreamChunkMsg:
		m.chatView.Update(msg)
		return m, nil

	case views.StreamErrorMsg:
		m.chatView.Update(msg)
		m.streamCancel = nil
		return m, nil

	case views.RepoListMsg:
		m.dashboardView.Update(msg)
		return m, nil

	case views.RepoSelectMsg:
		rc := &repo.RepoContext{
			Owner:         msg.Repo.Owner,
			Name:          msg.Repo.Name,
			FullName:      msg.Repo.FullName,
			LocalPaths:    msg.Repo.LocalPaths,
			IsLocal:       msg.Repo.IsLocal,
			IsReadOnly:    !msg.Repo.IsLocal,
			DefaultBranch: msg.Repo.DefaultBranch,
			IsFork:        msg.Repo.Fork,
		}
		var upstreamURL string
		var remoteLines []string
		if msg.Repo.IsLocal && msg.Repo.LocalPath() != "" {
			top, up, lines := m.gitRemoteLines(msg.Repo.LocalPath())
			rc.RemoteTopology = top
			rc.UpstreamURL = up
			upstreamURL = up
			remoteLines = lines
		}
		m.activeRepo = rc
		m.explorerView.Files().SetEditable(msg.Repo.IsLocal)
		m.explorerView.Files().SetRepository(msg.Repo)
		m.explorerView.Commits().SetEditable(msg.Repo.IsLocal)
		m.explorerView.Branches().SetEditable(msg.Repo.IsLocal)
		repoPath := ""
		if msg.Repo.LocalPath() != "" {
			repoPath = msg.Repo.LocalPath()
		}
		m.explorerView.Releases().SetRepoContext(repoPath, msg.Repo.Owner, msg.Repo.Name)
		if msg.Repo.IsLocal && msg.Repo.LocalPath() != "" {
			m.reflogView.SetRepoPath(msg.Repo.LocalPath())
		}
		m.header.SetRepo(msg.Repo.FullName)
		m.statusBar.SetRepoName(msg.Repo.FullName)
		m.statusBar.SetBranch(msg.Repo.DefaultBranch)
		m.chatView.AppendMessage(views.Message{
			Role:      views.RoleSystem,
			Content:   fmt.Sprintf("Entered repository %s", msg.Repo.FullName),
			Timestamp: time.Now(),
		})

		var fetchCmds []tea.Cmd
		if m.ghClient != nil && msg.Repo.Owner != "" {
			fetchCmds = append(fetchCmds,
				m.fetchRepoPRs(msg.Repo.Owner, msg.Repo.Name),
				m.fetchRepoIssues(msg.Repo.Owner, msg.Repo.Name),
				m.fetchRepoDetail(msg.Repo.Owner, msg.Repo.Name),
				m.buildRepoSummary(msg.Repo.Owner, msg.Repo.Name),
			)
		}
		if msg.Repo.IsLocal && msg.Repo.LocalPath() != "" {
			fetchCmds = append(fetchCmds, m.loadFileTree())
		} else if m.ghClient != nil && msg.Repo.Owner != "" {
			fetchCmds = append(fetchCmds, m.loadRemoteFileTree(msg.Repo.Owner, msg.Repo.Name, msg.Repo.DefaultBranch))
		}
		fetchCmds = append(fetchCmds, m.loadCommitLog(), m.loadCommitGraph(), m.loadBranchTree())
		if m.ghClient != nil && msg.Repo.Owner != "" {
			fetchCmds = append(fetchCmds, m.loadWorkflowRuns(), m.loadDeployments(), m.loadReleases())
		}

		m.inspectorPane.SetRepoDetail(panes.RepoDetailData{
			Name:        msg.Repo.FullName,
			Description: msg.Repo.Description,
			Language:    msg.Repo.Language,
			Stars:       msg.Repo.Stars,
			OpenPRs:     msg.Repo.OpenPRs,
			OpenIssues:  msg.Repo.OpenIssues,
			IsLocal:     msg.Repo.IsLocal,
			LocalPaths:  msg.Repo.LocalPaths,
			IsFork:      msg.Repo.Fork,
			UpstreamURL: upstreamURL,
			GitRemotes:  remoteLines,
		})
		m.inspectorPane.Show()
		m.syncChrome()

		return m, tea.Batch(fetchCmds...)

	case views.CloneRepoRequestMsg:
		targetPath := strings.TrimSpace(msg.TargetPath)
		if targetPath == "" {
			targetPath = m.defaultCloneTarget(msg.Repo)
		}
		host := m.effectiveGitHubHost()
		cloneURL := fmt.Sprintf("https://%s/%s/%s.git", host, msg.Repo.Owner, msg.Repo.Name)
		m.chatView.AppendMessage(views.Message{
			Role:      views.RoleSystem,
			Content:   fmt.Sprintf("Cloning %s into %s", msg.Repo.FullName, targetPath),
			Timestamp: time.Now(),
		})
		return m, m.cloneRepoCmd(cloneURL, targetPath, msg.Repo)

	case views.CloneCompleteMsg:
		if msg.Err != nil {
			m.chatView.AppendMessage(views.Message{
				Role:      views.RoleError,
				Content:   fmt.Sprintf("Clone failed for %s: %v", msg.Repo.FullName, msg.Err),
				Timestamp: time.Now(),
			})
			return m, nil
		}
		return m, m.finishCloneSuccess(msg.Repo, msg.TargetPath, fmt.Sprintf("Cloned %s to %s", msg.Repo.FullName, msg.TargetPath))

	case views.CloneRepoResultMsg:
		if msg.Err != nil {
			m.chatView.AppendMessage(views.Message{
				Role:      views.RoleError,
				Content:   fmt.Sprintf("Clone failed for %s: %v", msg.Repo.FullName, msg.Err),
				Timestamp: time.Now(),
			})
			return m, nil
		}
		return m, m.finishCloneSuccess(msg.Repo, msg.TargetPath, fmt.Sprintf("Cloned %s to %s", msg.Repo.FullName, msg.TargetPath))

	case views.RepoDetailMsg:
		m.inspectorPane.EnrichRepoDetail(panes.RepoDetailData{
			Description:   msg.Description,
			Stars:         msg.Stars,
			Forks:         msg.Forks,
			Language:      msg.Language,
			License:       msg.License,
			Topics:        msg.Topics,
			DefaultBranch: msg.DefaultBranch,
			IsPrivate:     msg.IsPrivate,
			CreatedAt:     msg.CreatedAt,
			HTMLURL:       msg.HTMLURL,
			OpenPRs:       msg.OpenPRs,
			OpenIssues:    msg.OpenIssues,
		})
		return m, nil

	case views.ConfigSaveMsg:
		return m, m.handleConfigSave(msg)

	case views.RequestFileContentMsg:
		if m.activeRepo != nil && m.activeRepo.IsLocal && m.activeRepo.LocalPath() != "" {
			return m, m.loadFileContent(m.activeRepo.LocalPath() + "/" + msg.Path)
		} else if m.activeRepo != nil && m.ghClient != nil && m.activeRepo.Owner != "" {
			return m, m.loadRemoteFileContent(m.activeRepo.Owner, m.activeRepo.Name, msg.Path, m.activeRepo.DefaultBranch)
		}
		return m, m.loadFileContent(msg.Path)

	case views.RequestFileDiffMsg:
		return m, m.loadFileDiff(msg.Path, msg.Cached)

	case views.RequestApplyGitPatchMsg:
		return m, m.applyGitPatch(msg)

	case views.RequestGitStageFileMsg:
		return m, m.gitStageFile(msg)

	case views.RequestFileEditMsg:
		if m.activeRepo == nil || !m.activeRepo.IsLocal || m.activeRepo.LocalPath() == "" {
			return m, func() tea.Msg {
				return views.FileSavedMsg{Path: msg.Path, Err: fmt.Errorf("editing is only available for local repositories")}
			}
		}
		target := msg.Path
		if !filepath.IsAbs(target) {
			target = filepath.Join(m.activeRepo.LocalPath(), msg.Path)
		}
		return m, m.loadFileEditContent(target)

	case views.RequestFileSaveMsg:
		return m, m.saveFileContent(msg.Path, msg.Content)

	case views.RequestFileOpMsg:
		return m, m.handleFileOp(msg)

	case views.RequestBatchFileOpMsg:
		return m, m.handleBatchFileOp(msg)

	case views.BatchFileOpResultMsg:
		_, cmd := m.explorerView.Update(msg)
		var follow []tea.Cmd
		if cmd != nil {
			follow = append(follow, cmd)
		}
		if msg.Err == nil && m.activeRepo != nil && m.activeRepo.IsLocal {
			follow = append(follow, m.loadFileTree())
		}
		if len(follow) == 0 {
			return m, nil
		}
		return m, tea.Batch(follow...)

	case views.FileOpResultMsg:
		_, cmd := m.explorerView.Update(msg)
		followUps := []tea.Cmd{cmd, m.loadFileTree()}
		if msg.Err == nil && msg.Kind == views.FileOpCreateFile && strings.TrimSpace(msg.Target) != "" {
			followUps = append(followUps, m.loadFileEditContent(filepath.Join(m.repoRoot(), filepath.FromSlash(msg.Target))))
		}
		return m, tea.Batch(followUps...)

	case views.RequestPRDetailMsg:
		return m, m.loadPRDetail(msg.Number)

	case views.RequestIssueDetailMsg:
		return m, m.loadIssueDetail(msg.Number)

	case views.RequestPRActionMsg:
		return m, m.handlePRAction(msg)

	case views.RequestIssueActionMsg:
		return m, m.handleIssueAction(msg)

	case views.RequestCommitDetailMsg:
		return m, m.loadCommitDetail(msg.Hash)

	case views.RequestCommitActionMsg:
		return m, m.handleCommitAction(msg)

	case views.RequestBranchCheckoutMsg:
		return m, m.checkoutBranch(msg.Name)

	case views.RequestBranchActionMsg:
		return m, m.handleBranchAction(msg)

	case views.RequestWorkflowActionMsg:
		return m, m.handleWorkflowAction(msg)

	case views.RequestWorkflowDispatchMsg:
		return m, m.handleWorkflowDispatch(msg)

	case views.RequestReleaseOpMsg:
		return m, m.handleReleaseOp(msg)

	case views.RequestBranchProtectionMsg:
		return m, m.loadBranchProtection(msg.Branch)
	}

	switch m.focus {
	case FocusComposer:
		cmd := m.composer.Update(msg)
		cmds = append(cmds, cmd)
	case FocusNav:
		if m.navPane != nil {
			_, cmd := m.navPane.Update(msg)
			cmds = append(cmds, cmd)
		}
	case FocusInspector:
		if m.inspectorPane != nil {
			cmd := m.inspectorPane.Update(msg)
			cmds = append(cmds, cmd)
		}
	case FocusContent:
		cmd := m.router.Update(msg)
		cmds = append(cmds, cmd)
	default:
		cmd := m.router.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() tea.View {
	if !m.ready {
		v := tea.NewView("Starting Gitdex...")
		v.AltScreen = true
		v.MouseMode = tea.MouseModeCellMotion
		return v
	}

	var content string
	if m.showHelp {
		content = m.renderHelp()
	} else {
		content = m.renderApp()
	}

	v := tea.NewView(content)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

func (m *Model) renderApp() string {
	var b strings.Builder

	m.header.SetTabs(m.router)
	b.WriteString(m.header.Render())
	b.WriteString("\n")

	navStr := ""
	if m.dims.ShowNav() {
		navStr = m.navPane.View()
	}

	mainStr := m.router.Render()

	inspStr := ""
	if m.dims.ShowInspector() && m.inspectorPane.IsVisible() {
		inspStr = m.inspectorPane.View()
	}

	columns := layout.RenderColumns(m.dims, navStr, mainStr, inspStr, m.theme.Divider())
	b.WriteString(columns)
	b.WriteString("\n")

	b.WriteString(m.composer.Render())
	b.WriteString("\n")

	b.WriteString(m.statusBar.Render())

	if m.cmdPalette.IsVisible() {
		return m.cmdPalette.Render(m.dims.Width, m.dims.Height)
	}

	return b.String()
}

func (m *Model) renderHelp() string {
	var b strings.Builder

	t := m.theme
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(t.Primary())
	b.WriteString(titleStyle.Render("  Gitdex Keyboard Shortcuts"))
	b.WriteString("\n\n")

	sections := []struct {
		title string
		items []struct{ key, desc string }
	}{
		{
			title: "Navigation",
			items: []struct{ key, desc string }{
				{"F1", "Dashboard (Overview + Health)"},
				{"F2", "Chat (AI conversation)"},
				{"F3", "Explorer (PRs + Issues + Files + Commits + Branches + Workflows + Deployments)"},
				{"F4", "Workspace (Plans + Tasks + Evidence + Cruise + Approvals)"},
				{"F5", "Settings (Configuration)"},
				{"1-7", "Switch sub-tab within current view"},
				{"Ctrl+P", "Command palette"},
				{"Ctrl+T", "Cycle theme"},
				{"Ctrl+I", "Toggle inspector"},
			},
		},
		{
			title: "Focus",
			items: []struct{ key, desc string }{
				{"Tab", "Cycle focus: Input -> Content -> Inspector"},
				{"Ctrl+2", "Focus main content (navigate with arrows)"},
				{"Ctrl+3", "Focus inspector"},
			},
		},
		{
			title: "Content Navigation",
			items: []struct{ key, desc string }{
				{"Arrows or k/j", "Navigate items"},
				{"Enter", "Open detail / expand"},
				{"Esc", "Close detail / go back"},
				{"d", "View diff (in Files)"},
				{"g/G", "Jump to top / bottom"},
				{"PgUp/PgDn", "Scroll content"},
			},
		},
		{
			title: "Input",
			items: []struct{ key, desc string }{
				{"Enter", "Submit command or message"},
				{"Up/Down", "Browse input history"},
				{"Ctrl+A", "Move to start of line"},
				{"Ctrl+E", "Move to end of line"},
				{"Ctrl+U", "Clear to start"},
				{"Ctrl+K", "Clear to end"},
			},
		},
		{
			title: "Commands",
			items: []struct{ key, desc string }{
				{"/help", "List available commands"},
				{"/dashboard", "Switch to Dashboard"},
				{"/chat", "Switch to Chat"},
				{"/explorer", "Switch to Explorer (PRs/Issues/Files)"},
				{"/workspace", "Switch to Workspace (Plans/Tasks/Evidence/Cruise/Approvals)"},
				{"/settings", "Open settings"},
				{"/clear", "Clear chat history"},
				{"/theme [name]", "List or switch theme"},
				{"/quit", "Exit Gitdex"},
			},
		},
	}

	sectionStyle := lipgloss.NewStyle().Bold(true).Underline(true)
	keyStyle := lipgloss.NewStyle().Foreground(t.FocusBorderColor()).Bold(true).Width(16)
	descStyle := lipgloss.NewStyle().Foreground(t.Fg())

	for _, section := range sections {
		b.WriteString("  ")
		b.WriteString(sectionStyle.Render(section.title))
		b.WriteString("\n")
		for _, item := range section.items {
			b.WriteString("  ")
			b.WriteString(keyStyle.Render(item.key))
			b.WriteString(descStyle.Render(item.desc))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	b.WriteString(lipgloss.NewStyle().Foreground(t.DimText()).Italic(true).Render("  Press ? or Esc to close"))

	return b.String()
}

func (m *Model) switchView(id views.ID) {
	m.router.SwitchTo(id)
	m.header.SetTabs(m.router)
	m.syncChrome()
}

func (m *Model) switchViewWithLoad(id views.ID) tea.Cmd {
	m.switchView(id)
	switch id {
	case views.ViewExplorer:
		return m.loadFileTree()
	default:
		return nil
	}
}

func (m *Model) cycleTheme() {
	names := theme.PaletteNames()
	if len(names) == 0 {
		return
	}
	m.paletteIdx = (m.paletteIdx + 1) % len(names)
	m.paletteName = names[m.paletteIdx]
	paletteFn := theme.BuiltinPalettes[m.paletteName]
	palette := paletteFn()
	*m.theme = theme.NewTheme(m.theme.IsDark, palette)
	m.styles = theme.NewStyles(*m.theme)
	m.refreshStyles()
	m.statusBar.SetThemeName(m.paletteName)
	m.syncChrome()
}

func (m *Model) refreshStyles() {
	if m.navPane != nil {
		m.navPane.SetStyles(m.styles)
	}
	if m.inspectorPane != nil {
		m.inspectorPane.SetStyles(m.styles)
	}
}

func defaultPaletteIdx() int {
	for i, n := range theme.PaletteNames() {
		if n == "default" {
			return i
		}
	}
	return 0
}

func navPathToViewID(path string) views.ID {
	switch path {
	case "dashboard":
		return views.ViewDashboard
	case "chat":
		return views.ViewChat
	case "explorer":
		return views.ViewExplorer
	case "workspace":
		return views.ViewWorkspace
	case "settings":
		return views.ViewSettings
	case "reflog":
		return views.ViewReflog
	default:
		return ""
	}
}

func (m *Model) setFocusArea(area FocusArea) {
	m.focus = area
	m.composer.SetFocused(area == FocusComposer)
	if m.navPane != nil {
		m.navPane.SetFocused(area == FocusNav)
	}
	if m.inspectorPane != nil {
		m.inspectorPane.SetFocused(area == FocusInspector)
	}
	m.syncChrome()
}

func (m *Model) toggleFocus() {
	switch m.focus {
	case FocusComposer:
		m.setFocusArea(FocusContent)
		m.statusBar.SetMode("NORMAL")
	case FocusContent:
		if m.dims.ShowInspector() && m.inspectorPane != nil && m.inspectorPane.IsVisible() {
			m.setFocusArea(FocusInspector)
			m.statusBar.SetMode("INSPECT")
		} else {
			m.setFocusArea(FocusComposer)
			m.statusBar.SetMode("INSERT")
		}
	case FocusInspector:
		m.setFocusArea(FocusComposer)
		m.statusBar.SetMode("INSERT")
	case FocusNav:
		m.setFocusArea(FocusComposer)
		m.statusBar.SetMode("INSERT")
	default:
		m.setFocusArea(FocusComposer)
		m.statusBar.SetMode("INSERT")
	}
}

func (m *Model) resizeAll() {
	m.header.SetWidth(m.dims.Width)
	m.composer.SetWidth(m.dims.Width)
	m.statusBar.SetWidth(m.dims.Width)
	m.cmdPalette.SetSize(m.dims.Width, m.dims.Height)

	contentHeight := m.dims.ContentHeight()
	mainWidth := m.dims.MainWidth()

	m.router.SetSize(mainWidth, contentHeight)

	if m.dims.ShowNav() && m.navPane != nil {
		m.navPane.SetSize(m.dims.NavWidth(), contentHeight)
	}
	if m.dims.ShowInspector() && m.inspectorPane != nil {
		m.inspectorPane.SetSize(m.dims.InspectorWidth(), contentHeight)
	}
}

func (m *Model) handleSubmit(msg components.SubmitMsg) tea.Cmd {
	userMsg := views.Message{
		Role:      views.RoleUser,
		Content:   msg.Input,
		Timestamp: time.Now(),
	}
	m.chatView.AppendMessage(userMsg)

	if m.router.ActiveID() != views.ViewChat {
		m.switchView(views.ViewChat)
	}

	if msg.IsCommand {
		return m.executeCommand(msg.Input)
	}

	if msg.IsIntent {
		return m.executeIntent(msg.Input)
	}

	if m.chatView.IsStreaming() {
		m.chatView.AppendMessage(views.Message{
			Role:      views.RoleInfo,
			Content:   "Waiting for the previous response to finish.",
			Timestamp: time.Now(),
		})
		return nil
	}

	if m.llmProvider == nil {
		m.chatView.AppendMessage(views.Message{
			Role:      views.RoleError,
			Content:   "LLM is not configured. Open Settings (F5) and set provider, model, and credentials.",
			Timestamp: time.Now(),
		})
		return nil
	}

	m.chatSession.AddMessage("user", msg.Input)
	m.chatView.BeginStream()

	if m.streamCancel != nil {
		m.streamCancel()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	m.streamCancel = cancel

	messages := m.chatSession.GetContext()
	req := adapter.ChatRequest{
		Messages: messages,
		Stream:   true,
	}

	return func() tea.Msg {
		ch, err := m.llmProvider.StreamChatCompletion(ctx, req)
		if err != nil {
			cancel()
			return views.StreamErrorMsg{Error: fmt.Errorf("LLM stream failed: %w", err)}
		}
		return streamStartMsg{ch: ch, ctx: ctx}
	}
}

type autonomyResultMsg struct {
	result autonomyexec.Result
	err    error
}

func (m *Model) executeIntent(input string) tea.Cmd {
	intent := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(input), "!"))
	if intent == "" {
		m.chatView.AppendMessage(views.Message{
			Role:      views.RoleInfo,
			Content:   "Usage: !<intent>",
			Timestamp: time.Now(),
		})
		return nil
	}

	if m.chatView.IsStreaming() {
		m.chatView.AppendMessage(views.Message{
			Role:      views.RoleInfo,
			Content:   "Waiting for the previous response to finish.",
			Timestamp: time.Now(),
		})
		return nil
	}

	app := m.currentBootstrapApp()
	owner, repoName := m.activeGitHubCoordinates()
	repoRoot := ""
	if m.activeRepo != nil && m.activeRepo.IsLocal && m.activeRepo.LocalPath() != "" {
		repoRoot = m.activeRepo.LocalPath()
	}
	repoRoot = autonomyexec.SelectRepoRootForRemote(app, repoRoot, owner, repoName)

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		result, err := autonomyexec.Run(ctx, app, autonomyexec.Request{
			RepoRoot:          repoRoot,
			Owner:             owner,
			Repo:              repoName,
			Intent:            intent,
			Execute:           true,
			AutoThreshold:     autonomy.RiskHigh,
			ApprovalThreshold: autonomy.RiskCritical,
		})
		return autonomyResultMsg{result: result, err: err}
	}
}

func (m *Model) handleAutonomyResult(msg autonomyResultMsg) tea.Cmd {
	if msg.err != nil {
		m.chatView.AppendMessage(views.Message{
			Role:      views.RoleError,
			Content:   fmt.Sprintf("Execute failed: %v", msg.err),
			Timestamp: time.Now(),
		})
		return nil
	}

	var buf bytes.Buffer
	if err := autonomyexec.RenderResult(&buf, msg.result); err != nil {
		m.chatView.AppendMessage(views.Message{
			Role:      views.RoleError,
			Content:   fmt.Sprintf("Render failed: %v", err),
			Timestamp: time.Now(),
		})
		return nil
	}

	content := strings.TrimSpace(buf.String())
	m.chatView.AppendMessage(views.Message{
		Role:      views.RoleAssistant,
		Content:   content,
		Timestamp: time.Now(),
	})
	m.chatSession.AddMessage("assistant", content)

	if msg.result.Owner != "" && msg.result.Repo != "" {
		fullName := msg.result.Owner + "/" + msg.result.Repo
		if m.activeRepo == nil {
			m.activeRepo = &repo.RepoContext{
				Owner:      msg.result.Owner,
				Name:       msg.result.Repo,
				FullName:   fullName,
				IsLocal:    msg.result.RepoRoot != "",
				IsReadOnly: msg.result.RepoRoot == "",
			}
		}
		if m.activeRepo.FullName == "" {
			m.activeRepo.FullName = fullName
		}
		if m.activeRepo.Owner == "" {
			m.activeRepo.Owner = msg.result.Owner
		}
		if m.activeRepo.Name == "" {
			m.activeRepo.Name = msg.result.Repo
		}
		m.header.SetRepo(fullName)
		m.statusBar.SetRepoName(fullName)
	}

	if strings.TrimSpace(msg.result.RepoRoot) != "" {
		if m.activeRepo == nil {
			m.activeRepo = &repo.RepoContext{
				LocalPaths: []string{msg.result.RepoRoot},
				IsLocal:    true,
			}
		} else {
			m.activeRepo.LocalPaths = []string{msg.result.RepoRoot}
			m.activeRepo.IsLocal = true
			m.activeRepo.IsReadOnly = false
		}
		m.explorerView.Files().SetEditable(true)
	}

	return m.refreshAfterAutonomy(msg.result)
}

type streamStartMsg struct {
	ch  <-chan adapter.ChatResponse
	ctx context.Context
}

func (m *Model) handleStreamStart(msg streamStartMsg) tea.Cmd {
	return func() tea.Msg {
		select {
		case <-msg.ctx.Done():
			return views.StreamErrorMsg{Error: fmt.Errorf("request timed out or was canceled")}
		case resp, ok := <-msg.ch:
			if !ok {
				return views.StreamChunkMsg{Done: true}
			}
			return streamNextMsg{
				chunk: resp,
				ch:    msg.ch,
				ctx:   msg.ctx,
			}
		}
	}
}

type streamNextMsg struct {
	chunk adapter.ChatResponse
	ch    <-chan adapter.ChatResponse
	ctx   context.Context
}

func (m *Model) handleStreamNext(msg streamNextMsg) tea.Cmd {
	return func() tea.Msg {
		select {
		case <-msg.ctx.Done():
			return views.StreamErrorMsg{Error: fmt.Errorf("request timed out or was canceled")}
		case resp, ok := <-msg.ch:
			if !ok {
				return views.StreamChunkMsg{Done: true}
			}
			return streamNextMsg{
				chunk: resp,
				ch:    msg.ch,
				ctx:   msg.ctx,
			}
		}
	}
}

func (m *Model) tryLoadLLMProvider() {
	cfg, err := config.Load(config.Options{})
	if err != nil {
		m.llmProvider = nil
		return
	}
	providerName := strings.TrimSpace(cfg.LLM.Provider)
	if providerName == "" {
		m.llmProvider = nil
		return
	}
	if strings.ToLower(providerName) != "ollama" && strings.TrimSpace(cfg.LLM.APIKey) == "" {
		m.llmProvider = nil
		return
	}
	provider, err := adapter.NewProviderFromConfig(cfg.LLM.Provider, cfg.LLM.Model, cfg.LLM.APIKey, cfg.LLM.Endpoint)
	if err != nil {
		m.llmProvider = nil
		return
	}
	m.llmProvider = provider
}

func (m *Model) tryLoadGitHubClient() {
	cfg, err := config.Load(config.Options{})
	if err != nil {
		m.ghClient = nil
		return
	}
	tr, err := identity.ResolveTransport(cfg.Identity, nil)
	if err != nil {
		m.ghClient = nil
		return
	}
	httpClient := &http.Client{Transport: tr.Transport}
	if tr.Host != "" && tr.Host != "github.com" {
		baseURL := fmt.Sprintf("https://%s/api/v3", tr.Host)
		c, err := ghclient.NewClientWithBaseURL(httpClient, baseURL)
		if err != nil {
			return
		}
		m.ghClient = c
	} else {
		m.ghClient = ghclient.NewClient(httpClient)
	}
}

func (m *Model) tryLoadSettingsFromConfig() {
	cfg, err := config.Load(config.Options{})
	if err != nil {
		return
	}
	m.settingsView.LoadRuntimeConfig(cfg)
	m.syncChrome()
}

func (m *Model) refreshCurrentViewCmd() tea.Cmd {
	switch m.router.ActiveID() {
	case views.ViewDashboard:
		var cmds []tea.Cmd
		if m.ghClient != nil {
			cmds = append(cmds, m.fetchRepos())
		}
		if m.activeRepo != nil && m.ghClient != nil && m.activeRepo.Owner != "" {
			cmds = append(cmds, m.buildRepoSummary(m.activeRepo.Owner, m.activeRepo.Name))
		}
		return tea.Batch(cmds...)
	case views.ViewExplorer:
		return tea.Batch(m.cmdsForExplorerRefresh()...)
	case views.ViewWorkspace:
		var cmds []tea.Cmd
		if m.activeRepo != nil && m.ghClient != nil && m.activeRepo.Owner != "" {
			cmds = append(cmds, m.buildRepoSummary(m.activeRepo.Owner, m.activeRepo.Name))
		}
		if m.currentBootstrapApp().StorageProvider != nil {
			cmds = append(cmds, m.loadWorkspaceFromStores())
		}
		return tea.Batch(cmds...)
	case views.ViewChat:
		return nil
	case views.ViewSettings:
		m.tryLoadSettingsFromConfig()
		return nil
	case views.ViewReflog:
		if m.activeRepo != nil && strings.TrimSpace(m.activeRepo.LocalPath()) != "" {
			return views.LoadReflogCmd(m.activeRepo.LocalPath())
		}
		return nil
	default:
		return nil
	}
}

func (m *Model) cmdsForExplorerRefresh() []tea.Cmd {
	var exploreCmds []tea.Cmd
	if m.activeRepo != nil && m.activeRepo.IsLocal {
		exploreCmds = append(exploreCmds, m.loadFileTree())
	} else if m.activeRepo != nil && m.ghClient != nil && m.activeRepo.Owner != "" {
		exploreCmds = append(exploreCmds, m.loadRemoteFileTree(m.activeRepo.Owner, m.activeRepo.Name, m.activeRepo.DefaultBranch))
	}
	if m.activeRepo != nil {
		exploreCmds = append(exploreCmds,
			m.loadCommitLog(),
			m.loadCommitGraph(),
			m.loadBranchTree(),
		)
	}
	path, wantDiff := m.explorerView.Files().ReloadPathForRefresh()
	if path != "" && m.activeRepo != nil {
		if wantDiff {
			exploreCmds = append(exploreCmds, m.loadFileDiff(path))
		} else if m.activeRepo.IsLocal && m.activeRepo.LocalPath() != "" {
			abs := path
			if !filepath.IsAbs(path) {
				abs = filepath.Join(m.activeRepo.LocalPath(), filepath.FromSlash(path))
			}
			exploreCmds = append(exploreCmds, m.loadFileContent(abs))
		} else if m.ghClient != nil && m.activeRepo.Owner != "" {
			rel := path
			if filepath.IsAbs(path) {
				rel = strings.TrimPrefix(path, m.activeRepo.LocalPath())
				rel = strings.TrimLeft(rel, `/\`)
			}
			rel = filepath.ToSlash(rel)
			exploreCmds = append(exploreCmds, m.loadRemoteFileContent(m.activeRepo.Owner, m.activeRepo.Name, rel, m.activeRepo.DefaultBranch))
		}
	}
	if m.activeRepo != nil && m.ghClient != nil && m.activeRepo.Owner != "" {
		exploreCmds = append(exploreCmds,
			m.fetchRepoPRs(m.activeRepo.Owner, m.activeRepo.Name),
			m.fetchRepoIssues(m.activeRepo.Owner, m.activeRepo.Name),
			m.loadWorkflowRuns(),
			m.loadDeployments(),
			m.loadReleases(),
		)
	}
	return exploreCmds
}

func (m *Model) fetchRepos() tea.Cmd {
	client := m.ghClient
	if client == nil {
		return nil
	}
	idx := m.localIndex
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		if idx != nil {
			workspaceRoots := []string(nil)
			if cfg, err := config.Load(config.Options{}); err == nil {
				workspaceRoots = append(workspaceRoots, cfg.Git.WorkspaceRoots...)
			}
			idx.BuildWithRoots(ctx, workspaceRoots)
		}

		repos, err := client.ListUserRepositories(ctx)
		if err != nil {
			return views.RepoListMsg{Repos: nil}
		}

		items := make([]views.RepoListItem, 0, len(repos))
		for _, r := range repos {
			item := views.RepoListItem{
				Owner:         r.GetOwner().GetLogin(),
				Name:          r.GetName(),
				FullName:      r.GetFullName(),
				Description:   r.GetDescription(),
				Language:      r.GetLanguage(),
				Stars:         r.GetStargazersCount(),
				Fork:          r.GetFork(),
				DefaultBranch: r.GetDefaultBranch(),
				OpenIssues:    r.GetOpenIssuesCount(),
			}
			if r.GetUpdatedAt().Time.IsZero() {
				item.UpdatedAt = ""
			} else {
				item.UpdatedAt = r.GetUpdatedAt().Time.Format("2006-01-02")
			}

			if idx != nil {
				paths := idx.LookupByOwnerName(r.GetOwner().GetLogin(), r.GetName())
				if len(paths) > 0 {
					item.LocalPaths = paths
					item.IsLocal = true
				}
			}

			items = append(items, item)
		}

		cwdRoot, _ := config.ResolveRepositoryRoot("")
		if cwdRoot != "" {
			cwdRoot = strings.ReplaceAll(cwdRoot, "\\", "/")
			matched := false
			for i := range items {
				for _, lp := range items[i].LocalPaths {
					if strings.EqualFold(lp, cwdRoot) {
						matched = true
						break
					}
				}
				if matched {
					break
				}
			}
			if !matched {
				cwdName := inferRepoName(cwdRoot)
				items = append([]views.RepoListItem{{
					Name:       cwdName,
					FullName:   cwdName,
					LocalPaths: []string{cwdRoot},
					IsLocal:    true,
					UpdatedAt:  time.Now().Format("2006-01-02"),
				}}, items...)
			}
		}

		cwd, _ := os.Getwd()
		if cwd != "" {
			cwdNorm := strings.ReplaceAll(cwd, "\\", "/")
			cwdDirName := strings.ToLower(inferRepoName(cwdNorm))
			for i := range items {
				if items[i].IsLocal {
					continue
				}
				if strings.EqualFold(items[i].Name, cwdDirName) {
					items[i].LocalPaths = append(items[i].LocalPaths, cwdNorm)
					items[i].IsLocal = true
					break
				}
			}
		}

		return views.RepoListMsg{Repos: items}
	}
}

func (m *Model) fetchRepoPRs(owner, name string) tea.Cmd {
	client := m.ghClient
	if client == nil {
		return nil
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		prs, err := client.ListOpenPullRequests(ctx, owner, name)
		if err != nil {
			return views.PullsDataMsg{Items: nil}
		}
		return views.PullsDataMsg{Items: prs}
	}
}

func (m *Model) fetchRepoIssues(owner, name string) tea.Cmd {
	client := m.ghClient
	if client == nil {
		return nil
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		issues, err := client.ListOpenIssues(ctx, owner, name)
		if err != nil {
			return views.IssuesDataMsg{Items: nil}
		}
		return views.IssuesDataMsg{Items: issues}
	}
}

func (m *Model) fetchRepoDetail(owner, name string) tea.Cmd {
	client := m.ghClient
	if client == nil {
		return nil
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		detail, err := client.GetRepositoryDetail(ctx, owner, name)
		if err != nil {
			return views.RepoDetailMsg{}
		}
		return views.RepoDetailMsg{
			Description:   detail.Description,
			Stars:         detail.Stars,
			Forks:         detail.Forks,
			Language:      detail.Language,
			License:       detail.License,
			Topics:        detail.Topics,
			DefaultBranch: detail.DefaultBranch,
			IsPrivate:     detail.IsPrivate,
			CreatedAt:     detail.CreatedAt,
			HTMLURL:       detail.HTMLURL,
			OpenPRs:       0,
			OpenIssues:    detail.OpenIssues,
		}
	}
}

func (m *Model) buildRepoSummary(owner, name string) tea.Cmd {
	client := m.ghClient
	activeRepo := m.activeRepo
	return func() tea.Msg {
		summary := &repo.RepoSummary{
			Owner:        owner,
			Repo:         name,
			OverallLabel: repo.Unknown,
			Timestamp:    time.Now(),
		}

		if activeRepo != nil && activeRepo.IsLocal {
			summary.Local.Label = repo.Healthy
			summary.Local.Branch = activeRepo.DefaultBranch
			summary.Local.Detail = "Local clone available"
		} else {
			summary.Local.Label = repo.Unknown
			summary.Local.Detail = "No local clone"
		}

		if client != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()

			remote, err := client.GetRepository(ctx, owner, name)
			if err == nil && remote != nil {
				summary.Remote = *remote
				summary.Remote.Label = repo.Healthy
				summary.Remote.Detail = "Repository accessible"
			} else {
				summary.Remote.Label = repo.Degraded
				summary.Remote.Detail = "Failed to fetch remote info"
			}

			prs, err := client.ListOpenPullRequests(ctx, owner, name)
			if err == nil {
				summary.Collaboration.PullRequests = prs
				summary.Collaboration.OpenPRCount = len(prs)
				stalePRs := 0
				for _, pr := range prs {
					if pr.StaleDays > 14 {
						stalePRs++
					}
				}
				if stalePRs > 0 {
					summary.Collaboration.Label = repo.Drifting
					summary.Collaboration.Detail = fmt.Sprintf("%d stale PRs (>14 days)", stalePRs)
				} else {
					summary.Collaboration.Label = repo.Healthy
					summary.Collaboration.Detail = fmt.Sprintf("%d open PRs", len(prs))
				}
			}

			issues, err := client.ListOpenIssues(ctx, owner, name)
			if err == nil {
				summary.Collaboration.OpenIssueCount = len(issues)
				if summary.Collaboration.Detail != "" {
					summary.Collaboration.Detail += fmt.Sprintf(", %d open issues", len(issues))
				}
			}

			runs, err := client.ListWorkflowRuns(ctx, owner, name)
			if err == nil {
				summary.Workflows.Runs = runs
				if len(runs) == 0 {
					summary.Workflows.Label = repo.Unknown
					summary.Workflows.Detail = "No workflow runs found"
				} else {
					allPass := true
					for _, r := range runs {
						if r.Conclusion != "success" && r.Status != "completed" {
							allPass = false
							break
						}
					}
					if allPass {
						summary.Workflows.Label = repo.Healthy
						summary.Workflows.Detail = fmt.Sprintf("%d runs, all passing", len(runs))
					} else {
						summary.Workflows.Label = repo.Degraded
						summary.Workflows.Detail = fmt.Sprintf("%d runs, some failing", len(runs))
					}
				}
			} else {
				summary.Workflows.Label = repo.Unknown
				summary.Workflows.Detail = "Could not fetch workflow status"
			}

			deploys, err := client.ListDeployments(ctx, owner, name)
			if err == nil {
				summary.Deployments.Deployments = deploys
				if len(deploys) == 0 {
					summary.Deployments.Label = repo.Unknown
					summary.Deployments.Detail = "No deployments"
				} else {
					hasFailure := false
					for _, d := range deploys {
						if d.State == "failure" || d.State == "error" {
							hasFailure = true
							break
						}
					}
					if hasFailure {
						summary.Deployments.Label = repo.Degraded
					} else {
						summary.Deployments.Label = repo.Healthy
					}
					summary.Deployments.Detail = fmt.Sprintf("%d deployments", len(deploys))
				}
			} else {
				summary.Deployments.Label = repo.Unknown
				summary.Deployments.Detail = "Could not fetch deployments"
			}

			summary.OverallLabel = repo.WorstLabel(
				summary.Local.Label,
				summary.Remote.Label,
				summary.Collaboration.Label,
				summary.Workflows.Label,
				summary.Deployments.Label,
			)
		}

		return views.StatusDataMsg{Summary: summary}
	}
}

func inferRepoName(repoPath string) string {
	parts := strings.Split(strings.ReplaceAll(repoPath, "\\", "/"), "/")
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] != "" {
			return parts[i]
		}
	}
	return "unknown"
}

func (m *Model) queuePostCommand(cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	if m.postCommand == nil {
		m.postCommand = cmd
		return
	}
	m.postCommand = tea.Batch(m.postCommand, cmd)
}

func (m *Model) drainPostCommand() tea.Cmd {
	cmd := m.postCommand
	m.postCommand = nil
	return cmd
}

func (m *Model) effectiveGitHubHost() string {
	cfg, err := config.Load(config.Options{})
	if err == nil {
		if tr, err := identity.ResolveTransport(cfg.Identity, nil); err == nil && strings.TrimSpace(tr.Host) != "" {
			return strings.TrimSpace(tr.Host)
		}
		if host := strings.TrimSpace(cfg.Identity.GitHubApp.Host); host != "" {
			host = strings.TrimPrefix(host, "https://")
			host = strings.TrimPrefix(host, "http://")
			if idx := strings.Index(host, "/"); idx >= 0 {
				host = host[:idx]
			}
			if host != "" {
				return host
			}
		}
	}
	return "github.com"
}

func (m *Model) defaultCloneTarget(repoItem views.RepoListItem) string {
	if cfg, err := config.Load(config.Options{}); err == nil {
		if len(cfg.Git.WorkspaceRoots) > 0 && strings.TrimSpace(cfg.Git.WorkspaceRoots[0]) != "" {
			root := cfg.Git.WorkspaceRoots[0]
			if repoItem.Owner != "" {
				return filepath.Join(root, repoItem.Owner, repoItem.Name)
			}
			return filepath.Join(root, repoItem.Name)
		}
	}
	if cwd, err := os.Getwd(); err == nil && cwd != "" {
		if repoItem.Owner != "" {
			return filepath.Join(cwd, repoItem.Owner, repoItem.Name)
		}
		return filepath.Join(cwd, repoItem.Name)
	}
	if repoItem.Owner != "" {
		return filepath.Join(repoItem.Owner, repoItem.Name)
	}
	return repoItem.Name
}

// gitRemoteLines reads git remotes from a local clone for inspector + RepoContext.
func (m *Model) gitRemoteLines(repoRoot string) (map[string]string, string, []string) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	rm := gitops.NewRemoteManager(gitops.NewGitExecutor())
	remotes, err := rm.ListRemotes(ctx, repoRoot)
	if err != nil || len(remotes) == 0 {
		return nil, "", nil
	}
	top := make(map[string]string)
	var upstream string
	var lines []string
	for _, r := range remotes {
		url := strings.TrimSpace(r.FetchURL)
		if url == "" {
			url = strings.TrimSpace(r.PushURL)
		}
		top[r.Name] = url
		lines = append(lines, fmt.Sprintf("%s → %s", r.Name, url))
		if strings.EqualFold(r.Name, "upstream") {
			upstream = url
		}
	}
	sort.Strings(lines)
	return top, upstream, lines
}

func (m *Model) cloneRepoCmd(repoURL, targetPath string, repoItem views.RepoListItem) tea.Cmd {
	targetPath = strings.TrimSpace(targetPath)
	if targetPath == "" {
		targetPath = m.defaultCloneTarget(repoItem)
	}
	branch := strings.TrimSpace(repoItem.DefaultBranch)
	opts := gitops.CloneOptions{}
	if branch != "" {
		opts.Branch = branch
		opts.SingleBranch = true
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return views.CloneCompleteMsg{Repo: repoItem, TargetPath: targetPath, URL: repoURL, Err: err}
		}
		err := m.cloneRemote(ctx, repoURL, targetPath, opts)
		return views.CloneCompleteMsg{Repo: repoItem, TargetPath: targetPath, URL: repoURL, Err: err}
	}
}

func (m *Model) finishCloneSuccess(cloned views.RepoListItem, targetPath, chatLine string) tea.Cmd {
	cloned.LocalPaths = []string{targetPath}
	cloned.IsLocal = true
	top, upstream, lines := m.gitRemoteLines(targetPath)
	m.activeRepo = &repo.RepoContext{
		Owner:          cloned.Owner,
		Name:           cloned.Name,
		FullName:       cloned.FullName,
		LocalPaths:     cloned.LocalPaths,
		IsLocal:        true,
		IsReadOnly:     false,
		DefaultBranch:  cloned.DefaultBranch,
		IsFork:         cloned.Fork,
		RemoteTopology: top,
		UpstreamURL:    upstream,
	}
	m.header.SetRepo(cloned.FullName)
	m.explorerView.Files().SetEditable(true)
	m.explorerView.Files().SetRepository(cloned)
	m.explorerView.Commits().SetEditable(true)
	m.explorerView.Branches().SetEditable(true)
	m.statusBar.SetRepoName(cloned.FullName)
	m.statusBar.SetBranch(cloned.DefaultBranch)
	m.inspectorPane.EnrichRepoDetail(panes.RepoDetailData{
		Name:        cloned.FullName,
		Language:    cloned.Language,
		Stars:       cloned.Stars,
		OpenPRs:     cloned.OpenPRs,
		OpenIssues:  cloned.OpenIssues,
		IsLocal:     true,
		LocalPaths:  cloned.LocalPaths,
		IsFork:      cloned.Fork,
		UpstreamURL: upstream,
		GitRemotes:  lines,
	})
	m.inspectorPane.Show()
	m.syncChrome()
	m.chatView.AppendMessage(views.Message{
		Role:      views.RoleSystem,
		Content:   chatLine,
		Timestamp: time.Now(),
	})
	return tea.Batch(
		m.loadFileTree(),
		m.loadCommitLog(),
		m.loadCommitGraph(),
		m.loadBranchTree(),
		m.loadWorkflowRuns(),
		m.loadDeployments(),
		m.loadReleases(),
		m.fetchRepoPRs(cloned.Owner, cloned.Name),
		m.fetchRepoIssues(cloned.Owner, cloned.Name),
		m.fetchRepoDetail(cloned.Owner, cloned.Name),
		m.buildRepoSummary(cloned.Owner, cloned.Name),
		m.fetchRepos(),
	)
}

func (m *Model) cloneRepo(repoItem views.RepoListItem, targetPath string) tea.Cmd {
	targetPath = strings.TrimSpace(targetPath)
	if targetPath == "" {
		targetPath = m.defaultCloneTarget(repoItem)
	}
	host := m.effectiveGitHubHost()
	cloneURL := fmt.Sprintf("https://%s/%s/%s.git", host, repoItem.Owner, repoItem.Name)
	branch := strings.TrimSpace(repoItem.DefaultBranch)
	opts := gitops.CloneOptions{}
	if branch != "" {
		opts.Branch = branch
		opts.SingleBranch = true
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return views.CloneRepoResultMsg{Repo: repoItem, TargetPath: targetPath, Err: err}
		}
		err := m.cloneRemote(ctx, cloneURL, targetPath, opts)
		return views.CloneRepoResultMsg{
			Repo:       repoItem,
			TargetPath: targetPath,
			Err:        err,
		}
	}
}

func (m *Model) executeCommand(input string) tea.Cmd {
	// Rebind command closures against the current model snapshot. New() returns the model by
	// value, so handlers registered during construction can otherwise capture stale state.
	m.registerBuiltinCommands()
	m.registerRepoCommands()
	m.registerHelpUpdate()

	parts := strings.SplitN(strings.TrimPrefix(input, "/"), " ", 2)
	cmd := parts[0]
	args := ""
	if len(parts) > 1 {
		args = parts[1]
	}

	if handler, ok := m.cmdHandlers[cmd]; ok {
		result := handler(args)
		postCmd := m.drainPostCommand()
		m.chatView.AppendMessage(views.Message{
			Role:      views.RoleSystem,
			Content:   result,
			Timestamp: time.Now(),
		})

		var refreshCmds []tea.Cmd
		switch cmd {
		case "explorer":
			refreshCmds = append(refreshCmds, m.loadFileTree())
		case "new", "edit", "rm", "find", "search", "chmod", "symlink", "archive", "mkdir", "mv", "cp":
			if m.activeRepo != nil && m.activeRepo.IsLocal {
				refreshCmds = append(refreshCmds, m.loadFileTree())
			}
		case "pr", "issue", "release", "actions":
			if owner, name := m.activeRepoCoordinates(); owner != "" && name != "" && m.ghClient != nil {
				refreshCmds = append(refreshCmds,
					m.fetchRepoPRs(owner, name),
					m.fetchRepoIssues(owner, name),
					m.fetchRepoDetail(owner, name),
					m.buildRepoSummary(owner, name),
				)
			}
		case "status", "add", "reset", "restore", "commit", "amend", "branch", "checkout", "merge", "rebase", "fetch", "pull", "push", "remote", "stash", "log", "blame", "tag", "worktree", "gc", "clean", "cherry-pick", "bisect", "submodule", "reflog", "patch":
			if m.activeRepo != nil && m.activeRepo.IsLocal {
				refreshCmds = append(refreshCmds, m.loadFileTree())
			}
			if owner, name := m.activeRepoCoordinates(); owner != "" && name != "" && m.ghClient != nil {
				refreshCmds = append(refreshCmds, m.buildRepoSummary(owner, name))
			}
		}
		if len(refreshCmds) > 0 {
			if postCmd != nil {
				refreshCmds = append(refreshCmds, postCmd)
			}
			return tea.Batch(refreshCmds...)
		}
		return postCmd
	}

	m.chatView.AppendMessage(views.Message{
		Role:      views.RoleError,
		Content:   fmt.Sprintf("Unknown command: /%s\nEnter /help to see available commands.", cmd),
		Timestamp: time.Now(),
	})
	return nil
}

func (m *Model) registerBuiltinCommands() {
	m.cmdHandlers["help"] = func(_ string) string {
		var b bytes.Buffer
		b.WriteString("Available commands:\n\n")
		b.WriteString("  /help          Show help\n")
		b.WriteString("  /dashboard     Switch to Dashboard\n")
		b.WriteString("  /chat          Switch to Chat\n")
		b.WriteString("  /explorer      Switch to Explorer (PRs/Issues/Files/Commits/Branches/Workflows/Deployments)\n")
		b.WriteString("  /workspace     Switch to Workspace (tabs 1–5: plans, tasks, evidence, cruise, approvals)\n")
		b.WriteString("  /settings      Open Settings\n")
		b.WriteString("  /clear         Clear the chat history\n")
		b.WriteString("  /theme [name]  List or switch themes\n")
		b.WriteString("  /view <name>   Switch views\n")
		b.WriteString("  /clone [path]  Clone the active remote repository locally\n")
		b.WriteString("  /quit          Exit Gitdex\n")
		b.WriteString("\nType natural language directly to talk with Gitdex.\n")
		b.WriteString("Press Tab to move focus into the main content area for navigation.")
		return b.String()
	}

	m.cmdHandlers["clear"] = func(_ string) string {
		m.chatView = views.NewChatView(m.theme)
		contentHeight := m.dims.ContentHeight()
		if contentHeight < 5 {
			contentHeight = 5
		}
		m.chatView.SetSize(m.dims.MainWidth(), contentHeight)
		m.router = views.NewRouter(views.ViewDashboard, m.dashboardView, m.chatView, m.explorerView, m.workspaceView, m.settingsView, m.reflogView)
		return "Chat cleared."
	}

	m.cmdHandlers["theme"] = func(args string) string {
		args = strings.TrimSpace(args)
		if args == "" {
			names := theme.PaletteNames()
			return "Available themes: " + strings.Join(names, ", ")
		}
		if fn, ok := theme.BuiltinPalettes[args]; ok {
			m.paletteName = args
			for i, n := range theme.PaletteNames() {
				if n == args {
					m.paletteIdx = i
					break
				}
			}
			palette := fn()
			*m.theme = theme.NewTheme(m.theme.IsDark, palette)
			m.styles = theme.NewStyles(*m.theme)
			m.refreshStyles()
			m.statusBar.SetThemeName(m.paletteName)
			return "Switched theme to " + args
		}
		return fmt.Sprintf("Unknown theme: %s\nAvailable: %s", args, strings.Join(theme.PaletteNames(), ", "))
	}

	m.cmdHandlers["dashboard"] = func(_ string) string {
		m.switchView(views.ViewDashboard)
		return "Switched to Dashboard. Press Tab to enter the content area, then use 1/2 for the dashboard subtabs."
	}

	m.cmdHandlers["chat"] = func(_ string) string {
		m.switchView(views.ViewChat)
		return "Switched to Chat."
	}

	m.cmdHandlers["explorer"] = func(_ string) string {
		m.switchView(views.ViewExplorer)
		return "Switched to Explorer. Press Tab to enter the content area, then use 1-7 for PRs, Issues, Files, Commits, Branches, Workflows, and Deployments."
	}

	m.cmdHandlers["workspace"] = func(_ string) string {
		m.switchView(views.ViewWorkspace)
		return "Switched to Workspace. Press Tab to enter the content area, then use 1/2/3 for Plans, Tasks, and Evidence."
	}

	m.cmdHandlers["settings"] = func(_ string) string {
		m.switchView(views.ViewSettings)
		return "Switched to Settings. Use Tab for sections, Up/Down for fields, Enter to edit, and Left/Right or Space to cycle recommended options."
	}

	m.cmdHandlers["view"] = func(args string) string {
		target := strings.TrimSpace(args)
		if id := navPathToViewID(target); id != "" {
			m.switchView(id)
			return "Switched to " + target + " view."
		}
		return fmt.Sprintf("Unknown view: %s\nAvailable: dashboard, chat, explorer, workspace, settings, reflog", target)
	}

	m.cmdHandlers["status"] = func(_ string) string {
		m.switchView(views.ViewDashboard)
		return "Switched to Dashboard view."
	}

	m.cmdHandlers["quit"] = func(_ string) string {
		return "Exiting..."
	}
}

func (m *Model) handleConfigSave(msg views.ConfigSaveMsg) tea.Cmd {
	target := msg.Target
	if target == "" {
		target = views.SaveTargetGlobal
	}

	cfgPath := ""
	targetLabel := "global"
	switch target {
	case views.SaveTargetRepo:
		cfg, err := config.Load(config.Options{})
		if err != nil {
			m.chatView.AppendMessage(views.Message{
				Role:      views.RoleError,
				Content:   fmt.Sprintf("Could not resolve repository config target: %v", err),
				Timestamp: time.Now(),
			})
			return nil
		}
		if !cfg.Paths.RepositoryDetected || strings.TrimSpace(cfg.Paths.RepoConfig) == "" {
			m.chatView.AppendMessage(views.Message{
				Role:      views.RoleError,
				Content:   "Repository config is unavailable because no repository context was detected.",
				Timestamp: time.Now(),
			})
			return nil
		}
		cfgPath = cfg.Paths.RepoConfig
		targetLabel = "repo"
	default:
		path, err := config.ResolveGlobalConfigPath("")
		if err != nil {
			m.chatView.AppendMessage(views.Message{
				Role:      views.RoleError,
				Content:   fmt.Sprintf("Could not resolve global config path: %v", err),
				Timestamp: time.Now(),
			})
			return nil
		}
		cfgPath = path
	}

	dirtyKeys := msg.DirtyKeys
	if len(dirtyKeys) == 0 {
		dirtyKeys = make([]string, 0, len(msg.Fields))
		for _, field := range msg.Fields {
			dirtyKeys = append(dirtyKeys, field.Key)
		}
	}

	fieldsToWrite := views.FilterConfigFields(msg.Fields, dirtyKeys)
	if len(fieldsToWrite) == 0 {
		m.chatView.AppendMessage(views.Message{
			Role:      views.RoleInfo,
			Content:   "No settings changes to save.",
			Timestamp: time.Now(),
		})
		return nil
	}

	base := config.FileConfig{}
	if _, err := os.Stat(cfgPath); err == nil {
		existing, err := config.ReadFile(cfgPath)
		if err != nil {
			m.chatView.AppendMessage(views.Message{
				Role:      views.RoleError,
				Content:   fmt.Sprintf("Could not read existing config before save: %v", err),
				Timestamp: time.Now(),
			})
			return nil
		}
		base = existing
	}

	fc := views.ApplyFieldsToFileConfig(base, fieldsToWrite)
	normalizedStorage := storage.Config{
		Type:         storage.BackendType(fc.Storage.Type),
		DSN:          fc.Storage.DSN,
		MaxOpenConns: fc.Storage.MaxOpenConns,
		MaxIdleConns: fc.Storage.MaxIdleConns,
		AutoMigrate:  fc.Storage.AutoMigrate,
	}.Normalized(filepath.Dir(cfgPath))
	fc.Storage.Type = string(normalizedStorage.Type)
	fc.Storage.DSN = normalizedStorage.DSN
	if err := config.WriteFile(cfgPath, fc); err != nil {
		m.chatView.AppendMessage(views.Message{
			Role:      views.RoleError,
			Content:   fmt.Sprintf("Config write failed: %v", err),
			Timestamp: time.Now(),
		})
		return nil
	}

	m.chatView.AppendMessage(views.Message{
		Role:      views.RoleSystem,
		Content:   fmt.Sprintf("Saved %d setting(s) to %s config:\n%s", len(fieldsToWrite), targetLabel, views.FormatConfigSaveMsg(fieldsToWrite)),
		Timestamp: time.Now(),
	})

	m.tryLoadSettingsFromConfig()
	m.tryLoadLLMProvider()
	m.tryLoadGitHubClient()

	if m.ghClient != nil {
		m.chatView.AppendMessage(views.Message{
			Role:      views.RoleInfo,
			Content:   "GitHub authentication is ready. Refreshing repositories.",
			Timestamp: time.Now(),
		})
		return m.fetchRepos()
	}
	return nil
}

const (
	maxFilePreviewBytes = 1 << 27 // 128 MiB
	sniffBinaryBytes    = 8192
	hexPreviewBytes     = 256
	largeFileThreshold  = 10 << 20 // 10 MiB -- warn user about external editor
)

func formatHexDump(data []byte) string {
	var b strings.Builder
	for i := 0; i < len(data); i += 16 {
		end := i + 16
		if end > len(data) {
			end = len(data)
		}
		b.WriteString(fmt.Sprintf("%04x  ", i))
		for j := i; j < end; j++ {
			b.WriteString(fmt.Sprintf("%02x ", data[j]))
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func readFileForPreview(absPath string) views.FileContentMsg {
	info, err := os.Stat(absPath)
	if err != nil {
		return views.FileContentMsg{Path: absPath, Content: fmt.Sprintf("Error reading file: %v", err)}
	}
	if info.IsDir() {
		return views.FileContentMsg{Path: absPath, Content: "Error: path is a directory"}
	}
	size := info.Size()
	readN := size
	if readN > maxFilePreviewBytes {
		readN = maxFilePreviewBytes
	}
	f, err := os.Open(absPath)
	if err != nil {
		return views.FileContentMsg{Path: absPath, Content: fmt.Sprintf("Error reading file: %v", err)}
	}
	defer f.Close()
	buf := make([]byte, readN)
	n, err := io.ReadFull(f, buf)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return views.FileContentMsg{Path: absPath, Content: fmt.Sprintf("Error reading file: %v", err)}
	}
	data := buf[:n]
	sniff := data
	if len(sniff) > sniffBinaryBytes {
		sniff = sniff[:sniffBinaryBytes]
	}
	isBinary := bytes.IndexByte(sniff, 0) >= 0
	truncated := size > maxFilePreviewBytes

	hexLen := hexPreviewBytes
	if len(data) < hexLen {
		hexLen = len(data)
	}
	hexDump := ""
	if hexLen > 0 {
		hexDump = formatHexDump(data[:hexLen])
	}

	msg := views.FileContentMsg{
		Path:      absPath,
		SizeBytes: size,
		Truncated: truncated,
		IsBinary:  isBinary,
		HexDump:   hexDump,
	}
	if isBinary {
		msg.Content = fmt.Sprintf("Binary file (%d bytes)", size)
		return msg
	}
	content := string(data)
	if truncated {
		mb := float64(size) / (1024 * 1024)
		content += fmt.Sprintf("\n\n[truncated - file is %.2f MB]", mb)
	}
	msg.Content = content
	return msg
}

func (m *Model) loadFileContent(path string) tea.Cmd {
	return func() tea.Msg {
		return readFileForPreview(path)
	}
}

func (m *Model) loadFileEditContent(path string) tea.Cmd {
	return func() tea.Msg {
		info, err := os.Stat(path)
		if err != nil {
			return views.FileSavedMsg{Path: path, Err: fmt.Errorf("error reading file for edit: %w", err)}
		}
		if info.IsDir() {
			return views.FileSavedMsg{Path: path, Err: fmt.Errorf("path is a directory")}
		}
		readN := info.Size()
		if readN > maxFilePreviewBytes {
			readN = maxFilePreviewBytes
		}
		f, err := os.Open(path)
		if err != nil {
			return views.FileSavedMsg{Path: path, Err: fmt.Errorf("error reading file for edit: %w", err)}
		}
		defer f.Close()
		buf := make([]byte, readN)
		n, err := io.ReadFull(f, buf)
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			return views.FileSavedMsg{Path: path, Err: fmt.Errorf("error reading file for edit: %w", err)}
		}
		data := buf[:n]
		sniff := data
		if len(sniff) > sniffBinaryBytes {
			sniff = sniff[:sniffBinaryBytes]
		}
		if bytes.IndexByte(sniff, 0) >= 0 {
			return views.FileEditMsg{Path: path, Content: "Binary file — cannot edit here safely. Open preview (Enter) and use h (hex) or o (external)."}
		}
		content := string(data)
		if info.Size() > maxFilePreviewBytes {
			mb := float64(info.Size()) / (1024 * 1024)
			content += fmt.Sprintf("\n\n[truncated - file is %.2f MB; first 128 MiB loaded]", mb)
		}
		if info.Size() > largeFileThreshold {
			mb := float64(info.Size()) / (1024 * 1024)
			content = fmt.Sprintf("--- Large file (%.1f MB) loaded. Press 'o' for external editor. ---\n\n", mb) + content
		}
		return views.FileEditMsg{Path: path, Content: content}
	}
}

func (m *Model) saveFileContent(path, content string) tea.Cmd {
	return func() tea.Msg {
		mode := os.FileMode(0o644)
		if info, err := os.Stat(path); err == nil {
			mode = info.Mode().Perm()
		}
		if err := os.WriteFile(path, []byte(content), mode); err != nil {
			return views.FileSavedMsg{Path: path, Content: content, Err: err}
		}
		return views.FileSavedMsg{Path: path, Content: content}
	}
}

func (m *Model) handleFileOp(msg views.RequestFileOpMsg) tea.Cmd {
	return func() tea.Msg {
		root := ""
		if m.activeRepo != nil && m.activeRepo.IsLocal && m.activeRepo.LocalPath() != "" {
			root = m.activeRepo.LocalPath()
		}
		if strings.TrimSpace(root) == "" {
			return views.FileOpResultMsg{Kind: msg.Kind, Path: msg.Path, Target: msg.Target, Err: fmt.Errorf("file operations are only available for local repositories")}
		}

		resolve := func(rel string) string {
			rel = filepath.FromSlash(strings.TrimSpace(rel))
			if rel == "" {
				return root
			}
			if filepath.IsAbs(rel) {
				return rel
			}
			return filepath.Join(root, rel)
		}
		withinRoot := func(path string) bool {
			rootClean := filepath.Clean(root)
			pathClean := filepath.Clean(path)
			rootKey := strings.ToLower(filepath.ToSlash(rootClean))
			pathKey := strings.ToLower(filepath.ToSlash(pathClean))
			if pathKey == rootKey {
				return true
			}
			return strings.HasPrefix(pathKey, rootKey+"/")
		}

		sourcePath := resolve(msg.Path)
		targetPath := resolve(msg.Target)
		if msg.Kind != views.FileOpCreateFile && msg.Kind != views.FileOpCreateDir && !withinRoot(sourcePath) {
			return views.FileOpResultMsg{Kind: msg.Kind, Path: msg.Path, Target: msg.Target, Err: fmt.Errorf("path escapes repository root")}
		}
		if msg.Kind != views.FileOpDelete && strings.TrimSpace(msg.Target) != "" && !withinRoot(targetPath) {
			return views.FileOpResultMsg{Kind: msg.Kind, Path: msg.Path, Target: msg.Target, Err: fmt.Errorf("target escapes repository root")}
		}

		var err error
		switch msg.Kind {
		case views.FileOpCreateFile:
			err = os.MkdirAll(filepath.Dir(targetPath), 0o755)
			if err == nil {
				var f *os.File
				f, err = os.OpenFile(targetPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
				if err == nil {
					err = f.Close()
				}
			}
		case views.FileOpCreateDir:
			err = os.MkdirAll(targetPath, 0o755)
		case views.FileOpMove:
			err = os.MkdirAll(filepath.Dir(targetPath), 0o755)
			if err == nil {
				err = os.Rename(sourcePath, targetPath)
			}
		case views.FileOpDelete:
			err = os.RemoveAll(sourcePath)
		default:
			err = fmt.Errorf("unsupported file operation %q", msg.Kind)
		}

		return views.FileOpResultMsg{
			Kind:   msg.Kind,
			Path:   filepath.ToSlash(strings.TrimSpace(msg.Path)),
			Target: filepath.ToSlash(strings.TrimSpace(msg.Target)),
			Err:    err,
		}
	}
}

func (m *Model) handleBatchFileOp(msg views.RequestBatchFileOpMsg) tea.Cmd {
	return func() tea.Msg {
		root := ""
		if m.activeRepo != nil && m.activeRepo.IsLocal && m.activeRepo.LocalPath() != "" {
			root = m.activeRepo.LocalPath()
		}
		if root == "" {
			return views.BatchFileOpResultMsg{Err: fmt.Errorf("file operations require a local repository")}
		}
		paths := append([]string(nil), msg.Paths...)
		if len(paths) == 0 {
			return views.BatchFileOpResultMsg{Err: fmt.Errorf("no files selected")}
		}

		resolve := func(rel string) string {
			rel = filepath.FromSlash(strings.TrimSpace(rel))
			if rel == "" {
				return root
			}
			return filepath.Join(root, rel)
		}

		switch msg.Kind {
		case "rename":
			parts := strings.SplitN(msg.Pattern, "->", 2)
			if len(parts) != 2 {
				return views.BatchFileOpResultMsg{Err: fmt.Errorf("pattern must be like \"*.txt -> *.md\"")}
			}
			oldGlob := strings.TrimSpace(parts[0])
			newGlob := strings.TrimSpace(parts[1])
			if oldGlob == "" || newGlob == "" {
				return views.BatchFileOpResultMsg{Err: fmt.Errorf("invalid rename pattern")}
			}
			var n int
			for _, rel := range paths {
				newRel, ok := batchRenameTarget(oldGlob, newGlob, rel)
				if !ok {
					continue
				}
				src := resolve(rel)
				dst := resolve(newRel)
				if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
					return views.BatchFileOpResultMsg{Err: err}
				}
				if err := os.Rename(src, dst); err != nil {
					return views.BatchFileOpResultMsg{Err: err}
				}
				n++
			}
			if n == 0 {
				return views.BatchFileOpResultMsg{Message: "No files matched the rename pattern."}
			}
			return views.BatchFileOpResultMsg{Message: fmt.Sprintf("Renamed %d file(s).", n)}

		case "copy":
			destDir := filepath.ToSlash(strings.TrimSpace(msg.TargetDir))
			if destDir == "" {
				return views.BatchFileOpResultMsg{Err: fmt.Errorf("destination directory is required")}
			}
			dstBase, err := ensureRepoPath(root, destDir)
			if err != nil {
				return views.BatchFileOpResultMsg{Err: err}
			}
			if err := os.MkdirAll(dstBase, 0o755); err != nil {
				return views.BatchFileOpResultMsg{Err: err}
			}
			for _, rel := range paths {
				src := resolve(rel)
				dst := filepath.Join(dstBase, filepath.Base(rel))
				if err := copyPath(src, dst); err != nil {
					return views.BatchFileOpResultMsg{Err: err}
				}
			}
			return views.BatchFileOpResultMsg{Message: fmt.Sprintf("Copied %d file(s) to %s.", len(paths), destDir)}

		case "move":
			destDir := filepath.ToSlash(strings.TrimSpace(msg.TargetDir))
			if destDir == "" {
				return views.BatchFileOpResultMsg{Err: fmt.Errorf("destination directory is required")}
			}
			dstBase, err := ensureRepoPath(root, destDir)
			if err != nil {
				return views.BatchFileOpResultMsg{Err: err}
			}
			if err := os.MkdirAll(dstBase, 0o755); err != nil {
				return views.BatchFileOpResultMsg{Err: err}
			}
			for _, rel := range paths {
				src := resolve(rel)
				dst := filepath.Join(dstBase, filepath.Base(rel))
				if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
					return views.BatchFileOpResultMsg{Err: err}
				}
				if err := os.Rename(src, dst); err != nil {
					return views.BatchFileOpResultMsg{Err: err}
				}
			}
			return views.BatchFileOpResultMsg{Message: fmt.Sprintf("Moved %d file(s) to %s.", len(paths), destDir)}

		case "delete":
			for _, rel := range paths {
				src := resolve(rel)
				if err := os.RemoveAll(src); err != nil {
					return views.BatchFileOpResultMsg{Err: err}
				}
			}
			return views.BatchFileOpResultMsg{Message: fmt.Sprintf("Deleted %d path(s).", len(paths))}

		default:
			return views.BatchFileOpResultMsg{Err: fmt.Errorf("unknown batch operation %q", msg.Kind)}
		}
	}
}

func batchRenameTarget(oldGlob, newGlob, relPath string) (newRel string, ok bool) {
	relPath = filepath.ToSlash(relPath)
	base := filepath.Base(relPath)
	dir := filepath.Dir(relPath)
	if dir == "." {
		dir = ""
	}
	oldGlob = filepath.ToSlash(strings.TrimSpace(oldGlob))
	newGlob = filepath.ToSlash(strings.TrimSpace(newGlob))

	if strings.HasPrefix(oldGlob, "*.") && strings.HasPrefix(newGlob, "*.") {
		extOld := strings.TrimPrefix(oldGlob, "*")
		extNew := strings.TrimPrefix(newGlob, "*")
		if strings.HasSuffix(base, extOld) {
			stem := strings.TrimSuffix(base, extOld)
			newBase := stem + extNew
			if dir == "" || dir == "." {
				return newBase, true
			}
			return filepath.ToSlash(filepath.Join(dir, newBase)), true
		}
		return "", false
	}
	matched, _ := filepath.Match(oldGlob, base)
	if !matched {
		return "", false
	}
	if strings.Contains(newGlob, "*") {
		return "", false
	}
	if dir == "" || dir == "." {
		return newGlob, true
	}
	return filepath.ToSlash(filepath.Join(dir, newGlob)), true
}

// loadFileDiff loads a file diff; optional cached selects git diff --cached (staged).
func (m *Model) loadFileDiff(path string, cached ...bool) tea.Cmd {
	useCached := false
	if len(cached) > 0 {
		useCached = cached[0]
	}
	activeRepo := m.activeRepo
	return func() tea.Msg {
		if activeRepo == nil {
			return views.FileDiffMsg{Diff: "(no repository selected)", Cached: useCached}
		}
		if !activeRepo.IsLocal || activeRepo.LocalPath() == "" {
			return views.FileDiffMsg{Diff: "(diff only available for local repositories)", Cached: useCached}
		}
		executor := gitops.NewGitExecutor()
		pm := gitops.NewPatchManager(executor)
		diffPath := path
		if filepath.IsAbs(diffPath) {
			if rel, err := filepath.Rel(activeRepo.LocalPath(), diffPath); err == nil {
				diffPath = filepath.ToSlash(rel)
			}
		}
		diff, err := pm.Diff(context.Background(), activeRepo.LocalPath(), &gitops.DiffOptions{
			Paths:  []string{diffPath},
			Cached: useCached,
		})
		if err != nil {
			return views.FileDiffMsg{Diff: fmt.Sprintf("Error: %v", err), Cached: useCached}
		}
		if diff == "" {
			return views.FileDiffMsg{Diff: "(no changes)", Cached: useCached}
		}
		return views.FileDiffMsg{Diff: diff, Cached: useCached}
	}
}

func (m *Model) applyGitPatch(msg views.RequestApplyGitPatchMsg) tea.Cmd {
	activeRepo := m.activeRepo
	return func() tea.Msg {
		if activeRepo == nil || activeRepo.LocalPath() == "" {
			return views.FileDiffMsg{Diff: "(no repository selected)", Cached: msg.Cached}
		}
		pm := gitops.NewPatchManager(gitops.NewGitExecutor())
		err := pm.ApplyPatchCachedFromStdin(context.Background(), activeRepo.LocalPath(), strings.NewReader(msg.Patch), msg.Reverse)
		if err != nil {
			return views.FileDiffMsg{Diff: fmt.Sprintf("Error applying patch: %v", err), Cached: msg.Cached}
		}
		diff, err := pm.Diff(context.Background(), activeRepo.LocalPath(), &gitops.DiffOptions{
			Paths:  []string{msg.FilePath},
			Cached: msg.Cached,
		})
		if err != nil {
			return views.FileDiffMsg{Diff: fmt.Sprintf("Error reloading diff: %v", err), Cached: msg.Cached}
		}
		if diff == "" {
			return views.FileDiffMsg{Diff: "(no changes)", Cached: msg.Cached}
		}
		return views.FileDiffMsg{Diff: diff, Cached: msg.Cached}
	}
}

func (m *Model) gitStageFile(msg views.RequestGitStageFileMsg) tea.Cmd {
	activeRepo := m.activeRepo
	return func() tea.Msg {
		if activeRepo == nil || activeRepo.LocalPath() == "" {
			return views.FileDiffMsg{Diff: "(no repository selected)", Cached: msg.Cached}
		}
		cm := gitops.NewCommitManager(gitops.NewGitExecutor())
		var err error
		if msg.Unstage {
			err = cm.RestoreStaged(context.Background(), activeRepo.LocalPath(), msg.Path)
		} else {
			err = cm.Add(context.Background(), activeRepo.LocalPath(), msg.Path)
		}
		if err != nil {
			return views.FileDiffMsg{Diff: fmt.Sprintf("Error: %v", err), Cached: msg.Cached}
		}
		pm := gitops.NewPatchManager(gitops.NewGitExecutor())
		diff, err := pm.Diff(context.Background(), activeRepo.LocalPath(), &gitops.DiffOptions{
			Paths:  []string{msg.Path},
			Cached: msg.Cached,
		})
		if err != nil {
			return views.FileDiffMsg{Diff: fmt.Sprintf("Error reloading diff: %v", err), Cached: msg.Cached}
		}
		if diff == "" {
			return views.FileDiffMsg{Diff: "(no changes)", Cached: msg.Cached}
		}
		return views.FileDiffMsg{Diff: diff, Cached: msg.Cached}
	}
}

func (m *Model) loadFileTree() tea.Cmd {
	repoPath := ""
	if m.activeRepo != nil && m.activeRepo.LocalPath() != "" {
		repoPath = m.activeRepo.LocalPath()
	}
	return func() tea.Msg {
		root := repoPath
		if root == "" {
			var err error
			root, err = config.ResolveRepositoryRoot("")
			if err != nil {
				return views.FileTreeDataMsg{Root: nil}
			}
		}
		executor := gitops.NewGitExecutor()
		hi := gitops.NewHistoryInspector(executor)
		entries, err := hi.LsTree(context.Background(), root, "HEAD", true)
		if err != nil {
			return views.FileTreeDataMsg{Root: nil}
		}
		paths := make([]string, len(entries))
		for i, e := range entries {
			paths[i] = e.Path
		}
		treeRoot := views.BuildFileTree(paths)
		return views.FileTreeDataMsg{Root: treeRoot}
	}
}

func (m *Model) activeGitHubCoordinates() (string, string) {
	if m.activeRepo == nil {
		return "", ""
	}
	if strings.TrimSpace(m.activeRepo.Owner) != "" && strings.TrimSpace(m.activeRepo.Name) != "" {
		return strings.TrimSpace(m.activeRepo.Owner), strings.TrimSpace(m.activeRepo.Name)
	}
	if root := strings.TrimSpace(m.activeRepo.LocalPath()); root != "" {
		return parseRemoteOwnerRepo(root)
	}
	return "", ""
}

func (m *Model) loadPRDetail(number int) tea.Cmd {
	activeRepo := m.activeRepo
	client := m.ghClient
	owner, name := m.activeGitHubCoordinates()
	return func() tea.Msg {
		if activeRepo == nil || client == nil || owner == "" || name == "" {
			return views.PRDetailMsg{Err: fmt.Errorf("PR detail requires a GitHub-backed repository")}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		pr, err := client.GetPullRequest(ctx, owner, name, number)
		if err != nil {
			return views.PRDetailMsg{Err: err}
		}
		reviews, _ := client.ListPRReviews(ctx, owner, name, number)
		files, _ := client.ListPRFiles(ctx, owner, name, number)
		comments, _ := client.ListPRComments(ctx, owner, name, number)

		labels := make([]string, 0, len(pr.Labels))
		for _, label := range pr.Labels {
			labels = append(labels, label.GetName())
		}

		detail := views.PRDetail{
			Number: pr.GetNumber(),
			Title:  pr.GetTitle(),
			Author: pr.GetUser().GetLogin(),
			State:  pr.GetState(),
			Body:   pr.GetBody(),
			Labels: labels,
		}
		if pr.GetMerged() {
			detail.State = "merged"
		}

		for _, review := range reviews {
			if review == nil {
				continue
			}
			detail.Reviews = append(detail.Reviews, views.PRReviewItem{
				User:  review.GetUser().GetLogin(),
				State: review.GetState(),
			})
		}

		for _, file := range files {
			if file == nil {
				continue
			}
			detail.Files = append(detail.Files, views.PRFileItem{
				Filename:  file.GetFilename(),
				Status:    file.GetStatus(),
				Additions: file.GetAdditions(),
				Deletions: file.GetDeletions(),
			})
		}

		for _, comment := range comments {
			if comment == nil {
				continue
			}
			created := ""
			if comment.CreatedAt != nil {
				created = comment.CreatedAt.Time.Format("2006-01-02")
			}
			detail.Comments = append(detail.Comments, views.PRCommentItem{
				User:    comment.GetUser().GetLogin(),
				Body:    comment.GetBody(),
				Created: created,
			})
		}

		return views.PRDetailMsg{Detail: detail}
	}
}

func (m *Model) loadIssueDetail(number int) tea.Cmd {
	activeRepo := m.activeRepo
	client := m.ghClient
	owner, name := m.activeGitHubCoordinates()
	return func() tea.Msg {
		if activeRepo == nil || client == nil || owner == "" || name == "" {
			return views.IssueDetailMsg{Err: fmt.Errorf("Issue detail requires a GitHub-backed repository")}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		issue, err := client.GetIssue(ctx, owner, name, number)
		if err != nil {
			return views.IssueDetailMsg{Err: err}
		}
		comments, _ := client.ListIssueComments(ctx, owner, name, number)

		labels := make([]string, 0, len(issue.Labels))
		for _, label := range issue.Labels {
			labels = append(labels, label.GetName())
		}
		assignees := make([]string, 0, len(issue.Assignees))
		for _, assignee := range issue.Assignees {
			assignees = append(assignees, assignee.GetLogin())
		}
		milestone := ""
		if issue.Milestone != nil {
			milestone = issue.Milestone.GetTitle()
		}

		detail := views.IssueDetail{
			Number:    issue.GetNumber(),
			Title:     issue.GetTitle(),
			Author:    issue.GetUser().GetLogin(),
			State:     issue.GetState(),
			Body:      issue.GetBody(),
			Labels:    labels,
			Assignees: assignees,
			Milestone: milestone,
		}

		for _, comment := range comments {
			if comment == nil {
				continue
			}
			created := ""
			if comment.CreatedAt != nil {
				created = comment.CreatedAt.Time.Format("2006-01-02")
			}
			detail.Comments = append(detail.Comments, views.IssueCommentItem{
				User:    comment.GetUser().GetLogin(),
				Body:    comment.GetBody(),
				Created: created,
			})
		}

		return views.IssueDetailMsg{Detail: detail}
	}
}

func (m *Model) handlePRAction(msg views.RequestPRActionMsg) tea.Cmd {
	activeRepo := m.activeRepo
	client := m.ghClient
	owner, name := m.activeGitHubCoordinates()
	return func() tea.Msg {
		if activeRepo == nil || client == nil || owner == "" || name == "" {
			return views.PRActionResultMsg{Number: msg.Number, Kind: msg.Kind, Err: fmt.Errorf("PR action requires a GitHub-backed repository"), Message: "PR action failed: repository or GitHub runtime unavailable"}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		switch msg.Kind {
		case views.PRActionComment:
			comment, err := client.CreateComment(ctx, owner, name, msg.Number, strings.TrimSpace(msg.Body))
			if err != nil {
				return views.PRActionResultMsg{Number: msg.Number, Kind: msg.Kind, Err: err, Message: fmt.Sprintf("Failed to comment on PR #%d: %v", msg.Number, err)}
			}
			return views.PRActionResultMsg{Number: msg.Number, Kind: msg.Kind, Message: fmt.Sprintf("Commented on PR #%d%s", msg.Number, previewURL(comment.GetHTMLURL()))}
		case views.PRActionApprove:
			review, err := client.SubmitPRReview(ctx, owner, name, msg.Number, "APPROVE", strings.TrimSpace(msg.Body))
			if err != nil {
				return views.PRActionResultMsg{Number: msg.Number, Kind: msg.Kind, Err: err, Message: fmt.Sprintf("Failed to approve PR #%d: %v", msg.Number, err)}
			}
			return views.PRActionResultMsg{Number: msg.Number, Kind: msg.Kind, Message: fmt.Sprintf("Approved PR #%d%s", msg.Number, previewURL(review.GetHTMLURL()))}
		case views.PRActionRequestChanges:
			review, err := client.SubmitPRReview(ctx, owner, name, msg.Number, "REQUEST_CHANGES", strings.TrimSpace(msg.Body))
			if err != nil {
				return views.PRActionResultMsg{Number: msg.Number, Kind: msg.Kind, Err: err, Message: fmt.Sprintf("Failed to request changes on PR #%d: %v", msg.Number, err)}
			}
			return views.PRActionResultMsg{Number: msg.Number, Kind: msg.Kind, Message: fmt.Sprintf("Requested changes on PR #%d%s", msg.Number, previewURL(review.GetHTMLURL()))}
		case views.PRActionMerge:
			method := strings.TrimSpace(msg.MergeMethod)
			if method == "" {
				method = "merge"
			}
			result, err := client.MergePullRequest(ctx, owner, name, msg.Number, strings.TrimSpace(msg.Body), method)
			if err != nil {
				return views.PRActionResultMsg{Number: msg.Number, Kind: msg.Kind, Err: err, Message: fmt.Sprintf("Failed to merge PR #%d: %v", msg.Number, err)}
			}
			if !result.GetMerged() {
				mergeErr := fmt.Errorf("%s", result.GetMessage())
				return views.PRActionResultMsg{Number: msg.Number, Kind: msg.Kind, Err: mergeErr, Message: fmt.Sprintf("PR #%d was not merged: %s", msg.Number, result.GetMessage())}
			}
			return views.PRActionResultMsg{Number: msg.Number, Kind: msg.Kind, Message: fmt.Sprintf("Merged PR #%d using %s", msg.Number, method)}
		case views.PRActionClose:
			if err := client.CloseIssue(ctx, owner, name, msg.Number); err != nil {
				return views.PRActionResultMsg{Number: msg.Number, Kind: msg.Kind, Err: err, Message: fmt.Sprintf("Failed to close PR #%d: %v", msg.Number, err)}
			}
			return views.PRActionResultMsg{Number: msg.Number, Kind: msg.Kind, Message: fmt.Sprintf("Closed PR #%d", msg.Number)}
		default:
			err := fmt.Errorf("unsupported PR action %q", msg.Kind)
			return views.PRActionResultMsg{Number: msg.Number, Kind: msg.Kind, Err: err, Message: err.Error()}
		}
	}
}

func (m *Model) handleIssueAction(msg views.RequestIssueActionMsg) tea.Cmd {
	activeRepo := m.activeRepo
	client := m.ghClient
	owner, name := m.activeGitHubCoordinates()
	return func() tea.Msg {
		if activeRepo == nil || client == nil || owner == "" || name == "" {
			return views.IssueActionResultMsg{Number: msg.Number, Kind: msg.Kind, Err: fmt.Errorf("Issue action requires a GitHub-backed repository"), Message: "Issue action failed: repository or GitHub runtime unavailable"}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		switch msg.Kind {
		case views.IssueActionComment:
			comment, err := client.CreateComment(ctx, owner, name, msg.Number, strings.TrimSpace(msg.Body))
			if err != nil {
				return views.IssueActionResultMsg{Number: msg.Number, Kind: msg.Kind, Err: err, Message: fmt.Sprintf("Failed to comment on issue #%d: %v", msg.Number, err)}
			}
			return views.IssueActionResultMsg{Number: msg.Number, Kind: msg.Kind, Message: fmt.Sprintf("Commented on issue #%d%s", msg.Number, previewURL(comment.GetHTMLURL()))}
		case views.IssueActionLabel:
			req := &ghclientIssueRequestAdapter{Labels: msg.Values}
			issue, err := client.UpdateIssue(ctx, owner, name, msg.Number, req.asIssueRequest())
			if err != nil {
				return views.IssueActionResultMsg{Number: msg.Number, Kind: msg.Kind, Err: err, Message: fmt.Sprintf("Failed to set labels on issue #%d: %v", msg.Number, err)}
			}
			return views.IssueActionResultMsg{Number: msg.Number, Kind: msg.Kind, Message: fmt.Sprintf("Updated labels on issue #%d (%d labels)", msg.Number, len(issue.Labels))}
		case views.IssueActionAssign:
			if err := client.SetAssignees(ctx, owner, name, msg.Number, msg.Values); err != nil {
				return views.IssueActionResultMsg{Number: msg.Number, Kind: msg.Kind, Err: err, Message: fmt.Sprintf("Failed to set assignees on issue #%d: %v", msg.Number, err)}
			}
			return views.IssueActionResultMsg{Number: msg.Number, Kind: msg.Kind, Message: fmt.Sprintf("Updated assignees on issue #%d", msg.Number)}
		case views.IssueActionClose:
			if err := client.CloseIssue(ctx, owner, name, msg.Number); err != nil {
				return views.IssueActionResultMsg{Number: msg.Number, Kind: msg.Kind, Err: err, Message: fmt.Sprintf("Failed to close issue #%d: %v", msg.Number, err)}
			}
			return views.IssueActionResultMsg{Number: msg.Number, Kind: msg.Kind, Message: fmt.Sprintf("Closed issue #%d", msg.Number)}
		case views.IssueActionReopen:
			if err := client.ReopenIssue(ctx, owner, name, msg.Number); err != nil {
				return views.IssueActionResultMsg{Number: msg.Number, Kind: msg.Kind, Err: err, Message: fmt.Sprintf("Failed to reopen issue #%d: %v", msg.Number, err)}
			}
			return views.IssueActionResultMsg{Number: msg.Number, Kind: msg.Kind, Message: fmt.Sprintf("Reopened issue #%d", msg.Number)}
		default:
			err := fmt.Errorf("unsupported issue action %q", msg.Kind)
			return views.IssueActionResultMsg{Number: msg.Number, Kind: msg.Kind, Err: err, Message: err.Error()}
		}
	}
}

func (m *Model) refreshPRAction(msg views.PRActionResultMsg) tea.Cmd {
	if msg.Err != nil || m.activeRepo == nil {
		return nil
	}
	owner, name := m.activeGitHubCoordinates()
	if owner == "" || name == "" {
		return nil
	}
	cmds := []tea.Cmd{
		m.loadPRDetail(msg.Number),
		m.fetchRepoPRs(owner, name),
		m.fetchRepoDetail(owner, name),
		m.buildRepoSummary(owner, name),
	}
	return tea.Batch(cmds...)
}

func (m *Model) refreshIssueAction(msg views.IssueActionResultMsg) tea.Cmd {
	if msg.Err != nil || m.activeRepo == nil {
		return nil
	}
	owner, name := m.activeGitHubCoordinates()
	if owner == "" || name == "" {
		return nil
	}
	cmds := []tea.Cmd{
		m.loadIssueDetail(msg.Number),
		m.fetchRepoIssues(owner, name),
		m.fetchRepoDetail(owner, name),
		m.buildRepoSummary(owner, name),
	}
	return tea.Batch(cmds...)
}

type ghclientIssueRequestAdapter struct {
	Labels []string
}

func (a *ghclientIssueRequestAdapter) asIssueRequest() *gh.IssueRequest {
	labels := append([]string(nil), a.Labels...)
	return &gh.IssueRequest{Labels: &labels}
}

func (m *Model) loadCommitDetail(hash string) tea.Cmd {
	activeRepo := m.activeRepo
	client := m.ghClient
	owner, name := m.activeGitHubCoordinates()
	return func() tea.Msg {
		if activeRepo == nil || strings.TrimSpace(hash) == "" {
			return views.CommitDetailMsg{Err: fmt.Errorf("commit detail unavailable")}
		}

		if activeRepo.IsLocal && activeRepo.LocalPath() != "" {
			executor := gitops.NewGitExecutor()
			hi := gitops.NewHistoryInspector(executor)
			show, err := hi.Show(context.Background(), activeRepo.LocalPath(), hash)
			if err != nil {
				return views.CommitDetailMsg{Hash: hash, Err: err}
			}
			return views.CommitDetailMsg{Hash: hash, Content: show}
		}

		if client != nil && owner != "" && name != "" {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			commit, err := client.GetCommit(ctx, owner, name, hash)
			if err != nil {
				return views.CommitDetailMsg{Hash: hash, Err: err}
			}
			var b strings.Builder
			if commit.Commit != nil {
				b.WriteString(fmt.Sprintf("commit %s\n", commit.GetSHA()))
				if commit.Commit.Author != nil {
					b.WriteString(fmt.Sprintf("Author: %s <%s>\n", commit.Commit.Author.GetName(), commit.Commit.Author.GetEmail()))
					if commit.Commit.Author.Date != nil {
						b.WriteString(fmt.Sprintf("Date:   %s\n\n", commit.Commit.Author.GetDate().Time.Format(time.RFC3339)))
					}
				}
				b.WriteString(strings.TrimSpace(commit.Commit.GetMessage()))
				b.WriteString("\n\n")
			}
			for _, file := range commit.Files {
				if file == nil {
					continue
				}
				b.WriteString(fmt.Sprintf("%s  %s  +%d -%d\n", strings.ToUpper(file.GetStatus()), file.GetFilename(), file.GetAdditions(), file.GetDeletions()))
			}
			return views.CommitDetailMsg{Hash: hash, Content: strings.TrimSpace(b.String())}
		}

		return views.CommitDetailMsg{Hash: hash, Err: fmt.Errorf("commit detail unavailable for current repository mode")}
	}
}

func (m *Model) handleCommitAction(msg views.RequestCommitActionMsg) tea.Cmd {
	activeRepo := m.activeRepo
	return func() tea.Msg {
		if activeRepo == nil || !activeRepo.IsLocal || activeRepo.LocalPath() == "" {
			err := fmt.Errorf("commit action requires a local repository")
			return views.CommitActionResultMsg{Hash: msg.Hash, Kind: msg.Kind, Err: err, Message: err.Error()}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		switch msg.Kind {
		case views.CommitActionCherryPick:
			bm := gitops.NewBranchManager(gitops.NewGitExecutor())
			if err := bm.CherryPick(ctx, activeRepo.LocalPath(), msg.Hash); err != nil {
				return views.CommitActionResultMsg{Hash: msg.Hash, Kind: msg.Kind, Err: err, Message: fmt.Sprintf("Cherry-pick failed for %s: %v", msg.Hash, err)}
			}
			return views.CommitActionResultMsg{Hash: msg.Hash, Kind: msg.Kind, Message: fmt.Sprintf("Cherry-picked %s", msg.Hash)}
		case views.CommitActionRevert:
			cm := gitops.NewCommitManager(gitops.NewGitExecutor())
			if err := cm.Revert(ctx, activeRepo.LocalPath(), msg.Hash, nil); err != nil {
				return views.CommitActionResultMsg{Hash: msg.Hash, Kind: msg.Kind, Err: err, Message: fmt.Sprintf("Revert failed for %s: %v", msg.Hash, err)}
			}
			return views.CommitActionResultMsg{Hash: msg.Hash, Kind: msg.Kind, Message: fmt.Sprintf("Reverted %s", msg.Hash)}
		default:
			err := fmt.Errorf("unsupported commit action %q", msg.Kind)
			return views.CommitActionResultMsg{Hash: msg.Hash, Kind: msg.Kind, Err: err, Message: err.Error()}
		}
	}
}

func (m *Model) refreshCommitAction(msg views.CommitActionResultMsg) tea.Cmd {
	if msg.Err != nil {
		return nil
	}
	cmds := []tea.Cmd{m.loadCommitLog(), m.loadBranchTree()}
	owner, name := m.activeGitHubCoordinates()
	if owner != "" && name != "" {
		cmds = append(cmds, m.buildRepoSummary(owner, name))
	}
	return tea.Batch(cmds...)
}

func (m *Model) handleBranchAction(msg views.RequestBranchActionMsg) tea.Cmd {
	activeRepo := m.activeRepo
	return func() tea.Msg {
		if activeRepo == nil || !activeRepo.IsLocal || activeRepo.LocalPath() == "" {
			err := fmt.Errorf("branch action requires a local repository")
			return views.BranchActionResultMsg{Kind: msg.Kind, Name: msg.Name, Target: msg.Target, Err: err, Message: err.Error()}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		bm := gitops.NewBranchManager(gitops.NewGitExecutor())

		switch msg.Kind {
		case views.BranchActionCreate:
			if err := bm.CreateBranch(ctx, activeRepo.LocalPath(), msg.Name, msg.Target); err != nil {
				return views.BranchActionResultMsg{Kind: msg.Kind, Name: msg.Name, Target: msg.Target, Err: err, Message: fmt.Sprintf("Create branch failed for %s: %v", msg.Name, err)}
			}
			return views.BranchActionResultMsg{Kind: msg.Kind, Name: msg.Name, Target: msg.Target, Message: fmt.Sprintf("Created branch %s", msg.Name)}
		case views.BranchActionRename:
			if err := bm.RenameBranch(ctx, activeRepo.LocalPath(), msg.Name, msg.Target); err != nil {
				return views.BranchActionResultMsg{Kind: msg.Kind, Name: msg.Name, Target: msg.Target, Err: err, Message: fmt.Sprintf("Rename branch failed for %s: %v", msg.Name, err)}
			}
			if activeRepo.DefaultBranch == msg.Name {
				activeRepo.DefaultBranch = msg.Target
			}
			return views.BranchActionResultMsg{Kind: msg.Kind, Name: msg.Name, Target: msg.Target, Message: fmt.Sprintf("Renamed branch %s -> %s", msg.Name, msg.Target)}
		case views.BranchActionDelete:
			if err := bm.DeleteBranch(ctx, activeRepo.LocalPath(), msg.Name, msg.Force); err != nil {
				return views.BranchActionResultMsg{Kind: msg.Kind, Name: msg.Name, Err: err, Message: fmt.Sprintf("Delete branch failed for %s: %v", msg.Name, err)}
			}
			return views.BranchActionResultMsg{Kind: msg.Kind, Name: msg.Name, Message: fmt.Sprintf("Deleted branch %s", msg.Name)}
		default:
			err := fmt.Errorf("unsupported branch action %q", msg.Kind)
			return views.BranchActionResultMsg{Kind: msg.Kind, Name: msg.Name, Target: msg.Target, Err: err, Message: err.Error()}
		}
	}
}

func (m *Model) refreshBranchAction(msg views.BranchActionResultMsg) tea.Cmd {
	if msg.Err != nil {
		return nil
	}
	cmds := []tea.Cmd{m.loadBranchTree()}
	owner, name := m.activeGitHubCoordinates()
	if owner != "" && name != "" {
		cmds = append(cmds, m.buildRepoSummary(owner, name))
	}
	return tea.Batch(cmds...)
}

func (m *Model) handleWorkflowDispatch(msg views.RequestWorkflowDispatchMsg) tea.Cmd {
	activeRepo := m.activeRepo
	client := m.ghClient
	return func() tea.Msg {
		if activeRepo == nil || client == nil || activeRepo.Owner == "" || activeRepo.Name == "" {
			err := fmt.Errorf("workflow dispatch requires a GitHub-backed repository")
			return views.WorkflowDispatchResultMsg{WorkflowID: msg.WorkflowID, Ref: msg.Ref, Err: err, Message: err.Error()}
		}
		if msg.WorkflowID == 0 {
			err := fmt.Errorf("workflow ID is required")
			return views.WorkflowDispatchResultMsg{WorkflowID: msg.WorkflowID, Ref: msg.Ref, Err: err, Message: err.Error()}
		}
		ref := strings.TrimSpace(msg.Ref)
		if ref == "" {
			ref = activeRepo.DefaultBranch
		}
		if ref == "" {
			ref = "main"
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		if err := client.TriggerWorkflow(ctx, activeRepo.Owner, activeRepo.Name, msg.WorkflowID, ref); err != nil {
			return views.WorkflowDispatchResultMsg{WorkflowID: msg.WorkflowID, Ref: ref, Err: err, Message: fmt.Sprintf("Workflow dispatch failed for %d: %v", msg.WorkflowID, err)}
		}
		return views.WorkflowDispatchResultMsg{WorkflowID: msg.WorkflowID, Ref: ref, Message: fmt.Sprintf("Dispatched workflow %d on %s/%s @ %s", msg.WorkflowID, activeRepo.Owner, activeRepo.Name, ref)}
	}
}

func (m *Model) handleWorkflowAction(msg views.RequestWorkflowActionMsg) tea.Cmd {
	activeRepo := m.activeRepo
	client := m.ghClient
	return func() tea.Msg {
		if activeRepo == nil || client == nil || activeRepo.Owner == "" || activeRepo.Name == "" {
			err := fmt.Errorf("workflow action requires a GitHub-backed repository")
			return views.WorkflowActionResultMsg{RunID: msg.RunID, Kind: msg.Kind, Err: err, Message: err.Error()}
		}
		if msg.RunID == 0 {
			err := fmt.Errorf("workflow run ID is required")
			return views.WorkflowActionResultMsg{RunID: msg.RunID, Kind: msg.Kind, Err: err, Message: err.Error()}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		switch msg.Kind {
		case views.WorkflowActionRerun:
			if err := client.RerunWorkflowRun(ctx, activeRepo.Owner, activeRepo.Name, msg.RunID); err != nil {
				return views.WorkflowActionResultMsg{RunID: msg.RunID, Kind: msg.Kind, Err: err, Message: fmt.Sprintf("Workflow rerun failed for %d: %v", msg.RunID, err)}
			}
			return views.WorkflowActionResultMsg{RunID: msg.RunID, Kind: msg.Kind, Message: fmt.Sprintf("Rerun requested for workflow run %d on %s/%s", msg.RunID, activeRepo.Owner, activeRepo.Name)}
		case views.WorkflowActionCancel:
			if err := client.CancelWorkflowRun(ctx, activeRepo.Owner, activeRepo.Name, msg.RunID); err != nil {
				return views.WorkflowActionResultMsg{RunID: msg.RunID, Kind: msg.Kind, Err: err, Message: fmt.Sprintf("Workflow cancel failed for %d: %v", msg.RunID, err)}
			}
			return views.WorkflowActionResultMsg{RunID: msg.RunID, Kind: msg.Kind, Message: fmt.Sprintf("Cancel requested for workflow run %d on %s/%s", msg.RunID, activeRepo.Owner, activeRepo.Name)}
		default:
			err := fmt.Errorf("unsupported workflow action %q", msg.Kind)
			return views.WorkflowActionResultMsg{RunID: msg.RunID, Kind: msg.Kind, Err: err, Message: err.Error()}
		}
	}
}

func (m *Model) refreshWorkflowDispatch(msg views.WorkflowDispatchResultMsg) tea.Cmd {
	if msg.Err != nil {
		return nil
	}
	cmds := []tea.Cmd{m.loadWorkflowRuns(), m.loadDeployments(), m.loadReleases()}
	owner, name := m.activeGitHubCoordinates()
	if owner != "" && name != "" {
		cmds = append(cmds, m.buildRepoSummary(owner, name))
	}
	return tea.Batch(cmds...)
}

func (m *Model) refreshWorkflowAction(msg views.WorkflowActionResultMsg) tea.Cmd {
	if msg.Err != nil {
		return nil
	}
	cmds := []tea.Cmd{m.loadWorkflowRuns(), m.loadDeployments(), m.loadReleases()}
	owner, name := m.activeGitHubCoordinates()
	if owner != "" && name != "" {
		cmds = append(cmds, m.buildRepoSummary(owner, name))
	}
	return tea.Batch(cmds...)
}

func (m *Model) checkoutBranch(name string) tea.Cmd {
	activeRepo := m.activeRepo
	return func() tea.Msg {
		if activeRepo == nil || !activeRepo.IsLocal || activeRepo.LocalPath() == "" {
			return views.BranchCheckoutResultMsg{Name: name, Err: fmt.Errorf("branch checkout requires a local repository")}
		}
		bm := gitops.NewBranchManager(gitops.NewGitExecutor())
		if err := bm.SwitchBranch(context.Background(), activeRepo.LocalPath(), name); err != nil {
			return views.BranchCheckoutResultMsg{Name: name, Err: err}
		}
		activeRepo.DefaultBranch = name
		return views.BranchCheckoutResultMsg{Name: name}
	}
}

func (m *Model) loadCommitLog() tea.Cmd {
	activeRepo := m.activeRepo
	client := m.ghClient
	return func() tea.Msg {
		if activeRepo == nil {
			return views.CommitLogDataMsg{}
		}

		if activeRepo.IsLocal && activeRepo.LocalPath() != "" {
			executor := gitops.NewGitExecutor()
			hi := gitops.NewHistoryInspector(executor)
			entries, err := hi.Log(context.Background(), activeRepo.LocalPath(), &gitops.LogOptions{MaxCount: 50})
			if err != nil {
				return views.CommitLogDataMsg{}
			}
			commits := make([]views.CommitEntry, 0, len(entries))
			for _, entry := range entries {
				commits = append(commits, views.CommitEntry{
					Hash:    entry.SHA,
					Author:  entry.Author,
					Date:    entry.Date.Format("2006-01-02"),
					Message: entry.Subject,
				})
			}
			return views.CommitLogDataMsg{Commits: commits}
		}

		if client != nil && activeRepo.Owner != "" {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			entries, err := client.ListCommits(ctx, activeRepo.Owner, activeRepo.Name)
			if err != nil {
				return views.CommitLogDataMsg{}
			}
			commits := make([]views.CommitEntry, 0, len(entries))
			for _, entry := range entries {
				if entry == nil {
					continue
				}
				hash := entry.GetSHA()
				author := ""
				if entry.Author != nil {
					author = entry.Author.GetLogin()
				}
				if author == "" && entry.Commit != nil && entry.Commit.Author != nil {
					author = entry.Commit.Author.GetName()
				}
				date := ""
				if entry.Commit != nil && entry.Commit.Author != nil && entry.Commit.Author.Date != nil {
					date = entry.Commit.Author.GetDate().Time.Format("2006-01-02")
				}
				message := ""
				if entry.Commit != nil {
					message = strings.Split(strings.TrimSpace(entry.Commit.GetMessage()), "\n")[0]
				}
				commits = append(commits, views.CommitEntry{
					Hash:    hash,
					Author:  author,
					Date:    date,
					Message: message,
				})
			}
			return views.CommitLogDataMsg{Commits: commits}
		}

		return views.CommitLogDataMsg{}
	}
}

func (m *Model) loadCommitGraph() tea.Cmd {
	if m.activeRepo == nil || !m.activeRepo.IsLocal || strings.TrimSpace(m.activeRepo.LocalPath()) == "" {
		return nil
	}
	return views.LoadCommitGraphCmd(m.activeRepo.LocalPath())
}

func (m *Model) loadBranchTree() tea.Cmd {
	activeRepo := m.activeRepo
	client := m.ghClient
	return func() tea.Msg {
		if activeRepo == nil {
			return views.BranchTreeDataMsg{}
		}

		if activeRepo.IsLocal && activeRepo.LocalPath() != "" {
			executor := gitops.NewGitExecutor()
			bm := gitops.NewBranchManager(executor)
			localBranches, err := bm.ListBranches(context.Background(), activeRepo.LocalPath(), false)
			if err != nil {
				return views.BranchTreeDataMsg{}
			}
			remoteBranches, _ := bm.ListBranches(context.Background(), activeRepo.LocalPath(), true)
			current := ""
			if res, err := executor.Run(context.Background(), activeRepo.LocalPath(), "rev-parse", "--abbrev-ref", "HEAD"); err == nil {
				current = strings.TrimSpace(res.Stdout)
			}

			entries := make([]views.BranchEntry, 0, len(localBranches)+len(remoteBranches))
			for _, branch := range localBranches {
				entries = append(entries, views.BranchEntry{
					Name:       branch.Name,
					SHA:        branch.SHA,
					Upstream:   branch.Upstream,
					LastCommit: branch.LastCommit,
					Ahead:      branch.Ahead,
					Behind:     branch.Behind,
					IsCurrent:  branch.Name == current,
				})
			}
			for _, branch := range remoteBranches {
				entries = append(entries, views.BranchEntry{
					Name:       branch.Name,
					SHA:        branch.SHA,
					Upstream:   branch.Upstream,
					LastCommit: branch.LastCommit,
					Ahead:      branch.Ahead,
					Behind:     branch.Behind,
					IsRemote:   true,
				})
			}
			return views.BranchTreeDataMsg{Branches: entries}
		}

		if client != nil && activeRepo.Owner != "" {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			branches, err := client.ListBranches(ctx, activeRepo.Owner, activeRepo.Name)
			if err != nil {
				return views.BranchTreeDataMsg{}
			}
			entries := make([]views.BranchEntry, 0, len(branches))
			for _, branch := range branches {
				if branch == nil {
					continue
				}
				protected := false
				if branch.Protected != nil {
					protected = *branch.Protected
				}
				entries = append(entries, views.BranchEntry{
					Name:      branch.GetName(),
					SHA:       branch.GetCommit().GetSHA(),
					IsRemote:  true,
					IsCurrent: branch.GetName() == activeRepo.DefaultBranch,
					Protected: protected,
				})
			}
			return views.BranchTreeDataMsg{Branches: entries}
		}

		return views.BranchTreeDataMsg{}
	}
}

func (m *Model) loadWorkflowRuns() tea.Cmd {
	activeRepo := m.activeRepo
	client := m.ghClient
	return func() tea.Msg {
		if activeRepo == nil || client == nil || activeRepo.Owner == "" || activeRepo.Name == "" {
			return views.WorkflowRunsDataMsg{}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		runs, err := client.ListWorkflowRuns(ctx, activeRepo.Owner, activeRepo.Name)
		if err != nil {
			return views.WorkflowRunsDataMsg{}
		}
		items := make([]views.WorkflowRunEntry, 0, len(runs))
		for _, run := range runs {
			items = append(items, views.WorkflowRunEntry{
				RunID:      run.RunID,
				WorkflowID: run.WorkflowID,
				Name:       run.Name,
				Status:     run.Status,
				Conclusion: run.Conclusion,
				Branch:     run.Branch,
				Event:      run.Event,
				CreatedAt:  run.CreatedAt,
				URL:        run.URL,
			})
		}
		return views.WorkflowRunsDataMsg{Runs: items}
	}
}

func (m *Model) loadDeployments() tea.Cmd {
	activeRepo := m.activeRepo
	client := m.ghClient
	return func() tea.Msg {
		if activeRepo == nil || client == nil || activeRepo.Owner == "" || activeRepo.Name == "" {
			return views.DeploymentDataMsg{}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		deployments, err := client.ListDeployments(ctx, activeRepo.Owner, activeRepo.Name)
		if err != nil {
			return views.DeploymentDataMsg{}
		}
		items := make([]views.DeploymentEntry, 0, len(deployments))
		for _, item := range deployments {
			items = append(items, views.DeploymentEntry{
				ID:          item.ID,
				Environment: item.Environment,
				State:       item.State,
				Ref:         item.Ref,
				CreatedAt:   item.CreatedAt,
				URL:         item.URL,
			})
		}
		return views.DeploymentDataMsg{Deployments: items}
	}
}

// LoadReleasesCmd loads GitHub releases for the active repository (Explorer Releases tab).
func (m *Model) LoadReleasesCmd() tea.Cmd { return m.loadReleases() }

func (m *Model) loadReleases() tea.Cmd {
	activeRepo := m.activeRepo
	client := m.ghClient
	return func() tea.Msg {
		if activeRepo == nil || client == nil || activeRepo.Owner == "" || activeRepo.Name == "" {
			return views.ReleaseListMsg{}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()
		rels, err := client.ListReleases(ctx, activeRepo.Owner, activeRepo.Name)
		if err != nil {
			return views.ReleaseListMsg{Err: err}
		}
		return views.ReleaseListMsg{Releases: rels}
	}
}

func (m *Model) loadBranchProtection(branch string) tea.Cmd {
	activeRepo := m.activeRepo
	client := m.ghClient
	b := strings.TrimSpace(branch)
	return func() tea.Msg {
		if activeRepo == nil || client == nil || activeRepo.Owner == "" || activeRepo.Name == "" || b == "" {
			return views.BranchProtectionDataMsg{Branch: b, Err: fmt.Errorf("branch protection unavailable")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()
		bp, err := client.GetBranchProtection(ctx, activeRepo.Owner, activeRepo.Name, b)
		if err != nil {
			return views.BranchProtectionDataMsg{Branch: b, Err: err}
		}
		rules, _ := client.ListRulesets(ctx, activeRepo.Owner, activeRepo.Name)
		lines := []string{
			fmt.Sprintf("Required reviews: %d", bp.RequiredReviews),
			fmt.Sprintf("Signed commits: %v", bp.RequireSignedCommits),
			fmt.Sprintf("Enforce admins: %v", bp.EnforceAdmins),
			fmt.Sprintf("Linear history: %v", bp.RequireLinearHistory),
		}
		for _, r := range rules {
			lines = append(lines, fmt.Sprintf("Ruleset %q (target=%s): %s", r.Name, r.Target, r.Enforcement))
		}
		return views.BranchProtectionDataMsg{Branch: b, Lines: lines}
	}
}

func (m *Model) handleReleaseOp(msg views.RequestReleaseOpMsg) tea.Cmd {
	client := m.ghClient
	if client == nil {
		return func() tea.Msg {
			return views.ReleaseOpResultMsg{Kind: msg.Kind, Err: fmt.Errorf("GitHub client not configured")}
		}
	}
	owner := strings.TrimSpace(msg.OwnerHint)
	repoName := strings.TrimSpace(msg.RepoHint)
	if m.activeRepo != nil {
		if owner == "" {
			owner = m.activeRepo.Owner
		}
		if repoName == "" {
			repoName = m.activeRepo.Name
		}
	}
	if owner == "" || repoName == "" {
		return func() tea.Msg {
			return views.ReleaseOpResultMsg{Kind: msg.Kind, Err: fmt.Errorf("owner/repo required")}
		}
	}
	o, r := owner, repoName
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		switch msg.Kind {
		case views.ReleaseOpCreate:
			_, err := client.CreateRelease(ctx, o, r, msg.Tag, msg.Name, msg.Body, msg.Draft, msg.Prerelease)
			if err != nil {
				return views.ReleaseOpResultMsg{Kind: msg.Kind, Err: err}
			}
			return views.ReleaseOpResultMsg{Kind: msg.Kind, Message: "Release created"}
		case views.ReleaseOpUpdate:
			_, err := client.UpdateRelease(ctx, o, r, msg.ReleaseID, msg.Tag, msg.Name, msg.Body, msg.Draft, msg.Prerelease)
			if err != nil {
				return views.ReleaseOpResultMsg{Kind: msg.Kind, Err: err}
			}
			return views.ReleaseOpResultMsg{Kind: msg.Kind, Message: "Release updated"}
		case views.ReleaseOpPublish:
			_, err := client.PublishRelease(ctx, o, r, msg.ReleaseID)
			if err != nil {
				return views.ReleaseOpResultMsg{Kind: msg.Kind, Err: err}
			}
			return views.ReleaseOpResultMsg{Kind: msg.Kind, Message: "Release published"}
		case views.ReleaseOpDelete:
			err := client.DeleteRelease(ctx, o, r, msg.ReleaseID)
			if err != nil {
				return views.ReleaseOpResultMsg{Kind: msg.Kind, Err: err}
			}
			return views.ReleaseOpResultMsg{Kind: msg.Kind, Message: "Release deleted"}
		default:
			return views.ReleaseOpResultMsg{Kind: msg.Kind, Err: fmt.Errorf("unknown release op")}
		}
	}
}

func (m *Model) loadRemoteFileContent(owner, name, path, defaultBranch string) tea.Cmd {
	client := m.ghClient
	if client == nil {
		return nil
	}
	ref := defaultBranch
	if ref == "" {
		ref = "HEAD"
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		content, err := client.GetFileContent(ctx, owner, name, path, ref)
		if err != nil {
			return views.FileContentMsg{Path: path, Content: fmt.Sprintf("Error: %v", err)}
		}
		return views.FileContentMsg{Path: path, Content: content}
	}
}

func (m *Model) loadRemoteFileTree(owner, name, defaultBranch string) tea.Cmd {
	client := m.ghClient
	if client == nil {
		return nil
	}
	ref := defaultBranch
	if ref == "" {
		ref = "HEAD"
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		paths, err := client.GetTreeRecursive(ctx, owner, name, ref)
		if err != nil {
			return views.FileTreeDataMsg{Root: nil}
		}
		treeRoot := views.BuildFileTree(paths)
		return views.FileTreeDataMsg{Root: treeRoot}
	}
}

func (m *Model) SetSummary(s *repo.RepoSummary) {
	m.dashboardView.Cockpit().SetSummary(s)
	m.dashboardView.Status().SetSummary(s)
	m.inspectorPane.UpdateRiskSummary(panes.StatusUpdateMsg{Summary: s})
}

func (m *Model) SetBootstrapApp(app bootstrap.App) {
	m.runtimeApp = app
}

func (m *Model) RegisterCommand(name string, handler CommandHandler) {
	m.cmdHandlers[name] = handler
}

func (m *Model) currentBootstrapApp() bootstrap.App {
	app := m.runtimeApp

	repoRoot := ""
	if m.activeRepo != nil && m.activeRepo.IsLocal && m.activeRepo.LocalPath() != "" {
		repoRoot = m.activeRepo.LocalPath()
	}
	if repoRoot == "" {
		repoRoot = strings.TrimSpace(app.RepoRoot)
	}

	loadOpts := config.Options{}
	if repoRoot != "" {
		loadOpts.RepoRoot = repoRoot
		loadOpts.WorkingDir = repoRoot
	} else if wd := strings.TrimSpace(app.Config.Paths.WorkingDir); wd != "" {
		loadOpts.WorkingDir = wd
	}

	if cfg, err := config.Load(loadOpts); err == nil {
		app.Config = cfg
		if repoRoot == "" {
			repoRoot = strings.TrimSpace(cfg.Paths.RepositoryRoot)
		}
	}

	if repoRoot == "" {
		repoRoot = firstNonEmpty(app.Config.Paths.RepositoryRoot, app.RepoRoot)
	}
	app.RepoRoot = repoRoot
	return app
}

func (m *Model) refreshAfterAutonomy(result autonomyexec.Result) tea.Cmd {
	var cmds []tea.Cmd

	if result.Owner != "" && result.Repo != "" && m.ghClient != nil {
		cmds = append(cmds,
			m.fetchRepoDetail(result.Owner, result.Repo),
			m.fetchRepoPRs(result.Owner, result.Repo),
			m.fetchRepoIssues(result.Owner, result.Repo),
			m.buildRepoSummary(result.Owner, result.Repo),
		)
	}

	if result.RepoRoot != "" {
		cmds = append(cmds, m.loadFileTree(), m.loadCommitLog(), m.loadCommitGraph(), m.loadBranchTree())
	} else if m.activeRepo != nil {
		cmds = append(cmds, m.loadCommitLog(), m.loadCommitGraph(), m.loadBranchTree())
	}

	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

func (m *Model) syncInspectorForExplorerTab(tabIdx int) {
	if m.inspectorPane == nil {
		return
	}
	switch tabIdx {
	case 0, 1, 2, 3, 4, 5, 6, 7:
		m.inspectorPane.SetMode(panes.ModeRepoDetail)
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func boolToYesNo(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}
