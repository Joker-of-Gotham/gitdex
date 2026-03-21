package autonomy

import "strings"

type RiskLevel int

const (
	RiskLow      RiskLevel = 1
	RiskMedium   RiskLevel = 2
	RiskHigh     RiskLevel = 3
	RiskCritical RiskLevel = 4
)

func (r RiskLevel) String() string {
	switch r {
	case RiskLow:
		return "low"
	case RiskMedium:
		return "medium"
	case RiskHigh:
		return "high"
	case RiskCritical:
		return "critical"
	default:
		return "unknown"
	}
}

func ParseRiskLevel(s string) RiskLevel {
	switch strings.ToLower(s) {
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

var actionRiskMap = map[string]RiskLevel{
	"file.mkdir":                   RiskLow,
	"file.write":                   RiskHigh,
	"file.append":                  RiskMedium,
	"file.delete":                  RiskHigh,
	"file.move":                    RiskMedium,
	"file.copy":                    RiskLow,
	"git.status":                   RiskLow,
	"git.branch.delete":            RiskLow,
	"git.tag":                      RiskLow,
	"git.gc":                       RiskLow,
	"git.clean":                    RiskMedium,
	"git.fetch":                    RiskLow,
	"git.stash":                    RiskMedium,
	"git.add":                      RiskLow,
	"git.commit":                   RiskMedium,
	"git.push":                     RiskHigh,
	"git.push.force":               RiskCritical,
	"git.reset.hard":               RiskCritical,
	"git.branch.delete.force":      RiskCritical,
	"git.pull":                     RiskMedium,
	"git.merge":                    RiskMedium,
	"git.rebase":                   RiskHigh,
	"git.branch.create":            RiskHigh,
	"git.branch.rename":            RiskMedium,
	"git.checkout":                 RiskLow,
	"git.cherry-pick":              RiskHigh,
	"git.commit.amend":             RiskMedium,
	"git.log":                      RiskLow,
	"git.restore":                  RiskMedium,
	"git.reset":                    RiskMedium,
	"github.pr.create":             RiskMedium,
	"github.pr.merge":              RiskMedium,
	"github.pr.close":              RiskMedium,
	"github.pr.comment":            RiskLow,
	"github.pr.review":             RiskMedium,
	"github.issue.create":          RiskLow,
	"github.issue.close":           RiskLow,
	"github.issue.reopen":          RiskLow,
	"github.issue.comment":         RiskLow,
	"github.issue.label":           RiskLow,
	"github.issue.assign":          RiskLow,
	"github.release.create":        RiskMedium,
	"github.workflow.trigger":      RiskMedium,
	"github.repo.delete":           RiskCritical,
	"github.branch.protection.set": RiskCritical,
}

var blockedActions = map[string]string{
	"git.push.force":               "Force push 被安全护栏拦截",
	"git.reset.hard":               "Hard reset 被安全护栏拦截 — 可能导致未提交更改永久丢失",
	"git.branch.delete.force":      "强制删除分支被安全护栏拦截",
	"github.repo.delete":           "仓库删除被安全护栏拦截",
	"github.branch.protection.set": "分支保护修改被安全护栏拦截",
}

type Guardrails struct {
	customRisks   map[string]RiskLevel
	customBlocked map[string]string
}

func NewGuardrails() *Guardrails {
	return &Guardrails{
		customRisks:   make(map[string]RiskLevel),
		customBlocked: make(map[string]string),
	}
}

func (g *Guardrails) SetActionRisk(action string, level RiskLevel) {
	g.customRisks[action] = level
}

func (g *Guardrails) BlockAction(action, reason string) {
	g.customBlocked[action] = reason
}

func (g *Guardrails) EvaluateRisk(plan ActionPlan) RiskLevel {
	if plan.RiskLevel > 0 {
		return plan.RiskLevel
	}

	worst := RiskLow
	for _, step := range plan.Steps {
		risk := g.actionRisk(step.Action)
		if risk > worst {
			worst = risk
		}
	}
	return worst
}

func (g *Guardrails) CheckPolicy(plan ActionPlan) (allowed bool, reason string) {
	for _, step := range plan.Steps {
		if r, ok := g.customBlocked[step.Action]; ok {
			return false, r
		}
		if r, ok := blockedActions[step.Action]; ok {
			return false, r
		}
	}
	return true, ""
}

func (g *Guardrails) actionRisk(action string) RiskLevel {
	if r, ok := g.customRisks[action]; ok {
		return r
	}
	if r, ok := actionRiskMap[action]; ok {
		return r
	}
	return RiskHigh
}
