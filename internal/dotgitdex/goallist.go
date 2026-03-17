package dotgitdex

import (
	"fmt"
	"os"
	"strings"
)

// Goal represents a top-level goal with sub-tasks.
type Goal struct {
	Title     string
	Completed bool
	Todos     []Todo
}

// Todo is a sub-task under a goal.
type Todo struct {
	Title     string
	Completed bool
}

// ParseGoalList parses a markdown goal-list.md into structured goals.
// Expected format:
//
//	# Goal-List
//	- [ ] Goal A
//	  - [ ] Todo A-1
//	  - [x] Todo A-2
//	- [x] Goal B
//	  - [x] Todo B-1
func ParseGoalList(content string) ([]Goal, error) {
	var goals []Goal
	var current *Goal
	for _, rawLine := range strings.Split(content, "\n") {
		line := strings.TrimRight(rawLine, "\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		completed, title := parseCheckbox(trimmed)
		if title == "" {
			continue
		}
		if indent < 2 {
			goals = append(goals, Goal{Title: title, Completed: completed})
			current = &goals[len(goals)-1]
		} else if current != nil {
			current.Todos = append(current.Todos, Todo{Title: title, Completed: completed})
		}
	}
	return goals, nil
}

func parseCheckbox(s string) (completed bool, title string) {
	s = strings.TrimPrefix(s, "- ")
	s = strings.TrimPrefix(s, "* ")
	if strings.HasPrefix(s, "[x] ") || strings.HasPrefix(s, "[X] ") {
		return true, strings.TrimSpace(s[4:])
	}
	if strings.HasPrefix(s, "[ ] ") {
		return false, strings.TrimSpace(s[4:])
	}
	return false, strings.TrimSpace(s)
}

// WriteGoalList writes goals to goal-list/goal-list.md.
func (m *Manager) WriteGoalList(goals []Goal) error {
	var b strings.Builder
	b.WriteString("# Goal-List\n\n")
	for _, g := range goals {
		b.WriteString(fmt.Sprintf("- [%s] %s\n", checkMark(g.Completed), g.Title))
		for _, t := range g.Todos {
			b.WriteString(fmt.Sprintf("  - [%s] %s\n", checkMark(t.Completed), t.Title))
		}
	}
	return os.WriteFile(m.GoalListPath(), []byte(b.String()), 0o644)
}

// ReadGoalList reads and parses goal-list.md.
func (m *Manager) ReadGoalList() ([]Goal, error) {
	data, err := os.ReadFile(m.GoalListPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return ParseGoalList(string(data))
}

// PendingGoals returns only incomplete goals with their incomplete todos.
func PendingGoals(goals []Goal) []Goal {
	var out []Goal
	for _, g := range goals {
		if g.Completed {
			continue
		}
		pending := Goal{Title: g.Title}
		for _, t := range g.Todos {
			if !t.Completed {
				pending.Todos = append(pending.Todos, t)
			}
		}
		out = append(out, pending)
	}
	return out
}

// FormatPendingGoals renders pending goals as markdown text for LLM consumption.
func FormatPendingGoals(goals []Goal) string {
	pending := PendingGoals(goals)
	if len(pending) == 0 {
		return ""
	}
	var b strings.Builder
	for _, g := range pending {
		b.WriteString(fmt.Sprintf("- [ ] %s\n", g.Title))
		for _, t := range g.Todos {
			b.WriteString(fmt.Sprintf("  - [ ] %s\n", t.Title))
		}
	}
	return b.String()
}

// FormatSingleGoalTodos renders one goal's sub-tasks with completion status.
// Unlike FormatPendingGoals, this shows BOTH completed and pending todos
// so the LLM can see what's already done.
func FormatSingleGoalTodos(g Goal) string {
	if len(g.Todos) == 0 {
		return fmt.Sprintf("- [ ] %s\n  (no sub-tasks defined)\n", g.Title)
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Goal: %s\n", g.Title))
	for _, t := range g.Todos {
		mark := " "
		if t.Completed {
			mark = "x"
		}
		b.WriteString(fmt.Sprintf("  - [%s] %s\n", mark, t.Title))
	}
	return b.String()
}

// GoalProgress returns (completed, total) counts for a goal's sub-tasks.
func GoalProgress(g Goal) (int, int) {
	done := 0
	for _, t := range g.Todos {
		if t.Completed {
			done++
		}
	}
	return done, len(g.Todos)
}

func checkMark(done bool) string {
	if done {
		return "x"
	}
	return " "
}
