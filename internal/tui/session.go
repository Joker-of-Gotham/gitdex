package tui

import (
	"strings"
	"time"

	"github.com/Joker-of-Gotham/gitdex/internal/llm/prompt"
)

type GoalRecord struct {
	Goal      string    `json:"goal"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

type SessionContext struct {
	ActiveGoal     string            `json:"active_goal,omitempty"`
	GoalHistory    []GoalRecord      `json:"goal_history,omitempty"`
	SkippedActions []string          `json:"skipped_actions,omitempty"`
	Preferences    map[string]string `json:"preferences,omitempty"`
}

const maxGoalHistory = 20

func (s SessionContext) ToPromptContext() prompt.SessionContext {
	out := prompt.SessionContext{
		ActiveGoal:       strings.TrimSpace(s.ActiveGoal),
		ActiveGoalStatus: "",
		SkippedActions:   append([]string(nil), s.SkippedActions...),
		Preferences:      map[string]string{},
	}
	out.GoalHistory = make([]prompt.GoalRecord, 0, len(s.GoalHistory))
	for _, h := range s.GoalHistory {
		out.GoalHistory = append(out.GoalHistory, prompt.GoalRecord{
			Goal:      h.Goal,
			Status:    h.Status,
			Timestamp: h.Timestamp.Format(time.RFC3339),
		})
	}
	for k, v := range s.Preferences {
		out.Preferences[k] = v
	}
	return out
}

func (s *SessionContext) markGoalStatus(status string) {
	if s == nil {
		return
	}
	goal := strings.TrimSpace(s.ActiveGoal)
	if goal == "" {
		return
	}
	s.GoalHistory = append(s.GoalHistory, GoalRecord{
		Goal:      goal,
		Status:    strings.TrimSpace(status),
		Timestamp: time.Now(),
	})
	if len(s.GoalHistory) > maxGoalHistory {
		s.GoalHistory = s.GoalHistory[len(s.GoalHistory)-maxGoalHistory:]
	}
	if status == "completed" {
		s.ActiveGoal = ""
	}
}
