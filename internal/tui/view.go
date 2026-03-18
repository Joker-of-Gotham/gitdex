package tui

import (
	"fmt"
	"strings"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/Joker-of-Gotham/gitdex/internal/dotgitdex"
	"github.com/Joker-of-Gotham/gitdex/internal/observability"
	tuictx "github.com/Joker-of-Gotham/gitdex/internal/tui/context"
	"github.com/Joker-of-Gotham/gitdex/internal/tui/theme"
)

// ── Theme-aware style helpers ──────────────────────────────────────────
// All styles are computed from theme.Current so the entire TUI
// recolors uniformly when the user switches themes.

func ts() *theme.Theme {
	if theme.Current == nil {
		theme.Init("catppuccin")
	}
	return theme.Current
}

func s(hex string) lipgloss.Style { return lipgloss.NewStyle().Foreground(lipgloss.Color(hex)) }
func sb(hex string) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(hex)).Bold(true)
}

func titleStyle() lipgloss.Style         { return sb(ts().Primary) }
func subtitleStyle() lipgloss.Style      { return s(ts().TextMuted) }
func modeManual() lipgloss.Style         { return sb(ts().Success) }
func modeAuto() lipgloss.Style           { return sb(ts().Warning) }
func modeCruise() lipgloss.Style         { return sb(ts().Danger) }
func flowStyle() lipgloss.Style          { return s(ts().Accent) }
func successStyle() lipgloss.Style       { return s(ts().Success) }
func successBoldStyle() lipgloss.Style   { return sb(ts().Success) }
func warningStyle() lipgloss.Style       { return s(ts().Warning) }
func warningBoldStyle() lipgloss.Style   { return sb(ts().Warning) }
func dangerStyle() lipgloss.Style        { return s(ts().Danger) }
func dangerBoldStyle() lipgloss.Style    { return sb(ts().Danger) }
func infoStyle() lipgloss.Style          { return s(ts().Info) }
func accentStyle() lipgloss.Style        { return s(ts().Accent) }
func commandStyle() lipgloss.Style       { return s(ts().Secondary) }
func tsTimeStyle() lipgloss.Style        { return s(ts().Accent) }
func keyStyle() lipgloss.Style           { return sb(ts().Warning) }
func valueStyle() lipgloss.Style         { return s(ts().Text) }
func mutedStyle() lipgloss.Style         { return s(ts().TextMuted) }
func dimStyle() lipgloss.Style           { return s(ts().TextMuted) }
func sectionHeaderStyle() lipgloss.Style { return sb(ts().Text) }
func focusedHeaderStyle() lipgloss.Style { return sb(ts().Primary) }
func borderStyle() lipgloss.Style        { return s(ts().Border) }
func panelBorderStyle() lipgloss.Style   { return s(ts().Secondary) }
func activeInputStyle() lipgloss.Style   { return sb(ts().Accent) }
func inactiveInputStyle() lipgloss.Style { return s(ts().TextMuted) }
func configActiveStyle() lipgloss.Style  { return sb(ts().Accent) }
func configItemStyle() lipgloss.Style    { return s(ts().Text) }
func configCheckStyle() lipgloss.Style   { return sb(ts().Success) }
func configHelpStyle() lipgloss.Style    { return s(ts().TextMuted) }
func cursorStyle() lipgloss.Style        { return sb(ts().Warning) }

// ── i18n labels ───────────────────────────────────────────────────────

var configTexts = map[string]map[string]string{
	"config_title":          {"en": "Configuration", "zh": "配置", "ja": "設定"},
	"config_model":          {"en": "Model Configuration", "zh": "模型配置", "ja": "モデル設定"},
	"config_mode":           {"en": "Mode Settings", "zh": "运行模式", "ja": "動作モード"},
	"config_lang":           {"en": "Language", "zh": "语言设置", "ja": "言語設定"},
	"config_theme":          {"en": "Theme", "zh": "主题设置", "ja": "テーマ"},
	"mode_manual":           {"en": "Manual", "zh": "手动", "ja": "手動"},
	"mode_auto":             {"en": "Auto", "zh": "自动", "ja": "自動"},
	"mode_cruise":           {"en": "Cruise", "zh": "巡航", "ja": "クルーズ"},
	"cruise_interval_title": {"en": "Cruise Patrol Interval", "zh": "巡航巡查间隔", "ja": "クルーズ巡回間隔"},
	"cruise_interval_label": {"en": "Interval:", "zh": "间隔:", "ja": "間隔:"},
	"cruise_interval_desc":  {"en": "Time between cruise patrol scans (in seconds, min 60)", "zh": "巡航模式定期巡查的时间间隔（秒，最小60）", "ja": "クルーズモードの巡回間隔（秒、最小60）"},
	"mode_manual_desc":      {"en": "Review each suggestion with /run accept or /run all", "zh": "使用 /run accept 或 /run all 逐条确认", "ja": "/run accept または /run all で確認"},
	"mode_auto_desc":        {"en": "Fully automatic analysis-suggest-execute loop", "zh": "全自动 分析→建议→执行 循环", "ja": "全自動 分析→提案→実行 ループ"},
	"mode_cruise_desc":      {"en": "Auto + periodic monitoring and self-check", "zh": "自动 + 定时巡检和自主检查", "ja": "自動 + 定期監視と自己チェック"},
	"back":                  {"en": "Back", "zh": "返回", "ja": "戻る"},
	"navigate":              {"en": "Navigate", "zh": "导航", "ja": "操作"},
	"select":                {"en": "Select", "zh": "选择", "ja": "選択"},
	"no_goal":               {"en": "No active goal", "zh": "无目标", "ja": "目標なし"},
	"dark":                  {"en": "Dark", "zh": "暗色", "ja": "ダーク"},
	"light":                 {"en": "Light", "zh": "亮色", "ja": "ライト"},
	"save":                  {"en": "Save", "zh": "保存", "ja": "保存"},
	"cancel":                {"en": "Cancel", "zh": "取消", "ja": "キャンセル"},
}

func configText(key, lang string) string {
	if m, ok := configTexts[key]; ok {
		if v, ok := m[lang]; ok {
			return v
		}
		if v, ok := m["en"]; ok {
			return v
		}
	}
	return key
}

// ── Layout geometry ───────────────────────────────────────────────────

type layoutGeo struct {
	headerH, inputH, contentH int
	leftW, rightW             int
	gitH, goalH, logH         int
}

func (m Model) calcLayout() layoutGeo {
	w, h := m.width, m.height
	if w <= 0 {
		w = 80
	}
	if h <= 0 {
		h = 24
	}
	tok := layoutTokensForWidth(w)
	contentH := h - 8
	if contentH < tok.minContentH {
		contentH = tok.minContentH
	}
	leftW := w * tok.leftRatioPct / 100
	if leftW < tok.minLeftW {
		leftW = tok.minLeftW
	}
	rightW := w - leftW - 1
	if rightW < tok.minRightW {
		rightW = tok.minRightW
	}
	avail := contentH - 2
	if avail < tok.minPanelH*3 {
		avail = tok.minPanelH * 3
	}
	gitH := avail * tok.gitRatioPct / 100
	if gitH < tok.minPanelH {
		gitH = tok.minPanelH
	}
	goalH := avail * tok.goalRatioPct / 100
	if goalH < tok.minPanelH {
		goalH = tok.minPanelH
	}
	logH := avail - gitH - goalH
	if logH < tok.minPanelH {
		logH = tok.minPanelH
	}
	return layoutGeo{1, 2, contentH, leftW, rightW, gitH, goalH, logH}
}

// ── View entry ────────────────────────────────────────────────────────

func (m Model) View() tea.View {
	if !m.ready {
		return tea.NewView("Initializing Gitdex...")
	}
	var content string
	switch m.page {
	case PageConfig, PageConfigModel, PageConfigMode, PageConfigLang, PageConfigTheme:
		content = m.renderConfigView()
	default:
		content = m.renderMainView()
	}
	v := tea.NewView(content)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

// ── Config pages ──────────────────────────────────────────────────────

func (m Model) renderConfigView() string {
	w, h := m.width, m.height
	if w <= 0 {
		w = 80
	}
	if h <= 0 {
		h = 24
	}
	header := m.renderConfigHeader(w)
	divider := borderStyle().Render(strings.Repeat("─", w))

	var body string
	switch m.page {
	case PageConfigModel:
		body = m.renderConfigModelPage(w)
	case PageConfigMode:
		body = m.renderConfigModePage(w)
	case PageConfigLang:
		body = m.renderConfigLangPage(w)
	case PageConfigTheme:
		body = m.renderConfigThemePage(w)
	default:
		body = m.renderConfigMainPage(w)
	}

	bodyH := h - 3
	if bodyH < 3 {
		bodyH = 3
	}
	bodyLines := strings.Split(body, "\n")
	scrollBody := applyPanelScroll(bodyLines, m.panelScrolls[FocusLeft], w, bodyH)
	return header + "\n" + divider + "\n" + scrollBody
}

func (m Model) renderConfigHeader(w int) string {
	left := titleStyle().Render(" Gitdex") + " " + accentStyle().Render("◆ "+configText("config_title", m.language))
	sub := ""
	switch m.page {
	case PageConfigModel:
		sub = " / " + configText("config_model", m.language)
	case PageConfigMode:
		sub = " / " + configText("config_mode", m.language)
	case PageConfigLang:
		sub = " / " + configText("config_lang", m.language)
	case PageConfigTheme:
		sub = " / " + configText("config_theme", m.language)
	}
	if sub != "" {
		left += subtitleStyle().Render(sub)
	}
	right := configHelpStyle().Render("Esc:" + configText("back", m.language))
	pad := w - lipgloss.Width(left) - lipgloss.Width(right)
	if pad < 1 {
		pad = 1
	}
	return left + strings.Repeat(" ", pad) + right
}

func (m Model) renderConfigMainPage(w int) string {
	lang := m.language
	helperSum := m.configInfo.Helper.Provider + " / " + m.configInfo.Helper.Model
	plannerSum := m.configInfo.Planner.Provider
	if m.configInfo.Planner.Model != "" {
		plannerSum += " / " + m.configInfo.Planner.Model
	} else {
		plannerSum += " (same as helper)"
	}
	modeDisplay := m.mode
	if m.mode == "cruise" {
		modeDisplay += fmt.Sprintf("  (%s)", formatDuration(m.cruiseIntervalS))
	}
	items := []struct {
		label, value string
	}{
		{configText("config_model", lang), "Helper: " + helperSum + "  |  Planner: " + plannerSum},
		{configText("config_mode", lang), modeDisplay},
		{configText("config_lang", lang), m.configInfo.Language},
		{configText("config_theme", lang), m.configInfo.Theme},
	}
	var lines []string
	lines = append(lines, "")
	for i, item := range items {
		prefix := "    "
		style := configItemStyle()
		if i == m.configMenuIdx {
			prefix = "  " + accentStyle().Render(">") + " "
			style = configActiveStyle()
		}
		lines = append(lines, prefix+style.Render(item.label)+mutedStyle().Render("  "+item.value))
	}
	lines = append(lines, "")
	lines = append(lines, configHelpStyle().Render(fmt.Sprintf("  ↑↓ %s   Enter: %s   Esc: %s",
		configText("navigate", lang), configText("select", lang), configText("back", lang))))
	return strings.Join(lines, "\n")
}

// ── Model config page (old TUI style: bordered boxes + chips) ─────────

func (m Model) renderConfigModelPage(w int) string {
	boxW := clampInt(w-8, 24, 72)

	t := ts()
	boxActive := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(t.BorderFoc)).
		Padding(0, 1).Width(boxW)
	boxIdle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(t.Border)).
		Padding(0, 1).Width(boxW)
	choiceOn := lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.BgPanel)).
		Background(lipgloss.Color(t.Warning)).
		Bold(true).Padding(0, 1)
	choiceOff := lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Text)).
		Background(lipgloss.Color(t.Border)).
		Padding(0, 1)

	fi := m.configDraft.FieldIdx
	provID := draftProviders[m.configDraft.ProviderIdx]
	meta := providerMetaFor(provID)

	var b strings.Builder
	b.WriteString("\n")

	// Role selector (fi == 0)
	roleName := "Helper LLM"
	if m.configDraft.Role == RolePlanner {
		roleName = "Planner LLM"
	}
	roleBox := boxIdle
	if fi == 0 {
		roleBox = boxActive
	}
	helperChip := choiceOff.Render("Helper")
	plannerChip := choiceOff.Render("Planner")
	if m.configDraft.Role == RoleHelper {
		helperChip = choiceOn.Render("Helper")
	} else {
		plannerChip = choiceOn.Render("Planner")
	}
	b.WriteString(keyStyle().Render("  Role") + "  " + mutedStyle().Render("(which LLM to configure)") + "\n")
	b.WriteString("  " + roleBox.Render(helperChip+" "+plannerChip) + "\n")
	b.WriteString("  " + mutedStyle().Render("Configuring: "+roleName) + "\n\n")

	// Provider chips (fi == 1)
	b.WriteString(keyStyle().Render("  Provider") + "\n")
	var chips []string
	for _, p := range draftProviders {
		st := choiceOff
		if p == provID {
			st = choiceOn
		}
		chips = append(chips, st.Render(p))
	}
	chipLine := strings.Join(chips, " ")
	provBox := boxIdle
	if fi == 1 {
		provBox = boxActive
	}
	b.WriteString("  " + provBox.Render(chipLine) + "\n")
	b.WriteString("  " + mutedStyle().Render(meta.Label+" - "+meta.Kind) + "\n")
	if len(meta.RecommendedModels) > 0 {
		b.WriteString("  " + mutedStyle().Render("Recommended: "+strings.Join(meta.RecommendedModels, ", ")) + "\n")
	}
	b.WriteString("\n")

	// Model field (fi == 2)
	b.WriteString(keyStyle().Render("  Model") + "\n")
	if provID == "ollama" {
		b.WriteString(m.renderOllamaModelSelector(fi == 2, boxActive, boxIdle) + "\n\n")
	} else {
		b.WriteString("  " + m.renderTextField(m.configDraft.Model, fi == 2, boxActive, boxIdle) + "\n\n")
	}

	// Endpoint field (fi == 3)
	b.WriteString(keyStyle().Render("  Endpoint") + "\n")
	b.WriteString("  " + m.renderTextField(m.configDraft.Endpoint, fi == 3, boxActive, boxIdle) + "\n\n")

	// API Key Env field (fi == 4, only for non-ollama)
	if provID != "ollama" {
		b.WriteString(keyStyle().Render("  API Key Env") + "\n")
		b.WriteString("  " + m.renderTextField(m.configDraft.APIKeyEnv, fi == 4, boxActive, boxIdle) + "\n")
		if meta.APIKeyEnv != "" {
			b.WriteString("  " + configHelpStyle().Render("Set "+meta.APIKeyEnv+" env var") + "\n")
		}
		b.WriteString("\n")
	}

	// Footer
	if m.configEditing {
		b.WriteString(configHelpStyle().Render("  Type to edit   Esc: stop   Tab: next   Enter: "+configText("save", m.language)+"   Ctrl+V: paste") + "\n")
	} else {
		b.WriteString(configHelpStyle().Render("  ←→: role/provider   Tab/↑↓: fields   Enter: edit/select   Esc: back   Ctrl+V: paste") + "\n")
	}
	return b.String()
}

func (m Model) renderOllamaModelSelector(active bool, boxActive, boxIdle lipgloss.Style) string {
	box := boxIdle
	if active {
		box = boxActive
	}
	if m.ollamaFetching {
		return "  " + box.Render(mutedStyle().Render("Loading models..."))
	}
	if m.ollamaFetchError != "" {
		return "  " + box.Render(dangerStyle().Render("Error: "+m.ollamaFetchError)+"\n"+
			mutedStyle().Render("  Fallback: ")+valueStyle().Render(m.configDraft.Model))
	}
	if len(m.ollamaModels) == 0 {
		return "  " + box.Render(mutedStyle().Render("No local models found. Run: ollama pull <model>"))
	}
	var lines []string
	for i, om := range m.ollamaModels {
		prefix := "  "
		st := configItemStyle()
		if i == m.ollamaModelIdx {
			prefix = accentStyle().Render("> ")
			st = configActiveStyle()
		}
		check := ""
		if m.configDraft.Model == om.Name {
			check = " " + configCheckStyle().Render("●")
		}
		detail := ""
		if om.ParamSize != "" {
			detail = " " + mutedStyle().Render("("+om.ParamSize)
			if om.Quant != "" {
				detail += mutedStyle().Render(" " + om.Quant)
			}
			detail += mutedStyle().Render(")")
		}
		lines = append(lines, prefix+st.Render(om.Name)+detail+check)
	}
	return "  " + box.Render(strings.Join(lines, "\n"))
}

func (m Model) renderTextField(value string, active bool, boxActive, boxIdle lipgloss.Style) string {
	if !active {
		display := value
		if display == "" {
			display = mutedStyle().Render("(empty)")
		}
		return boxIdle.Render(display)
	}
	if m.configEditing {
		before, after := splitAtRunePos(value, m.configDraft.CursorAt)
		cursor := cursorStyle().Render("|")
		display := before + cursor + after
		if value == "" {
			display = cursor + mutedStyle().Render("type here...")
		}
		return boxActive.Render(display)
	}
	display := value
	if display == "" {
		display = mutedStyle().Render("(empty)")
	}
	return boxActive.Render(display + "  " + accentStyle().Render("← Enter to edit"))
}

func (m Model) renderConfigModePage(w int) string {
	lang := m.language
	modes := []struct {
		key, desc, val string
	}{
		{"mode_manual", "mode_manual_desc", "manual"},
		{"mode_auto", "mode_auto_desc", "auto"},
		{"mode_cruise", "mode_cruise_desc", "cruise"},
	}
	var lines []string
	lines = append(lines, "")
	for i, md := range modes {
		prefix := "    "
		style := configItemStyle()
		if i == m.configModeIdx {
			prefix = "  " + accentStyle().Render(">") + " "
			style = configActiveStyle()
		}
		check := ""
		if m.mode == md.val {
			check = " " + configCheckStyle().Render("[*]")
		}
		lines = append(lines, prefix+style.Render(configText(md.key, lang))+check)
		lines = append(lines, "      "+mutedStyle().Render(configText(md.desc, lang)))
	}

	lines = append(lines, "")
	lines = append(lines, sectionHeaderStyle().Render("  "+configText("cruise_interval_title", lang)))

	intervalPrefix := "    "
	intervalStyle := configItemStyle()
	if m.configModeIdx == 3 {
		intervalPrefix = "  " + accentStyle().Render(">") + " "
		intervalStyle = configActiveStyle()
	}

	intervalDisplay := ""
	if m.editingInterval {
		cursor := cursorStyle().Render("│")
		intervalDisplay = intervalStyle.Render(m.intervalBuf+cursor) +
			mutedStyle().Render(" s")
	} else {
		intervalDisplay = intervalStyle.Render(fmt.Sprintf("%d", m.cruiseIntervalS)) +
			mutedStyle().Render(" s  ("+formatDuration(m.cruiseIntervalS)+")")
	}
	lines = append(lines, intervalPrefix+keyStyle().Render(configText("cruise_interval_label", lang))+
		"  "+intervalDisplay)
	lines = append(lines, "      "+mutedStyle().Render(configText("cruise_interval_desc", lang)))

	lines = append(lines, "")
	help := fmt.Sprintf("  ↑↓ %s   Enter: %s   Esc: %s",
		configText("navigate", lang), configText("select", lang), configText("back", lang))
	lines = append(lines, configHelpStyle().Render(help))
	return strings.Join(lines, "\n")
}

func formatDuration(seconds int) string {
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	m := seconds / 60
	s := seconds % 60
	if s == 0 {
		return fmt.Sprintf("%dmin", m)
	}
	return fmt.Sprintf("%dmin%ds", m, s)
}

func (m Model) renderConfigLangPage(w int) string {
	lang := m.language
	langs := []struct {
		label, val string
	}{{"English", "en"}, {"中文", "zh"}, {"日本語", "ja"}}
	var lines []string
	lines = append(lines, "")
	for i, l := range langs {
		prefix := "    "
		style := configItemStyle()
		if i == m.configLangIdx {
			prefix = "  " + accentStyle().Render(">") + " "
			style = configActiveStyle()
		}
		check := ""
		if m.language == l.val {
			check = " " + configCheckStyle().Render("[*]")
		}
		lines = append(lines, prefix+style.Render(l.label)+check)
	}
	lines = append(lines, "")
	lines = append(lines, configHelpStyle().Render(fmt.Sprintf("  ↑↓ %s   Enter: %s   Esc: %s",
		configText("navigate", lang), configText("select", lang), configText("back", lang))))
	return strings.Join(lines, "\n")
}

func (m Model) renderConfigThemePage(w int) string {
	lang := m.language
	allThemes := theme.Names()
	var lines []string
	lines = append(lines, "")
	for i, name := range allThemes {
		prefix := "    "
		st := configItemStyle()
		if i == m.configThemeIdx {
			prefix = "  " + accentStyle().Render(">") + " "
			st = configActiveStyle()
		}
		check := ""
		if m.configInfo.Theme == name {
			check = " " + configCheckStyle().Render("[*]")
		}
		lines = append(lines, prefix+st.Render(name)+check)
	}
	lines = append(lines, "")
	lines = append(lines, configHelpStyle().Render(fmt.Sprintf("  ↑↓ %s   Enter: %s   Esc: %s",
		configText("navigate", lang), configText("select", lang), configText("back", lang))))
	return strings.Join(lines, "\n")
}

// ── Main view ─────────────────────────────────────────────────────────

func (m Model) renderMainView() string {
	geo := m.calcLayout()
	w := m.width
	if w <= 0 {
		w = 80
	}
	m.tabsComp.SetWidth(w)
	tabBar := m.tabsComp.View()
	input := m.renderInput(w)

	contentH := geo.contentH
	if contentH < 3 {
		contentH = 3
	}

	activeView := m.tabsComp.CurrentView()
	var content string

	switch activeView {
	case tuictx.GitView:
		content = m.renderGitFullView(w, contentH, geo)
	case tuictx.WorkspaceView:
		content = m.renderWorkspaceFullView(w, contentH)
	case tuictx.GitHubView:
		content = m.renderGitHubFullView(w, contentH)
	default:
		mainW := geo.leftW
		sideW := geo.rightW
		m.agentTable.SetDimensions(mainW, contentH)

		left := m.agentTable.View()
		right := m.renderRightPanel(sideW, contentH, geo)
		sep := buildVSep(contentH)
		content = lipgloss.JoinHorizontal(lipgloss.Top, left, sep, right)
	}

	thinDiv := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ts().Border)).
		Render(strings.Repeat("─", w))

	m.footerComp.SetWidth(w)
	footerView := m.footerComp.View()
	statusLine := m.renderStatusLine(w)
	base := tabBar + "\n" + thinDiv + "\n" + content + "\n" + thinDiv + "\n" + input + "\n" + statusLine + "\n" + footerView
	if m.showCommandPalette {
		base += "\n" + thinDiv + "\n" + m.renderCommandPalette(w)
	}
	if m.showHelpOverlay {
		return base + "\n" + thinDiv + "\n" + m.renderHelpOverlay(w)
	}
	return base
}

// ── Tab-specific full views ───────────────────────────────────────────

func (m Model) renderGitFullView(w, h int, geo layoutGeo) string {
	mainW := geo.leftW
	sideW := geo.rightW

	gitContent := m.renderGitPanel(mainW-2, h-2)
	left := m.panelBox(FocusGit, mainW, h).Render(
		panelTitle("Repository", true) + "\n" + gitContent,
	)

	var sb strings.Builder
	sb.WriteString(sectionHeaderStyle().Render(" ◆ Branches") + "\n")
	for _, b := range m.gitInfo.LocalBranches {
		marker := mutedStyle().Render("  ")
		if b.IsCurrent {
			marker = successBoldStyle().Render("● ")
		}
		line := marker + accentStyle().Render(b.Name)
		if b.Upstream != "" {
			line += mutedStyle().Render(" → " + b.Upstream)
		}
		if b.Ahead > 0 {
			line += warningStyle().Render(fmt.Sprintf(" ↑%d", b.Ahead))
		}
		if b.Behind > 0 {
			line += dangerStyle().Render(fmt.Sprintf(" ↓%d", b.Behind))
		}
		sb.WriteString(" " + line + "\n")
	}
	if m.gitInfo.Stash > 0 {
		sb.WriteString("\n" + sectionHeaderStyle().Render(" ◆ Stash") + "\n")
		sb.WriteString(fmt.Sprintf("  %d entries\n", m.gitInfo.Stash))
	}
	if len(m.gitInfo.Tags) > 0 {
		sb.WriteString("\n" + sectionHeaderStyle().Render(" ◆ Tags") + "\n")
		for _, t := range m.gitInfo.Tags {
			sb.WriteString("  " + mutedStyle().Render(t) + "\n")
		}
	}
	right := m.panelBox(FocusGoals, sideW, h).Render(sb.String())
	sep := buildVSep(h)
	return lipgloss.JoinHorizontal(lipgloss.Top, left, sep, right)
}

func (m Model) renderWorkspaceFullView(w, h int) string {
	var lines []string
	lines = append(lines, sectionHeaderStyle().Render(" ◆ Working Tree"))
	if m.gitInfo.WorkingDirty == 0 {
		lines = append(lines, mutedStyle().Render("  Clean — no modified files"))
	} else {
		for _, f := range m.gitInfo.WorkingFiles {
			lines = append(lines, "  "+warningStyle().Render("M")+" "+f)
		}
	}
	lines = append(lines, "")
	lines = append(lines, sectionHeaderStyle().Render(" ◆ Staging Area"))
	if m.gitInfo.StagingDirty == 0 {
		lines = append(lines, mutedStyle().Render("  Nothing staged"))
	} else {
		for _, f := range m.gitInfo.StagingFiles {
			lines = append(lines, "  "+successStyle().Render("A")+" "+f)
		}
	}
	return applyPanelScroll(lines, m.panelScrolls[FocusLeft], w, h)
}

func (m Model) renderGitHubFullView(w, h int) string {
	var lines []string
	lines = append(lines, sectionHeaderStyle().Render(" ◆ GitHub"))
	lines = append(lines, "")
	lines = append(lines, mutedStyle().Render("  GitHub view — use /creative to generate goals from issues & PRs"))
	lines = append(lines, "")
	lines = append(lines, dimLabel("repo")+"  "+valueStyle().Render(m.configInfo.RepoRoot))
	return applyPanelScroll(lines, m.panelScrolls[FocusLeft], w, h)
}

func buildVSep(h int) string {
	ch := borderStyle().Render("│")
	lines := make([]string, h)
	for i := range lines {
		lines[i] = ch
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderStatusLine(w int) string {
	t := ts()
	parts := []string{
		modePill(m.mode),
		statusPill(m.analyzing, m.executing),
	}
	if m.activeFlow != "idle" {
		parts = append(parts, lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.Accent)).
			Render("flow:"+m.activeFlow))
	}
	if m.activeGoal != "" {
		goalLabel := m.activeGoal
		if len(goalLabel) > 30 {
			goalLabel = goalLabel[:27] + "..."
		}
		parts = append(parts, lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.Info)).
			Render("goal:"+goalLabel))
	}
	if m.lastTokenMax > 0 {
		pct := m.lastTokenUsed * 100 / m.lastTokenMax
		parts = append(parts, lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.TextMuted)).
			Render(fmt.Sprintf("ctx:%s/%s(%d%%)",
				formatTokenCount(m.lastTokenUsed),
				formatTokenCount(m.lastTokenMax), pct)))
	}
	left := strings.Join(parts, "  ")
	pad := w - lipgloss.Width(left)
	if pad < 0 {
		pad = 0
	}
	return left + strings.Repeat(" ", pad)
}

func modePill(mode string) string {
	t := ts()
	var fg, bg string
	label := strings.ToUpper(mode)
	switch mode {
	case "manual":
		fg, bg = t.BgPanel, t.Success
	case "auto":
		fg, bg = t.BgPanel, t.Warning
	case "cruise":
		fg, bg = t.BgPanel, t.Danger
	default:
		fg, bg = t.Text, t.Border
	}
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(fg)).
		Background(lipgloss.Color(bg)).
		Bold(true).
		Padding(0, 1).
		Render(label)
}

func statusPill(analyzing, executing bool) string {
	t := ts()
	if analyzing {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.BgPanel)).
			Background(lipgloss.Color(t.Warning)).
			Padding(0, 1).
			Render("◉ ANALYZING")
	}
	if executing {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.BgPanel)).
			Background(lipgloss.Color(t.Warning)).
			Padding(0, 1).
			Render("◉ EXECUTING")
	}
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.BgPanel)).
		Background(lipgloss.Color(t.Success)).
		Padding(0, 1).
		Render("● READY")
}

func infoPill(label, value string) string {
	t := ts()
	lbl := lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.TextMuted)).
		Render(label)
	val := lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Text)).
		Render(value)
	return lbl + " " + val
}

func (m Model) renderHeader(w int) string {
	title := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ts().Primary)).
		Bold(true).
		Render(" ◆ Gitdex")

	mp := modePill(m.mode)
	sp := statusPill(m.analyzing, m.executing)

	cruiseInfo := ""
	if m.mode == "cruise" {
		status := "idle"
		if m.cruiseCycleActive {
			status = "active"
		}
		cruiseInfo = " " + mutedStyle().Render(fmt.Sprintf("⏱%s [%s]", formatDuration(m.cruiseIntervalS), status))
	}

	left := title + " " + mp + cruiseInfo + " " + sp

	var rightParts []string

	if m.lastTokenMax > 0 {
		pct := m.lastTokenUsed * 100 / m.lastTokenMax
		barStr := renderTokenBar(m.lastTokenUsed, m.lastTokenMax, 12)
		pctLabel := fmt.Sprintf("%d%%", pct)
		ctxStyle := mutedStyle()
		if pct > 80 {
			ctxStyle = warningStyle()
		}
		if pct > 95 {
			ctxStyle = dangerStyle()
		}
		rightParts = append(rightParts, infoPill("ctx", barStr+" "+ctxStyle.Render(pctLabel)))
	}

	if m.activeFlow != "idle" {
		rightParts = append(rightParts, infoPill("flow", flowStyle().Render(m.activeFlow)))
	}

	metrics := observability.SnapshotMetrics()
	if metrics.CommandsTotal > 0 {
		rightParts = append(rightParts, infoPill("cmd",
			fmt.Sprintf("%d/%d", metrics.CommandsSucceeded, metrics.CommandsTotal)))
	}
	if metrics.ReplanAttempts > 0 {
		rightParts = append(rightParts, warningStyle().Render(fmt.Sprintf("replan:%d", metrics.ReplanAttempts)))
	}

	if metrics.ProviderAvailable == 1 {
		rightParts = append(rightParts, successStyle().Render("llm ●"))
	} else {
		rightParts = append(rightParts, dangerStyle().Render("llm ○"))
	}

	right := strings.Join(rightParts, "  ")
	pad := w - lipgloss.Width(left) - lipgloss.Width(right)
	if pad < 1 {
		pad = 1
	}
	return left + strings.Repeat(" ", pad) + right
}

func renderTokenBar(used, max, barW int) string {
	if max == 0 || barW <= 0 {
		return ""
	}
	filled := used * barW / max
	if filled > barW {
		filled = barW
	}
	t := ts()
	fillClr := t.Success
	pct := used * 100 / max
	if pct > 80 {
		fillClr = t.Warning
	}
	if pct > 95 {
		fillClr = t.Danger
	}
	fillStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(fillClr))
	emptyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Border))
	return fillStyle.Render(strings.Repeat("█", filled)) + emptyStyle.Render(strings.Repeat("░", barW-filled))
}

func formatTokenCount(n int) string {
	if n >= 1000 {
		return fmt.Sprintf("%.1fk", float64(n)/1000)
	}
	return fmt.Sprintf("%d", n)
}

func (m Model) renderInput(w int) string {
	t := ts()
	prefix := ""
	if m.composerFocus {
		prefix = lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.BgPanel)).
			Background(lipgloss.Color(t.Accent)).
			Bold(true).
			Padding(0, 1).
			Render("❯")
		prefix += " "
	} else {
		prefix = inactiveInputStyle().Render(" ❯ ")
	}
	text := m.composerText
	cursor := ""
	if m.composerFocus {
		cursor = cursorStyle().Render("│")
	}
	hint := ""
	if text == "" && !m.composerFocus {
		hint = mutedStyle().Render("Click here or Tab to focus")
	} else if text == "" && m.composerFocus {
		hint = mutedStyle().Render("/goal <text>  /run  /mode  /creative  /config  /palette  /help")
	}
	inputLine := prefix + text + cursor + hint

	helpItems := []string{"/goal", "/run", "/mode", "/creative", "/config", "/palette", "/analyze", "/help", "/clear"}
	var helpParts []string
	for _, item := range helpItems {
		helpParts = append(helpParts, dimStyle().Render(item))
	}
	separator := dimStyle().Render("  ")
	helpLine := " " + strings.Join(helpParts, separator)
	helpLine += "   " + keyStyle().Render("Tab") + dimStyle().Render(":focus")
	helpLine += "  " + keyStyle().Render("?") + dimStyle().Render(":help")
	helpLine += "  " + keyStyle().Render("q") + dimStyle().Render(":quit")

	return inputLine + "\n" + helpLine
}

// ── Left panel ────────────────────────────────────────────────────────

func (m Model) renderLeftPanel(w, h int) string {
	var lines []string
	goalTotal := len(m.goals)
	goalDone := 0
	for _, g := range m.goals {
		if g.Completed {
			goalDone++
		}
	}
	if goalTotal > 0 {
		barW := w - 18
		if barW < 5 {
			barW = 5
		}
		lines = append(lines, fmt.Sprintf(" %s %s %d/%d",
			keyStyle().Render("Goal"), renderProgressBar(goalDone, goalTotal, barW), goalDone, goalTotal))
		todoTotal, todoDone := 0, 0
		for _, g := range m.goals {
			if !g.Completed && g.Title == m.activeGoal {
				for _, t := range g.Todos {
					todoTotal++
					if t.Completed {
						todoDone++
					}
				}
				break
			}
		}
		if todoTotal > 0 {
			lines = append(lines, fmt.Sprintf(" %s %s %d/%d",
				keyStyle().Render("Todo"), renderProgressBar(todoDone, todoTotal, barW), todoDone, todoTotal))
		}
	} else {
		lines = append(lines, mutedStyle().Render(" "+configText("no_goal", m.language)))
	}
	lines = append(lines, "")

	wrapW := w - 10
	if wrapW < 10 {
		wrapW = 10
	}

	if len(m.suggestions) > 0 {
		hdr := sectionHeaderStyle()
		if m.focusZone == FocusLeft {
			hdr = focusedHeaderStyle()
		}
		lines = append(lines, hdr.Render(" ◆ Suggestions")+
			mutedStyle().Render(fmt.Sprintf(" (%d)", len(m.suggestions))))
		lines = append(lines, "")
		for idx, sg := range m.suggestions {
			icon, tag := suggStatusIcon(sg.Status)
			name := firstLine(sg.Item.Name)
			if name == "" {
				name = "(unnamed)"
			}
			toolLabel := toolTypePill(sg.Item.Action.ToolLabel())

			numStr := mutedStyle().Render(fmt.Sprintf("#%d", idx+1))
			headerLine := fmt.Sprintf(" %s %s %s %s", icon, tag, numStr, toolLabel)
			lines = append(lines, headerLine)

			nameW := wrapW - 4
			if nameW < 20 {
				nameW = 20
			}
			for _, wl := range wrapText(name, nameW) {
				lines = append(lines, "   "+valueStyle().Render(wl))
			}

			cmdStr := ""
			if sg.Item.Action.Command != "" {
				cmdStr = sg.Item.Action.Command
			} else if sg.Item.Action.FilePath != "" {
				cmdStr = fmt.Sprintf("%s %s", sg.Item.Action.FileOp, sg.Item.Action.FilePath)
			}
			if cmdStr != "" {
				for _, wl := range wrapText(cmdStr, wrapW-6) {
					lines = append(lines, "   "+commandStyle().Render("$ "+wl))
				}
			}
			if sg.Item.Reason != "" {
				reason := firstLine(sg.Item.Reason)
				for _, wl := range wrapText(reason, wrapW-6) {
					lines = append(lines, "   "+infoStyle().Render("→ "+wl))
				}
			}
			if sg.Status == StatusDone && sg.Output != "" {
				out := firstLine(sg.Output)
				for _, wl := range wrapText(out, wrapW-4) {
					lines = append(lines, "   "+successStyle().Render(wl))
				}
			}
			if sg.Status == StatusFailed && sg.Error != "" {
				errMsg := firstLine(sg.Error)
				for _, wl := range wrapText(errMsg, wrapW-4) {
					lines = append(lines, "   "+dangerStyle().Render("✗ "+wl))
				}
			}
			lines = append(lines, "")
		}
	}

	if len(m.roundHistory) > 0 {
		lines = append(lines, sectionHeaderStyle().Render(" ◆ Completed"))
		shown := m.roundHistory
		if len(shown) > 5 {
			shown = shown[len(shown)-5:]
		}
		for _, r := range shown {
			for _, cmd := range r.Commands {
				for _, wl := range wrapText(cmd, w-6) {
					lines = append(lines, " "+successStyle().Render("✓ ")+mutedStyle().Render(wl))
				}
			}
		}
		lines = append(lines, "")
	}

	if len(lines) <= 2 && len(m.suggestions) == 0 {
		lines = append(lines, "")
		lines = append(lines, mutedStyle().Render("  Use /goal <text> to set a goal."))
		lines = append(lines, mutedStyle().Render("  Type /help for commands."))
		lines = append(lines, "")
	}

	return applyPanelScroll(lines, m.panelScrolls[FocusLeft], w, h)
}

func suggStatusIcon(st SuggestionStatus) (string, string) {
	t := ts()
	mkPill := func(icon, label, fg, bg string) (string, string) {
		iconStr := lipgloss.NewStyle().Foreground(lipgloss.Color(fg)).Render(icon)
		pillStr := lipgloss.NewStyle().
			Foreground(lipgloss.Color(fg)).
			Background(lipgloss.Color(bg)).
			Padding(0, 1).
			Render(label)
		return iconStr, pillStr
	}
	switch st {
	case StatusExecuting:
		return mkPill("◉", "RUN", t.BgPanel, t.Warning)
	case StatusDone:
		return mkPill("✓", "OK", t.BgPanel, t.Success)
	case StatusFailed:
		return mkPill("✗", "ERR", t.BgPanel, t.Danger)
	case StatusSkipped:
		return mkPill("○", "SKIP", t.TextMuted, t.Border)
	default:
		return mkPill("◌", "WAIT", t.TextMuted, t.Border)
	}
}

func toolTypePill(toolType string) string {
	t := ts()
	clr := t.Accent
	switch toolType {
	case "git_command":
		clr = t.Success
	case "shell_command":
		clr = t.Warning
	case "file_write", "file_read":
		clr = t.Info
	case "github_op":
		clr = t.Secondary
	}
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(clr)).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(clr)).
		Padding(0, 1).
		Render(toolType)
}

// ── Right panel ───────────────────────────────────────────────────────

func (m Model) renderRightPanel(w, h int, geo layoutGeo) string {
	innerW := w - 4
	if innerW < 10 {
		innerW = 10
	}
	innerGitH := geo.gitH - 2
	innerGoalH := geo.goalH - 2
	innerLogH := geo.logH - 2
	if innerGitH < 1 {
		innerGitH = 1
	}
	if innerGoalH < 1 {
		innerGoalH = 1
	}
	if innerLogH < 1 {
		innerLogH = 1
	}

	git := m.renderGitPanel(innerW, innerGitH)
	goal := m.renderGoalPanel(innerW, innerGoalH)
	log := m.renderLogPanel(innerW, innerLogH)

	gitTitle := panelTitle("Repository", m.focusZone == FocusGit)
	goalTitle := panelTitle("Goals", m.focusZone == FocusGoals)
	logTitle := panelTitle("Log", m.focusZone == FocusLog)

	gitBox := m.panelBox(FocusGit, w, geo.gitH).Render(gitTitle + "\n" + git)
	goalBox := m.panelBox(FocusGoals, w, geo.goalH).Render(goalTitle + "\n" + goal)
	logBox := m.panelBox(FocusLog, w, geo.logH).Render(logTitle + "\n" + log)

	return gitBox + "\n" + goalBox + "\n" + logBox
}

func panelTitle(title string, focused bool) string {
	t := ts()
	if focused {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.Primary)).
			Bold(true).
			Render(" ◆ " + title)
	}
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.TextMuted)).
		Bold(true).
		Render(" ◇ " + title)
}

func (m Model) panelBox(zone FocusZone, w, h int) lipgloss.Style {
	t := ts()
	borderClr := lipgloss.Color(t.Border)
	if m.focusZone == zone {
		borderClr = lipgloss.Color(t.BorderFoc)
	}
	boxW := w - 2
	boxH := h - 2
	if boxW < 1 {
		boxW = 1
	}
	if boxH < 1 {
		boxH = 1
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderClr).
		Width(boxW).
		Height(boxH)
}

func (m Model) renderGitPanel(w, h int) string {
	var lines []string
	gi := m.gitInfo

	if gi.Branch != "" {
		br := successBoldStyle().Render(gi.Branch)
		if gi.Detached {
			br += " " + statusTag("detached", ts().Danger)
		}
		lines = append(lines, fmt.Sprintf(" %s %s", dimLabel("branch"), br))
	}

	wtLabel := statusTag("clean", ts().Success)
	if gi.WorkingDirty > 0 {
		wtLabel = statusTag(fmt.Sprintf("%d changed", gi.WorkingDirty), ts().Warning)
	}
	saLabel := statusTag("clean", ts().Success)
	if gi.StagingDirty > 0 {
		saLabel = statusTag(fmt.Sprintf("%d staged", gi.StagingDirty), ts().Accent)
	}
	lines = append(lines, fmt.Sprintf(" %s %s  %s %s",
		dimLabel("work"), wtLabel, dimLabel("stage"), saLabel))

	if gi.Ahead > 0 || gi.Behind > 0 {
		var parts []string
		if gi.Ahead > 0 {
			parts = append(parts, statusTag(fmt.Sprintf("↑%d ahead", gi.Ahead), ts().Warning))
		}
		if gi.Behind > 0 {
			parts = append(parts, statusTag(fmt.Sprintf("↓%d behind", gi.Behind), ts().Danger))
		}
		lines = append(lines, " "+dimLabel("sync")+" "+strings.Join(parts, " "))
	}

	var flags []string
	if gi.MergeInProgress {
		flags = append(flags, statusTag("merge", ts().Danger))
	}
	if gi.RebaseInProgress {
		flags = append(flags, statusTag("rebase", ts().Danger))
	}
	if gi.CherryInProgress {
		flags = append(flags, statusTag("cherry-pick", ts().Danger))
	}
	if gi.BisectInProgress {
		flags = append(flags, statusTag("bisect", ts().Danger))
	}
	if len(flags) > 0 {
		lines = append(lines, " "+dimLabel("state")+" "+strings.Join(flags, " "))
	}

	if len(gi.Remotes) > 0 {
		lines = append(lines, "")
		lines = append(lines, " "+dimLabel("remotes"))
		nameW := 0
		for _, r := range gi.Remotes {
			if len(r.Name) > nameW {
				nameW = len(r.Name)
			}
		}
		for _, r := range gi.Remotes {
			padded := r.Name + strings.Repeat(" ", nameW-len(r.Name))
			lines = append(lines, fmt.Sprintf("   %s  %s",
				accentStyle().Render(padded), mutedStyle().Render(r.FetchURL)))
		}
	}

	if len(gi.LocalBranches) > 0 {
		lines = append(lines, "")
		lines = append(lines, " "+dimLabel("branches"))
		for _, b := range gi.LocalBranches {
			marker := mutedStyle().Render("  ")
			if b.IsCurrent {
				marker = successBoldStyle().Render("● ")
			}
			brLine := marker + accentStyle().Render(b.Name)
			if b.Upstream != "" {
				brLine += mutedStyle().Render(" → " + b.Upstream)
			}
			lines = append(lines, " "+brLine)
		}
	}

	if gi.Stash > 0 {
		lines = append(lines, " "+dimLabel("stash")+" "+mutedStyle().Render(fmt.Sprintf("%d entries", gi.Stash)))
	}
	if gi.UserName != "" {
		lines = append(lines, " "+dimLabel("user")+" "+valueStyle().Render(gi.UserName+" <"+gi.UserEmail+">"))
	}

	return applyPanelScroll(lines, m.panelScrolls[FocusGit], w, h)
}

func dimLabel(label string) string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(ts().TextMuted)).
		Bold(true).
		Render(label + ":")
}

func statusTag(text, color string) string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(color)).
		Render(text)
}

func (m Model) renderGoalPanel(w, h int) string {
	var lines []string
	if len(m.goals) == 0 {
		lines = append(lines, mutedStyle().Render("  "+configText("no_goal", m.language)))
		return applyPanelScroll(lines, m.panelScrolls[FocusGoals], w, h)
	}

	totalDone, totalGoals := 0, 0
	for _, g := range m.goals {
		if g.Completed {
			totalDone++
		}
		totalGoals++
	}

	barW := w - 20
	if barW < 5 {
		barW = 5
	}
	lines = append(lines, fmt.Sprintf(" %s %d/%d",
		renderProgressBar(totalDone, totalGoals, barW), totalDone, totalGoals))
	lines = append(lines, "")

	var completed, pending []int
	activeIdx := -1
	for i, g := range m.goals {
		if g.Completed {
			completed = append(completed, i)
		} else if activeIdx == -1 && g.Title == m.activeGoal {
			activeIdx = i
		} else if !g.Completed {
			pending = append(pending, i)
		}
	}
	if activeIdx == -1 && len(pending) > 0 {
		activeIdx = pending[0]
		pending = pending[1:]
	}

	show := completed
	if len(show) > 3 {
		show = show[len(show)-3:]
	}
	goalW := w - 6
	if goalW < 20 {
		goalW = 20
	}
	for _, idx := range show {
		for i, wl := range wrapText(m.goals[idx].Title, goalW) {
			if i == 0 {
				lines = append(lines, " "+successStyle().Render("✓")+mutedStyle().Render(" "+wl))
			} else {
				lines = append(lines, "     "+mutedStyle().Render(wl))
			}
		}
	}

	if activeIdx >= 0 {
		g := m.goals[activeIdx]
		todoDone, todoTotal := dotgitdex.GoalProgress(g)

		lines = append(lines, "")
		activeTag := lipgloss.NewStyle().
			Foreground(lipgloss.Color(ts().BgPanel)).
			Background(lipgloss.Color(ts().Warning)).
			Bold(true).
			Padding(0, 1).
			Render("ACTIVE")
		for i, wl := range wrapText(g.Title, goalW-lipgloss.Width(activeTag)-2) {
			if i == 0 {
				lines = append(lines, " "+activeTag+" "+accentStyle().Render(wl))
			} else {
				lines = append(lines, "     "+accentStyle().Render(wl))
			}
		}

		if todoTotal > 0 {
			todoBarW := goalW - 12
			if todoBarW < 5 {
				todoBarW = 5
			}
			lines = append(lines, fmt.Sprintf("   %s %d/%d",
				renderProgressBar(todoDone, todoTotal, todoBarW), todoDone, todoTotal))
		}

		todoW := w - 8
		if todoW < 16 {
			todoW = 16
		}
		for _, td := range g.Todos {
			if td.Completed {
				for i, wl := range wrapText(td.Title, todoW) {
					if i == 0 {
						lines = append(lines, "   "+successStyle().Render("✓")+mutedStyle().Render(" "+wl))
					} else {
						lines = append(lines, "       "+mutedStyle().Render(wl))
					}
				}
			} else {
				for i, wl := range wrapText(td.Title, todoW) {
					if i == 0 {
						lines = append(lines, "   "+mutedStyle().Render("○")+valueStyle().Render(" "+wl))
					} else {
						lines = append(lines, "       "+valueStyle().Render(wl))
					}
				}
			}
		}
	}

	if len(pending) > 0 {
		lines = append(lines, "")
	}
	for _, idx := range pending {
		for i, wl := range wrapText(m.goals[idx].Title, goalW) {
			if i == 0 {
				lines = append(lines, " "+mutedStyle().Render("○ "+wl))
			} else {
				lines = append(lines, "     "+mutedStyle().Render(wl))
			}
		}
	}

	return applyPanelScroll(lines, m.panelScrolls[FocusGoals], w, h)
}

func (m Model) renderLogPanel(w, h int) string {
	var lines []string
	entries := m.opLog.Entries()
	if len(entries) == 0 {
		lines = append(lines, mutedStyle().Render("  No log entries yet"))
		return applyPanelScroll(lines, m.panelScrolls[FocusLog], w, h)
	}
	shown := entries
	if len(shown) > 30 {
		shown = shown[len(shown)-30:]
	}

	if !m.detailPaneOpen {
		summaryW := w - 4
		if summaryW < 20 {
			summaryW = 20
		}
		detailW := w - 6
		if detailW < 16 {
			detailW = 16
		}
		for _, e := range shown {
			ts := tsTimeStyle().Render(e.Timestamp.Format("15:04:05"))
			icon := e.Icon()
			summary := firstLine(e.Summary)
			for i, wl := range wrapText(summary, summaryW-10) {
				if i == 0 {
					lines = append(lines, fmt.Sprintf(" %s %s %s", icon, ts, valueStyle().Render(wl)))
				} else {
					lines = append(lines, "            "+valueStyle().Render(wl))
				}
			}
			if e.Detail != "" {
				detail := firstLine(e.Detail)
				for _, wl := range wrapText(detail, detailW) {
					lines = append(lines, "     "+mutedStyle().Render(wl))
				}
			}
		}
		return applyPanelScroll(lines, m.panelScrolls[FocusLog], w, h)
	}

	// Detail pane mode: list on left, expanded selected detail on right.
	listW := w * 45 / 100
	if listW < 20 {
		listW = 20
	}
	detailW := w - listW - 1
	if detailW < 20 {
		detailW = 20
		listW = w - detailW - 1
	}
	if listW < 12 {
		listW = 12
	}
	selected := m.logCursor
	if selected < 0 {
		selected = 0
	}
	if selected >= len(shown) {
		selected = len(shown) - 1
	}

	listLines := []string{sectionHeaderStyle().Render(" List")}
	for i, e := range shown {
		icon := e.Icon()
		summary := firstLine(e.Summary)
		prefix := "  "
		st := valueStyle()
		if i == selected {
			prefix = " " + accentStyle().Render(">")
			st = accentStyle()
		}
		for j, wl := range wrapText(summary, listW-8) {
			if j == 0 {
				listLines = append(listLines, fmt.Sprintf("%s %s %s", prefix, icon, st.Render(wl)))
			} else {
				listLines = append(listLines, "    "+st.Render(wl))
			}
		}
	}
	left := applyPanelScroll(listLines, 0, listW, h)

	sel := shown[selected]
	detailLines := []string{
		sectionHeaderStyle().Render(" Detail"),
		"",
		mutedStyle().Render(sel.Timestamp.Format("15:04:05")) + " " + valueStyle().Render(sel.Summary),
	}
	detailText := strings.TrimSpace(sel.Detail)
	if detailText == "" {
		detailText = "(no detail)"
	}
	detailLines = append(detailLines, "")
	for _, wl := range wrapText(detailText, detailW-2) {
		detailLines = append(detailLines, " "+mutedStyle().Render(wl))
	}
	detailLines = append(detailLines, "")
	detailLines = append(detailLines, mutedStyle().Render(" Enter: collapse  ↑/↓: move"))
	right := applyPanelScroll(detailLines, 0, detailW, h)

	return lipgloss.JoinHorizontal(lipgloss.Top, left, buildVSep(h), right)
}

// ── Helpers ───────────────────────────────────────────────────────────

func applyPanelScroll(lines []string, offset, w, h int) string {
	for len(lines) < h {
		lines = append(lines, "")
	}
	if offset > len(lines)-h {
		offset = len(lines) - h
	}
	if offset < 0 {
		offset = 0
	}
	visible := lines[offset:]
	if len(visible) > h {
		visible = visible[:h]
	}
	result := make([]string, len(visible))
	for i, line := range visible {
		lw := lipgloss.Width(line)
		if lw < w {
			result[i] = line + strings.Repeat(" ", w-lw)
		} else {
			result[i] = line
		}
	}
	return strings.Join(result, "\n")
}

func padLine(line string, w int) string {
	if lw := lipgloss.Width(line); lw < w {
		return line + strings.Repeat(" ", w-lw)
	}
	return line
}

func firstLine(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	s = strings.SplitN(s, "\n", 2)[0]
	return strings.TrimSpace(s)
}

// truncStr is kept for backward-compatible tests; V3 never truncates.
func truncStr(s string, _ int) string {
	return firstLine(s)
}

func wrapText(s string, maxW int) []string {
	if maxW <= 0 {
		maxW = 40
	}
	s = strings.ReplaceAll(s, "\r\n", "\n")
	rawLines := strings.Split(s, "\n")
	var result []string
	for _, line := range rawLines {
		if line == "" {
			result = append(result, "")
			continue
		}
		runes := []rune(line)
		for len(runes) > maxW {
			result = append(result, string(runes[:maxW]))
			runes = runes[maxW:]
		}
		result = append(result, string(runes))
	}
	if len(result) == 0 {
		result = []string{""}
	}
	return result
}

func renderProgressBar(done, total, width int) string {
	if width <= 0 {
		width = 10
	}
	if total == 0 {
		return mutedStyle().Render(strings.Repeat("─", width))
	}
	filled := (done * width) / total
	if filled > width {
		filled = width
	}
	t := ts()
	pct := done * 100 / total
	fillClr := t.Success
	if pct < 50 {
		fillClr = t.Warning
	}
	if done == total {
		fillClr = t.Success
	}
	fillStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(fillClr))
	emptyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Border))
	return fillStyle.Render(strings.Repeat("█", filled)) + emptyStyle.Render(strings.Repeat("░", width-filled))
}

func splitAtRunePos(text string, pos int) (string, string) {
	n := utf8.RuneCountInString(text)
	if pos < 0 {
		pos = 0
	}
	if pos > n {
		pos = n
	}
	runes := []rune(text)
	return string(runes[:pos]), string(runes[pos:])
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func (m Model) renderCommandPalette(w int) string {
	lines := []string{
		sectionHeaderStyle().Render(" ◆ Command Palette"),
		"",
		keyStyle().Render(" Query: ") + valueStyle().Render(m.paletteQuery),
		"",
	}
	items := m.filteredCommandPaletteItems()
	if len(items) == 0 {
		lines = append(lines, mutedStyle().Render("  (no command matches)"))
	} else {
		max := len(items)
		if max > 8 {
			max = 8
		}
		for i := 0; i < max; i++ {
			prefix := "  "
			style := valueStyle()
			if i == m.paletteIdx {
				prefix = " " + accentStyle().Render(">")
				style = accentStyle()
			}
			for j, wl := range wrapText(items[i], w-6) {
				if j == 0 {
					lines = append(lines, prefix+" "+style.Render(wl))
				} else {
					lines = append(lines, "    "+style.Render(wl))
				}
			}
		}
	}
	lines = append(lines, "")
	lines = append(lines, mutedStyle().Render(" Enter: select   Tab: autofill   Esc: close   Ctrl+P: toggle"))
	return applyPanelScroll(lines, 0, w, 12)
}

func (m Model) renderHelpOverlay(w int) string {
	lines := []string{
		sectionHeaderStyle().Render(" ◆ Help"),
		"",
		valueStyle().Render(" 全局"),
		mutedStyle().Render("  ?: 显示/隐藏帮助"),
		mutedStyle().Render("  Tab: 切换焦点区"),
		mutedStyle().Render("  Esc: 退出输入焦点或关闭帮助"),
		mutedStyle().Render("  /help: 打开帮助"),
		mutedStyle().Render("  Ctrl+P 或 /palette: 打开命令面板"),
		"",
		valueStyle().Render(" 滚动"),
		mutedStyle().Render("  ↑/↓ 或 j/k: 行滚动"),
		mutedStyle().Render("  PgUp/PgDn: 页滚动"),
		mutedStyle().Render("  Ctrl+u / Ctrl+d: 半页滚动"),
		mutedStyle().Render("  鼠标滚轮: 当前区域滚动"),
		"",
		valueStyle().Render(" 执行"),
		mutedStyle().Render("  /goal <text>: 提交目标"),
		mutedStyle().Render("  /run accept | /run all: 执行建议"),
		mutedStyle().Render("  /mode manual|auto|cruise: 切模式"),
		mutedStyle().Render("  /creative: 手动触发创造流程"),
		mutedStyle().Render("  /test: LLM 连通性探测"),
		mutedStyle().Render("  /failures: 查看失败分类看板"),
		mutedStyle().Render("  /replay: 生成最近执行重放脚本"),
	}
	return applyPanelScroll(lines, 0, w, 14)
}
