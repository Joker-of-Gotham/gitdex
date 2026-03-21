package views

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/your-org/gitdex/internal/tui/render"
	"github.com/your-org/gitdex/internal/tui/theme"
)

type BranchEntry struct {
	Name       string
	SHA        string
	Upstream   string
	LastCommit string
	IsRemote   bool
	IsCurrent  bool
	Protected  bool
	Ahead      int
	Behind     int
}

type BranchTreeDataMsg struct {
	Branches []BranchEntry
}

type BranchCheckoutResultMsg struct {
	Name string
	Err  error
}

type BranchSelectedMsg struct {
	Branch BranchEntry
}

type branchPromptKind int

const (
	branchPromptNone branchPromptKind = iota
	branchPromptCreate
	branchPromptRename
	branchPromptDelete
)

type BranchTreeView struct {
	t          *theme.Theme
	branches   []BranchEntry
	cursor     int
	width      int
	height     int
	detail     bool
	statusLine string
	editable   bool
	prompt     textinput.Model
	promptKind branchPromptKind
}

func NewBranchTreeView(t *theme.Theme) *BranchTreeView {
	prompt := textinput.New()
	prompt.Prompt = ""
	prompt.CharLimit = 256
	return &BranchTreeView{t: t, prompt: prompt}
}

func (v *BranchTreeView) ID() ID        { return "branch_tree" }
func (v *BranchTreeView) Title() string { return "Branches" }
func (v *BranchTreeView) Init() tea.Cmd { return nil }

func (v *BranchTreeView) SetBranches(b []BranchEntry) {
	v.branches = b
	v.cursor = 0
}

func (v *BranchTreeView) SetEditable(editable bool) {
	v.editable = editable
}

func (v *BranchTreeView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case BranchTreeDataMsg:
		v.branches = msg.Branches
		v.cursor = 0
	case BranchCheckoutResultMsg:
		if msg.Err != nil {
			v.statusLine = fmt.Sprintf("Checkout failed: %v", msg.Err)
		} else {
			v.statusLine = fmt.Sprintf("Checked out %s", msg.Name)
		}
		return v, nil
	case BranchActionResultMsg:
		if msg.Err != nil {
			v.statusLine = msg.Err.Error()
			return v, nil
		}
		v.statusLine = msg.Message
		v.closePrompt()
		return v, nil
	case tea.KeyPressMsg:
		return v.handleKey(msg)
	}
	return v, nil
}

func (v *BranchTreeView) handleKey(msg tea.KeyPressMsg) (View, tea.Cmd) {
	if v.promptKind != branchPromptNone {
		return v.handlePromptKey(msg)
	}
	prev := v.cursor
	switch msg.String() {
	case "up", "k":
		if v.cursor > 0 {
			v.cursor--
		}
	case "down", "j":
		if v.cursor < len(v.branches)-1 {
			v.cursor++
		}
	case "g":
		v.cursor = 0
	case "G":
		if len(v.branches) > 0 {
			v.cursor = len(v.branches) - 1
		}
	case "pgup":
		viewH := maxInt(1, v.height-5)
		step := maxInt(1, viewH/2)
		v.cursor -= step
		if v.cursor < 0 {
			v.cursor = 0
		}
	case "pgdown":
		viewH := maxInt(1, v.height-5)
		step := maxInt(1, viewH/2)
		v.cursor += step
		if v.cursor >= len(v.branches) {
			v.cursor = maxInt(0, len(v.branches)-1)
		}
	case "enter":
		v.detail = !v.detail
		if branch := v.selected(); branch != nil {
			cmds := []tea.Cmd{
				func() tea.Msg { return BranchSelectedMsg{Branch: *branch} },
			}
			if v.detail {
				if cmd := v.branchProtectionCmd(); cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
			return v, tea.Batch(cmds...)
		}
	case "esc":
		if v.detail {
			v.detail = false
			return v, nil
		}
	case "c":
		if branch := v.selected(); branch != nil {
			if branch.IsRemote {
				v.statusLine = "Remote-only branch. Clone locally to switch."
				return v, nil
			}
			if !v.editable {
				v.statusLine = "Checkout requires a local repository."
				return v, nil
			}
			if branch.IsCurrent {
				v.statusLine = "Already on selected branch."
				return v, nil
			}
			return v, func() tea.Msg { return RequestBranchCheckoutMsg{Name: branch.Name} }
		}
	case "n":
		if !v.ensureEditable("Create branch requires a local repository.") {
			return v, nil
		}
		return v.openPrompt(branchPromptCreate, "")
	case "r":
		if !v.ensureEditable("Rename branch requires a local repository.") {
			return v, nil
		}
		if branch := v.selected(); branch != nil {
			if branch.IsRemote {
				v.statusLine = "Remote branches cannot be renamed locally here."
				return v, nil
			}
			return v.openPrompt(branchPromptRename, branch.Name)
		}
	case "x":
		if !v.ensureEditable("Delete branch requires a local repository.") {
			return v, nil
		}
		if branch := v.selected(); branch != nil {
			if branch.IsRemote {
				v.statusLine = "Remote branches cannot be deleted from this view."
				return v, nil
			}
			if branch.IsCurrent {
				v.statusLine = "Cannot delete the current branch."
				return v, nil
			}
			return v.openPrompt(branchPromptDelete, "")
		}
	}
	if v.cursor != prev {
		if branch := v.selected(); branch != nil {
			cmds := []tea.Cmd{
				func() tea.Msg { return BranchSelectedMsg{Branch: *branch} },
			}
			if cmd := v.branchProtectionCmd(); cmd != nil {
				cmds = append(cmds, cmd)
			}
			return v, tea.Batch(cmds...)
		}
	}
	return v, nil
}

func (v *BranchTreeView) branchProtectionCmd() tea.Cmd {
	b := v.selected()
	if b == nil || !b.Protected || !b.IsRemote {
		return nil
	}
	return func() tea.Msg {
		return RequestBranchProtectionMsg{Branch: b.Name}
	}
}

func (v *BranchTreeView) Render() string {
	if len(v.branches) == 0 {
		return lipgloss.NewStyle().Foreground(v.t.DimText()).Render("  No branches loaded")
	}
	base := v.renderList(v.width)
	if v.promptKind != branchPromptNone {
		base += "\n\n" + v.renderPromptPanel()
	}
	return base
}

func (v *BranchTreeView) renderList(width int) string {
	var lines []string
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(v.t.Primary())
	hintStyle := lipgloss.NewStyle().Foreground(v.t.DimText()).Italic(true)
	lines = append(lines, headerStyle.Render(fmt.Sprintf("  Branches (%d)", len(v.branches))))
	lines = append(lines, hintStyle.Render("  Up/Down navigate  Enter inspect  c checkout  n create  r rename  x delete  Esc close detail"))
	if v.statusLine != "" {
		lines = append(lines, lipgloss.NewStyle().Foreground(v.t.Warning()).Render("  "+v.statusLine))
	}
	lines = append(lines, "")

	viewH := max(1, v.height-5)
	start := 0
	if v.cursor >= viewH {
		start = v.cursor - viewH + 1
	}
	end := start + viewH
	if end > len(v.branches) {
		end = len(v.branches)
	}

	for i := start; i < end; i++ {
		branch := v.branches[i]
		name := branch.Name
		if branch.IsRemote {
			name += " [remote]"
		}
		if branch.Protected {
			name += " [protected]"
		}
		if branch.Ahead > 0 || branch.Behind > 0 {
			name += fmt.Sprintf("  +%d/-%d", branch.Ahead, branch.Behind)
		}
		if branch.IsCurrent {
			name = "* " + name
		} else {
			name = "  " + name
		}
		name = truncate(name, maxInt(20, width-2))

		style := lipgloss.NewStyle().Foreground(v.t.Fg())
		if branch.IsCurrent {
			style = style.Bold(true).Foreground(v.t.Success())
		} else if branch.IsRemote {
			style = style.Foreground(v.t.DimText())
		}
		line := style.Render(name)
		if i == v.cursor {
			line = render.FillBlock(name, maxInt(20, width-2), lipgloss.NewStyle().Bold(true).Foreground(v.t.Fg()).Background(v.t.Selection()))
		}
		lines = append(lines, line)
	}
	if v.detail {
		lines = append(lines, "")
		lines = append(lines, hintStyle.Render("  Inspector detail active. Use Ctrl+3 or Ctrl+I to review the selected branch."))
	}
	return strings.Join(lines, "\n")
}

func (v *BranchTreeView) renderDetail() string {
	title := lipgloss.NewStyle().Bold(true).Foreground(v.t.Secondary()).Render("Branch Detail")
	branch := v.selected()
	if branch == nil {
		return title + "\n\n" + lipgloss.NewStyle().Foreground(v.t.DimText()).Render("Select a branch to inspect.")
	}
	lines := []string{
		title,
		"",
		"Name: " + branch.Name,
		"Scope: " + ternary(branch.IsRemote, "remote", "local"),
		"Current: " + ternary(branch.IsCurrent, "yes", "no"),
	}
	if branch.Upstream != "" {
		lines = append(lines, "Upstream: "+branch.Upstream)
	}
	if branch.SHA != "" {
		lines = append(lines, "SHA: "+truncate(branch.SHA, 16))
	}
	if branch.LastCommit != "" {
		lines = append(lines, "Last commit: "+branch.LastCommit)
	}
	if branch.Protected && branch.IsRemote {
		lines = append(lines, "Protected: yes (see inspector for API details when selected)")
	}
	if branch.Ahead > 0 || branch.Behind > 0 {
		lines = append(lines, fmt.Sprintf("Divergence: +%d / -%d", branch.Ahead, branch.Behind))
	}
	mode := "remote read-only"
	if v.editable {
		mode = "local writable"
	}
	lines = append(lines, "Repository mode: "+mode)
	if !branch.IsRemote {
		lines = append(lines, "", "Actions: c checkout  r rename  x delete")
	}
	lines = append(lines, "Create: n")
	return strings.Join(lines, "\n")
}

func (v *BranchTreeView) selected() *BranchEntry {
	if v.cursor >= 0 && v.cursor < len(v.branches) {
		return &v.branches[v.cursor]
	}
	return nil
}

func (v *BranchTreeView) splitWidths() (int, int) {
	listW := max(40, v.width*46/100)
	detailW := max(32, v.width-listW-1)
	return listW, detailW
}

func (v *BranchTreeView) SetSize(w, h int) {
	v.width = w
	v.height = h
}

func (v *BranchTreeView) ensureEditable(message string) bool {
	if v.editable {
		return true
	}
	v.statusLine = message
	return false
}

func (v *BranchTreeView) openPrompt(kind branchPromptKind, initial string) (View, tea.Cmd) {
	v.promptKind = kind
	v.prompt.SetValue(initial)
	v.prompt.Placeholder = initial
	return v, v.prompt.Focus()
}

func (v *BranchTreeView) closePrompt() {
	v.promptKind = branchPromptNone
	v.prompt.SetValue("")
	v.prompt.Blur()
}

func (v *BranchTreeView) handlePromptKey(msg tea.KeyPressMsg) (View, tea.Cmd) {
	switch msg.String() {
	case "esc":
		v.closePrompt()
		v.statusLine = "Branch action canceled."
		return v, nil
	case "enter":
		branch := v.selected()
		switch v.promptKind {
		case branchPromptCreate:
			raw := strings.TrimSpace(v.prompt.Value())
			if raw == "" {
				v.statusLine = "Branch name cannot be empty."
				return v, nil
			}
			name, start := parseBranchCreatePrompt(raw)
			return v, func() tea.Msg {
				return RequestBranchActionMsg{Kind: BranchActionCreate, Name: name, Target: start}
			}
		case branchPromptRename:
			if branch == nil {
				v.closePrompt()
				return v, nil
			}
			newName := strings.TrimSpace(v.prompt.Value())
			if newName == "" {
				v.statusLine = "New branch name cannot be empty."
				return v, nil
			}
			return v, func() tea.Msg {
				return RequestBranchActionMsg{Kind: BranchActionRename, Name: branch.Name, Target: newName}
			}
		case branchPromptDelete:
			if branch == nil {
				v.closePrompt()
				return v, nil
			}
			if !strings.EqualFold(strings.TrimSpace(v.prompt.Value()), "delete") {
				v.statusLine = "Type delete to confirm."
				return v, nil
			}
			return v, func() tea.Msg {
				return RequestBranchActionMsg{Kind: BranchActionDelete, Name: branch.Name}
			}
		}
	}

	var cmd tea.Cmd
	v.prompt, cmd = v.prompt.Update(msg)
	return v, cmd
}

func (v *BranchTreeView) renderPromptPanel() string {
	title := "Branch Action"
	hint := ""
	switch v.promptKind {
	case branchPromptCreate:
		title = "Create Branch"
		hint = "Use `<name>` or `<name> <start-point>`."
	case branchPromptRename:
		title = "Rename Branch"
		hint = "Enter the new branch name."
	case branchPromptDelete:
		title = "Delete Branch"
		hint = "Type delete to confirm deleting the selected branch."
	}
	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(v.t.Primary()).Render(title),
		lipgloss.NewStyle().Foreground(v.t.MutedFg()).Render(hint),
		"",
		v.prompt.View(),
		"",
		lipgloss.NewStyle().Foreground(v.t.DimText()).Render("Enter submit  Esc cancel"),
	}
	return render.SurfacePanel(strings.Join(lines, "\n"), max(36, v.width), v.t.Surface(), v.t.BorderColor())
}

func parseBranchCreatePrompt(raw string) (string, string) {
	parts := strings.Fields(strings.TrimSpace(raw))
	if len(parts) == 0 {
		return "", ""
	}
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], strings.Join(parts[1:], " ")
}
