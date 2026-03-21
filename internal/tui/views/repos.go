package views

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/your-org/gitdex/internal/tui/render"
	"github.com/your-org/gitdex/internal/tui/theme"
)

const linesPerRepo = 7

type ReposView struct {
	t        *theme.Theme
	items    []RepoListItem
	filtered []RepoListItem
	cursor   int
	width    int
	height   int
	search   string
	loading  bool
}

func NewReposView(t *theme.Theme) *ReposView {
	return &ReposView{t: t, loading: true}
}

func (v *ReposView) ID() ID        { return "repos" }
func (v *ReposView) Title() string { return "Repositories" }
func (v *ReposView) Init() tea.Cmd { return nil }

func (v *ReposView) SetItems(items []RepoListItem) {
	v.items = items
	v.loading = false
	v.applyFilter()
}

func (v *ReposView) viewportHeight() int {
	overhead := 5
	h := v.height - overhead
	if h < linesPerRepo {
		h = linesPerRepo
	}
	return max(1, h/linesPerRepo)
}

func (v *ReposView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case RepoListMsg:
		v.SetItems(msg.Repos)
		return v, nil
	case tea.KeyPressMsg:
		switch msg.String() {
		case "up", "k":
			if v.cursor > 0 {
				v.cursor--
			}
		case "down", "j":
			if v.cursor < len(v.filtered)-1 {
				v.cursor++
			}
		case "pgup":
			v.cursor -= v.viewportHeight()
			if v.cursor < 0 {
				v.cursor = 0
			}
		case "pgdown":
			v.cursor += v.viewportHeight()
			if v.cursor >= len(v.filtered) {
				v.cursor = max(0, len(v.filtered)-1)
			}
		case "g":
			v.cursor = 0
		case "G":
			if len(v.filtered) > 0 {
				v.cursor = len(v.filtered) - 1
			}
		case "enter":
			if v.cursor < len(v.filtered) {
				return v, func() tea.Msg { return RepoSelectMsg{Repo: v.filtered[v.cursor]} }
			}
		case "c":
			if v.cursor < len(v.filtered) && !v.filtered[v.cursor].IsLocal {
				return v, func() tea.Msg { return CloneRepoRequestMsg{Repo: v.filtered[v.cursor]} }
			}
		case "backspace":
			if len(v.search) > 0 {
				v.search = v.search[:len(v.search)-1]
				v.applyFilter()
			}
		default:
			r := []rune(msg.String())
			if len(r) == 1 && r[0] >= 32 {
				v.search += msg.String()
				v.applyFilter()
			}
		}
	}
	return v, nil
}

func (v *ReposView) Render() string {
	if v.width < 10 || v.height < 3 {
		return ""
	}
	if v.loading {
		return render.SurfacePanel(
			lipgloss.NewStyle().Bold(true).Foreground(v.t.Primary()).Render("Repository Index")+"\n\n"+
				lipgloss.NewStyle().Foreground(v.t.MutedFg()).Render(theme.Icons.Spinner[0]+" Loading repositories..."),
			maxInt(24, v.width),
			v.t.Surface(),
			v.t.BorderColor(),
		)
	}
	if len(v.items) == 0 {
		return render.SurfacePanel(
			lipgloss.NewStyle().Bold(true).Foreground(v.t.Primary()).Render("Repository Index")+"\n\n"+
				lipgloss.NewStyle().Foreground(v.t.DimText()).Render("No repositories detected yet.")+"\n"+
				lipgloss.NewStyle().Foreground(v.t.MutedFg()).Width(maxInt(24, v.width-8)).
					Render("Configure GitHub access in Settings and keep a local clone open when you want file, branch, and worktree context to join remote repository metadata."),
			maxInt(24, v.width),
			v.t.Surface(),
			v.t.BorderColor(),
		)
	}

	if v.width >= 110 {
		listWidth := maxInt(40, v.width*55/100)
		detailWidth := maxInt(32, v.width-listWidth-1)
		return lipgloss.JoinHorizontal(
			lipgloss.Top,
			v.renderList(listWidth),
			" ",
			v.renderDetail(detailWidth),
		)
	}
	return strings.Join([]string{
		v.renderList(v.width),
		v.renderDetail(v.width),
	}, "\n\n")
}

func (v *ReposView) renderList(width int) string {
	rows := []string{
		lipgloss.NewStyle().Bold(true).Foreground(v.t.Primary()).Render("Repository Index"),
		lipgloss.NewStyle().Foreground(v.t.DimText()).Render("Type to filter  |  Enter open  |  C clone remote repo  |  PgUp/PgDn page"),
		lipgloss.NewStyle().Foreground(v.t.MutedFg()).Render("Query: " + valueOrDash(v.search)),
		"",
	}

	visible := v.viewportHeight()
	start := 0
	if v.cursor >= visible {
		start = v.cursor - visible + 1
	}
	end := minInt(len(v.filtered), start+visible)
	cardWidth := maxInt(28, width-4)

	for i := start; i < end; i++ {
		rows = append(rows, v.renderRepoCard(v.filtered[i], i == v.cursor, cardWidth))
	}
	rows = append(rows, "", lipgloss.NewStyle().Foreground(v.t.DimText()).Render(fmt.Sprintf("%d/%d shown", len(v.filtered), len(v.items))))
	return render.SurfacePanel(strings.Join(rows, "\n"), width, v.t.Surface(), v.t.BorderColor())
}

func (v *ReposView) renderDetail(width int) string {
	selected := v.SelectedRepo()
	if selected == nil {
		return render.SurfacePanel(
			lipgloss.NewStyle().Bold(true).Foreground(v.t.Primary()).Render("Repository Detail")+"\n\n"+
				lipgloss.NewStyle().Foreground(v.t.DimText()).Render("Select a repository to inspect remote signals and local clone context."),
			width,
			v.t.Surface(),
			v.t.BorderColor(),
		)
	}

	locality := "remote-only"
	if selected.IsLocal {
		locality = "local writable"
		if len(selected.LocalPaths) > 1 {
			locality = fmt.Sprintf("local writable (%d paths)", len(selected.LocalPaths))
		}
	}

	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(v.t.Primary()).Render("Repository Detail"),
		"",
		lipgloss.NewStyle().Bold(true).Foreground(v.t.Fg()).Render(selected.FullName),
		lipgloss.NewStyle().Foreground(v.t.MutedFg()).Width(maxInt(20, width-4)).Render(valueOrDash(selected.Description)),
		"",
		lipgloss.NewStyle().Foreground(v.t.DimText()).Render("Language"),
		lipgloss.NewStyle().Foreground(v.t.Fg()).Render(valueOrDash(selected.Language)),
		"",
		lipgloss.NewStyle().Foreground(v.t.DimText()).Render("Default branch"),
		lipgloss.NewStyle().Foreground(v.t.Fg()).Render(valueOrDash(selected.DefaultBranch)),
		"",
		lipgloss.NewStyle().Foreground(v.t.DimText()).Render("Signals"),
		lipgloss.NewStyle().Foreground(v.t.Fg()).Render(fmt.Sprintf("%d stars  |  %d PRs  |  %d issues", selected.Stars, selected.OpenPRs, selected.OpenIssues)),
		"",
		lipgloss.NewStyle().Foreground(v.t.DimText()).Render("Availability"),
		lipgloss.NewStyle().Foreground(v.t.Fg()).Render(locality),
	}
	if len(selected.LocalPaths) > 0 {
		lines = append(lines, "", lipgloss.NewStyle().Bold(true).Foreground(v.t.Secondary()).Render("Local Paths"))
		for _, path := range selected.LocalPaths {
			lines = append(lines, lipgloss.NewStyle().Foreground(v.t.MutedFg()).Width(maxInt(20, width-4)).Render(path))
		}
	}
	lines = append(lines, "", lipgloss.NewStyle().Foreground(v.t.MutedFg()).Width(maxInt(20, width-4)).
		Render("Selecting a repository refreshes the cockpit, explorer, workspace, and inspector together so the operator stays in one coherent context."))
	if !selected.IsLocal {
		lines = append(lines, "", lipgloss.NewStyle().Foreground(v.t.Secondary()).Render("Press C to clone into the default workspace root, or use /clone after opening the repository."))
	}
	return render.SurfacePanel(strings.Join(lines, "\n"), width, v.t.Surface(), v.t.BorderColor())
}

func (v *ReposView) renderRepoCard(r RepoListItem, selected bool, width int) string {
	badge := "remote-only"
	if r.IsLocal {
		badge = "local"
	}
	name := r.FullName
	if r.Fork {
		name += " (fork)"
	}
	panelFrame := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1).GetHorizontalFrameSize()
	innerW := maxInt(12, width-panelFrame)

	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(v.t.Fg()).Render(truncate(name, innerW)),
		lipgloss.NewStyle().Foreground(v.t.DimText()).Render(truncate(fmt.Sprintf("%s  |  %s  |  branch %s", badge, valueOrDash(r.Language), valueOrDash(r.DefaultBranch)), innerW)),
	}
	if r.IsLocal && len(r.LocalPaths) > 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(v.t.MutedFg()).Render(truncate(r.LocalPaths[0], innerW)))
	}
	lines = append(lines,
		lipgloss.NewStyle().Foreground(v.t.DimText()).Render(truncate(fmt.Sprintf("%s %d  |  %s %d  |  updated %s", theme.Icons.PullRequest, r.OpenPRs, theme.Icons.Issue, r.OpenIssues, valueOrDash(r.UpdatedAt)), innerW)),
		lipgloss.NewStyle().Foreground(v.t.MutedFg()).Render(truncate(valueOrDash(r.Description), innerW)),
	)

	block := strings.Join(lines, "\n")
	border := v.t.BorderColor()
	if selected {
		block = render.FillBlock(block, innerW, lipgloss.NewStyle().Background(v.t.Selection()))
		border = v.t.FocusBorderColor()
	}
	return render.SurfacePanel(block, width, v.t.Surface(), border)
}

func (v *ReposView) SetSize(width, height int) {
	v.width = width
	v.height = height
}

func (v *ReposView) applyFilter() {
	if v.search == "" {
		v.filtered = v.items
	} else {
		query := strings.ToLower(v.search)
		v.filtered = nil
		for _, item := range v.items {
			if strings.Contains(strings.ToLower(item.FullName), query) ||
				strings.Contains(strings.ToLower(item.Language), query) ||
				strings.Contains(strings.ToLower(item.Description), query) {
				v.filtered = append(v.filtered, item)
			}
		}
	}
	if v.cursor >= len(v.filtered) {
		v.cursor = max(0, len(v.filtered)-1)
	}
}

func (v *ReposView) SelectedRepo() *RepoListItem {
	if v.cursor < len(v.filtered) {
		item := v.filtered[v.cursor]
		return &item
	}
	return nil
}
