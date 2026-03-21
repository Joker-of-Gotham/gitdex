package command

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/your-org/gitdex/internal/app/bootstrap"
	"github.com/your-org/gitdex/internal/campaign"
	clioutput "github.com/your-org/gitdex/internal/cli/output"
)

var campaignStoreOverride campaign.CampaignStore

// SetCampaignStoreForTest allows integration tests to inject a custom store.
func SetCampaignStoreForTest(s campaign.CampaignStore) func() {
	prev := campaignStoreOverride
	campaignStoreOverride = s
	return func() { campaignStoreOverride = prev }
}

func getCampaignStore(appFn func() bootstrap.App) campaign.CampaignStore {
	if campaignStoreOverride != nil {
		return campaignStoreOverride
	}
	return appFn().StorageProvider.CampaignStore()
}

func newCampaignGroupCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "campaign",
		Short: "Define and manage governed multi-repository campaigns",
	}
	cmd.AddCommand(newCampaignCreateCommand(flags, appFn))
	cmd.AddCommand(newCampaignShowCommand(flags, appFn))
	cmd.AddCommand(newCampaignListCommand(flags, appFn))
	cmd.AddCommand(newCampaignAddRepoCommand(flags, appFn))
	cmd.AddCommand(newCampaignRemoveRepoCommand(flags, appFn))
	cmd.AddCommand(newCampaignMatrixCommand(flags, appFn))
	cmd.AddCommand(newCampaignStatusCommand(flags, appFn))
	cmd.AddCommand(newCampaignApproveCommand(flags, appFn))
	cmd.AddCommand(newCampaignExcludeCommand(flags, appFn))
	cmd.AddCommand(newCampaignRetryCommand(flags, appFn))
	cmd.AddCommand(newCampaignInterveneCommand(flags, appFn))
	return cmd
}

func newCampaignCreateCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new campaign",
		Long:  "Create a campaign with --name and optional --description.",
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			campaignStore := getCampaignStore(appFn)
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			name, _ := cmd.Flags().GetString("name")
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			desc, _ := cmd.Flags().GetString("description")

			c := &campaign.Campaign{
				Name:        name,
				Description: desc,
				Status:      campaign.StatusDraft,
				TargetRepos: []campaign.RepoTarget{},
				CreatedBy:   "cli",
			}
			if err := campaignStore.SaveCampaign(c); err != nil {
				return fmt.Errorf("failed to create campaign: %w", err)
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, c)
			}
			return renderCampaignText(cmd.OutOrStdout(), c)
		},
	}
	cmd.Flags().String("name", "", "Campaign name (required)")
	cmd.Flags().String("description", "", "Campaign description")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newCampaignShowCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	return &cobra.Command{
		Use:   "show <campaign_id>",
		Short: "Show campaign details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			campaignStore := getCampaignStore(appFn)
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			c, err := campaignStore.GetCampaign(args[0])
			if err != nil {
				return err
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, c)
			}
			return renderCampaignText(cmd.OutOrStdout(), c)
		},
	}
}

func newCampaignListCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all campaigns",
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			campaignStore := getCampaignStore(appFn)
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			list, err := campaignStore.ListCampaigns()
			if err != nil {
				return err
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, map[string]any{"campaigns": list})
			}
			return renderCampaignListText(cmd.OutOrStdout(), list)
		},
	}
}

func newCampaignAddRepoCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-repo <campaign_id>",
		Short: "Add a repository target to a campaign",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			campaignStore := getCampaignStore(appFn)
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			repo, _ := cmd.Flags().GetString("repo")
			if repo == "" {
				return fmt.Errorf("--repo is required (format: owner/repo)")
			}
			owner, name := parseOwnerRepo(repo)
			if owner == "" || name == "" {
				return fmt.Errorf("--repo must be in format owner/repo")
			}

			c, err := campaignStore.GetCampaign(args[0])
			if err != nil {
				return err
			}
			for _, t := range c.TargetRepos {
				if t.Owner == owner && t.Repo == name {
					if clioutput.IsStructured(format) {
						return clioutput.WriteValue(cmd.OutOrStdout(), format, c)
					}
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Repo %s already in campaign\n", repo)
					return nil
				}
			}
			c.TargetRepos = append(c.TargetRepos, campaign.RepoTarget{
				Owner:           owner,
				Repo:            name,
				InclusionStatus: campaign.InclusionPending,
			})
			if err := campaignStore.UpdateCampaign(c); err != nil {
				return err
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, c)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Added %s to campaign %s\n", repo, c.CampaignID)
			return nil
		},
	}
	cmd.Flags().String("repo", "", "Repository in owner/repo format (required)")
	_ = cmd.MarkFlagRequired("repo")
	return cmd
}

func newCampaignRemoveRepoCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove-repo <campaign_id>",
		Short: "Remove a repository target from a campaign",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			campaignStore := getCampaignStore(appFn)
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			repo, _ := cmd.Flags().GetString("repo")
			if repo == "" {
				return fmt.Errorf("--repo is required (format: owner/repo)")
			}
			owner, name := parseOwnerRepo(repo)
			if owner == "" || name == "" {
				return fmt.Errorf("--repo must be in format owner/repo")
			}

			c, err := campaignStore.GetCampaign(args[0])
			if err != nil {
				return err
			}
			filtered := make([]campaign.RepoTarget, 0, len(c.TargetRepos))
			removed := false
			for _, t := range c.TargetRepos {
				if t.Owner == owner && t.Repo == name {
					removed = true
					continue
				}
				filtered = append(filtered, t)
			}
			if !removed {
				if clioutput.IsStructured(format) {
					return clioutput.WriteValue(cmd.OutOrStdout(), format, c)
				}
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Repo %s not in campaign\n", repo)
				return nil
			}
			c.TargetRepos = filtered
			if err := campaignStore.UpdateCampaign(c); err != nil {
				return err
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, c)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Removed %s from campaign %s\n", repo, c.CampaignID)
			return nil
		},
	}
	cmd.Flags().String("repo", "", "Repository in owner/repo format (required)")
	_ = cmd.MarkFlagRequired("repo")
	return cmd
}

func newCampaignMatrixCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	engine := campaign.NewDefaultMatrixEngine()
	return &cobra.Command{
		Use:   "matrix <campaign_id>",
		Short: "Show campaign matrix view",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			campaignStore := getCampaignStore(appFn)
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			c, err := campaignStore.GetCampaign(args[0])
			if err != nil {
				return err
			}

			mat, err := engine.Build(cmd.Context(), c)
			if err != nil {
				return fmt.Errorf("build matrix: %w", err)
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, mat)
			}
			return renderCampaignMatrixText(cmd.OutOrStdout(), mat)
		},
	}
}

func newCampaignStatusCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	engine := campaign.NewDefaultMatrixEngine()
	return &cobra.Command{
		Use:   "status <campaign_id>",
		Short: "Show campaign status summary",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			campaignStore := getCampaignStore(appFn)
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			c, err := campaignStore.GetCampaign(args[0])
			if err != nil {
				return err
			}

			mat, err := engine.Build(cmd.Context(), c)
			if err != nil {
				return fmt.Errorf("build matrix: %w", err)
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, mat.Summary)
			}
			return renderCampaignSummaryText(cmd.OutOrStdout(), c, mat.Summary)
		},
	}
}

func newCampaignApproveCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "approve <campaign_id>",
		Short: "Approve a repository in a campaign",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			campaignStore := getCampaignStore(appFn)
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			repo, _ := cmd.Flags().GetString("repo")
			if repo == "" {
				return fmt.Errorf("--repo is required (format: owner/repo)")
			}
			owner, name := parseOwnerRepo(repo)
			if owner == "" || name == "" {
				return fmt.Errorf("--repo must be in format owner/repo")
			}

			engine := campaign.NewDefaultInterventionEngine(campaignStore)
			req := campaign.InterventionRequest{
				InterventionType: campaign.InterventionApproveRepo,
				CampaignID:       args[0],
				Owner:            owner,
				Repo:             name,
				Actor:            "cli",
			}
			result, err := engine.Execute(cmd.Context(), req)
			if err != nil {
				return err
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, result)
			}
			return renderInterventionResult(cmd.OutOrStdout(), result)
		},
	}
	cmd.Flags().String("repo", "", "Repository in owner/repo format (required)")
	_ = cmd.MarkFlagRequired("repo")
	return cmd
}

func newCampaignExcludeCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exclude <campaign_id>",
		Short: "Exclude a repository from a campaign",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			campaignStore := getCampaignStore(appFn)
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			repo, _ := cmd.Flags().GetString("repo")
			if repo == "" {
				return fmt.Errorf("--repo is required (format: owner/repo)")
			}
			owner, name := parseOwnerRepo(repo)
			if owner == "" || name == "" {
				return fmt.Errorf("--repo must be in format owner/repo")
			}
			reason, _ := cmd.Flags().GetString("reason")

			engine := campaign.NewDefaultInterventionEngine(campaignStore)
			req := campaign.InterventionRequest{
				InterventionType: campaign.InterventionExcludeRepo,
				CampaignID:       args[0],
				Owner:            owner,
				Repo:             name,
				Reason:           reason,
				Actor:            "cli",
			}
			result, err := engine.Execute(cmd.Context(), req)
			if err != nil {
				return err
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, result)
			}
			return renderInterventionResult(cmd.OutOrStdout(), result)
		},
	}
	cmd.Flags().String("repo", "", "Repository in owner/repo format (required)")
	cmd.Flags().String("reason", "", "Reason for excluding")
	_ = cmd.MarkFlagRequired("repo")
	return cmd
}

func newCampaignRetryCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "retry <campaign_id>",
		Short: "Retry a failed repository in a campaign",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			campaignStore := getCampaignStore(appFn)
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			repo, _ := cmd.Flags().GetString("repo")
			if repo == "" {
				return fmt.Errorf("--repo is required (format: owner/repo)")
			}
			owner, name := parseOwnerRepo(repo)
			if owner == "" || name == "" {
				return fmt.Errorf("--repo must be in format owner/repo")
			}

			engine := campaign.NewDefaultInterventionEngine(campaignStore)
			req := campaign.InterventionRequest{
				InterventionType: campaign.InterventionRetryRepo,
				CampaignID:       args[0],
				Owner:            owner,
				Repo:             name,
				Actor:            "cli",
			}
			result, err := engine.Execute(cmd.Context(), req)
			if err != nil {
				return err
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, result)
			}
			return renderInterventionResult(cmd.OutOrStdout(), result)
		},
	}
	cmd.Flags().String("repo", "", "Repository in owner/repo format (required)")
	_ = cmd.MarkFlagRequired("repo")
	return cmd
}

func newCampaignInterveneCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "intervene <campaign_id>",
		Short: "Intervene on a repository (pause, resume, etc.)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			campaignStore := getCampaignStore(appFn)
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)

			repo, _ := cmd.Flags().GetString("repo")
			if repo == "" {
				return fmt.Errorf("--repo is required (format: owner/repo)")
			}
			owner, name := parseOwnerRepo(repo)
			if owner == "" || name == "" {
				return fmt.Errorf("--repo must be in format owner/repo")
			}
			action, _ := cmd.Flags().GetString("action")
			if action == "" {
				return fmt.Errorf("--action is required (pause, resume, override_plan)")
			}

			var itype campaign.InterventionType
			switch strings.ToLower(action) {
			case "pause":
				itype = campaign.InterventionPauseRepo
			case "resume":
				itype = campaign.InterventionResumeRepo
			case "override_plan":
				itype = campaign.InterventionOverridePlan
			default:
				return fmt.Errorf("invalid action %q; use pause, resume, or override_plan", action)
			}

			engine := campaign.NewDefaultInterventionEngine(campaignStore)
			req := campaign.InterventionRequest{
				InterventionType: itype,
				CampaignID:       args[0],
				Owner:            owner,
				Repo:             name,
				Actor:            "cli",
			}
			if itype == campaign.InterventionOverridePlan {
				overrides, _ := cmd.Flags().GetStringSlice("overrides")
				if len(overrides) > 0 {
					req.Overrides = make(map[string]string)
					for _, ov := range overrides {
						k, v, ok := strings.Cut(ov, "=")
						if ok && strings.TrimSpace(k) != "" {
							req.Overrides[strings.TrimSpace(k)] = strings.TrimSpace(v)
						}
					}
				}
			}
			result, err := engine.Execute(cmd.Context(), req)
			if err != nil {
				return err
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, result)
			}
			return renderInterventionResult(cmd.OutOrStdout(), result)
		},
	}
	cmd.Flags().String("repo", "", "Repository in owner/repo format (required)")
	cmd.Flags().String("action", "", "Action: pause, resume, override_plan (required)")
	cmd.Flags().StringSlice("overrides", []string{}, "Key=value overrides for override_plan action")
	_ = cmd.MarkFlagRequired("repo")
	_ = cmd.MarkFlagRequired("action")
	return cmd
}

func parseOwnerRepo(s string) (owner, repo string) {
	parts := strings.SplitN(s, "/", 2)
	if len(parts) != 2 {
		return "", ""
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
}

func renderCampaignText(out io.Writer, c *campaign.Campaign) error {
	_, _ = fmt.Fprintf(out, "Campaign ID:   %s\n", c.CampaignID)
	_, _ = fmt.Fprintf(out, "Name:          %s\n", c.Name)
	_, _ = fmt.Fprintf(out, "Description:   %s\n", c.Description)
	_, _ = fmt.Fprintf(out, "Status:        %s\n", c.Status)
	_, _ = fmt.Fprintf(out, "Created by:    %s\n", c.CreatedBy)
	_, _ = fmt.Fprintf(out, "Created at:    %s\n", c.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))
	_, _ = fmt.Fprintf(out, "Updated at:    %s\n", c.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"))
	if len(c.TargetRepos) > 0 {
		_, _ = fmt.Fprintf(out, "\nTarget repos:\n")
		for _, t := range c.TargetRepos {
			_, _ = fmt.Fprintf(out, "  - %s/%s (%s)\n", t.Owner, t.Repo, t.InclusionStatus)
		}
	}
	return nil
}

func renderCampaignListText(out io.Writer, list []*campaign.Campaign) error {
	if len(list) == 0 {
		_, _ = fmt.Fprintln(out, "No campaigns found.")
		return nil
	}
	_, _ = fmt.Fprintf(out, "%-24s %-20s %-12s %s\n", "Campaign ID", "Name", "Status", "Repos")
	_, _ = fmt.Fprintln(out, strings.Repeat("─", 80))
	for _, c := range list {
		n := len(c.TargetRepos)
		_, _ = fmt.Fprintf(out, "%-24s %-20s %-12s %d\n", c.CampaignID, c.Name, c.Status, n)
	}
	return nil
}

func renderCampaignMatrixText(out io.Writer, mat *campaign.CampaignMatrix) error {
	_, _ = fmt.Fprintf(out, "Campaign Matrix: %s\n\n", mat.CampaignID)
	s := mat.Summary
	_, _ = fmt.Fprintf(out, "Total: %d | Succeeded: %d | Failed: %d | Pending: %d | Excluded: %d\n\n",
		s.Total, s.Succeeded, s.Failed, s.Pending, s.Excluded)
	_, _ = fmt.Fprintf(out, "%-20s %-16s %-12s %-12s %s\n",
		"Owner/Repo", "Status", "Plan ID", "Task ID", "Last Updated")
	_, _ = fmt.Fprintln(out, strings.Repeat("─", 80))
	for _, e := range mat.Entries {
		_, _ = fmt.Fprintf(out, "%-20s %-16s %-12s %-12s %s\n",
			e.Owner+"/"+e.Repo, e.Status, e.PlanID, e.TaskID, e.LastUpdated.Format("2006-01-02 15:04"))
	}
	return nil
}

func renderCampaignSummaryText(out io.Writer, c *campaign.Campaign, s campaign.MatrixSummary) error {
	_, _ = fmt.Fprintf(out, "Campaign: %s (%s)\n", c.Name, c.CampaignID)
	_, _ = fmt.Fprintf(out, "Status: %s\n", c.Status)
	_, _ = fmt.Fprintf(out, "Repos: total=%d succeeded=%d failed=%d pending=%d excluded=%d\n",
		s.Total, s.Succeeded, s.Failed, s.Pending, s.Excluded)
	return nil
}

func renderInterventionResult(out io.Writer, r *campaign.InterventionResult) error {
	status := "OK"
	if !r.Success {
		status = "FAILED"
	}
	_, _ = fmt.Fprintf(out, "Status: %s\n", status)
	_, _ = fmt.Fprintf(out, "Message: %s\n", r.Message)
	if r.PreviousStatus != "" {
		_, _ = fmt.Fprintf(out, "Previous status: %s\n", r.PreviousStatus)
	}
	if r.NewStatus != "" {
		_, _ = fmt.Fprintf(out, "New status: %s\n", r.NewStatus)
	}
	return nil
}
