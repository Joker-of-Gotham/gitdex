package tui

import "strings"

func sliceVisibleLines(content string, height, offset int) string {
	if height <= 0 {
		return ""
	}
	lines := strings.Split(content, "\n")
	offset = clampOffset(len(lines), height, offset)
	end := offset + height
	if end > len(lines) {
		end = len(lines)
	}
	visible := append([]string(nil), lines[offset:end]...)
	for len(visible) < height {
		visible = append(visible, "")
	}
	return strings.Join(visible, "\n")
}

func clampOffset(totalLines, height, offset int) int {
	if height <= 0 || totalLines <= height {
		return 0
	}
	if offset < 0 {
		return 0
	}
	max := totalLines - height
	if offset > max {
		return max
	}
	return offset
}

func sliceVisibleLinesFromEnd(content string, height, offset int) string {
	if height <= 0 {
		return ""
	}
	lines := strings.Split(content, "\n")
	if len(lines) == 0 {
		lines = []string{""}
	}
	if offset < 0 {
		offset = 0
	}
	maxOffset := 0
	if len(lines) > height {
		maxOffset = len(lines) - height
	}
	if offset > maxOffset {
		offset = maxOffset
	}
	end := len(lines) - offset
	if end < 0 {
		end = 0
	}
	start := end - height
	if start < 0 {
		start = 0
	}
	visible := append([]string(nil), lines[start:end]...)
	for len(visible) < height {
		visible = append(visible, "")
	}
	return strings.Join(visible, "\n")
}

func lineCount(content string) int {
	if content == "" {
		return 0
	}
	return len(strings.Split(content, "\n"))
}
