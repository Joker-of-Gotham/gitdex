package tui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/Joker-of-Gotham/gitdex/internal/git"
	gitplatform "github.com/Joker-of-Gotham/gitdex/internal/platform"
	"github.com/mattn/go-runewidth"
)

const (
	resultKindGeneric       = "generic"
	resultKindStatus        = "status"
	resultKindLog           = "log"
	resultKindRemoteShow    = "remote_show"
	resultKindShow          = "show"
	resultKindDiff          = "diff"
	resultKindFileWrite     = "file_write"
	resultKindPlatformAdmin = "platform_admin"
)

type commandResultRenderer func(Model, commandTrace, int) []string

var commandResultRenderers = map[string]commandResultRenderer{
	resultKindStatus: func(m Model, trace commandTrace, width int) []string { return m.renderStatusResult(width) },
	resultKindLog:    func(_ Model, trace commandTrace, width int) []string { return renderLogResult(trace.Output, width) },
	resultKindRemoteShow: func(_ Model, trace commandTrace, width int) []string {
		return renderRemoteShowResult(trace.Output, width)
	},
	resultKindShow: func(_ Model, trace commandTrace, width int) []string { return renderShowResult(trace.Output, width) },
	resultKindDiff: func(_ Model, trace commandTrace, width int) []string {
		return renderDiffLikeResult(trace.Output, width)
	},
	resultKindFileWrite: func(_ Model, trace commandTrace, width int) []string { return renderFileWriteResult(trace, width) },
	resultKindPlatformAdmin: func(m Model, trace commandTrace, width int) []string {
		return renderPlatformAdminResult(m, trace, width)
	},
	resultKindGeneric: func(_ Model, trace commandTrace, width int) []string { return renderNotepadResult(trace.Output, width) },
}

func detectResultKind(command []string) string {
	if len(command) < 2 || !strings.EqualFold(strings.TrimSpace(command[0]), "git") {
		return resultKindGeneric
	}
	switch strings.ToLower(strings.TrimSpace(command[1])) {
	case "status":
		return resultKindStatus
	case "log":
		return resultKindLog
	case "diff":
		return resultKindDiff
	case "show":
		return resultKindShow
	case "remote":
		if len(command) > 2 && strings.EqualFold(strings.TrimSpace(command[2]), "show") {
			return resultKindRemoteShow
		}
	}
	return resultKindGeneric
}

func (m Model) renderLatestResultSpotlight(width int) string {
	if m.lastCommand.Title == "" {
		return ""
	}
	panelWidth := width
	if panelWidth < 30 {
		panelWidth = 30
	}
	borderStyle := panelStyleForStatus(m.lastCommand.Status).
		Padding(0, 1)
	innerWidth, _ := panelInnerSize(borderStyle, panelWidth, 1)

	lines := []string{
		keyStyle().Render(localizedLatestResultTitle()),
	}
	lines = append(lines, renderWrappedField(localizedText("Status: ", "状态: ", "Status: "), keyStyle(), localizedStatusText(m.lastCommand.Status), statusStyleForText(m.lastCommand.Status), innerWidth)...)
	lines = append(lines, renderWrappedField(localizedText("Target: ", "目标: ", "Target: "), keyStyle(), m.latestResultTarget(), commandStyle(), innerWidth)...)
	switch m.lastCommand.ResultKind {
	case resultKindFileWrite:
		lines = append(lines, renderWrappedField(localizedText("File: ", "文件: ", "File: "), keyStyle(), fmt.Sprintf("%s (%s)", valueOr(filepath.Base(m.lastCommand.FilePath), m.lastCommand.FilePath), m.lastCommand.FileOperation), valueStyle(), innerWidth)...)
	case resultKindPlatformAdmin:
		lines = append(lines, renderWrappedField(localizedText("Capability: ", "能力: ", "Capability: "), keyStyle(), humanCapabilityLabel(m.lastCommand.PlatformCapability), commandStyle(), innerWidth)...)
		lines = append(lines, renderWrappedField(localizedText("Coverage: ", "覆盖度: ", "Coverage: "), keyStyle(), m.capabilityCoverageLabel(m.lastCommand.PlatformCapability), infoStyle(), innerWidth)...)
		if strings.TrimSpace(m.lastCommand.PlatformAdapter) != "" {
			lines = append(lines, renderWrappedField(localizedText("Adapter: ", "适配器: ", "Adapter: "), keyStyle(), adapterDisplayLabel(gitplatform.AdapterKind(m.lastCommand.PlatformAdapter)), valueStyle(), innerWidth)...)
		}
		if strings.TrimSpace(m.lastCommand.PlatformRollback) != "" {
			lines = append(lines, renderWrappedField(localizedText("Rollback: ", "回滚: ", "Rollback: "), keyStyle(), m.lastCommand.PlatformRollback, infoStyle(), innerWidth)...)
		}
		lines = append(lines, renderWrappedField(localizedText("Flow: ", "流程: ", "Flow: "), keyStyle(), m.lastCommand.PlatformFlow, statusStyleForText(m.lastCommand.Status), innerWidth)...)
		if m.lastCommand.PlatformApproval {
			lines = append(lines, renderWrappedField(localizedText("Approval: ", "审批: ", "Approval: "), keyStyle(), localizedText("required", "需要", "required"), warnStyle(), innerWidth)...)
		}
		if strings.TrimSpace(m.lastCommand.PlatformResourceID) != "" {
			lines = append(lines, renderWrappedField(localizedText("Resource: ", "资源: ", "Resource: "), keyStyle(), m.lastCommand.PlatformResourceID, valueStyle(), innerWidth)...)
		}
		if strings.TrimSpace(m.lastCommand.PlatformLedgerID) != "" {
			lines = append(lines, renderWrappedField(localizedText("Ledger: ", "账本: ", "Ledger: "), keyStyle(), m.lastCommand.PlatformLedgerID, tsStyle(), innerWidth)...)
		}
		if strings.TrimSpace(m.lastCommand.Output) != "" {
			lines = append(lines, renderWrappedField(localizedText("Summary: ", "摘要: ", "Summary: "), keyStyle(), m.lastCommand.Output, infoStyle(), innerWidth)...)
		}
	case resultKindStatus:
		lines = append(lines, infoStyle().Render(localizedText("Working tree and staging table updated below in Result inspector.", "工作区和暂存区详情已在结果检查面板中更新。", "Working tree and staging table updated below in Result inspector.")))
	case resultKindLog:
		lines = append(lines, infoStyle().Render(localizedText("Commit list parsed into structured rows in Result inspector.", "提交列表已在结果检查面板中结构化展示。", "Commit list parsed into structured rows in Result inspector.")))
	default:
		if summary := firstNonEmptyLine(m.lastCommand.Output); summary != "" {
			lines = append(lines, renderWrappedField(localizedText("Summary: ", "摘要: ", "Summary: "), keyStyle(), summary, infoStyle(), innerWidth)...)
		}
	}
	return borderStyle.Width(panelWidth).Render(strings.Join(lines, "\n"))
}

func (m Model) latestResultTarget() string {
	if m.lastCommand.ResultKind == resultKindPlatformAdmin && strings.TrimSpace(m.lastCommand.PlatformCapability) != "" {
		parts := []string{humanCapabilityLabel(m.lastCommand.PlatformCapability)}
		if flow := strings.TrimSpace(m.lastCommand.PlatformFlow); flow != "" {
			parts = append(parts, flow)
		}
		if op := strings.TrimSpace(m.lastCommand.PlatformOperation); op != "" {
			parts = append(parts, op)
		}
		if resource := strings.TrimSpace(m.lastCommand.PlatformResourceID); resource != "" {
			parts = append(parts, resource)
		}
		return strings.Join(parts, " / ")
	}
	return m.lastCommand.Title
}

func panelStyleForStatus(status string) lipgloss.Style {
	switch {
	case strings.Contains(strings.ToLower(status), "success"), strings.Contains(strings.ToLower(status), "done"), strings.Contains(strings.ToLower(status), "viewed"):
		return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#57D48A"))
	case strings.Contains(strings.ToLower(status), "fail"), strings.Contains(strings.ToLower(status), "error"):
		return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#FF8C73"))
	default:
		return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#6FC3DF"))
	}
}

func firstNonEmptyLine(text string) string {
	for _, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return ""
}

func (m Model) renderStructuredCommandResult(width int) []string {
	if renderer, ok := commandResultRenderers[m.lastCommand.ResultKind]; ok {
		return renderer(m, m.lastCommand, width)
	}
	return commandResultRenderers[resultKindGeneric](m, m.lastCommand, width)
}

func (m Model) renderStatusResult(width int) []string {
	if m.gitState == nil {
		return renderNotepadResult(m.lastCommand.Output, width)
	}
	lines := []string{keyStyle().Render(localizedText("Repository status", "仓库状态", "Repository status"))}
	lines = append(lines, renderWrappedField(localizedText("Branch: ", "分支: ", "Branch: "), keyStyle(), m.gitState.LocalBranch.Name, commandStyle(), width)...)
	if m.gitState.LocalBranch.Upstream != "" {
		lines = append(lines, renderWrappedField(localizedText("Upstream: ", "上游: ", "Upstream: "), keyStyle(), m.gitState.LocalBranch.Upstream, infoStyle(), width)...)
	}
	lines = append(lines, "")
	lines = append(lines, keyStyle().Render(localizedText("Staging area", "暂存区", "Staging area")))
	if len(m.gitState.StagingArea) == 0 {
		lines = append(lines, mutedStyle().Render("  "+localizedText("clean", "干净", "clean")))
	} else {
		for _, file := range m.gitState.StagingArea {
			row := fmt.Sprintf("  %s  %s", string(file.StagingCode), file.Path)
			lines = append(lines, statusStyle(file.StagingCode).Render(row))
		}
	}
	lines = append(lines, "")
	lines = append(lines, keyStyle().Render(localizedText("Working tree", "工作区", "Working tree")))
	if len(m.gitState.WorkingTree) == 0 {
		lines = append(lines, mutedStyle().Render("  "+localizedText("clean", "干净", "clean")))
	} else {
		for _, file := range m.gitState.WorkingTree {
			row := fmt.Sprintf("  %s  %s", string(file.WorktreeCode), file.Path)
			lines = append(lines, statusStyle(file.WorktreeCode).Render(row))
		}
	}
	return lines
}

func renderLogResult(output string, width int) []string {
	lines := []string{keyStyle().Render(localizedText("Commit log", "提交记录", "Commit log"))}
	if strings.TrimSpace(output) == "" {
		return append(lines, mutedStyle().Render("(empty)"))
	}
	for _, raw := range strings.Split(strings.ReplaceAll(output, "\r\n", "\n"), "\n") {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		hash, rest, found := strings.Cut(raw, " ")
		if !found {
			lines = append(lines, valueStyle().Render(raw))
			continue
		}
		lines = append(lines, renderWrappedField(hash+" ", tsStyle(), strings.TrimSpace(rest), valueStyle(), width)...)
	}
	return lines
}

func renderNotepadResult(output string, width int) []string {
	lines := []string{keyStyle().Render(localizedText("Output", "输出", "Output"))}
	if strings.TrimSpace(output) == "" {
		return append(lines, mutedStyle().Render("(empty)"))
	}
	lines = append(lines, wrapPlainText(output, width)...)
	return lines
}

func renderShowResult(output string, width int) []string {
	lines := []string{keyStyle().Render(localizedText("Show output", "显示输出", "Show output"))}
	if strings.TrimSpace(output) == "" {
		return append(lines, mutedStyle().Render("(empty)"))
	}
	lines = append(lines, wrapPlainText(output, width)...)
	return lines
}

func renderRemoteShowResult(output string, width int) []string {
	lines := []string{keyStyle().Render(localizedText("Remote details", "远端详情", "Remote details"))}
	if strings.TrimSpace(output) == "" {
		return append(lines, mutedStyle().Render("(empty)"))
	}
	for _, raw := range strings.Split(strings.ReplaceAll(output, "\r\n", "\n"), "\n") {
		raw = strings.TrimRight(raw, " ")
		if strings.TrimSpace(raw) == "" {
			lines = append(lines, "")
			continue
		}
		if key, value, ok := strings.Cut(raw, ":"); ok && strings.TrimSpace(key) != "" && strings.TrimSpace(value) != "" {
			lines = append(lines, renderWrappedField(strings.TrimSpace(key)+": ", keyStyle(), strings.TrimSpace(value), valueStyle(), width)...)
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(raw), "*") || strings.HasPrefix(strings.TrimSpace(raw), "Local") || strings.HasPrefix(strings.TrimSpace(raw), "Remote") {
			lines = append(lines, infoStyle().Render(strings.TrimSpace(raw)))
			continue
		}
		lines = append(lines, wrapPlainText(raw, width)...)
	}
	return lines
}

func renderDiffLikeResult(output string, width int) []string {
	lines := []string{keyStyle().Render(localizedText("Diff / show output", "Diff / show 输出", "Diff / show output"))}
	if strings.TrimSpace(output) == "" {
		return append(lines, mutedStyle().Render("(empty)"))
	}
	for _, raw := range strings.Split(strings.ReplaceAll(output, "\r\n", "\n"), "\n") {
		switch {
		case strings.HasPrefix(raw, "+") && !strings.HasPrefix(raw, "+++"):
			lines = append(lines, successStyle().Render(raw))
		case strings.HasPrefix(raw, "-") && !strings.HasPrefix(raw, "---"):
			lines = append(lines, dangerStyle().Render(raw))
		case strings.HasPrefix(raw, "@@"):
			lines = append(lines, warnStyle().Render(raw))
		default:
			lines = append(lines, wrapPlainText(raw, width)...)
		}
	}
	return lines
}

func renderFileWriteResult(trace commandTrace, width int) []string {
	lines := []string{keyStyle().Render(localizedText("File mutation", "文件变更", "File mutation"))}
	lines = append(lines, renderWrappedField(localizedText("Path: ", "路径: ", "Path: "), keyStyle(), trace.FilePath, commandStyle(), width)...)
	lines = append(lines, renderWrappedField(localizedText("Operation: ", "操作: ", "Operation: "), keyStyle(), trace.FileOperation, statusStyleForText(trace.Status), width)...)
	lines = append(lines, "")

	if width >= 80 {
		lines = append(lines, renderSideBySideSnapshots("Before", trace.BeforeContent, "After", trace.AfterContent, width)...)
		return lines
	}

	lines = append(lines, keyStyle().Render("Before"))
	if strings.TrimSpace(trace.BeforeContent) == "" {
		lines = append(lines, mutedStyle().Render("(empty)"))
	} else {
		lines = append(lines, wrapPlainText(trace.BeforeContent, width)...)
	}
	lines = append(lines, "")
	lines = append(lines, keyStyle().Render("After"))
	if strings.TrimSpace(trace.AfterContent) == "" {
		lines = append(lines, mutedStyle().Render("(empty)"))
	} else {
		lines = append(lines, wrapPlainText(trace.AfterContent, width)...)
	}
	return lines
}

func renderPlatformAdminResult(m Model, trace commandTrace, width int) []string {
	lines := []string{keyStyle().Render(localizedText("Platform admin result", "平台操作结果", "Platform admin result"))}
	lines = append(lines, renderWrappedField(localizedText("Capability: ", "能力: ", "Capability: "), keyStyle(), trace.PlatformCapability, commandStyle(), width)...)
	platformID := m.detectedPlatform()
	lines = append(lines, renderWrappedField(localizedText("Coverage: ", "覆盖度: ", "Coverage: "), keyStyle(), valueOr(tracePlatformCoverage(platformID, trace.PlatformCapability), "full"), infoStyle(), width)...)
	if strings.TrimSpace(trace.PlatformAdapter) != "" {
		lines = append(lines, renderWrappedField(localizedText("Adapter: ", "适配器: ", "Adapter: "), keyStyle(), adapterDisplayLabel(gitplatform.AdapterKind(trace.PlatformAdapter)), valueStyle(), width)...)
	}
	if strings.TrimSpace(trace.PlatformRollback) != "" {
		lines = append(lines, renderWrappedField(localizedText("Rollback: ", "回滚: ", "Rollback: "), keyStyle(), trace.PlatformRollback, infoStyle(), width)...)
	}
	lines = append(lines, renderWrappedField(localizedText("Flow: ", "流程: ", "Flow: "), keyStyle(), trace.PlatformFlow, statusStyleForText(trace.Status), width)...)
	if boundary := strings.TrimSpace(firstNonEmpty(trace.PlatformBoundary, platformBoundaryReason(platformID, trace.PlatformCapability))); boundary != "" {
		lines = append(lines, renderWrappedField(localizedText("Boundary: ", "边界: ", "Boundary: "), keyStyle(), boundary, infoStyle(), width)...)
	}
	if strings.TrimSpace(trace.PlatformOperation) != "" {
		lines = append(lines, renderWrappedField(localizedText("Operation: ", "操作: ", "Operation: "), keyStyle(), trace.PlatformOperation, valueStyle(), width)...)
	}
	if strings.TrimSpace(trace.PlatformResourceID) != "" {
		lines = append(lines, renderWrappedField(localizedText("Resource: ", "资源: ", "Resource: "), keyStyle(), trace.PlatformResourceID, valueStyle(), width)...)
	}
	if trace.PlatformApproval {
		lines = append(lines, renderWrappedField(localizedText("Approval: ", "审批: ", "Approval: "), keyStyle(), localizedText("required", "需要", "required"), warnStyle(), width)...)
	}
	if strings.TrimSpace(trace.PlatformLedgerID) != "" {
		lines = append(lines, renderWrappedField(localizedText("Ledger: ", "账本: ", "Ledger: "), keyStyle(), trace.PlatformLedgerID, tsStyle(), width)...)
	}
	if strings.TrimSpace(trace.Output) != "" {
		lines = append(lines, renderWrappedField(localizedText("Summary: ", "摘要: ", "Summary: "), keyStyle(), trace.Output, infoStyle(), width)...)
	}
	if strings.TrimSpace(trace.PlatformCompensation) != "" {
		lines = append(lines, renderWrappedField(localizedText("Compensation: ", "补偿: ", "Compensation: "), keyStyle(), trace.PlatformCompensation, warnStyle(), width)...)
	}
	if entry := findLedgerEntry(m.mutationLedger, trace.PlatformLedgerID); entry != nil {
		if entry.RequestRevision > 0 {
			lines = append(lines, renderWrappedField(localizedText("Revision: ", "修订: ", "Revision: "), keyStyle(), fmt.Sprintf("%d", entry.RequestRevision), tsStyle(), width)...)
		}
		if grade := rollbackGrade(entry); grade != "" {
			lines = append(lines, renderWrappedField(localizedText("Rollback grade: ", "回滚等级: ", "Rollback grade: "), keyStyle(), grade, infoStyle(), width)...)
		}
		if len(entry.Metadata) > 0 {
			lines = append(lines, renderAdapterLedgerFields(entry.Metadata, width)...)
		}
		if len(entry.Metadata) > 0 && strings.EqualFold(trace.PlatformCapability, "release") {
			lines = append(lines, renderReleaseLedgerFields(entry.Metadata, width)...)
		}
	}
	if m.lastPlatform != nil &&
		strings.EqualFold(strings.TrimSpace(m.lastPlatform.CapabilityID), strings.TrimSpace(trace.PlatformCapability)) &&
		len(m.lastPlatform.LedgerChain) > 1 {
		lines = append(lines, renderWrappedField(localizedText("Ledger chain: ", "账本链: ", "Ledger chain: "), keyStyle(), strings.Join(m.lastPlatform.LedgerChain, " -> "), tsStyle(), width)...)
	}
	lines = append(lines, "")

	switch {
	case len(trace.PlatformBefore) > 0 || len(trace.PlatformAfter) > 0:
		before := prettyJSON(trace.PlatformBefore)
		after := prettyJSON(trace.PlatformAfter)
		if width >= 90 {
			lines = append(lines, renderSideBySideSnapshots("Before", before, "After", after, width)...)
			return lines
		}
		lines = append(lines, keyStyle().Render(localizedText("Before", "变更前", "Before")))
		lines = append(lines, wrapPlainText(before, width)...)
		lines = append(lines, "")
		lines = append(lines, keyStyle().Render(localizedText("After", "变更后", "After")))
		lines = append(lines, wrapPlainText(after, width)...)
	case len(trace.PlatformInspect) > 0:
		lines = append(lines, keyStyle().Render(localizedText("Snapshot", "快照", "Snapshot")))
		lines = append(lines, wrapPlainText(prettyJSON(trace.PlatformInspect), width)...)
	case len(trace.PlatformSnapshot) > 0:
		lines = append(lines, keyStyle().Render(localizedText("Snapshot", "快照", "Snapshot")))
		lines = append(lines, wrapPlainText(prettyJSON(trace.PlatformSnapshot), width)...)
	default:
		lines = append(lines, mutedStyle().Render("(no snapshot returned)"))
	}
	lines = append(lines, renderPlatformSpecializedSnapshot(trace, width)...)
	return lines
}

func findLedgerEntry(entries []gitplatform.MutationLedgerEntry, ledgerID string) *gitplatform.MutationLedgerEntry {
	ledgerID = strings.TrimSpace(ledgerID)
	if ledgerID == "" {
		return nil
	}
	for idx := range entries {
		if strings.EqualFold(strings.TrimSpace(entries[idx].ID), ledgerID) {
			return &entries[idx]
		}
	}
	return nil
}

func rollbackGrade(entry *gitplatform.MutationLedgerEntry) string {
	if entry == nil {
		return ""
	}
	if grade := strings.TrimSpace(entry.Metadata["rollback_grade"]); grade != "" {
		return grade
	}
	switch entry.ExecMeta.Rollback {
	case gitplatform.RollbackReversible:
		return "reversible"
	case gitplatform.RollbackCompensating:
		return "compensating"
	default:
		return ""
	}
}

func renderReleaseLedgerFields(metadata map[string]string, width int) []string {
	lines := []string{keyStyle().Render("Release asset")}
	if name := strings.TrimSpace(metadata["asset_name"]); name != "" {
		lines = append(lines, renderWrappedField("Asset: ", keyStyle(), name, commandStyle(), width)...)
	}
	if label := strings.TrimSpace(metadata["asset_label"]); label != "" {
		lines = append(lines, renderWrappedField("Label: ", keyStyle(), label, valueStyle(), width)...)
	}
	if contentType := strings.TrimSpace(metadata["content_type"]); contentType != "" {
		lines = append(lines, renderWrappedField("Content-Type: ", keyStyle(), contentType, infoStyle(), width)...)
	}
	if size := strings.TrimSpace(metadata["asset_size"]); size != "" {
		lines = append(lines, renderWrappedField("Size: ", keyStyle(), size+" bytes", valueStyle(), width)...)
	}
	if digest := strings.TrimSpace(metadata["asset_digest"]); digest != "" {
		lines = append(lines, renderWrappedField("Digest: ", keyStyle(), digest, infoStyle(), width)...)
	}
	if sourceKind := strings.TrimSpace(metadata["source_kind"]); sourceKind != "" {
		lines = append(lines, renderWrappedField("Source: ", keyStyle(), sourceKind, valueStyle(), width)...)
	}
	if stored := strings.TrimSpace(metadata["stored_bytes_ref"]); stored != "" {
		lines = append(lines, renderWrappedField("Stored bytes: ", keyStyle(), stored, tsStyle(), width)...)
	}
	if recoverable := strings.TrimSpace(metadata["recoverable"]); recoverable != "" {
		lines = append(lines, renderWrappedField("Recoverable: ", keyStyle(), recoverable, statusStyleForText(recoverable), width)...)
	}
	if grade := strings.TrimSpace(metadata["rollback_grade"]); grade != "" {
		lines = append(lines, renderWrappedField("Rollback grade: ", keyStyle(), grade, infoStyle(), width)...)
	}
	if partial := strings.TrimSpace(metadata["partial_restore_required"]); partial != "" {
		lines = append(lines, renderWrappedField("Partial restore: ", keyStyle(), partial, warnStyle(), width)...)
	}
	return lines
}

func renderAdapterLedgerFields(metadata map[string]string, width int) []string {
	lines := []string{}
	if backed := strings.TrimSpace(metadata["adapter_backed"]); backed != "" {
		lines = append(lines, renderWrappedField("Adapter route: ", keyStyle(), adapterDisplayLabel(gitplatform.AdapterKind(backed)), valueStyle(), width)...)
	}
	if transport := strings.TrimSpace(metadata["adapter_transport"]); transport != "" {
		lines = append(lines, renderWrappedField("Adapter transport: ", keyStyle(), transport, infoStyle(), width)...)
	}
	if binary := strings.TrimSpace(metadata["adapter_binary"]); binary != "" {
		lines = append(lines, renderWrappedField("Adapter binary: ", keyStyle(), binary, tsStyle(), width)...)
	}
	if driver := strings.TrimSpace(metadata["browser_driver"]); driver != "" {
		lines = append(lines, renderWrappedField("Browser driver: ", keyStyle(), driver, tsStyle(), width)...)
	}
	if manual := strings.TrimSpace(metadata["manual_completion_required"]); manual != "" {
		lines = append(lines, renderWrappedField("Manual completion: ", keyStyle(), manual, warnStyle(), width)...)
	}
	if validation := strings.TrimSpace(metadata["operator_validation_required"]); validation != "" {
		lines = append(lines, renderWrappedField("Operator validation: ", keyStyle(), validation, warnStyle(), width)...)
	}
	return lines
}

func renderPlatformSpecializedSnapshot(trace commandTrace, width int) []string {
	switch strings.ToLower(strings.TrimSpace(trace.PlatformCapability)) {
	case "release":
		return renderReleaseSnapshotSummary(trace, width)
	case "pages":
		return renderPagesSnapshotSummary(trace, width)
	default:
		return nil
	}
}

func renderReleaseSnapshotSummary(trace commandTrace, width int) []string {
	raw := firstNonEmptyRaw(trace.PlatformAfter, trace.PlatformSnapshot, trace.PlatformInspect)
	obj := decodeJSONObject(raw)
	if len(obj) == 0 {
		return nil
	}
	lines := []string{"", keyStyle().Render("Release summary")}
	if tag := strings.TrimSpace(stringFromAny(obj["tag_name"])); tag != "" {
		lines = append(lines, renderWrappedField("Tag: ", keyStyle(), tag, commandStyle(), width)...)
	}
	if name := strings.TrimSpace(stringFromAny(obj["name"])); name != "" {
		lines = append(lines, renderWrappedField("Name: ", keyStyle(), name, valueStyle(), width)...)
	}
	draft := fmt.Sprintf("%t", boolFromAnyValue(obj["draft"]))
	lines = append(lines, renderWrappedField("Draft: ", keyStyle(), draft, statusStyleForText(draft), width)...)
	if url := strings.TrimSpace(stringFromAny(obj["html_url"])); url != "" {
		lines = append(lines, renderWrappedField("URL: ", keyStyle(), url, infoStyle(), width)...)
	}
	return lines
}

func renderPagesSnapshotSummary(trace commandTrace, width int) []string {
	raw := firstNonEmptyRaw(trace.PlatformAfter, trace.PlatformSnapshot, trace.PlatformInspect)
	obj := decodeJSONObject(raw)
	if len(obj) == 0 {
		return nil
	}
	lines := []string{"", keyStyle().Render("Pages summary")}
	if cname := strings.TrimSpace(stringFromAny(obj["cname"])); cname != "" {
		lines = append(lines, renderWrappedField("Domain: ", keyStyle(), cname, commandStyle(), width)...)
	}
	if buildType := strings.TrimSpace(stringFromAny(obj["build_type"])); buildType != "" {
		lines = append(lines, renderWrappedField("Build type: ", keyStyle(), buildType, valueStyle(), width)...)
	}
	if state := strings.TrimSpace(stringFromAny(obj["protected_domain_state"])); state != "" {
		lines = append(lines, renderWrappedField("Domain state: ", keyStyle(), state, statusStyleForText(state), width)...)
	}
	if enforced := obj["https_enforced"]; enforced != nil {
		lines = append(lines, renderWrappedField("HTTPS enforced: ", keyStyle(), fmt.Sprintf("%t", boolFromAnyValue(enforced)), statusStyleForText(fmt.Sprintf("%t", boolFromAnyValue(enforced))), width)...)
	}
	if https := obj["https_certificate"]; https != nil {
		lines = append(lines, renderWrappedField("HTTPS: ", keyStyle(), "certificate metadata present", successStyle(), width)...)
	}
	if source, ok := obj["source"].(map[string]any); ok {
		sourceText := strings.TrimSpace(stringFromAny(source["branch"]))
		if path := strings.TrimSpace(stringFromAny(source["path"])); path != "" {
			sourceText = strings.TrimSpace(sourceText + ":" + path)
		}
		if sourceText != "" {
			lines = append(lines, renderWrappedField("Source: ", keyStyle(), sourceText, valueStyle(), width)...)
		}
	}
	if status := strings.TrimSpace(stringFromAny(obj["status"])); status != "" {
		lines = append(lines, renderWrappedField("Status: ", keyStyle(), status, statusStyleForText(status), width)...)
	}
	return lines
}

func firstNonEmptyRaw(values ...json.RawMessage) json.RawMessage {
	for _, value := range values {
		if len(value) > 0 {
			return value
		}
	}
	return nil
}

func decodeJSONObject(raw json.RawMessage) map[string]any {
	if len(raw) == 0 {
		return nil
	}
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil
	}
	return obj
}

func stringFromAny(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		return fmt.Sprintf("%v", value)
	}
}

func boolFromAnyValue(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		return strings.EqualFold(strings.TrimSpace(typed), "true")
	default:
		return false
	}
}

func renderPlatformRequestPreview(platformID gitplatform.Platform, op *git.PlatformExecInfo, width int) []string {
	lines := []string{keyStyle().Render(localizedText("Prepared platform request", "已准备的平台请求", "Prepared platform request"))}
	if op == nil {
		return append(lines, mutedStyle().Render("(empty)"))
	}
	lines = append(lines, renderWrappedField(localizedText("Capability: ", "能力: ", "Capability: "), keyStyle(), op.CapabilityID, commandStyle(), width)...)
	lines = append(lines, renderWrappedField(localizedText("Coverage: ", "覆盖度: ", "Coverage: "), keyStyle(), valueOr(tracePlatformCoverage(platformID, op.CapabilityID), "full"), infoStyle(), width)...)
	if meta := platformRequestMeta(platformID, op); meta.Adapter != "" {
		lines = append(lines, renderWrappedField(localizedText("Adapter: ", "适配器: ", "Adapter: "), keyStyle(), adapterDisplayLabel(meta.Adapter), valueStyle(), width)...)
		lines = append(lines, renderWrappedField(localizedText("Rollback: ", "回滚: ", "Rollback: "), keyStyle(), string(meta.Rollback), infoStyle(), width)...)
		if meta.ApprovalRequired {
			lines = append(lines, renderWrappedField(localizedText("Approval: ", "审批: ", "Approval: "), keyStyle(), localizedText("required", "需要", "required"), warnStyle(), width)...)
		}
	}
	lines = append(lines, renderWrappedField(localizedText("Flow: ", "流程: ", "Flow: "), keyStyle(), op.Flow, statusStyleForText(op.Flow), width)...)
	if boundary := strings.TrimSpace(platformBoundaryReason(platformID, op.CapabilityID)); boundary != "" {
		lines = append(lines, renderWrappedField(localizedText("Boundary: ", "边界: ", "Boundary: "), keyStyle(), boundary, infoStyle(), width)...)
	}
	if strings.TrimSpace(op.Operation) != "" {
		lines = append(lines, renderWrappedField(localizedText("Operation: ", "操作: ", "Operation: "), keyStyle(), op.Operation, valueStyle(), width)...)
	}
	if strings.TrimSpace(op.ResourceID) != "" {
		lines = append(lines, renderWrappedField(localizedText("Resource: ", "资源: ", "Resource: "), keyStyle(), op.ResourceID, valueStyle(), width)...)
	}
	lines = append(lines, "")
	lines = append(lines, wrapPlainText(prettyPlatformRequest(op), width)...)
	return lines
}

func tracePlatformCoverage(platformID gitplatform.Platform, capabilityID string) string {
	capabilityID = strings.TrimSpace(capabilityID)
	if capabilityID == "" {
		return ""
	}
	if boundary, ok := gitPlatformBoundary(platformID, capabilityID); ok {
		return boundary.Mode
	}
	return "full"
}

func platformBoundaryReason(platformID gitplatform.Platform, capabilityID string) string {
	if boundary, ok := gitPlatformBoundary(platformID, capabilityID); ok {
		return boundary.Reason
	}
	return ""
}

func platformRequestMeta(platformID gitplatform.Platform, op *git.PlatformExecInfo) gitplatform.ExecutionMeta {
	if op == nil {
		return gitplatform.ExecutionMeta{}
	}
	if platformID == gitplatform.PlatformUnknown {
		platformID = gitplatform.PlatformGitHub
	}
	return gitplatform.ExecutionMetaFor(platformID, op.CapabilityID, op.Flow, op.Operation)
}

func prettyJSON(raw json.RawMessage) string {
	if len(raw) == 0 {
		return "(empty)"
	}
	var formatted bytes.Buffer
	if err := json.Indent(&formatted, raw, "", "  "); err == nil {
		return formatted.String()
	}
	return strings.TrimSpace(string(raw))
}

func renderSideBySideSnapshots(leftTitle, left, rightTitle, right string, width int) []string {
	gap := 3
	colWidth := (width - gap) / 2
	if colWidth < 20 {
		colWidth = 20
	}
	leftLines := append([]string{leftTitle}, wrapPlainText(left, colWidth)...)
	rightLines := append([]string{rightTitle}, wrapPlainText(right, colWidth)...)
	total := len(leftLines)
	if len(rightLines) > total {
		total = len(rightLines)
	}
	out := make([]string, 0, total)
	for i := 0; i < total; i++ {
		l := ""
		r := ""
		if i < len(leftLines) {
			l = leftLines[i]
		}
		if i < len(rightLines) {
			r = rightLines[i]
		}
		leftText := runewidth.FillRight(l, colWidth)
		switch {
		case i == 0:
			out = append(out, keyStyle().Render(leftText)+"   "+keyStyle().Render(r))
		case strings.TrimSpace(l) == "" && strings.TrimSpace(r) == "":
			out = append(out, "")
		default:
			out = append(out, dangerStyle().Render(leftText)+"   "+successStyle().Render(r))
		}
	}
	return out
}
