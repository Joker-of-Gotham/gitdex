package tui

import "unicode/utf8"

func runeLen(text string) int {
	return utf8.RuneCountInString(text)
}

func clampRuneIndex(text string, pos int) int {
	if pos < 0 {
		return 0
	}
	if max := runeLen(text); pos > max {
		return max
	}
	return pos
}

func splitAtRune(text string, pos int) (string, string) {
	pos = clampRuneIndex(text, pos)
	runes := []rune(text)
	return string(runes[:pos]), string(runes[pos:])
}

func insertAtRune(text string, pos int, insert string) (string, int) {
	before, after := splitAtRune(text, pos)
	next := before + insert + after
	return next, pos + runeLen(insert)
}

func deleteRuneBefore(text string, pos int) (string, int) {
	pos = clampRuneIndex(text, pos)
	if pos == 0 {
		return text, 0
	}
	runes := []rune(text)
	return string(append(runes[:pos-1], runes[pos:]...)), pos - 1
}

func deleteRuneAt(text string, pos int) (string, int) {
	pos = clampRuneIndex(text, pos)
	runes := []rune(text)
	if pos >= len(runes) {
		return text, pos
	}
	return string(append(runes[:pos], runes[pos+1:]...)), pos
}

func cursorLine(text string, pos int) int {
	pos = clampRuneIndex(text, pos)
	line := 0
	for idx, r := range []rune(text) {
		if idx >= pos {
			break
		}
		if r == '\n' {
			line++
		}
	}
	return line
}

func cursorLineCol(text string, pos int) (int, int) {
	pos = clampRuneIndex(text, pos)
	line := 0
	col := 0
	for idx, r := range []rune(text) {
		if idx >= pos {
			break
		}
		if r == '\n' {
			line++
			col = 0
			continue
		}
		col++
	}
	return line, col
}

func runeIndexForLineCol(text string, targetLine, targetCol int) int {
	if targetLine < 0 {
		targetLine = 0
	}
	if targetCol < 0 {
		targetCol = 0
	}
	line := 0
	col := 0
	runes := []rune(text)
	for idx, r := range runes {
		if line == targetLine && col >= targetCol {
			return idx
		}
		if r == '\n' {
			if line == targetLine {
				return idx
			}
			line++
			col = 0
			continue
		}
		col++
	}
	return len(runes)
}

func moveCursorVertical(text string, pos, delta int) int {
	line, col := cursorLineCol(text, pos)
	return runeIndexForLineCol(text, line+delta, col)
}

func lineStart(text string, pos int) int {
	line, _ := cursorLineCol(text, pos)
	return runeIndexForLineCol(text, line, 0)
}

func lineEnd(text string, pos int) int {
	line, _ := cursorLineCol(text, pos)
	return runeIndexForLineCol(text, line+1, 0) - 1
}
