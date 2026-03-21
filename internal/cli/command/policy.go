package command

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/your-org/gitdex/internal/app/bootstrap"
	clioutput "github.com/your-org/gitdex/internal/cli/output"
	"github.com/your-org/gitdex/internal/policy"
)

func newPolicyGroupCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "policy",
		Short: "Configure policy bundles, risk tiers, and execution boundaries",
	}
	cmd.AddCommand(newPolicyShowCommand(flags, appFn))
	cmd.AddCommand(newPolicyListCommand(flags, appFn))
	cmd.AddCommand(newPolicyCreateCommand(flags, appFn))
	return cmd
}

func newPolicyShowCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show active policy bundle",
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)
			policyBundleStore := app.StorageProvider.PolicyBundleStore()

			active, err := policyBundleStore.GetActiveBundle()
			if err != nil {
				return fmt.Errorf("failed to get active bundle: %w", err)
			}
			if active == nil {
				if clioutput.IsStructured(format) {
					return clioutput.WriteValue(cmd.OutOrStdout(), format, map[string]string{
						"status":  "no_bundle",
						"message": "No policy bundle configured. Run 'gitdex policy create --name <name>' to create one.",
					})
				}
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No policy bundle configured. Run 'gitdex policy create --name <name>' to create one.")
				return nil
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, active)
			}
			return renderPolicyBundleText(cmd.OutOrStdout(), active)
		},
	}
}

func newPolicyListCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List policy bundles",
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)
			policyBundleStore := app.StorageProvider.PolicyBundleStore()

			bundles, err := policyBundleStore.ListBundles()
			if err != nil {
				return err
			}

			active, _ := policyBundleStore.GetActiveBundle()
			activeID := ""
			if active != nil {
				activeID = active.BundleID
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, map[string]any{
					"bundles":   bundles,
					"active_id": activeID,
				})
			}
			return renderPolicyBundleListText(cmd.OutOrStdout(), bundles, activeID)
		},
	}
}

func newPolicyCreateCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new policy bundle",
		Long:  "Create a new policy bundle. Use --name to set the bundle name.",
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)
			policyBundleStore := app.StorageProvider.PolicyBundleStore()

			name, _ := cmd.Flags().GetString("name")
			if name == "" {
				return fmt.Errorf("--name is required")
			}

			bundle := &policy.PolicyBundle{
				Name:      name,
				Version:   "1.0.0",
				CreatedAt: time.Now().UTC(),
			}

			if err := policyBundleStore.SaveBundle(bundle); err != nil {
				return fmt.Errorf("failed to create bundle: %w", err)
			}

			if err := policyBundleStore.SetActiveBundle(bundle.BundleID); err != nil {
				return err
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, bundle)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Policy bundle created: %s (id: %s)\n", bundle.Name, bundle.BundleID)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Set as active bundle.\n")
			return nil
		},
	}
	cmd.Flags().String("name", "", "Bundle name (required)")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func renderPolicyBundleText(out io.Writer, b *policy.PolicyBundle) error {
	_, _ = fmt.Fprintf(out, "═══ Active Policy Bundle ═══\n\n")
	_, _ = fmt.Fprintf(out, "Bundle ID:  %s\n", b.BundleID)
	_, _ = fmt.Fprintf(out, "Name:      %s\n", b.Name)
	_, _ = fmt.Fprintf(out, "Version:   %s\n", b.Version)
	_, _ = fmt.Fprintf(out, "Created:   %s\n", b.CreatedAt.Format(time.RFC3339))

	if len(b.CapabilityGrants) > 0 {
		_, _ = fmt.Fprintf(out, "\nCapability Grants:\n")
		for _, g := range b.CapabilityGrants {
			_, _ = fmt.Fprintf(out, "  - %s: %s\n", g.Scope, strings.Join(g.Capabilities, ", "))
		}
	}
	if len(b.ProtectedTargets) > 0 {
		_, _ = fmt.Fprintf(out, "\nProtected Targets:\n")
		for _, t := range b.ProtectedTargets {
			_, _ = fmt.Fprintf(out, "  - %s %q [%s]\n", t.TargetType, t.Pattern, t.ProtectionLevel)
		}
	}
	if len(b.ApprovalRules) > 0 {
		_, _ = fmt.Fprintf(out, "\nApproval Rules:\n")
		for _, r := range b.ApprovalRules {
			_, _ = fmt.Fprintf(out, "  - %s → %s (%s)\n", r.ActionPattern, strings.Join(r.RequiredApprovers, ", "), r.ApprovalType)
		}
	}
	return nil
}

func renderPolicyBundleListText(out io.Writer, bundles []*policy.PolicyBundle, activeID string) error {
	if len(bundles) == 0 {
		_, _ = fmt.Fprintln(out, "No policy bundles found.")
		return nil
	}

	_, _ = fmt.Fprintf(out, "%-24s %-20s %-10s %s\n", "Bundle ID", "Name", "Version", "Active")
	_, _ = fmt.Fprintln(out, strings.Repeat("─", 70))
	for _, b := range bundles {
		active := ""
		if b.BundleID == activeID {
			active = "*"
		}
		_, _ = fmt.Fprintf(out, "%-24s %-20s %-10s %s\n", b.BundleID, b.Name, b.Version, active)
	}
	return nil
}
