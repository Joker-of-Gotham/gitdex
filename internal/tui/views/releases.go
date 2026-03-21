package views

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	ghp "github.com/your-org/gitdex/internal/platform/github"
	"github.com/your-org/gitdex/internal/tui/render"
	"github.com/your-org/gitdex/internal/tui/theme"
)

// ReleasesView lists and manages GitHub releases for the active repository.
type ReleasesView struct {
	releases      []*ghp.Release
	cursor        int
	width, height int
	vp            viewport.Model
	repoPath      string
	owner         string
	repo          string
	statusMsg     string
	creating      bool
	editing       bool
	deleteConfirm bool
	tagInput      textinput.Model
	nameInput     textinput.Model
	bodyInput     textinput.Model
	formFocus     int
	draft         bool
	prerelease    bool
	detail        bool
	t             *theme.Theme
}

func NewReleasesView(t *theme.Theme) *ReleasesView {
	ti := func() textinput.Model {
		m := textinput.New()
		m.Prompt = ""
		m.CharLimit = 4096
		return m
	}
	return &ReleasesView{
		t:         t,
		tagInput:  ti(),
		nameInput: ti(),
		bodyInput: ti(),
	}
}

func (v *ReleasesView) ID() ID        { return "releases" }
func (v *ReleasesView) Title() string { return "Releases" }
func (v *ReleasesView) Init() tea.Cmd { return nil }

func (v *ReleasesView) SetRepoContext(repoPath, owner, repo string) {
	v.repoPath = repoPath
	v.owner = owner
	v.repo = repo
}

func (v *ReleasesView) SetSize(w, h int) {
	v.width = w
	v.height = h
	v.vp = viewport.New(viewport.WithWidth(max(20, w-4)), viewport.WithHeight(max(3, h-10)))
}

func (v *ReleasesView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case ReleaseListMsg:
		v.releases = msg.Releases
		if v.cursor >= len(v.releases) {
			v.cursor = max(0, len(v.releases)-1)
		}
		if msg.Err != nil {
			v.statusMsg = msg.Err.Error()
		}
		return v, nil
	case ReleaseOpResultMsg:
		if msg.Err != nil {
			v.statusMsg = msg.Err.Error()
		} else {
			v.statusMsg = msg.Message
			v.closeForm()
		}
		return v, nil
	case tea.KeyPressMsg:
		return v.handleKey(msg)
	}
	return v, nil
}

func (v *ReleasesView) handleKey(msg tea.KeyPressMsg) (View, tea.Cmd) {
	if v.creating || v.editing {
		return v.handleFormKey(msg)
	}
	prev := v.cursor
	if msg.String() != "d" {
		v.deleteConfirm = false
	}
	switch msg.String() {
	case "up", "k":
		if v.cursor > 0 {
			v.cursor--
		}
	case "down", "j":
		if v.cursor < len(v.releases)-1 {
			v.cursor++
		}
	case "g":
		v.cursor = 0
	case "G":
		if len(v.releases) > 0 {
			v.cursor = len(v.releases) - 1
		}
	case "enter":
		rel := v.selected()
		v.detail = !v.detail
		if v.detail && rel != nil {
			v.refreshDetailViewport()
		}
		if rel != nil {
			return v, func() tea.Msg {
				return ReleaseSelectedMsg{
					ID:          rel.ID,
					TagName:     rel.TagName,
					Name:        rel.Name,
					Draft:       rel.Draft,
					Prerelease:  rel.Prerelease,
					CreatedAt:   timeString(rel.CreatedAt),
					PublishedAt: timeString(rel.PublishedAt),
					URL:         rel.HTMLURL,
					Body:        rel.Body,
				}
			}
		}
	case "esc":
		if v.detail {
			v.detail = false
			return v, nil
		}
	case "c":
		if v.owner == "" || v.repo == "" {
			v.statusMsg = "GitHub owner/repo required for releases."
			return v, nil
		}
		v.creating = true
		v.editing = false
		v.formFocus = 0
		v.draft = false
		v.prerelease = false
		v.tagInput.SetValue("")
		v.nameInput.SetValue("")
		v.bodyInput.SetValue("")
		return v, v.tagInput.Focus()
	case "e":
		rel := v.selected()
		if rel == nil || v.owner == "" || v.repo == "" {
			v.statusMsg = "Select a release and ensure GitHub repo context is set."
			return v, nil
		}
		v.editing = true
		v.creating = false
		v.formFocus = 0
		v.tagInput.SetValue(rel.TagName)
		v.nameInput.SetValue(rel.Name)
		v.bodyInput.SetValue(rel.Body)
		v.draft = rel.Draft
		v.prerelease = rel.Prerelease
		return v, v.tagInput.Focus()
	case "p":
		rel := v.selected()
		if rel == nil || v.owner == "" || v.repo == "" {
			v.statusMsg = "Select a draft release to publish."
			return v, nil
		}
		if !rel.Draft {
			v.statusMsg = "Selected release is not a draft."
			return v, nil
		}
		return v, func() tea.Msg {
			return RequestReleaseOpMsg{
				Kind:         ReleaseOpPublish,
				ReleaseID:    rel.ID,
				OwnerHint:    v.owner,
				RepoHint:     v.repo,
				RepoPathHint: v.repoPath,
			}
		}
	case "d":
		if v.deleteConfirm {
			v.deleteConfirm = false
			rel := v.selected()
			if rel == nil {
				return v, nil
			}
			return v, func() tea.Msg {
				return RequestReleaseOpMsg{
					Kind:         ReleaseOpDelete,
					ReleaseID:    rel.ID,
					OwnerHint:    v.owner,
					RepoHint:     v.repo,
					RepoPathHint: v.repoPath,
				}
			}
		}
		v.deleteConfirm = true
		v.statusMsg = "Press d again to delete this release."
		return v, nil
	}
	if v.cursor != prev && v.detail {
		if rel := v.selected(); rel != nil {
			return v, func() tea.Msg {
				return ReleaseSelectedMsg{
					ID:          rel.ID,
					TagName:     rel.TagName,
					Name:        rel.Name,
					Draft:       rel.Draft,
					Prerelease:  rel.Prerelease,
					CreatedAt:   timeString(rel.CreatedAt),
					PublishedAt: timeString(rel.PublishedAt),
					URL:         rel.HTMLURL,
					Body:        rel.Body,
				}
			}
		}
	}

	if v.detail && v.selected() != nil {
		var cmd tea.Cmd
		v.vp, cmd = v.vp.Update(msg)
		return v, cmd
	}
	return v, nil
}

func (v *ReleasesView) handleFormKey(msg tea.KeyPressMsg) (View, tea.Cmd) {
	switch msg.String() {
	case "esc":
		v.closeForm()
		v.statusMsg = "Canceled."
		return v, nil
	case "tab":
		v.formFocus = (v.formFocus + 1) % 3
		return v, v.focusFormField()
	case "shift+tab":
		v.formFocus = (v.formFocus + 2) % 3
		return v, v.focusFormField()
	case "ctrl+d":
		v.draft = !v.draft
		v.statusMsg = fmt.Sprintf("draft=%v prerelease=%v", v.draft, v.prerelease)
		return v, nil
	case "ctrl+r":
		v.prerelease = !v.prerelease
		v.statusMsg = fmt.Sprintf("draft=%v prerelease=%v", v.draft, v.prerelease)
		return v, nil
	case "enter":
		tag := strings.TrimSpace(v.tagInput.Value())
		if tag == "" {
			v.statusMsg = "Tag is required."
			return v, nil
		}
		name := strings.TrimSpace(v.nameInput.Value())
		if name == "" {
			name = tag
		}
		body := v.bodyInput.Value()
		kind := ReleaseOpCreate
		var rid int64
		if v.editing {
			kind = ReleaseOpUpdate
			if sel := v.selected(); sel != nil {
				rid = sel.ID
			}
		}
		return v, func() tea.Msg {
			return RequestReleaseOpMsg{
				Kind:         kind,
				Tag:          tag,
				Name:         name,
				Body:         body,
				Draft:        v.draft,
				Prerelease:   v.prerelease,
				ReleaseID:    rid,
				OwnerHint:    v.owner,
				RepoHint:     v.repo,
				RepoPathHint: v.repoPath,
			}
		}
	case "up", "down":
		var cmd tea.Cmd
		switch v.formFocus {
		case 0:
			v.tagInput, cmd = v.tagInput.Update(msg)
		case 1:
			v.nameInput, cmd = v.nameInput.Update(msg)
		default:
			v.bodyInput, cmd = v.bodyInput.Update(msg)
		}
		return v, cmd
	}
	var cmd tea.Cmd
	switch v.formFocus {
	case 0:
		v.tagInput, cmd = v.tagInput.Update(msg)
	case 1:
		v.nameInput, cmd = v.nameInput.Update(msg)
	default:
		v.bodyInput, cmd = v.bodyInput.Update(msg)
	}
	return v, cmd
}

func (v *ReleasesView) focusFormField() tea.Cmd {
	v.tagInput.Blur()
	v.nameInput.Blur()
	v.bodyInput.Blur()
	switch v.formFocus {
	case 0:
		return v.tagInput.Focus()
	case 1:
		return v.nameInput.Focus()
	default:
		return v.bodyInput.Focus()
	}
}

func (v *ReleasesView) closeForm() {
	v.creating = false
	v.editing = false
	v.tagInput.Blur()
	v.nameInput.Blur()
	v.bodyInput.Blur()
}

func (v *ReleasesView) refreshDetailViewport() {
	sel := v.selected()
	if sel == nil {
		return
	}
	var b strings.Builder
	b.WriteString(renderReleaseDetailText(sel))
	v.vp.SetContent(b.String())
	v.vp.GotoTop()
}

func (v *ReleasesView) selected() *ghp.Release {
	if v.cursor >= 0 && v.cursor < len(v.releases) {
		return v.releases[v.cursor]
	}
	return nil
}

func (v *ReleasesView) Render() string {
	if len(v.releases) == 0 && !v.creating && !v.editing {
		return lipgloss.NewStyle().Foreground(v.t.DimText()).Render("  No releases loaded (open a GitHub-backed repo).")
	}
	title := lipgloss.NewStyle().Bold(true).Foreground(v.t.Primary()).Render(fmt.Sprintf("  Releases (%d)", len(v.releases)))
	hint := lipgloss.NewStyle().Foreground(v.t.DimText()).Italic(true).Render("  c create  e edit  p publish draft  d delete  Enter details  Ctrl+D draft  Ctrl+R prerelease (form)")
	lines := []string{title, hint}
	if v.statusMsg != "" {
		lines = append(lines, lipgloss.NewStyle().Foreground(v.t.Warning()).Render("  "+v.statusMsg))
	}
	lines = append(lines, "")

	if v.creating || v.editing {
		mode := "Create release"
		if v.editing {
			mode = "Edit release"
		}
		lines = append(lines,
			lipgloss.NewStyle().Bold(true).Foreground(v.t.Info()).Render("  "+mode),
			"  Tab / Shift+Tab: fields  Enter: submit  Esc: cancel",
			"  "+v.tagInput.View(),
			"  "+v.nameInput.View(),
			"  "+v.bodyInput.View(),
			"  "+lipgloss.NewStyle().Foreground(v.t.DimText()).Render(fmt.Sprintf("draft=%v prerelease=%v", v.draft, v.prerelease)),
		)
		return strings.Join(lines, "\n")
	}

	listH := max(1, v.height-6)
	start := 0
	if v.cursor >= listH {
		start = v.cursor - listH + 1
	}
	end := start + listH
	if end > len(v.releases) {
		end = len(v.releases)
	}
	for i := start; i < end; i++ {
		r := v.releases[i]
		badges := releaseBadges(r)
		line := fmt.Sprintf("  %-8d %-18s %s", r.ID, truncate(r.TagName, 18), badges)
		if i == v.cursor {
			line = render.FillBlock(line, max(20, v.width-2), lipgloss.NewStyle().Bold(true).Foreground(v.t.Fg()).Background(v.t.Selection()))
		}
		lines = append(lines, line)
	}
	out := strings.Join(lines, "\n")
	if v.detail {
		out += "\n\n" + lipgloss.NewStyle().Foreground(v.t.DimText()).Italic(true).Render("  Inspector detail active. Use Ctrl+3 or Ctrl+I to review the selected release.")
	}
	return out
}

func releaseBadges(r *ghp.Release) string {
	if r == nil {
		return ""
	}
	var parts []string
	if r.Draft {
		parts = append(parts, "[draft]")
	} else {
		parts = append(parts, "[published]")
	}
	if r.Prerelease {
		parts = append(parts, "[pre]")
	}
	return strings.Join(parts, " ")
}

func (v *ReleasesView) renderDetailPanel(width int) string {
	sel := v.selected()
	if sel == nil {
		return lipgloss.NewStyle().Foreground(v.t.DimText()).Render("Select a release.")
	}
	v.vp.SetWidth(max(10, width-2))
	v.vp.SetHeight(max(3, v.height-8))
	v.vp.SetContent(renderReleaseDetailText(sel))
	return lipgloss.NewStyle().Width(width).Render(v.vp.View())
}

func renderReleaseDetailText(r *ghp.Release) string {
	if r == nil {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Tag: %s\n", r.TagName)
	fmt.Fprintf(&b, "Name: %s\n", r.Name)
	fmt.Fprintf(&b, "Draft: %v  Prerelease: %v\n", r.Draft, r.Prerelease)
	if !r.CreatedAt.IsZero() {
		fmt.Fprintf(&b, "Created: %s\n", r.CreatedAt.Format("2006-01-02 15:04"))
	}
	if !r.PublishedAt.IsZero() {
		fmt.Fprintf(&b, "Published: %s\n", r.PublishedAt.Format("2006-01-02 15:04"))
	}
	if r.HTMLURL != "" {
		fmt.Fprintf(&b, "URL: %s\n", r.HTMLURL)
	}
	if strings.TrimSpace(r.Body) != "" {
		b.WriteString("\n")
		b.WriteString(strings.TrimSpace(r.Body))
	}
	return b.String()
}

func timeString(ts time.Time) string {
	if ts.IsZero() {
		return ""
	}
	return ts.Format("2006-01-02 15:04")
}
