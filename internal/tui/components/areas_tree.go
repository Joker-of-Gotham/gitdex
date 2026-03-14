package components

import (
	"fmt"
	"net/url"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/mattn/go-runewidth"

	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
)

const (
	defaultTreeWidth    = 32
	defaultTreeMaxItems = 4
)

type AreasTree struct {
	state    *status.GitState
	width    int
	maxItems int
}

func NewAreasTree(state *status.GitState) AreasTree {
	return AreasTree{
		state:    state,
		width:    defaultTreeWidth,
		maxItems: defaultTreeMaxItems,
	}
}

func (t AreasTree) SetWidth(width int) AreasTree {
	t.width = width
	return t
}

func (t AreasTree) SetMaxItems(maxItems int) AreasTree {
	t.maxItems = maxItems
	return t
}

func (t AreasTree) View() string {
	if t.width <= 0 {
		t.width = defaultTreeWidth
	}
	if t.state == nil {
		return strings.Join(t.wrapLine("", "repo: state unavailable", treeMuted()), "\n")
	}

	var lines []string
	lines = append(lines, t.wrapLine("", "repo", treeRoot())...)
	lines = append(lines, t.workingTreeLines()...)
	lines = append(lines, t.stagingTreeLines()...)
	lines = append(lines, t.localRepoLines()...)
	lines = append(lines, t.remoteLines()...)
	return strings.Join(lines, "\n")
}

func (t AreasTree) workingTreeLines() []string {
	modified := 0
	untracked := 0
	var items []string
	for _, f := range t.state.WorkingTree {
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
	lines = append(lines, t.wrapLine("+- ", "Working Directory", treeSection())...)
	summary := fmt.Sprintf("%d modified, %d new", modified, untracked)
	if modified == 0 && untracked == 0 {
		summary = "clean"
	}
	lines = append(lines, t.wrapLine("|  +- ", summary, treeStatus(modified == 0 && untracked == 0))...)
	for _, item := range t.limitLines(items) {
		lines = append(lines, t.wrapLine("|  |  ", item, treeItem(item))...)
	}
	return lines
}

func (t AreasTree) stagingTreeLines() []string {
	var items []string
	for _, f := range t.state.StagingArea {
		if f.StagingCode == git.StatusUnmodified {
			continue
		}
		items = append(items, string(f.StagingCode)+" "+f.Path)
	}

	var lines []string
	lines = append(lines, t.wrapLine("+- ", "Staging Area", treeSection())...)
	summary := "clean"
	if len(items) > 0 {
		summary = fmt.Sprintf("%d staged", len(items))
	}
	lines = append(lines, t.wrapLine("|  +- ", summary, treeStatus(len(items) == 0))...)
	for _, item := range t.limitLines(items) {
		lines = append(lines, t.wrapLine("|  |  ", item, treeItem(item))...)
	}
	return lines
}

func (t AreasTree) localRepoLines() []string {
	branch := strings.TrimSpace(t.state.LocalBranch.Name)
	if branch == "" {
		branch = "(no branch)"
	}
	ahead, behind := t.state.LocalBranch.Ahead, t.state.LocalBranch.Behind
	if t.state.UpstreamState != nil {
		ahead = t.state.UpstreamState.Ahead
		behind = t.state.UpstreamState.Behind
	}

	var lines []string
	lines = append(lines, t.wrapLine("+- ", "Local Repository", treeSection())...)
	lines = append(lines, t.wrapLine("|  +- ", fmt.Sprintf("branch: %s", branch), treeValue())...)
	lines = append(lines, t.wrapLine("|  +- ", fmt.Sprintf("commits: %d", t.state.CommitCount), treeValue())...)
	if t.state.LocalBranch.Upstream != "" {
		lines = append(lines, t.wrapLine("|  +- ", fmt.Sprintf("ahead:%d behind:%d vs %s", ahead, behind, t.state.LocalBranch.Upstream), treeStatus(ahead == 0 && behind == 0))...)
	} else {
		lines = append(lines, t.wrapLine("|  +- ", fmt.Sprintf("ahead:%d behind:%d | no upstream", ahead, behind), treeWarn())...)
	}
	if t.state.LocalBranch.IsDetached {
		lines = append(lines, t.wrapLine("|  +- ", "detached HEAD", treeDanger())...)
	}
	if t.state.MergeInProgress || t.state.RebaseInProgress || t.state.CherryInProgress || t.state.BisectInProgress {
		lines = append(lines, t.wrapLine("|  +- ", "git operation in progress", treeDanger())...)
	}
	return lines
}

func (t AreasTree) remoteLines() []string {
	var lines []string
	lines = append(lines, t.wrapLine("`- ", "Remote", treeSection())...)
	if len(t.state.RemoteInfos) == 0 {
		lines = append(lines, t.wrapLine("   `- ", "not configured", treeMuted())...)
		return lines
	}
	for idx, remote := range t.state.RemoteInfos {
		nodePrefix := "   +- "
		childPrefix := "   |  "
		if idx == len(t.state.RemoteInfos)-1 {
			nodePrefix = "   `- "
			childPrefix = "      "
		}
		title := remote.Name
		if title == "" {
			title = "remote"
		}
		title += " (remote)"
		lines = append(lines, t.wrapLine(nodePrefix, title, treeSection())...)

		urlValue := remote.PushURL
		if strings.TrimSpace(urlValue) == "" {
			urlValue = remote.FetchURL
		}
		host := hostFromRemote(urlValue)
		if host == "" {
			host = "not configured"
		}
		lines = append(lines, t.wrapLine(childPrefix+"+- ", "host: "+host, treeValue())...)
		lines = append(lines, t.wrapLine(childPrefix+"+- ", "transport: "+remoteScheme(urlValue), treeValue())...)
		lines = append(lines, t.wrapLine(childPrefix+"`- ", remoteStatusText(remote), remoteStatusStyle(remote))...)
	}
	return lines
}

func (t AreasTree) wrapLine(prefix, text string, style lipgloss.Style) []string {
	available := t.width - runewidth.StringWidth(prefix)
	if available < 8 {
		available = t.width
		prefix = ""
	}
	wrapped := strings.Split(runewidth.Wrap(text, available), "\n")
	out := make([]string, 0, len(wrapped))
	for i, line := range wrapped {
		lead := prefix
		if i > 0 {
			lead = strings.Repeat(" ", runewidth.StringWidth(prefix))
		}
		out = append(out, lipgloss.NewStyle().Foreground(lipgloss.Color("#51606B")).Render(lead)+style.Render(line))
	}
	return out
}

func (t AreasTree) limitLines(lines []string) []string {
	if t.maxItems <= 0 || len(lines) <= t.maxItems {
		return lines
	}
	out := append([]string(nil), lines[:t.maxItems]...)
	out = append(out, fmt.Sprintf("+%d more item(s)", len(lines)-t.maxItems))
	return out
}

func remoteStatusText(r git.RemoteInfo) string {
	switch {
	case !r.FetchURLValid && !r.PushURLValid:
		return "status: invalid URL"
	case r.ReachabilityChecked && !r.Reachable:
		return "status: unreachable"
	case r.ReachabilityChecked:
		return "status: reachable"
	default:
		return "status: not probed"
	}
}

func remoteStatusStyle(r git.RemoteInfo) lipgloss.Style {
	switch {
	case !r.FetchURLValid && !r.PushURLValid:
		return treeDanger()
	case r.ReachabilityChecked && !r.Reachable:
		return treeDanger()
	case r.ReachabilityChecked:
		return treeSafe()
	default:
		return treeWarn()
	}
}

func treeRoot() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#7BD8FF")).Bold(true)
}
func treeSection() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#F2C572")).Bold(true)
}
func treeValue() lipgloss.Style { return lipgloss.NewStyle().Foreground(lipgloss.Color("#DCE7EF")) }
func treeMuted() lipgloss.Style { return lipgloss.NewStyle().Foreground(lipgloss.Color("#7A8B99")) }
func treeSafe() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#7BD389")).Bold(true)
}
func treeWarn() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#F4B942")).Bold(true)
}
func treeDanger() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#FF8C73")).Bold(true)
}

func treeStatus(clean bool) lipgloss.Style {
	if clean {
		return treeSafe()
	}
	return treeWarn()
}

func treeItem(item string) lipgloss.Style {
	switch {
	case strings.HasPrefix(item, "? "):
		return treeMuted()
	case strings.HasPrefix(item, "D "):
		return treeDanger()
	default:
		return treeWarn()
	}
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
