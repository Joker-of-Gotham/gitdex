package panes

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/your-org/gitdex/internal/tui/render"
	"github.com/your-org/gitdex/internal/tui/theme"
)

type InspectorMode int

const (
	ModeRepoDetail InspectorMode = iota
	ModeRisk
	ModeEvidence
	ModeAudit
	ModeCommitDetail
	ModeBranchDetail
	ModePRDetail
	ModeIssueDetail
	ModeFileDetail
	ModeWorkflowDetail
	ModeDeploymentDetail
	ModeReleaseDetail
)

type RepoDetailData struct {
	Name          string
	Description   string
	Stars         int
	Forks         int
	Language      string
	License       string
	Topics        []string
	DefaultBranch string
	IsPrivate     bool
	CreatedAt     string
	HTMLURL       string
	OpenPRs       int
	OpenIssues    int
	IsLocal       bool
	LocalPaths    []string
	IsFork        bool
	UpstreamURL   string
	GitRemotes    []string
}

type InspectorEvidence struct {
	Timestamp string
	Title     string
	Result    string
	Detail    string
	Success   bool
}

type InspectorContext struct {
	ActiveView string
	Focus      string
	ThemeName  string
	Repo       string
	Branch     string
	Location   string
}

type InspectorCommitData struct {
	Hash    string
	Author  string
	Date    string
	Message string
	Stats   string
	Content string
}

type InspectorBranchData struct {
	Name       string
	Upstream   string
	Ahead      string
	Behind     string
	LastCommit string
}

type InspectorPRData struct {
	Number   int
	Title    string
	State    string
	Author   string
	Reviews  string
	Checks   string
	Labels   string
	Body     string
	Files    []string
	Comments []string
}

type InspectorIssueData struct {
	Number    int
	Title     string
	State     string
	Labels    string
	Assignees string
	Milestone string
	Body      string
	Comments  []string
}

type InspectorFileData struct {
	Path         string
	Size         string
	Language     string
	LastModified string
	Mode         string
	Preview      string
}

type InspectorWorkflowData struct {
	Name       string
	RunID      string
	WorkflowID string
	Status     string
	Conclusion string
	Branch     string
	Event      string
	CreatedAt  string
	URL        string
}

type InspectorDeploymentData struct {
	ID          string
	Environment string
	State       string
	Ref         string
	CreatedAt   string
	URL         string
}

type InspectorReleaseData struct {
	ID          string
	TagName     string
	Name        string
	Draft       string
	Prerelease  string
	CreatedAt   string
	PublishedAt string
	URL         string
	Body        string
}

type InspectorSettingsData struct {
	CurrentSection     string
	Profile            string
	IdentityMode       string
	EffectiveAuth      string
	EffectiveHost      string
	SaveTarget         string
	RecommendedAction  string
	DirtyCount         int
	RepositoryDetected bool
	GlobalConfig       string
	RepoConfig         string
	ActiveFiles        []string
	OverrideFields     []string
	Warnings           []string
}

type InspectorPane struct {
	mode       InspectorMode
	riskPane   *RiskPane
	repoDetail RepoDetailData
	evidence   []InspectorEvidence
	context    InspectorContext
	settings   InspectorSettingsData
	theme      *theme.Theme
	styles     theme.Styles
	width      int
	height     int
	visible    bool
	focused    bool
	vp         viewport.Model

	branchProtectionBranch string
	branchProtectionLines  []string
	branchProtectionErr    error

	commitData InspectorCommitData
	branchData InspectorBranchData
	prData     InspectorPRData
	issueData  InspectorIssueData
	fileData   InspectorFileData
	workflow   InspectorWorkflowData
	deployment InspectorDeploymentData
	release    InspectorReleaseData
}

func NewInspectorPane(t *theme.Theme, s theme.Styles) *InspectorPane {
	rp := NewRiskPane(s)
	return &InspectorPane{
		mode:     ModeRepoDetail,
		riskPane: &rp,
		theme:    t,
		styles:   s,
		visible:  true,
	}
}

func (ip *InspectorPane) SetMode(m InspectorMode) { ip.mode = m }

func (ip *InspectorPane) Toggle() { ip.visible = !ip.visible }

func (ip *InspectorPane) Show() { ip.visible = true }

func (ip *InspectorPane) IsVisible() bool { return ip.visible }

func (ip *InspectorPane) SetSize(w, h int) {
	ip.width = w
	ip.height = h
	contentH := h - 6
	if contentH < 3 {
		contentH = 3
	}
	ip.vp = viewport.New(viewport.WithWidth(w), viewport.WithHeight(contentH))
	if ip.riskPane != nil {
		ip.riskPane.SetSize(w, h-5)
	}
}

func (ip *InspectorPane) SetFocused(f bool) {
	ip.focused = f
	if ip.riskPane != nil {
		ip.riskPane.SetFocused(f)
	}
}

func (ip *InspectorPane) SetStyles(s theme.Styles) {
	ip.styles = s
	if ip.riskPane != nil {
		ip.riskPane.SetStyles(s)
	}
}

func (ip *InspectorPane) SetRepoDetail(d RepoDetailData) {
	ip.repoDetail = d
	ip.mode = ModeRepoDetail
	ip.branchProtectionBranch = ""
	ip.branchProtectionLines = nil
	ip.branchProtectionErr = nil
}

// SetBranchProtection shows branch protection summary for a protected branch (from GitHub API).
func (ip *InspectorPane) SetBranchProtection(branch string, lines []string, err error) {
	ip.branchProtectionBranch = branch
	ip.branchProtectionLines = lines
	ip.branchProtectionErr = err
}

func (ip *InspectorPane) EnrichRepoDetail(d RepoDetailData) {
	if d.Description != "" {
		ip.repoDetail.Description = d.Description
	}
	if d.Stars > 0 {
		ip.repoDetail.Stars = d.Stars
	}
	if d.Forks > 0 {
		ip.repoDetail.Forks = d.Forks
	}
	if d.Language != "" {
		ip.repoDetail.Language = d.Language
	}
	if d.License != "" {
		ip.repoDetail.License = d.License
	}
	if len(d.Topics) > 0 {
		ip.repoDetail.Topics = d.Topics
	}
	if d.DefaultBranch != "" {
		ip.repoDetail.DefaultBranch = d.DefaultBranch
	}
	ip.repoDetail.IsPrivate = d.IsPrivate
	if d.CreatedAt != "" {
		ip.repoDetail.CreatedAt = d.CreatedAt
	}
	if d.HTMLURL != "" {
		ip.repoDetail.HTMLURL = d.HTMLURL
	}
	if d.OpenPRs > 0 {
		ip.repoDetail.OpenPRs = d.OpenPRs
	}
	if d.OpenIssues > 0 {
		ip.repoDetail.OpenIssues = d.OpenIssues
	}
	if d.IsFork {
		ip.repoDetail.IsFork = true
	}
	if strings.TrimSpace(d.UpstreamURL) != "" {
		ip.repoDetail.UpstreamURL = strings.TrimSpace(d.UpstreamURL)
	}
	if len(d.GitRemotes) > 0 {
		ip.repoDetail.GitRemotes = append([]string(nil), d.GitRemotes...)
	}
}

func (ip *InspectorPane) SetEvidence(entries []InspectorEvidence) {
	ip.evidence = entries
}

func (ip *InspectorPane) SetContext(ctx InspectorContext) {
	ip.context = ctx
}

func (ip *InspectorPane) SetSettings(data InspectorSettingsData) {
	ip.settings = data
}

func (ip *InspectorPane) SetCommitDetail(d InspectorCommitData) {
	ip.commitData = d
	ip.mode = ModeCommitDetail
}

func (ip *InspectorPane) SetBranchDetail(d InspectorBranchData) {
	ip.branchData = d
	ip.mode = ModeBranchDetail
}

func (ip *InspectorPane) SetPRDetail(d InspectorPRData) {
	ip.prData = d
	ip.mode = ModePRDetail
}

func (ip *InspectorPane) SetIssueDetail(d InspectorIssueData) {
	ip.issueData = d
	ip.mode = ModeIssueDetail
}

func (ip *InspectorPane) SetFileDetail(d InspectorFileData) {
	ip.fileData = d
	ip.mode = ModeFileDetail
}

func (ip *InspectorPane) SetWorkflowDetail(d InspectorWorkflowData) {
	ip.workflow = d
	ip.mode = ModeWorkflowDetail
}

func (ip *InspectorPane) SetDeploymentDetail(d InspectorDeploymentData) {
	ip.deployment = d
	ip.mode = ModeDeploymentDetail
}

func (ip *InspectorPane) SetReleaseDetail(d InspectorReleaseData) {
	ip.release = d
	ip.mode = ModeReleaseDetail
}

func (ip *InspectorPane) Update(msg tea.Msg) tea.Cmd {
	if !ip.focused {
		return nil
	}
	km, ok := msg.(tea.KeyPressMsg)
	if !ok {
		var cmd tea.Cmd
		ip.vp, cmd = ip.vp.Update(msg)
		return cmd
	}

	if ip.isDetailMode() {
		var cmd tea.Cmd
		ip.vp, cmd = ip.vp.Update(msg)
		return cmd
	}

	switch km.String() {
	case "1":
		ip.mode = ModeRepoDetail
	case "2":
		ip.mode = ModeRisk
	case "3":
		ip.mode = ModeEvidence
	case "4":
		ip.mode = ModeAudit
	default:
		var cmd tea.Cmd
		ip.vp, cmd = ip.vp.Update(msg)
		return cmd
	}

	if ip.mode == ModeRisk && ip.riskPane != nil {
		updated, cmd := ip.riskPane.Update(msg)
		ip.riskPane = &updated
		return cmd
	}
	return nil
}

func (ip *InspectorPane) View() string {
	if !ip.visible {
		return ""
	}

	settingsActive := strings.EqualFold(ip.context.ActiveView, "Settings")
	content := ""
	switch ip.mode {
	case ModeRepoDetail:
		if settingsActive {
			content = ip.renderSettingsPrimary()
		} else {
			content = ip.renderRepoDetail()
		}
	case ModeRisk:
		if settingsActive {
			content = ip.renderSettingsChecks()
		} else if ip.riskPane != nil {
			content = ip.riskPane.View()
		}
	case ModeEvidence:
		if settingsActive {
			content = ip.renderSettingsSources()
		} else {
			content = ip.renderEvidence()
		}
	case ModeAudit:
		content = ip.renderContext()
	default:
		content = ip.renderContext()
	}

	switch ip.mode {
	case ModeCommitDetail:
		content = ip.renderCommitDetail()
	case ModeBranchDetail:
		content = ip.renderBranchDetail()
	case ModePRDetail:
		content = ip.renderPRDetailContent()
	case ModeIssueDetail:
		content = ip.renderIssueDetailContent()
	case ModeFileDetail:
		content = ip.renderFileDetailContent()
	case ModeWorkflowDetail:
		content = ip.renderWorkflowDetailContent()
	case ModeDeploymentDetail:
		content = ip.renderDeploymentDetailContent()
	case ModeReleaseDetail:
		content = ip.renderReleaseDetailContent()
	}

	ip.vp.SetContent(content)
	footer := "1 Repo | 2 Risk | 3 Evidence | 4 Context"
	if settingsActive {
		footer = "1 Config | 2 Checks | 3 Sources | 4 Context"
	}
	switch ip.mode {
	case ModeCommitDetail:
		footer = "Commit Detail"
	case ModeBranchDetail:
		footer = "Branch Detail"
	case ModePRDetail:
		footer = "PR Detail"
	case ModeIssueDetail:
		footer = "Issue Detail"
	case ModeFileDetail:
		footer = "File Detail"
	case ModeWorkflowDetail:
		footer = "Workflow Detail"
	case ModeDeploymentDetail:
		footer = "Deployment Detail"
	case ModeReleaseDetail:
		footer = "Release Detail"
	}
	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(ip.theme.Primary()).Render(theme.Icons.Search + " Inspector"),
	}
	if tabs := ip.renderModeTabs(); tabs != "" {
		lines = append(lines, tabs)
	}
	lines = append(lines,
		ip.vp.View(),
		lipgloss.NewStyle().Foreground(ip.theme.DimText()).Render(footer),
	)
	return strings.Join(lines, "\n")
}

func (ip *InspectorPane) renderModeTabs() string {
	if ip.isDetailMode() {
		return lipgloss.NewStyle().
			Foreground(ip.theme.DimText()).
			Italic(true).
			Render("Context-locked detail surface. Use Esc in the main view to return to the parent list.")
	}

	type tab struct {
		key   string
		label string
		mode  InspectorMode
		icon  string
	}
	settingsActive := strings.EqualFold(ip.context.ActiveView, "Settings")
	tabs := []tab{
		{key: "1", label: ternary(settingsActive, "Config", "Repository"), mode: ModeRepoDetail, icon: theme.Icons.Branch},
		{key: "2", label: ternary(settingsActive, "Checks", "Risk"), mode: ModeRisk, icon: theme.Icons.Warning},
		{key: "3", label: ternary(settingsActive, "Sources", "Evidence"), mode: ModeEvidence, icon: theme.Icons.Evidence},
		{key: "4", label: "Audit", mode: ModeAudit, icon: theme.Icons.Help},
	}

	rendered := make([]string, 0, len(tabs))
	for _, item := range tabs {
		if item.mode == ip.mode {
			rendered = append(rendered, lipgloss.NewStyle().
				Bold(true).
				Foreground(ip.theme.OnPrimary()).
				Background(ip.theme.Secondary()).
				Padding(0, 1).
				Render(item.key+" "+item.icon+" "+item.label))
			continue
		}
		rendered = append(rendered, lipgloss.NewStyle().
			Foreground(ip.theme.MutedFg()).
			Background(ip.theme.Surface()).
			Padding(0, 1).
			Render(item.key+" "+item.label))
	}
	return strings.Join(rendered, " ")
}

func (ip *InspectorPane) isDetailMode() bool {
	switch ip.mode {
	case ModeCommitDetail, ModeBranchDetail, ModePRDetail, ModeIssueDetail, ModeFileDetail, ModeWorkflowDetail, ModeDeploymentDetail, ModeReleaseDetail:
		return true
	default:
		return false
	}
}

func (ip *InspectorPane) renderRepoDetail() string {
	if ip.repoDetail.Name == "" {
		return lipgloss.NewStyle().
			Foreground(ip.theme.DimText()).
			Render("Select a repository to open the live context panel.")
	}

	visibility := "Public"
	visibilityColor := ip.theme.Success()
	if ip.repoDetail.IsPrivate {
		visibility = "Private"
		visibilityColor = ip.theme.Warning()
	}

	parts := []string{
		ip.card("Identity", []string{
			lipgloss.NewStyle().Bold(true).Foreground(ip.theme.Fg()).Render(ip.repoDetail.Name),
			ip.badge(visibility, visibilityColor),
			ip.renderOptional(ip.repoDetail.Description),
		}),
		ip.card("Signals", []string{
			ip.metricRow("Stars", fmt.Sprintf("%d", ip.repoDetail.Stars)),
			ip.metricRow("Forks", fmt.Sprintf("%d", ip.repoDetail.Forks)),
			ip.metricRow("PRs", fmt.Sprintf("%d", ip.repoDetail.OpenPRs)),
			ip.metricRow("Issues", fmt.Sprintf("%d", ip.repoDetail.OpenIssues)),
			ip.metricRow("Language", ip.valueOrDash(ip.repoDetail.Language)),
			ip.metricRow("License", ip.valueOrDash(ip.repoDetail.License)),
		}),
		ip.contextCardLines(),
	}

	if len(ip.repoDetail.Topics) > 0 {
		parts = append(parts, ip.card("Topics", []string{
			lipgloss.NewStyle().Foreground(ip.theme.Info()).Render(strings.Join(ip.repoDetail.Topics, "  ")),
		}))
	}
	if ip.branchProtectionBranch != "" || ip.branchProtectionErr != nil || len(ip.branchProtectionLines) > 0 {
		blines := []string{}
		if ip.branchProtectionErr != nil {
			blines = append(blines, lipgloss.NewStyle().Foreground(ip.theme.Warning()).Render(ip.branchProtectionErr.Error()))
		} else {
			blines = append(blines, lipgloss.NewStyle().Foreground(ip.theme.Fg()).Render("Branch: "+ip.branchProtectionBranch))
			for _, line := range ip.branchProtectionLines {
				blines = append(blines, lipgloss.NewStyle().Foreground(ip.theme.MutedFg()).Render(line))
			}
		}
		parts = append(parts, ip.card("Branch protection", blines))
	}
	return strings.Join(parts, "\n\n")
}

func (ip *InspectorPane) contextCardLines() string {
	mode := "remote-only"
	if ip.repoDetail.IsLocal {
		mode = "local writable"
	}
	lines := []string{
		ip.metricRow("Branch", ip.valueOrDash(ip.repoDetail.DefaultBranch)),
		ip.metricRow("Created", ip.valueOrDash(ip.repoDetail.CreatedAt)),
		ip.metricRow("Mode", mode),
		ip.metricRow("Path", ip.valueOrDash(strings.Join(ip.repoDetail.LocalPaths, ", "))),
		ip.metricRow("URL", ip.valueOrDash(ip.repoDetail.HTMLURL)),
	}
	if ip.repoDetail.IsFork {
		lines = append(lines, ip.metricRow("Fork", "yes (GitHub fork)"))
	}
	if strings.TrimSpace(ip.repoDetail.UpstreamURL) != "" {
		lines = append(lines, ip.metricRow("Upstream", ip.repoDetail.UpstreamURL))
	}
	if len(ip.repoDetail.GitRemotes) > 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(ip.theme.DimText()).Render("Remotes:"))
		for _, r := range ip.repoDetail.GitRemotes {
			lines = append(lines, lipgloss.NewStyle().Foreground(ip.theme.MutedFg()).Render("  "+r))
		}
	}
	return ip.card("Context", lines)
}

func (ip *InspectorPane) renderSettingsPrimary() string {
	next := strings.TrimSpace(ip.settings.RecommendedAction)
	if next == "" {
		next = "No settings guidance available."
	}

	return strings.Join([]string{
		ip.card("Configuration", []string{
			ip.metricRow("Section", ip.valueOrDash(ip.settings.CurrentSection)),
			ip.metricRow("Profile", ip.valueOrDash(ip.settings.Profile)),
			ip.metricRow("Mode", ip.valueOrDash(ip.settings.IdentityMode)),
			ip.metricRow("Auth", ip.valueOrDash(ip.settings.EffectiveAuth)),
			ip.metricRow("Host", ip.valueOrDash(ip.settings.EffectiveHost)),
			ip.metricRow("Save", ip.valueOrDash(ip.settings.SaveTarget)),
			ip.metricRow("Dirty", fmt.Sprintf("%d", ip.settings.DirtyCount)),
			ip.metricRow("Repo", ternary(ip.settings.RepositoryDetected, "Detected", "Not detected")),
		}),
		ip.card("Next Action", []string{
			lipgloss.NewStyle().Foreground(ip.theme.Fg()).Render(next),
		}),
	}, "\n\n")
}

func (ip *InspectorPane) renderSettingsChecks() string {
	if len(ip.settings.Warnings) == 0 {
		return ip.card("Checks", []string{
			lipgloss.NewStyle().Foreground(ip.theme.Success()).Render(theme.Icons.Check + " No blocking configuration gaps detected."),
		})
	}

	lines := make([]string, 0, len(ip.settings.Warnings))
	for _, warning := range ip.settings.Warnings {
		lines = append(lines, lipgloss.NewStyle().Foreground(ip.theme.Warning()).Render(theme.Icons.Warning+" "+warning))
	}
	return ip.card("Checks", lines)
}

func (ip *InspectorPane) renderSettingsSources() string {
	activeFiles := []string{lipgloss.NewStyle().Foreground(ip.theme.DimText()).Render("No active config files detected.")}
	if len(ip.settings.ActiveFiles) > 0 {
		activeFiles = make([]string, 0, len(ip.settings.ActiveFiles))
		for _, path := range ip.settings.ActiveFiles {
			activeFiles = append(activeFiles, lipgloss.NewStyle().Foreground(ip.theme.Fg()).Render(path))
		}
	}

	overrides := []string{lipgloss.NewStyle().Foreground(ip.theme.DimText()).Render("No env or flag overrides detected.")}
	if len(ip.settings.OverrideFields) > 0 {
		overrides = make([]string, 0, len(ip.settings.OverrideFields))
		for _, label := range ip.settings.OverrideFields {
			overrides = append(overrides, lipgloss.NewStyle().Foreground(ip.theme.Warning()).Render(label))
		}
	}

	return strings.Join([]string{
		ip.card("Active Files", activeFiles),
		ip.card("Overrides", overrides),
		ip.card("Targets", []string{
			ip.metricRow("Global", ip.valueOrDash(ip.settings.GlobalConfig)),
			ip.metricRow("Repo", ip.valueOrDash(ip.settings.RepoConfig)),
		}),
	}, "\n\n")
}

func (ip *InspectorPane) renderEvidence() string {
	if len(ip.evidence) == 0 {
		return lipgloss.NewStyle().
			Foreground(ip.theme.DimText()).
			Render("Evidence appears here after repository health and activity signals are loaded.")
	}

	lines := make([]string, 0, len(ip.evidence)*3)
	for _, item := range ip.evidence {
		color := ip.theme.Success()
		icon := theme.Icons.Check
		if !item.Success {
			color = ip.theme.Warning()
			icon = theme.Icons.Warning
		}
		lines = append(lines,
			lipgloss.NewStyle().Foreground(ip.theme.Timestamp()).Render(item.Timestamp)+"  "+
				lipgloss.NewStyle().Bold(true).Foreground(ip.theme.Fg()).Render(item.Title),
			lipgloss.NewStyle().Foreground(color).Render(icon+" "+strings.ToUpper(item.Result)),
			lipgloss.NewStyle().Foreground(ip.theme.MutedFg()).Render(item.Detail),
			"",
		)
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func (ip *InspectorPane) renderContext() string {
	rows := []string{
		ip.metricRow("View", ip.valueOrDash(ip.context.ActiveView)),
		ip.metricRow("Focus", ip.valueOrDash(ip.context.Focus)),
		ip.metricRow("Theme", ip.valueOrDash(ip.context.ThemeName)),
		ip.metricRow("Repo", ip.valueOrDash(ip.context.Repo)),
		ip.metricRow("Branch", ip.valueOrDash(ip.context.Branch)),
		ip.metricRow("Location", ip.valueOrDash(ip.context.Location)),
	}

	hotkeys := []string{
		"F1-F5  switch workspaces",
		"Tab    cycle focus",
		"Ctrl+1 nav / Ctrl+2 main / Ctrl+3 inspector",
		"Ctrl+P command palette",
		"Ctrl+I toggle inspector",
	}

	return strings.Join([]string{
		ip.card("Session", rows),
		ip.card("Hotkeys", hotkeys),
	}, "\n\n")
}

func (ip *InspectorPane) metricRow(label, value string) string {
	return lipgloss.NewStyle().Foreground(ip.theme.DimText()).Render(label+": ") +
		lipgloss.NewStyle().Foreground(ip.theme.Fg()).Render(value)
}

func (ip *InspectorPane) renderOptional(value string) string {
	if strings.TrimSpace(value) == "" {
		return lipgloss.NewStyle().Foreground(ip.theme.DimText()).Italic(true).Render("No description")
	}
	return lipgloss.NewStyle().Foreground(ip.theme.MutedFg()).Italic(true).Render(value)
}

func (ip *InspectorPane) badge(label string, bg color.Color) string {
	return lipgloss.NewStyle().
		Foreground(ip.theme.OnPrimary()).
		Background(bg).
		Padding(0, 1).
		Render(label)
}

func (ip *InspectorPane) card(title string, lines []string) string {
	content := lipgloss.NewStyle().Bold(true).Foreground(ip.theme.Primary()).Render(title) + "\n" + strings.Join(lines, "\n")
	return render.SurfacePanel(content, max(24, ip.width-2), ip.theme.Surface(), ip.theme.BorderColor())
}

func (ip *InspectorPane) valueOrDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}

func (ip *InspectorPane) UpdateRiskSummary(msg StatusUpdateMsg) {
	if ip.riskPane != nil {
		updated, _ := ip.riskPane.Update(msg)
		ip.riskPane = &updated
	}
}

func ternary[T any](condition bool, whenTrue, whenFalse T) T {
	if condition {
		return whenTrue
	}
	return whenFalse
}

func (ip *InspectorPane) renderCommitDetail() string {
	d := ip.commitData
	parts := []string{
		ip.card("Commit", []string{
			ip.kv("Hash", ip.valueOrDash(d.Hash)),
			ip.kv("Author", ip.valueOrDash(d.Author)),
			ip.kv("Date", ip.valueOrDash(d.Date)),
		}),
		ip.card("Message", []string{
			lipgloss.NewStyle().Foreground(ip.theme.Fg()).Width(maxInt(16, ip.width-6)).Render(ip.valueOrDash(d.Message)),
		}),
		ip.card("Stats", []string{
			lipgloss.NewStyle().Foreground(ip.theme.MutedFg()).Width(maxInt(16, ip.width-6)).Render(ip.valueOrDash(d.Stats)),
		}),
	}
	if strings.TrimSpace(d.Content) != "" {
		parts = append(parts, ip.card("Raw Detail", []string{
			lipgloss.NewStyle().Foreground(ip.theme.Fg()).Width(maxInt(16, ip.width-6)).Render(d.Content),
		}))
	}
	return strings.Join(parts, "\n\n")
}

func (ip *InspectorPane) renderBranchDetail() string {
	d := ip.branchData
	return ip.card("Branch", []string{
		ip.kv("Name", ip.valueOrDash(d.Name)),
		ip.kv("Upstream", ip.valueOrDash(d.Upstream)),
		ip.kv("Ahead", ip.valueOrDash(d.Ahead)),
		ip.kv("Behind", ip.valueOrDash(d.Behind)),
		ip.kv("Last Commit", ip.valueOrDash(d.LastCommit)),
	})
}

func (ip *InspectorPane) renderPRDetailContent() string {
	d := ip.prData
	parts := []string{
		ip.card(fmt.Sprintf("PR #%d", d.Number), []string{
			ip.kv("Title", ip.valueOrDash(d.Title)),
			ip.kv("State", ip.valueOrDash(d.State)),
			ip.kv("Author", ip.valueOrDash(d.Author)),
			ip.kv("Reviews", ip.valueOrDash(d.Reviews)),
			ip.kv("Checks", ip.valueOrDash(d.Checks)),
			ip.kv("Labels", ip.valueOrDash(d.Labels)),
		}),
	}
	if strings.TrimSpace(d.Body) != "" {
		parts = append(parts, ip.card("Body", []string{
			lipgloss.NewStyle().Foreground(ip.theme.Fg()).Width(maxInt(16, ip.width-6)).Render(d.Body),
		}))
	}
	if len(d.Files) > 0 {
		parts = append(parts, ip.card("Changed Files", d.Files))
	}
	if len(d.Comments) > 0 {
		parts = append(parts, ip.card("Recent Comments", d.Comments))
	}
	return strings.Join(parts, "\n\n")
}

func (ip *InspectorPane) renderIssueDetailContent() string {
	d := ip.issueData
	parts := []string{
		ip.card(fmt.Sprintf("Issue #%d", d.Number), []string{
			ip.kv("Title", ip.valueOrDash(d.Title)),
			ip.kv("State", ip.valueOrDash(d.State)),
			ip.kv("Labels", ip.valueOrDash(d.Labels)),
			ip.kv("Assignees", ip.valueOrDash(d.Assignees)),
			ip.kv("Milestone", ip.valueOrDash(d.Milestone)),
		}),
	}
	if strings.TrimSpace(d.Body) != "" {
		parts = append(parts, ip.card("Body", []string{
			lipgloss.NewStyle().Foreground(ip.theme.Fg()).Width(maxInt(16, ip.width-6)).Render(d.Body),
		}))
	}
	if len(d.Comments) > 0 {
		parts = append(parts, ip.card("Recent Comments", d.Comments))
	}
	return strings.Join(parts, "\n\n")
}

func (ip *InspectorPane) renderFileDetailContent() string {
	d := ip.fileData
	parts := []string{
		ip.card("File", []string{
			ip.kv("Path", ip.valueOrDash(d.Path)),
			ip.kv("Size", ip.valueOrDash(d.Size)),
			ip.kv("Language", ip.valueOrDash(d.Language)),
			ip.kv("Modified", ip.valueOrDash(d.LastModified)),
			ip.kv("Mode", ip.valueOrDash(d.Mode)),
		}),
	}
	if strings.TrimSpace(d.Preview) != "" {
		parts = append(parts, ip.card("Preview", []string{
			lipgloss.NewStyle().Foreground(ip.theme.Fg()).Width(maxInt(16, ip.width-6)).Render(d.Preview),
		}))
	}
	return strings.Join(parts, "\n\n")
}

func (ip *InspectorPane) renderWorkflowDetailContent() string {
	d := ip.workflow
	return ip.card("Workflow", []string{
		ip.kv("Name", ip.valueOrDash(d.Name)),
		ip.kv("Run ID", ip.valueOrDash(d.RunID)),
		ip.kv("Workflow ID", ip.valueOrDash(d.WorkflowID)),
		ip.kv("Status", ip.valueOrDash(d.Status)),
		ip.kv("Conclusion", ip.valueOrDash(d.Conclusion)),
		ip.kv("Branch", ip.valueOrDash(d.Branch)),
		ip.kv("Event", ip.valueOrDash(d.Event)),
		ip.kv("Created", ip.valueOrDash(d.CreatedAt)),
		ip.kv("URL", ip.valueOrDash(d.URL)),
	})
}

func (ip *InspectorPane) renderDeploymentDetailContent() string {
	d := ip.deployment
	return ip.card("Deployment", []string{
		ip.kv("ID", ip.valueOrDash(d.ID)),
		ip.kv("Environment", ip.valueOrDash(d.Environment)),
		ip.kv("State", ip.valueOrDash(d.State)),
		ip.kv("Ref", ip.valueOrDash(d.Ref)),
		ip.kv("Created", ip.valueOrDash(d.CreatedAt)),
		ip.kv("URL", ip.valueOrDash(d.URL)),
	})
}

func (ip *InspectorPane) renderReleaseDetailContent() string {
	d := ip.release
	parts := []string{
		ip.card("Release", []string{
			ip.kv("ID", ip.valueOrDash(d.ID)),
			ip.kv("Tag", ip.valueOrDash(d.TagName)),
			ip.kv("Name", ip.valueOrDash(d.Name)),
			ip.kv("Draft", ip.valueOrDash(d.Draft)),
			ip.kv("Prerelease", ip.valueOrDash(d.Prerelease)),
			ip.kv("Created", ip.valueOrDash(d.CreatedAt)),
			ip.kv("Published", ip.valueOrDash(d.PublishedAt)),
			ip.kv("URL", ip.valueOrDash(d.URL)),
		}),
	}
	if strings.TrimSpace(d.Body) != "" {
		parts = append(parts, ip.card("Notes", []string{
			lipgloss.NewStyle().Foreground(ip.theme.Fg()).Width(maxInt(16, ip.width-6)).Render(d.Body),
		}))
	}
	return strings.Join(parts, "\n\n")
}

func (ip *InspectorPane) kv(key, val string) string {
	return lipgloss.NewStyle().Foreground(ip.theme.DimText()).Render(key+": ") +
		lipgloss.NewStyle().Foreground(ip.theme.Fg()).Render(val)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
