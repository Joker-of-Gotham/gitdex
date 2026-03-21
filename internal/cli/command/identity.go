package command

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/your-org/gitdex/internal/app/bootstrap"
	clioutput "github.com/your-org/gitdex/internal/cli/output"
	"github.com/your-org/gitdex/internal/identity"
)

func newIdentityGroupCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "identity",
		Short: "Authorize and manage Gitdex identities and scope",
	}
	cmd.AddCommand(newIdentityShowCommand(flags, appFn))
	cmd.AddCommand(newIdentityListCommand(flags, appFn))
	cmd.AddCommand(newIdentityRegisterCommand(flags, appFn))
	return cmd
}

func newIdentityShowCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show current identity and scope",
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)
			identityStore := app.StorageProvider.IdentityStore()

			current, err := identityStore.GetCurrentIdentity()
			if err != nil {
				return fmt.Errorf("failed to get current identity: %w", err)
			}
			if current == nil {
				if clioutput.IsStructured(format) {
					return clioutput.WriteValue(cmd.OutOrStdout(), format, map[string]string{
						"status":  "no_identity",
						"message": "No identity configured. Run 'gitdex identity register' to register one.",
					})
				}
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No identity configured. Run 'gitdex identity register' to register one.")
				return nil
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, current)
			}
			return renderIdentityText(cmd.OutOrStdout(), current)
		},
	}
}

func newIdentityListCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List identities",
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)
			identityStore := app.StorageProvider.IdentityStore()

			identities, err := identityStore.ListIdentities()
			if err != nil {
				return err
			}

			current, _ := identityStore.GetCurrentIdentity()
			currentID := ""
			if current != nil {
				currentID = current.IdentityID
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, map[string]any{
					"identities": identities,
					"current_id": currentID,
				})
			}
			return renderIdentityListText(cmd.OutOrStdout(), identities, currentID)
		},
	}
}

func newIdentityRegisterCommand(flags *runtimeOptions, appFn func() bootstrap.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register",
		Short: "Register an identity",
		Long:  "Register a Gitdex identity. Use --type github_app with --app-id and --installation-id for GitHub App.",
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			format := effectiveOutputFormat(cmd, *flags, app.Config.Output)
			identityStore := app.StorageProvider.IdentityStore()

			identityTypeStr, _ := cmd.Flags().GetString("type")
			appID, _ := cmd.Flags().GetString("app-id")
			installationID, _ := cmd.Flags().GetString("installation-id")
			orgScope, _ := cmd.Flags().GetString("org-scope")
			repoScope, _ := cmd.Flags().GetString("repo-scope")

			identityType := identity.IdentityType(identityTypeStr)
			if identityType != identity.IdentityTypeGitHubApp && identityType != identity.IdentityTypePAT && identityType != identity.IdentityTypeToken {
				return fmt.Errorf("invalid identity type %q; valid types: github_app, pat, token", identityTypeStr)
			}

			if identityType == identity.IdentityTypeGitHubApp {
				if appID == "" || installationID == "" {
					return fmt.Errorf("--app-id and --installation-id are required for github_app identity")
				}
			}

			ident := &identity.AppIdentity{
				IdentityType:   identityType,
				AppID:          appID,
				InstallationID: installationID,
				OrgScope:       orgScope,
				RepoScope:      repoScope,
				Capabilities:   []identity.Capability{identity.CapReadRepo, identity.CapReadIssues, identity.CapReadPRs},
				CreatedAt:      time.Now().UTC(),
			}

			if err := identityStore.SaveIdentity(ident); err != nil {
				return fmt.Errorf("failed to register identity: %w", err)
			}

			if err := identityStore.SetCurrentIdentity(ident.IdentityID); err != nil {
				return err
			}

			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, ident)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Identity registered: %s (type: %s)\n", ident.IdentityID, ident.IdentityType)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Set as current identity.\n")
			return nil
		},
	}
	cmd.Flags().String("type", "github_app", "Identity type: github_app, pat, token")
	cmd.Flags().String("app-id", "", "GitHub App ID (required for github_app)")
	cmd.Flags().String("installation-id", "", "GitHub Installation ID (required for github_app)")
	cmd.Flags().String("org-scope", "", "Organization scope")
	cmd.Flags().String("repo-scope", "", "Repository scope")
	return cmd
}

func renderIdentityText(out io.Writer, i *identity.AppIdentity) error {
	_, _ = fmt.Fprintf(out, "═══ Current Identity ═══\n\n")
	_, _ = fmt.Fprintf(out, "Identity ID:   %s\n", i.IdentityID)
	_, _ = fmt.Fprintf(out, "Type:          %s\n", i.IdentityType)
	if i.AppID != "" {
		_, _ = fmt.Fprintf(out, "App ID:        %s\n", i.AppID)
	}
	if i.InstallationID != "" {
		_, _ = fmt.Fprintf(out, "Installation:  %s\n", i.InstallationID)
	}
	if i.OrgScope != "" {
		_, _ = fmt.Fprintf(out, "Org Scope:     %s\n", i.OrgScope)
	}
	if i.RepoScope != "" {
		_, _ = fmt.Fprintf(out, "Repo Scope:    %s\n", i.RepoScope)
	}
	_, _ = fmt.Fprintf(out, "Capabilities:  %s\n", strings.Join(capStrings(i.Capabilities), ", "))
	_, _ = fmt.Fprintf(out, "Created:       %s\n", i.CreatedAt.Format(time.RFC3339))

	if len(i.ScopeGrants) > 0 {
		_, _ = fmt.Fprintf(out, "\nScope Grants:\n")
		for _, g := range i.ScopeGrants {
			_, _ = fmt.Fprintf(out, "  - %s/%s: %s\n", g.ScopeType, g.ScopeValue, strings.Join(capStrings(g.Capabilities), ", "))
		}
	}
	return nil
}

func renderIdentityListText(out io.Writer, identities []*identity.AppIdentity, currentID string) error {
	if len(identities) == 0 {
		_, _ = fmt.Fprintln(out, "No identities found.")
		return nil
	}

	_, _ = fmt.Fprintf(out, "%-24s %-12s %-12s %s\n", "Identity ID", "Type", "App ID", "Current")
	_, _ = fmt.Fprintln(out, strings.Repeat("─", 65))
	for _, i := range identities {
		current := ""
		if i.IdentityID == currentID {
			current = "*"
		}
		_, _ = fmt.Fprintf(out, "%-24s %-12s %-12s %s\n", i.IdentityID, i.IdentityType, i.AppID, current)
	}
	return nil
}

func capStrings(caps []identity.Capability) []string {
	s := make([]string, len(caps))
	for i, c := range caps {
		s[i] = string(c)
	}
	return s
}
