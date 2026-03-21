package autonomy

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type GateAction string

const (
	GateAuto    GateAction = "auto"
	GateManual  GateAction = "manual"
	GateBlocked GateAction = "blocked"
)

type MissionWindow struct {
	DaysOfWeek []time.Weekday
	StartHour  int
	EndHour    int
	Timezone   string
}

type PolicyBundle struct {
	Name     string
	Rules    []PolicyRule
	Priority int
}

type PolicyRule struct {
	Condition string
	Action    GateAction
	Reason    string
}

type PolicyGate struct {
	mu                 sync.RWMutex
	approvalThresholds map[string]int
	riskGates          map[RiskLevel]GateAction
	missionWindows     []MissionWindow
	policies           []PolicyBundle
}

type PolicyGateOption func(*PolicyGate)

func WithApprovalThresholds(m map[string]int) PolicyGateOption {
	return func(g *PolicyGate) {
		if m == nil {
			return
		}
		for k, v := range m {
			g.approvalThresholds[k] = v
		}
	}
}

func WithRiskGates(m map[RiskLevel]GateAction) PolicyGateOption {
	return func(g *PolicyGate) {
		if m == nil {
			return
		}
		for k, v := range m {
			g.riskGates[k] = v
		}
	}
}

func WithMissionWindows(w []MissionWindow) PolicyGateOption {
	return func(g *PolicyGate) {
		g.missionWindows = append([]MissionWindow(nil), w...)
	}
}

func WithPolicies(p []PolicyBundle) PolicyGateOption {
	return func(g *PolicyGate) {
		g.policies = append([]PolicyBundle(nil), p...)
	}
}

func NewPolicyGate(opts ...PolicyGateOption) *PolicyGate {
	g := &PolicyGate{
		approvalThresholds: make(map[string]int),
		riskGates: map[RiskLevel]GateAction{
			RiskLow:      GateAuto,
			RiskMedium:   GateAuto,
			RiskHigh:     GateManual,
			RiskCritical: GateBlocked,
		},
	}
	for _, o := range opts {
		o(g)
	}
	return g
}

func (g *PolicyGate) Evaluate(action string, risk RiskLevel) (GateAction, string) {
	if g == nil {
		return GateAuto, ""
	}

	g.mu.RLock()
	bundles := append([]PolicyBundle(nil), g.policies...)
	g.mu.RUnlock()

	sort.SliceStable(bundles, func(i, j int) bool {
		return bundles[i].Priority > bundles[j].Priority
	})

	for _, bundle := range bundles {
		for _, rule := range bundle.Rules {
			if g.ruleMatches(rule, action, risk) {
				if strings.TrimSpace(rule.Reason) != "" {
					return rule.Action, rule.Reason
				}
				return rule.Action, fmt.Sprintf("policy %q matched", bundle.Name)
			}
		}
	}

	g.mu.RLock()
	defer g.mu.RUnlock()
	if ga, ok := g.riskGates[risk]; ok {
		return ga, ""
	}
	return GateManual, "no risk gate configured for level"
}

func (g *PolicyGate) ruleMatches(rule PolicyRule, action string, risk RiskLevel) bool {
	c := strings.TrimSpace(strings.ToLower(rule.Condition))
	if c == "" {
		return false
	}

	if strings.HasPrefix(c, "risk >=") {
		want := parseRiskToken(strings.TrimSpace(strings.TrimPrefix(c, "risk >=")))
		return risk >= want
	}
	if strings.HasPrefix(c, "risk <=") {
		want := parseRiskToken(strings.TrimSpace(strings.TrimPrefix(c, "risk <=")))
		return risk <= want
	}
	if strings.HasPrefix(c, "risk ==") || strings.HasPrefix(c, "risk =") {
		rest := c
		if strings.HasPrefix(c, "risk ==") {
			rest = strings.TrimSpace(strings.TrimPrefix(c, "risk =="))
		} else {
			rest = strings.TrimSpace(strings.TrimPrefix(c, "risk ="))
		}
		want := parseRiskToken(rest)
		return risk == want
	}

	if strings.HasPrefix(c, "action ==") {
		want := strings.Trim(strings.TrimSpace(strings.TrimPrefix(c, "action ==")), `"'`)
		return strings.TrimSpace(action) == want
	}

	return false
}

func parseRiskToken(s string) RiskLevel {
	s = strings.Trim(strings.ToLower(strings.TrimSpace(s)), `"'`)
	switch s {
	case "low":
		return RiskLow
	case "medium":
		return RiskMedium
	case "high":
		return RiskHigh
	case "critical":
		return RiskCritical
	default:
		return RiskHigh
	}
}

func (g *PolicyGate) HasMissionWindows() bool {
	if g == nil {
		return false
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.missionWindows) > 0
}

func (g *PolicyGate) IsInMissionWindow() bool {
	if g == nil {
		return true
	}
	g.mu.RLock()
	windows := append([]MissionWindow(nil), g.missionWindows...)
	g.mu.RUnlock()

	if len(windows) == 0 {
		return true
	}

	for _, w := range windows {
		if g.matchesWindow(w) {
			return true
		}
	}
	return false
}

func (g *PolicyGate) matchesWindow(w MissionWindow) bool {
	loc := time.Local
	if tz := strings.TrimSpace(w.Timezone); tz != "" {
		if l, err := time.LoadLocation(tz); err == nil {
			loc = l
		}
	}
	now := time.Now().In(loc)
	wd := now.Weekday()
	if len(w.DaysOfWeek) > 0 {
		ok := false
		for _, d := range w.DaysOfWeek {
			if d == wd {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	h := now.Hour()
	start, end := w.StartHour, w.EndHour
	if start <= end {
		return h >= start && h < end
	}
	return h >= start || h < end
}

func (g *PolicyGate) RequiresApproval(action string) (bool, int) {
	if g == nil {
		return false, 0
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	n, ok := g.approvalThresholds[action]
	if !ok {
		return false, 0
	}
	return n > 0, n
}

func (g *PolicyGate) SetApprovalThreshold(action string, count int) {
	if g == nil {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.approvalThresholds == nil {
		g.approvalThresholds = make(map[string]int)
	}
	g.approvalThresholds[action] = count
}

func (g *PolicyGate) AddMissionWindow(w MissionWindow) {
	if g == nil {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.missionWindows = append(g.missionWindows, w)
}

func (g *PolicyGate) AddPolicy(p PolicyBundle) {
	if g == nil {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.policies = append(g.policies, p)
}
