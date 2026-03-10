package components

import (
	"fmt"
	"net/url"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
)

const (
	defaultTreeWidth    = 32
	defaultTreeMaxItems = 4
)

// AreasTree renders Working/Staging/Local/Remote state in a vertical flow.
type AreasTree struct {
	state    *status.GitState
	width    int
	maxItems int
}

// NewAreasTree creates an AreasTree renderer.
func NewAreasTree(state *status.GitState) AreasTree {
	return AreasTree{
		state:    state,
		width:    defaultTreeWidth,
		maxItems: defaultTreeMaxItems,
	}
}

// SetWidth sets panel width.
func (t AreasTree) SetWidth(width int) AreasTree {
	t.width = width
	return t
}

// SetMaxItems sets the max visible file items per section.
func (t AreasTree) SetMaxItems(maxItems int) AreasTree {
	t.maxItems = maxItems
	return t
}

// View renders the full Git areas tree.
func (t AreasTree) View() string {
	if t.width <= 0 {
		t.width = defaultTreeWidth
	}
	if t.maxItems <= 0 {
		t.maxItems = defaultTreeMaxItems
	}
	if t.state == nil {
		return t.renderBox("Git Areas", []string{"State unavailable"}, colorMuted)
	}

	var blocks []string
	blocks = append(blocks, t.renderWorkingDirectory())
	blocks = append(blocks, t.renderConnector("git add"))
	blocks = append(blocks, t.renderStagingArea())
	blocks = append(blocks, t.renderConnector("git commit"))
	blocks = append(blocks, t.renderLocalRepository())
	blocks = append(blocks, t.renderConnector("git push"))

	remoteBlocks := t.renderRemoteBlocks()
	blocks = append(blocks, remoteBlocks...)

	return strings.Join(blocks, "\n")
}

const (
	colorSafe    = "42"
	colorCaution = "214"
	colorDanger  = "196"
	colorMuted   = "241"
)

func (t AreasTree) renderWorkingDirectory() string {
	s := t.state
	modified := 0
	untracked := 0
	var items []string
	for _, f := range s.WorkingTree {
		switch f.WorktreeCode {
		case git.StatusUntracked:
			untracked++
			items = append(items, "? "+f.Path)
		case git.StatusUnmodified:
			continue
		default:
			modified++
			items = append(items, string(f.WorktreeCode)+" "+f.Path)
		}
	}
	var lines []string
	if modified == 0 && untracked == 0 {
		lines = append(lines, "clean")
	} else {
		lines = append(lines, fmt.Sprintf("%d modified 璺?%d new", modified, untracked))
		lines = append(lines, t.limitLines(items)...)
	}
	color := colorSafe
	if modified > 0 || untracked > 0 {
		color = colorCaution
	}
	return t.renderBox("Working Directory", lines, color)
}

func (t AreasTree) renderStagingArea() string {
	s := t.state
	var lines []string
	var items []string
	for _, f := range s.StagingArea {
		code := f.StagingCode
		if code == git.StatusUnmodified {
			continue
		}
		items = append(items, string(code)+" "+f.Path)
	}
	if len(items) == 0 {
		lines = append(lines, "clean")
	} else {
		lines = append(lines, fmt.Sprintf("%d staged", len(items)))
		lines = append(lines, t.limitLines(items)...)
	}
	color := colorSafe
	if len(items) > 0 {
		color = colorCaution
	}
	return t.renderBox("Staging Area", lines, color)
}

func (t AreasTree) renderLocalRepository() string {
	s := t.state
	branch := s.LocalBranch.Name
	if strings.TrimSpace(branch) == "" {
		branch = "(no branch)"
	}
	lines := []string{
		fmt.Sprintf("%s | %d commits", branch, s.CommitCount),
	}
	ahead, behind := s.LocalBranch.Ahead, s.LocalBranch.Behind
	if s.UpstreamState != nil {
		ahead = s.UpstreamState.Ahead
		behind = s.UpstreamState.Behind
	}
	if s.LocalBranch.Upstream != "" {
		lines = append(lines, fmt.Sprintf("ahead:%d behind:%d vs %s", ahead, behind, s.LocalBranch.Upstream))
	} else {
		lines = append(lines, fmt.Sprintf("ahead:%d behind:%d | no upstream", ahead, behind))
	}
	if s.LocalBranch.IsDetached {
		lines = append(lines, "DETACHED HEAD")
	}
	if s.MergeInProgress || s.RebaseInProgress || s.CherryInProgress || s.BisectInProgress {
		lines = append(lines, "operation in progress")
	}

	color := colorSafe
	if s.MergeInProgress || s.RebaseInProgress || s.CherryInProgress || s.BisectInProgress {
		color = colorDanger
	} else if ahead > 0 || behind > 0 {
		color = colorCaution
	}
	return t.renderBox("Local Repository", lines, color)
}

func (t AreasTree) renderRemoteBlocks() []string {
	s := t.state
	var blocks []string
	if len(s.RemoteInfos) == 0 {
		blocks = append(blocks, t.renderBox("Remote", []string{"not configured"}, colorMuted))
		return blocks
	}

	upstream := upstreamRemoteName(s.LocalBranch.Upstream)
	hasUpstream := false
	for idx, r := range s.RemoteInfos {
		if idx > 0 {
			blocks = append(blocks, t.renderConnector("sync"))
		}
		if r.Name == upstream && upstream != "" {
			hasUpstream = true
		}
		blocks = append(blocks, t.renderRemoteBox(r))
	}

	if upstream != "" && !hasUpstream {
		blocks = append(blocks, t.renderConnector("upstream"))
		blocks = append(blocks, t.renderBox(fmt.Sprintf("%s (remote)", upstream), []string{"not configured"}, colorMuted))
	}
	return blocks
}

func (t AreasTree) renderRemoteBox(r git.RemoteInfo) string {
	title := r.Name + " (remote)"
	urlValue := r.PushURL
	if strings.TrimSpace(urlValue) == "" {
		urlValue = r.FetchURL
	}
	host := hostFromRemote(urlValue)
	scheme := remoteScheme(urlValue)

	var lines []string
	if host == "" {
		lines = append(lines, "not configured")
		return t.renderBox(title, lines, colorMuted)
	}
	lines = append(lines, host)

	var color string
	var statusText string
	switch {
	case !r.FetchURLValid && !r.PushURLValid:
		color = colorDanger
		statusText = "invalid URL"
	case r.ReachabilityChecked && !r.Reachable:
		color = colorDanger
		statusText = "unreachable"
	default:
		if r.ReachabilityChecked {
			statusText = "reachable"
			color = colorSafe
		} else {
			statusText = "not probed"
			color = colorCaution
		}
	}
	lines = append(lines, fmt.Sprintf("%s | %s", scheme, statusText))
	if r.LastError != "" && statusText == "unreachable" {
		lines = append(lines, r.LastError)
	}
	return t.renderBox(title, lines, color)
}

func (t AreasTree) renderConnector(action string) string {
	text := "      -> " + strings.TrimSpace(action)
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorMuted)).
		Render(trimForPanel(text, t.width))
}

func (t AreasTree) renderBox(title string, lines []string, color string) string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(color))
	bodyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	rendered := []string{titleStyle.Render(title)}
	for _, line := range lines {
		rendered = append(rendered, bodyStyle.Render(trimForPanel(line, t.width-4)))
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(color)).
		Padding(0, 1).
		Width(t.width).
		Render(strings.Join(rendered, "\n"))
}

func (t AreasTree) limitLines(lines []string) []string {
	if len(lines) <= t.maxItems {
		return lines
	}
	out := append([]string(nil), lines[:t.maxItems]...)
	out = append(out, fmt.Sprintf("... +%d more", len(lines)-t.maxItems))
	return out
}

func trimForPanel(s string, width int) string {
	s = strings.TrimSpace(s)
	if width <= 0 || len(s) <= width {
		return s
	}
	if width <= 3 {
		return s[:width]
	}
	return s[:width-3] + "..."
}

func upstreamRemoteName(upstream string) string {
	upstream = strings.TrimSpace(upstream)
	if upstream == "" {
		return ""
	}
	if idx := strings.Index(upstream, "/"); idx > 0 {
		return upstream[:idx]
	}
	return ""
}

func hostFromRemote(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "git@") {
		rest := strings.TrimPrefix(raw, "git@")
		if idx := strings.Index(rest, ":"); idx > 0 {
			return rest[:idx]
		}
		return rest
	}
	u, err := url.Parse(raw)
	if err == nil && u.Host != "" {
		return u.Host
	}
	return raw
}

func remoteScheme(raw string) string {
	raw = strings.TrimSpace(strings.ToLower(raw))
	switch {
	case strings.HasPrefix(raw, "git@"):
		return "SSH"
	case strings.HasPrefix(raw, "ssh://"):
		return "SSH"
	case strings.HasPrefix(raw, "https://"):
		return "HTTPS"
	case strings.HasPrefix(raw, "http://"):
		return "HTTP"
	default:
		return "Unknown"
	}
}
