package autonomy

import (
	"fmt"
	"strings"
	"time"
)

type CruiseReport struct {
	CycleID   string           `json:"cycle_id"`
	StartTime time.Time        `json:"start_time"`
	EndTime   time.Time        `json:"end_time"`
	Executed  []ExecutedAction `json:"executed,omitempty"`
	Pending   []ActionPlan     `json:"pending,omitempty"`
	Blocked   []BlockedAction  `json:"blocked,omitempty"`
	Errors    []string         `json:"errors,omitempty"`
}

type ExecutedAction struct {
	Plan   ActionPlan      `json:"plan"`
	Result ExecutionResult `json:"result"`
}

type BlockedAction struct {
	Plan   ActionPlan `json:"plan"`
	Reason string     `json:"reason"`
}

type Reporter struct {
	reports []CruiseReport
	maxKeep int
}

func NewReporter(maxKeep int) *Reporter {
	if maxKeep <= 0 {
		maxKeep = 50
	}
	return &Reporter{maxKeep: maxKeep}
}

func (r *Reporter) Add(report CruiseReport) {
	r.reports = append(r.reports, report)
	if len(r.reports) > r.maxKeep {
		r.reports = r.reports[len(r.reports)-r.maxKeep:]
	}
}

func (r *Reporter) Latest() *CruiseReport {
	if len(r.reports) == 0 {
		return nil
	}
	rpt := r.reports[len(r.reports)-1]
	return &rpt
}

func (r *Reporter) List() []CruiseReport {
	out := make([]CruiseReport, len(r.reports))
	copy(out, r.reports)
	return out
}

func (r *Reporter) Count() int {
	return len(r.reports)
}

func FormatReport(rpt CruiseReport) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("# 巡航报告 %s\n", rpt.CycleID))
	b.WriteString(fmt.Sprintf("时间: %s → %s (耗时 %s)\n\n",
		rpt.StartTime.Format("15:04:05"),
		rpt.EndTime.Format("15:04:05"),
		rpt.EndTime.Sub(rpt.StartTime).Round(time.Second)))

	if len(rpt.Executed) > 0 {
		b.WriteString(fmt.Sprintf("## 已执行 (%d)\n", len(rpt.Executed)))
		for _, e := range rpt.Executed {
			status := "✅"
			if !e.Result.Success {
				status = "❌"
			}
			b.WriteString(fmt.Sprintf("  %s %s — %s\n", status, e.Plan.Description, e.Plan.Rationale))
		}
		b.WriteString("\n")
	}

	if len(rpt.Pending) > 0 {
		b.WriteString(fmt.Sprintf("## 待审批 (%d)\n", len(rpt.Pending)))
		for _, p := range rpt.Pending {
			b.WriteString(fmt.Sprintf("  ⏳ [%s] %s — %s\n", p.RiskLevel.String(), p.Description, p.Rationale))
		}
		b.WriteString("\n")
	}

	if len(rpt.Blocked) > 0 {
		b.WriteString(fmt.Sprintf("## 已拦截 (%d)\n", len(rpt.Blocked)))
		for _, bl := range rpt.Blocked {
			b.WriteString(fmt.Sprintf("  🛑 %s — %s\n", bl.Plan.Description, bl.Reason))
		}
		b.WriteString("\n")
	}

	if len(rpt.Errors) > 0 {
		b.WriteString("## 错误\n")
		for _, e := range rpt.Errors {
			b.WriteString(fmt.Sprintf("  ⚠️  %s\n", e))
		}
	}

	return b.String()
}
