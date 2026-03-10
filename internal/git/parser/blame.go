package parser

import (
	"regexp"
	"strconv"
	"strings"
)

// BlameLine represents a line from git blame --porcelain.
type BlameLine struct {
	Hash    string
	Author  string
	Date    string
	LineNo  int
	Content string
}

// ParseBlame parses `git blame --porcelain` output into BlameLine slice.
func ParseBlame(output string) []BlameLine {
	if strings.TrimSpace(output) == "" {
		return nil
	}
	lines := strings.Split(output, "\n")
	var result []BlameLine
	var current *BlameLine
	hashRe := regexp.MustCompile(`^([a-f0-9]{40})\s+(\d+)\s+(\d+)\s+(\d+)`)
	authorRe := regexp.MustCompile(`^author\s+(.+)$`)
	dateRe := regexp.MustCompile(`^author-time\s+(\d+)$`)

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if m := hashRe.FindStringSubmatch(line); len(m) >= 5 {
			if current != nil && current.Content != "" {
				result = append(result, *current)
			}
			lineNo, _ := strconv.Atoi(m[4])
			current = &BlameLine{Hash: m[1], LineNo: lineNo}
			continue
		}
		if current == nil {
			continue
		}
		if m := authorRe.FindStringSubmatch(line); len(m) >= 2 {
			current.Author = m[1]
			continue
		}
		if m := dateRe.FindStringSubmatch(line); len(m) >= 2 {
			current.Date = m[1]
			continue
		}
		if strings.HasPrefix(line, "\t") {
			current.Content = strings.TrimPrefix(line, "\t")
			result = append(result, *current)
			current = &BlameLine{Hash: current.Hash, Author: current.Author, Date: current.Date}
		}
	}
	if current != nil && current.Content != "" {
		result = append(result, *current)
	}
	return result
}
