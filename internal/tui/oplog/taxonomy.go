package oplog

import (
	"fmt"
	"sort"
	"strings"
)

type FailureDashboard struct {
	Total   int
	Buckets map[string]int
}

func BuildFailureDashboard(entries []Entry) FailureDashboard {
	buckets := map[string]int{}
	total := 0
	for _, e := range entries {
		if e.Type != EntryCmdFail && e.Type != EntryLLMError {
			continue
		}
		total++
		b := classifyFailure(e.Summary + "\n" + e.Detail)
		buckets[b]++
	}
	return FailureDashboard{Total: total, Buckets: buckets}
}

func (d FailureDashboard) Render() string {
	if d.Total == 0 {
		return "no failures recorded"
	}
	type row struct {
		name  string
		count int
	}
	rows := make([]row, 0, len(d.Buckets))
	for k, v := range d.Buckets {
		rows = append(rows, row{name: k, count: v})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].count == rows[j].count {
			return rows[i].name < rows[j].name
		}
		return rows[i].count > rows[j].count
	})
	var lines []string
	lines = append(lines, fmt.Sprintf("total_failures=%d", d.Total))
	for _, r := range rows {
		lines = append(lines, fmt.Sprintf("  - %s: %d", r.name, r.count))
	}
	return strings.Join(lines, "\n")
}

func classifyFailure(s string) string {
	low := strings.ToLower(s)
	switch {
	case strings.Contains(low, "http 401"), strings.Contains(low, "http 403"),
		strings.Contains(low, "unauthorized"), strings.Contains(low, "forbidden"),
		strings.Contains(low, "scope"):
		return "auth_permission"
	case strings.Contains(low, "http 404"), strings.Contains(low, "not found"):
		return "not_found"
	case strings.Contains(low, "http 409"), strings.Contains(low, "conflict"),
		strings.Contains(low, "already exists"), strings.Contains(low, "duplicate"):
		return "conflict_duplicate"
	case strings.Contains(low, "timeout"), strings.Contains(low, "connection"),
		strings.Contains(low, "temporarily"), strings.Contains(low, "http 500"),
		strings.Contains(low, "http 502"), strings.Contains(low, "http 503"):
		return "network_transient"
	case strings.Contains(low, "validation"), strings.Contains(low, "invalid"),
		strings.Contains(low, "preflight"):
		return "validation"
	default:
		return "unknown"
	}
}
