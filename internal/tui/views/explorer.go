package views

import (
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/your-org/gitdex/internal/tui/theme"
)

type ExplorerSubTabChangedMsg struct {
	TabName string
	TabIdx  int
}

type ExplorerView struct {
	theme       *theme.Theme
	width       int
	height      int
	pulls       *PullsView
	issues      *IssuesView
	files       *FilesView
	commits     *CommitLogView
	branches    *BranchTreeView
	workflows   *WorkflowsView
	deployments *DeploymentsView
	releases    *ReleasesView
	active      int
	tabs        []string
}

func NewExplorerView(t *theme.Theme) *ExplorerView {
	return &ExplorerView{
		theme:       t,
		pulls:       NewPullsView(t),
		issues:      NewIssuesView(t),
		files:       NewFilesView(t),
		commits:     NewCommitLogView(t),
		branches:    NewBranchTreeView(t),
		workflows:   NewWorkflowsView(t),
		deployments: NewDeploymentsView(t),
		releases:    NewReleasesView(t),
		tabs:        []string{"Pull Requests", "Issues", "Files", "Commits", "Branches", "Workflows", "Deployments", "Releases"},
	}
}

func (v *ExplorerView) ID() ID        { return ViewExplorer }
func (v *ExplorerView) Title() string { return "Explorer" }
func (v *ExplorerView) Init() tea.Cmd { return nil }

func (v *ExplorerView) Pulls() *PullsView             { return v.pulls }
func (v *ExplorerView) Issues() *IssuesView           { return v.issues }
func (v *ExplorerView) Files() *FilesView             { return v.files }
func (v *ExplorerView) Commits() *CommitLogView       { return v.commits }
func (v *ExplorerView) Branches() *BranchTreeView     { return v.branches }
func (v *ExplorerView) Workflows() *WorkflowsView     { return v.workflows }
func (v *ExplorerView) Deployments() *DeploymentsView { return v.deployments }
func (v *ExplorerView) Releases() *ReleasesView       { return v.releases }
func (v *ExplorerView) ActiveTab() int                { return v.active }

func (v *ExplorerView) SetSize(w, h int) {
	v.width = w
	v.height = h
	subH := max(h-3, 5)
	v.pulls.SetSize(w, subH)
	v.issues.SetSize(w, subH)
	v.files.SetSize(w, subH)
	v.commits.SetSize(w, subH)
	v.branches.SetSize(w, subH)
	v.workflows.SetSize(w, subH)
	v.deployments.SetSize(w, subH)
	v.releases.SetSize(w, subH)
}

func (v *ExplorerView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case PullsDataMsg:
		v.pulls.SetItems(msg.Items)
		return v, nil
	case PRDetailMsg:
		_, cmd := v.pulls.Update(msg)
		return v, cmd
	case PRActionResultMsg:
		_, cmd := v.pulls.Update(msg)
		return v, cmd
	case IssuesDataMsg:
		v.issues.SetItems(msg.Items)
		return v, nil
	case IssueDetailMsg:
		_, cmd := v.issues.Update(msg)
		return v, cmd
	case IssueActionResultMsg:
		_, cmd := v.issues.Update(msg)
		return v, cmd
	case FileTreeDataMsg:
		v.files.SetTree(msg.Root)
		return v, nil
	case FileContentMsg:
		v.files.Update(msg)
		return v, nil
	case FileDiffMsg:
		v.files.Update(msg)
		return v, nil
	case FileEditMsg:
		_, cmd := v.files.Update(msg)
		return v, cmd
	case FileSavedMsg:
		_, cmd := v.files.Update(msg)
		return v, cmd
	case FileOpResultMsg:
		_, cmd := v.files.Update(msg)
		return v, cmd
	case BatchFileOpResultMsg:
		_, cmd := v.files.Update(msg)
		return v, cmd
	case CommitLogDataMsg:
		v.commits.SetCommits(msg.Commits)
		return v, nil
	case CommitGraphMsg:
		_, cmd := v.commits.Update(msg)
		return v, cmd
	case CommitDetailMsg:
		_, cmd := v.commits.Update(msg)
		return v, cmd
	case CommitActionResultMsg:
		_, cmd := v.commits.Update(msg)
		return v, cmd
	case BranchTreeDataMsg:
		v.branches.SetBranches(msg.Branches)
		return v, nil
	case BranchCheckoutResultMsg:
		_, cmd := v.branches.Update(msg)
		return v, cmd
	case BranchActionResultMsg:
		_, cmd := v.branches.Update(msg)
		return v, cmd
	case WorkflowRunsDataMsg:
		v.workflows.SetRuns(msg.Runs)
		return v, nil
	case WorkflowDispatchResultMsg:
		_, cmd := v.workflows.Update(msg)
		return v, cmd
	case DeploymentDataMsg:
		v.deployments.SetDeployments(msg.Deployments)
		return v, nil
	case ReleaseListMsg:
		_, cmd := v.releases.Update(msg)
		return v, cmd
	case ReleaseOpResultMsg:
		_, cmd := v.releases.Update(msg)
		return v, cmd
	case tea.KeyPressMsg:
		k := msg.String()
		switch k {
		case "[":
			mega := explorerMegaForFlat(v.active)
			prevMega := (mega + len(explorerMegaGroups) - 1) % len(explorerMegaGroups)
			v.active = explorerMegaGroups[prevMega][0]
			return v, v.emitTabChanged()
		case "]":
			mega := explorerMegaForFlat(v.active)
			nextMega := (mega + 1) % len(explorerMegaGroups)
			v.active = explorerMegaGroups[nextMega][0]
			return v, v.emitTabChanged()
		case "left":
			mega := explorerMegaForFlat(v.active)
			g := explorerMegaGroups[mega]
			pos := explorerSlotForFlat(v.active)
			if pos > 0 {
				v.active = g[pos-1]
			} else {
				v.active = g[len(g)-1]
			}
			return v, v.emitTabChanged()
		case "right":
			mega := explorerMegaForFlat(v.active)
			g := explorerMegaGroups[mega]
			pos := explorerSlotForFlat(v.active)
			if pos < len(g)-1 {
				v.active = g[pos+1]
			} else {
				v.active = g[0]
			}
			return v, v.emitTabChanged()
		default:
			if len(k) == 1 && k[0] >= '1' && k[0] <= '9' {
				n, err := strconv.Atoi(k)
				if err == nil {
					mega := explorerMegaForFlat(v.active)
					g := explorerMegaGroups[mega]
					if n >= 1 && n <= len(g) {
						v.active = g[n-1]
						return v, v.emitTabChanged()
					}
				}
			}
		}
		var cmd tea.Cmd
		switch v.active {
		case 0:
			_, cmd = v.pulls.Update(msg)
		case 1:
			_, cmd = v.issues.Update(msg)
		case 2:
			_, cmd = v.files.Update(msg)
		case 3:
			_, cmd = v.commits.Update(msg)
		case 4:
			_, cmd = v.branches.Update(msg)
		case 5:
			_, cmd = v.workflows.Update(msg)
		case 6:
			_, cmd = v.deployments.Update(msg)
		case 7:
			_, cmd = v.releases.Update(msg)
		}
		return v, cmd
	}
	return v, nil
}

func (v *ExplorerView) emitTabChanged() tea.Cmd {
	name := ""
	if v.active >= 0 && v.active < len(v.tabs) {
		name = v.tabs[v.active]
	}
	idx := v.active
	return func() tea.Msg { return ExplorerSubTabChangedMsg{TabName: name, TabIdx: idx} }
}

func (v *ExplorerView) ActiveTabName() string {
	if v.active >= 0 && v.active < len(v.tabs) {
		return v.tabs[v.active]
	}
	return ""
}

func (v *ExplorerView) Render() string {
	if v.width == 0 || v.height == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(renderSubTabsWithOptions(v.tabs, v.active, v.theme, v.width, &SubTabRenderOptions{
		MegaGroups: explorerMegaGroups,
		MegaLabels: explorerMegaLabels,
	}))
	b.WriteString("\n")
	switch v.active {
	case 0:
		b.WriteString(v.pulls.Render())
	case 1:
		b.WriteString(v.issues.Render())
	case 2:
		b.WriteString(v.files.Render())
	case 3:
		b.WriteString(v.commits.Render())
	case 4:
		b.WriteString(v.branches.Render())
	case 5:
		b.WriteString(v.workflows.Render())
	case 6:
		b.WriteString(v.deployments.Render())
	case 7:
		b.WriteString(v.releases.Render())
	}
	return b.String()
}
