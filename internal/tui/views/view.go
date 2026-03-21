package views

import tea "charm.land/bubbletea/v2"

type ID string

const (
	ViewDashboard ID = "dashboard"
	ViewChat      ID = "chat"
	ViewExplorer  ID = "explorer"
	ViewWorkspace ID = "workspace"
	ViewSettings  ID = "settings"

	ViewStatus   ID = "status"
	ViewCockpit  ID = "cockpit"
	ViewPlans    ID = "plans"
	ViewTasks    ID = "tasks"
	ViewEvidence ID = "evidence"
	ViewPulls    ID = "pulls"
	ViewIssues   ID = "issues"
	ViewFiles    ID = "files"

	ViewReflog            ID = "reflog"
	ViewInteractiveRebase ID = "interactive_rebase"
	ViewBisect            ID = "bisect"

	ViewSubmodules  ID = "submodules"
	ViewCommitGraph ID = "commit_graph"
	ViewRemotes     ID = "remotes"
	ViewConflict    ID = "conflict"

	ViewCruiseStatus  ID = "cruise_status"
	ViewApprovalQueue ID = "approval_queue"
)

type View interface {
	Init() tea.Cmd
	Update(msg tea.Msg) (View, tea.Cmd)
	Render() string
	SetSize(width, height int)
	ID() ID
	Title() string
}

type SwitchViewMsg struct {
	Target ID
}
