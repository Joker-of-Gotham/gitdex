package command

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/your-org/gitdex/internal/app/setup"
	clioutput "github.com/your-org/gitdex/internal/cli/output"
)

type initOptions struct {
	nonInteractive       bool
	writeGlobal          bool
	writeRepo            bool
	defaultOutput        string
	defaultLogLevel      string
	defaultProfile       string
	identityMode         string
	gitHubHost           string
	gitHubAppID          string
	gitHubInstallationID string
	gitHubPrivateKeyPath string
}

func newInitCommand(flags *runtimeOptions) *cobra.Command {
	opts := initOptions{}

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Run terminal-first setup for identity, defaults, and config files",
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := setup.Run(setup.Options{
				In:                   cmd.InOrStdin(),
				Out:                  cmd.ErrOrStderr(),
				ConfigFile:           flags.configFile,
				NonInteractive:       opts.nonInteractive,
				DefaultOutput:        opts.defaultOutput,
				DefaultLogLevel:      opts.defaultLogLevel,
				DefaultProfile:       opts.defaultProfile,
				IdentityMode:         opts.identityMode,
				GitHubHost:           opts.gitHubHost,
				GitHubAppID:          opts.gitHubAppID,
				GitHubInstallationID: opts.gitHubInstallationID,
				GitHubPrivateKeyPath: opts.gitHubPrivateKeyPath,
				WriteGlobal:          opts.writeGlobal,
				WriteGlobalSet:       cmd.Flags().Changed("write-global"),
				WriteRepo:            opts.writeRepo,
				WriteRepoSet:         cmd.Flags().Changed("write-repo"),
			})
			if err != nil {
				return err
			}

			format := effectiveOutputFormat(cmd, *flags, result.Config.Config.Output)
			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, result)
			}

			return renderSetupText(cmd.OutOrStdout(), result)
		},
	}

	cmd.Flags().BoolVar(&opts.nonInteractive, "non-interactive", false, "Use flag values and defaults without interactive prompts")
	cmd.Flags().BoolVar(&opts.writeGlobal, "write-global", true, "Write the selected settings to the global config file")
	cmd.Flags().BoolVar(&opts.writeRepo, "write-repo", false, "Write a repo-local config file when a repository context is detected")
	cmd.Flags().StringVar(&opts.defaultOutput, "default-output", "text", "Default output format to persist in config")
	cmd.Flags().StringVar(&opts.defaultLogLevel, "default-log-level", "info", "Default log level to persist in config")
	cmd.Flags().StringVar(&opts.defaultProfile, "default-profile", "local", "Default runtime profile to persist in config")
	cmd.Flags().StringVar(&opts.identityMode, "identity-mode", "github-app", "Identity mode to persist in config")
	cmd.Flags().StringVar(&opts.gitHubHost, "github-host", "github.com", "GitHub host name for GitHub App connectivity")
	cmd.Flags().StringVar(&opts.gitHubAppID, "github-app-id", "", "GitHub App ID to persist in config")
	cmd.Flags().StringVar(&opts.gitHubInstallationID, "github-installation-id", "", "GitHub App installation ID to persist in config")
	cmd.Flags().StringVar(&opts.gitHubPrivateKeyPath, "github-private-key-path", "", "GitHub App private key path to persist in config")

	return markSkipBootstrap(cmd)
}

func renderSetupText(out io.Writer, result setup.Result) error {
	if _, err := fmt.Fprintln(out, "Setup complete"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(out, "Written files:"); err != nil {
		return err
	}
	for _, path := range result.WrittenFiles {
		if _, err := fmt.Fprintf(out, "- %s\n", path); err != nil {
			return err
		}
	}
	if err := renderKeyValueLine(out, "Output", fmt.Sprintf("%s (source: %s)", result.Config.Config.Output, result.Config.Sources["output"])); err != nil {
		return err
	}
	if err := renderKeyValueLine(out, "Log level", fmt.Sprintf("%s (source: %s)", result.Config.Config.LogLevel, result.Config.Sources["log_level"])); err != nil {
		return err
	}
	if err := renderKeyValueLine(out, "Profile", fmt.Sprintf("%s (source: %s)", result.Config.Config.Profile, result.Config.Sources["profile"])); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(out, "Next steps:"); err != nil {
		return err
	}
	for _, step := range result.NextSteps {
		if _, err := fmt.Fprintf(out, "- %s\n", strings.TrimSpace(step)); err != nil {
			return err
		}
	}
	return nil
}
