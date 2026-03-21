package command

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/your-org/gitdex/internal/app/bootstrap"
	clioutput "github.com/your-org/gitdex/internal/cli/output"
	"github.com/your-org/gitdex/internal/planning"
	"github.com/your-org/gitdex/internal/planning/compiler"
	"github.com/your-org/gitdex/internal/planning/intent"
	"github.com/your-org/gitdex/internal/planning/reviewer"
	"github.com/your-org/gitdex/internal/policy"
)

func newPlanGroupCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plan",
		Short: "Compile and manage structured execution plans",
	}
	cmd.AddCommand(newPlanCompileCommand(flags, appFn))
	cmd.AddCommand(newPlanShowCommand(flags, appFn))
	cmd.AddCommand(newPlanListCommand(flags, appFn))
	cmd.AddCommand(newPlanReviewCommand(flags, appFn))
	cmd.AddCommand(newPlanApproveCommand(flags, appFn))
	cmd.AddCommand(newPlanRejectCommand(flags, appFn))
	cmd.AddCommand(newPlanDeferCommand(flags, appFn))
	cmd.AddCommand(newPlanEditCommand(flags, appFn))
	return cmd
}

func newPlanCompileCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	return &cobra.Command{
		Use:   "compile [goal]",
		Short: "Compile a goal into a structured execution plan",
		Long:  "Turn a command or natural-language goal into a reviewed, risk-assessed structured execution plan.",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			planStore := app.StorageProvider.PlanStore()
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)
			goal := strings.Join(args, " ")

			repoRoot := app.RepoRoot
			if repoRoot == "" {
				repoRoot = app.Config.Paths.RepositoryRoot
			}
			owner, repoName := resolveOwnerRepo("", "", repoRoot)
			if owner == "" || repoName == "" {
				return fmt.Errorf("plan compile requires a git repository; run from a repo or configure repository_root in gitdex config")
			}

			i := intent.NewCommandIntent(goal, "plan", nil)
			comp := compiler.New(owner, repoName)
			plan, err := comp.Compile(context.Background(), i)
			if err != nil {
				return fmt.Errorf("plan compilation failed: %w", err)
			}

			eng := policy.NewDefaultEngine()
			result, err := eng.Evaluate(context.Background(), plan)
			if err != nil {
				return fmt.Errorf("policy evaluation failed: %w", err)
			}
			plan.PolicyResult = result

			switch result.Verdict {
			case planning.VerdictAllowed:
				plan.Status = planning.PlanReviewRequired
			case planning.VerdictEscalated:
				plan.Status = planning.PlanReviewRequired
			case planning.VerdictBlocked:
				plan.Status = planning.PlanBlocked
			case planning.VerdictDegraded:
				plan.Status = planning.PlanReviewRequired
			}

			if err := planStore.Save(plan); err != nil {
				return fmt.Errorf("failed to store plan: %w", err)
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, plan)
			}
			return renderPlanText(cmd.OutOrStdout(), plan)
		},
	}
}

func newPlanShowCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	return &cobra.Command{
		Use:   "show [plan_id]",
		Short: "Show details of a compiled plan",
		Long:  "Show details of a compiled plan. Note: plans are stored in memory and do not persist across CLI invocations.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			planStore := app.StorageProvider.PlanStore()
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			plan, err := planStore.Get(args[0])
			if err != nil {
				return fmt.Errorf("plan not found (plans are stored in memory for this session only): %w", err)
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, plan)
			}
			return renderPlanText(cmd.OutOrStdout(), plan)
		},
	}
}

func newPlanListCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List recently compiled plans",
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			planStore := app.StorageProvider.PlanStore()
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			plans, err := planStore.List()
			if err != nil {
				return err
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, plans)
			}
			return renderPlanListText(cmd.OutOrStdout(), plans)
		},
	}
}

func renderPlanText(out io.Writer, p *planning.Plan) error {
	_, _ = fmt.Fprintf(out, "Plan: %s\n", p.PlanID)
	_, _ = fmt.Fprintf(out, "Status: %s\n", p.Status)
	_, _ = fmt.Fprintf(out, "Risk Level: %s\n", p.RiskLevel)
	_, _ = fmt.Fprintf(out, "Created: %s\n\n", p.CreatedAt.Format(time.RFC3339))

	_, _ = fmt.Fprintf(out, "Intent:\n")
	_, _ = fmt.Fprintf(out, "  Source: %s\n", p.Intent.Source)
	_, _ = fmt.Fprintf(out, "  Input:  %s\n\n", p.Intent.RawInput)

	_, _ = fmt.Fprintf(out, "Scope:\n")
	if p.Scope.Owner != "" {
		_, _ = fmt.Fprintf(out, "  Repository: %s/%s\n", p.Scope.Owner, p.Scope.Repo)
	}
	if p.Scope.Branch != "" {
		_, _ = fmt.Fprintf(out, "  Branch: %s\n", p.Scope.Branch)
	}

	if len(p.Steps) > 0 {
		_, _ = fmt.Fprintf(out, "\nSteps:\n")
		for _, s := range p.Steps {
			rev := ""
			if !s.Reversible {
				rev = " [irreversible]"
			}
			_, _ = fmt.Fprintf(out, "  %d. [%s] %s → %s%s\n", s.Sequence, s.RiskLevel, s.Action, s.Target, rev)
			_, _ = fmt.Fprintf(out, "     %s\n", s.Description)
		}
	}

	if p.PolicyResult != nil {
		_, _ = fmt.Fprintf(out, "\nPolicy Verdict: %s\n", p.PolicyResult.Verdict)
		_, _ = fmt.Fprintf(out, "  Reason: %s\n", p.PolicyResult.Reason)
		_, _ = fmt.Fprintf(out, "  Explanation: %s\n", p.PolicyResult.Explanation)
		if len(p.PolicyResult.RiskFactors) > 0 {
			_, _ = fmt.Fprintf(out, "  Risk Factors:\n")
			for _, f := range p.PolicyResult.RiskFactors {
				_, _ = fmt.Fprintf(out, "    - %s\n", f)
			}
		}
		if len(p.PolicyResult.RequiredApprovals) > 0 {
			_, _ = fmt.Fprintf(out, "  Required Approvals:\n")
			for _, a := range p.PolicyResult.RequiredApprovals {
				_, _ = fmt.Fprintf(out, "    - %s\n", a)
			}
		}
	}

	return nil
}

func newPlanReviewCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	return &cobra.Command{
		Use:   "review [plan_id]",
		Short: "Show a plan's full review surface with evidence, blockers, and next actions",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			planStore := app.StorageProvider.PlanStore()
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			plan, err := planStore.Get(args[0])
			if err != nil {
				return fmt.Errorf("plan not found (plans are stored in memory for this session only): %w", err)
			}

			approvals, _ := planStore.GetApprovals(args[0])

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, map[string]any{
					"plan":      plan,
					"approvals": approvals,
				})
			}
			return renderPlanReviewText(cmd.OutOrStdout(), plan, approvals)
		},
	}
}

func newPlanApproveCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "approve [plan_id]",
		Short: "Approve a plan for execution",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			app := appFn()
			planStore := app.StorageProvider.PlanStore()
			format := effectiveOutputFormat(c, *flags, app.Config.Output)
			reason, _ := c.Flags().GetString("reason")
			modeStr, _ := c.Flags().GetString("mode")

			eng := policy.NewDefaultEngine()
			rev := reviewer.New(planStore, eng)

			var mode *planning.ExecutionMode
			if modeStr != "" {
				m := planning.ExecutionMode(modeStr)
				if !planning.ValidExecutionMode(m) {
					return fmt.Errorf("invalid execution mode %q; valid modes: observe, recommend, dry_run, execute", modeStr)
				}
				mode = &m
			}

			if err := rev.Approve(context.Background(), args[0], "operator", reason, mode); err != nil {
				return err
			}

			plan, _ := planStore.Get(args[0])
			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(c.OutOrStdout(), format, plan)
			}
			_, _ = fmt.Fprintf(c.OutOrStdout(), "Plan %s approved.\n", args[0])
			if plan != nil && plan.ExecutionMode != "" {
				_, _ = fmt.Fprintf(c.OutOrStdout(), "Execution mode: %s\n", plan.ExecutionMode)
			}
			return nil
		},
	}
	cmd.Flags().String("reason", "", "Reason for approval")
	cmd.Flags().String("mode", "", "Execution mode: observe, recommend, dry_run, execute")
	return cmd
}

func newPlanRejectCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reject [plan_id]",
		Short: "Reject a plan and block it from execution",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			app := appFn()
			planStore := app.StorageProvider.PlanStore()
			format := effectiveOutputFormat(c, *flags, app.Config.Output)
			reason, _ := c.Flags().GetString("reason")

			eng := policy.NewDefaultEngine()
			rev := reviewer.New(planStore, eng)

			if err := rev.Reject(context.Background(), args[0], "operator", reason); err != nil {
				return err
			}

			plan, _ := planStore.Get(args[0])
			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(c.OutOrStdout(), format, plan)
			}
			_, _ = fmt.Fprintf(c.OutOrStdout(), "Plan %s rejected.\n", args[0])
			_, _ = fmt.Fprintf(c.OutOrStdout(), "Reason: %s\n", reason)
			return nil
		},
	}
	cmd.Flags().String("reason", "", "Reason for rejection (required)")
	_ = cmd.MarkFlagRequired("reason")
	return cmd
}

func newPlanDeferCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "defer [plan_id]",
		Short: "Defer a plan for later review",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			app := appFn()
			planStore := app.StorageProvider.PlanStore()
			format := effectiveOutputFormat(c, *flags, app.Config.Output)
			reason, _ := c.Flags().GetString("reason")

			eng := policy.NewDefaultEngine()
			rev := reviewer.New(planStore, eng)

			if err := rev.Defer(context.Background(), args[0], "operator", reason); err != nil {
				return err
			}

			plan, _ := planStore.Get(args[0])
			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(c.OutOrStdout(), format, plan)
			}
			_, _ = fmt.Fprintf(c.OutOrStdout(), "Plan %s deferred to draft.\n", args[0])
			return nil
		},
	}
	cmd.Flags().String("reason", "", "Reason for deferring")
	return cmd
}

func newPlanEditCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit [plan_id]",
		Short: "Edit a plan's scope or execution mode and re-evaluate policy",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			app := appFn()
			planStore := app.StorageProvider.PlanStore()
			format := effectiveOutputFormat(c, *flags, app.Config.Output)
			branch, _ := c.Flags().GetString("branch")
			modeStr, _ := c.Flags().GetString("mode")

			edits := reviewer.PlanEdits{}
			if branch != "" {
				edits.Branch = &branch
			}
			if modeStr != "" {
				m := planning.ExecutionMode(modeStr)
				if !planning.ValidExecutionMode(m) {
					return fmt.Errorf("invalid execution mode %q; valid modes: observe, recommend, dry_run, execute", modeStr)
				}
				edits.ExecutionMode = &m
			}

			eng := policy.NewDefaultEngine()
			rev := reviewer.New(planStore, eng)

			if err := rev.Edit(context.Background(), args[0], "operator", edits); err != nil {
				return err
			}

			plan, _ := planStore.Get(args[0])
			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(c.OutOrStdout(), format, plan)
			}
			_, _ = fmt.Fprintf(c.OutOrStdout(), "Plan %s edited and re-evaluated.\n", args[0])
			if plan != nil && plan.PolicyResult != nil {
				_, _ = fmt.Fprintf(c.OutOrStdout(), "New policy verdict: %s\n", plan.PolicyResult.Verdict)
				_, _ = fmt.Fprintf(c.OutOrStdout(), "New status: %s\n", plan.Status)
			}
			return nil
		},
	}
	cmd.Flags().String("branch", "", "New target branch")
	cmd.Flags().String("mode", "", "Execution mode: observe, recommend, dry_run, execute")
	return cmd
}

func renderPlanReviewText(out io.Writer, p *planning.Plan, approvals []*planning.ApprovalRecord) error {
	_, _ = fmt.Fprintf(out, "═══ Plan Review ═══\n\n")
	_, _ = fmt.Fprintf(out, "Plan:   %s\n", p.PlanID)
	_, _ = fmt.Fprintf(out, "Status: %s\n", p.Status)
	_, _ = fmt.Fprintf(out, "Risk:   %s\n", p.RiskLevel)
	if p.ExecutionMode != "" {
		_, _ = fmt.Fprintf(out, "Mode:   %s\n", p.ExecutionMode)
	}
	_, _ = fmt.Fprintf(out, "Created: %s\n\n", p.CreatedAt.Format(time.RFC3339))

	_, _ = fmt.Fprintf(out, "── Intent ──\n")
	_, _ = fmt.Fprintf(out, "  Source: %s\n", p.Intent.Source)
	_, _ = fmt.Fprintf(out, "  Input:  %s\n\n", p.Intent.RawInput)

	_, _ = fmt.Fprintf(out, "── Scope ──\n")
	if p.Scope.Owner != "" {
		_, _ = fmt.Fprintf(out, "  Repository: %s/%s\n", p.Scope.Owner, p.Scope.Repo)
	}
	if p.Scope.Branch != "" {
		_, _ = fmt.Fprintf(out, "  Branch:     %s\n", p.Scope.Branch)
	}
	if p.Scope.Environment != "" {
		_, _ = fmt.Fprintf(out, "  Environment: %s\n", p.Scope.Environment)
	}

	if len(p.Steps) > 0 {
		_, _ = fmt.Fprintf(out, "\n── Steps ──\n")
		for _, s := range p.Steps {
			rev := ""
			if !s.Reversible {
				rev = " [irreversible]"
			}
			_, _ = fmt.Fprintf(out, "  %d. [%s] %s → %s%s\n", s.Sequence, s.RiskLevel, s.Action, s.Target, rev)
			_, _ = fmt.Fprintf(out, "     %s\n", s.Description)
		}
	}

	if p.PolicyResult != nil {
		_, _ = fmt.Fprintf(out, "\n── Policy Verdict ──\n")
		_, _ = fmt.Fprintf(out, "  Verdict:     %s\n", p.PolicyResult.Verdict)
		_, _ = fmt.Fprintf(out, "  Reason:      %s\n", p.PolicyResult.Reason)
		_, _ = fmt.Fprintf(out, "  Explanation: %s\n", p.PolicyResult.Explanation)
		if len(p.PolicyResult.RiskFactors) > 0 {
			_, _ = fmt.Fprintf(out, "  Risk Factors:\n")
			for _, f := range p.PolicyResult.RiskFactors {
				_, _ = fmt.Fprintf(out, "    - %s\n", f)
			}
		}
		if len(p.PolicyResult.RequiredApprovals) > 0 {
			_, _ = fmt.Fprintf(out, "  Required Approvals:\n")
			for _, a := range p.PolicyResult.RequiredApprovals {
				_, _ = fmt.Fprintf(out, "    - %s\n", a)
			}
		}
	}

	// Blockers
	_, _ = fmt.Fprintf(out, "\n── Current Blockers ──\n")
	if p.Status == planning.PlanBlocked && p.PolicyResult != nil {
		_, _ = fmt.Fprintf(out, "  BLOCKED: %s\n", p.PolicyResult.Reason)
		_, _ = fmt.Fprintf(out, "  Use 'gitdex plan edit %s --branch <branch>' to modify scope and re-evaluate.\n", p.PlanID)
	} else if p.Status == planning.PlanBlocked {
		_, _ = fmt.Fprintf(out, "  BLOCKED by review action.\n")
		_, _ = fmt.Fprintf(out, "  Use 'gitdex plan edit %s --branch <branch>' to modify scope and re-evaluate.\n", p.PlanID)
	} else if p.PolicyResult != nil && p.PolicyResult.Verdict == planning.VerdictEscalated {
		_, _ = fmt.Fprintf(out, "  ESCALATED: Requires approval from: %s\n", strings.Join(p.PolicyResult.RequiredApprovals, ", "))
	} else {
		_, _ = fmt.Fprintf(out, "  None\n")
	}

	// Evidence
	if len(p.EvidenceRefs) > 0 {
		_, _ = fmt.Fprintf(out, "\n── Evidence ──\n")
		for _, e := range p.EvidenceRefs {
			_, _ = fmt.Fprintf(out, "  - %s\n", e)
		}
	}

	// Approval history
	if len(approvals) > 0 {
		_, _ = fmt.Fprintf(out, "\n── Approval History ──\n")
		for _, a := range approvals {
			_, _ = fmt.Fprintf(out, "  %s | %s by %s | %s → %s",
				a.CreatedAt.Format(time.RFC3339), a.Action, a.Actor, a.PreviousStatus, a.NewStatus)
			if a.Reason != "" {
				_, _ = fmt.Fprintf(out, " | %s", a.Reason)
			}
			_, _ = fmt.Fprintln(out)
		}
	}

	// Next actionable path
	_, _ = fmt.Fprintf(out, "\n── Next Action ──\n")
	switch p.Status {
	case planning.PlanReviewRequired:
		_, _ = fmt.Fprintf(out, "  → gitdex plan approve %s    (approve for execution)\n", p.PlanID)
		_, _ = fmt.Fprintf(out, "  → gitdex plan reject %s --reason <reason>    (reject)\n", p.PlanID)
		_, _ = fmt.Fprintf(out, "  → gitdex plan defer %s     (defer for later)\n", p.PlanID)
		_, _ = fmt.Fprintf(out, "  → gitdex plan edit %s --branch <branch>    (edit and re-evaluate)\n", p.PlanID)
	case planning.PlanBlocked:
		_, _ = fmt.Fprintf(out, "  → gitdex plan edit %s --branch <branch>    (edit to reduce risk)\n", p.PlanID)
	case planning.PlanApproved:
		_, _ = fmt.Fprintf(out, "  → Plan is approved. Awaiting execution.\n")
	case planning.PlanDraft:
		_, _ = fmt.Fprintf(out, "  → Plan has been deferred. Recompile or edit when ready.\n")
	default:
		_, _ = fmt.Fprintf(out, "  → No actions available for status: %s\n", p.Status)
	}

	return nil
}

func renderPlanListText(out io.Writer, plans []*planning.Plan) error {
	if len(plans) == 0 {
		_, _ = fmt.Fprintln(out, "No plans found.")
		return nil
	}

	_, _ = fmt.Fprintf(out, "%-20s %-15s %-10s %s\n", "Plan ID", "Status", "Risk", "Intent")
	_, _ = fmt.Fprintln(out, strings.Repeat("─", 70))
	for _, p := range plans {
		inputPreview := p.Intent.RawInput
		if len(inputPreview) > 30 {
			inputPreview = inputPreview[:27] + "..."
		}
		_, _ = fmt.Fprintf(out, "%-20s %-15s %-10s %s\n", p.PlanID, p.Status, p.RiskLevel, inputPreview)
	}
	return nil
}
