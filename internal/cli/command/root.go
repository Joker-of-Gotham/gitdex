package command

import (
	"context"
	"fmt"
	"io"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/your-org/gitdex/internal/app/autonomyexec"
	"github.com/your-org/gitdex/internal/app/bootstrap"
	"github.com/your-org/gitdex/internal/app/session"
	"github.com/your-org/gitdex/internal/app/version"
	"github.com/your-org/gitdex/internal/autonomy"
	cliCompletion "github.com/your-org/gitdex/internal/cli/completion"
	"github.com/your-org/gitdex/internal/daemon/service"
	"github.com/your-org/gitdex/internal/llm/adapter"
	tuiapp "github.com/your-org/gitdex/internal/tui/app"
)

type commandOptions struct {
	in      io.Reader
	out     io.Writer
	errOut  io.Writer
	use     string
	version string
}

type runtimeOptions struct {
	configFile string
	output     string
	logLevel   string
	profile    string
}

func NewRootCommand() *cobra.Command {
	return newCommandTree(commandOptions{
		in:      os.Stdin,
		out:     os.Stdout,
		errOut:  os.Stderr,
		use:     version.CLIName,
		version: version.Version,
	}, true)
}

func NewDaemonBinaryRootCommand() *cobra.Command {
	return newCommandTree(commandOptions{
		in:      os.Stdin,
		out:     os.Stdout,
		errOut:  os.Stderr,
		use:     version.DaemonName,
		version: version.Version,
	}, false)
}

func newCommandTree(opts commandOptions, includeDaemonGroup bool) *cobra.Command {
	if opts.in == nil {
		opts.in = os.Stdin
	}
	if opts.out == nil {
		opts.out = os.Stdout
	}
	if opts.errOut == nil {
		opts.errOut = os.Stderr
	}
	if opts.version == "" {
		opts.version = version.Version
	}

	flags := runtimeOptions{}
	var app bootstrap.App
	var sessionCtx *session.TaskContext
	var llmProvider adapter.Provider

	root := &cobra.Command{
		Use:           opts.use,
		Short:         "Gitdex starter baseline",
		SilenceErrors: true,
		SilenceUsage:  true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if shouldSkipBootstrap(cmd) {
				return nil
			}

			loaded, err := bootstrap.Load(buildBootstrapOptions(flags, opts.version, func(name string) bool {
				return commandFlagChanged(cmd, name)
			}))
			if err != nil {
				return err
			}

			app = loaded
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return launchInteractiveTUI(cmd, func() bootstrap.App { return app }, &flags)
		},
	}

	root.SetIn(opts.in)
	root.SetOut(opts.out)
	root.SetErr(opts.errOut)

	root.PersistentFlags().StringVar(&flags.configFile, "config", "", "Path to a Gitdex config file")
	root.PersistentFlags().StringVar(&flags.output, "output", "text", "Output format")
	root.PersistentFlags().StringVar(&flags.logLevel, "log-level", "info", "Log level")
	root.PersistentFlags().StringVar(&flags.profile, "profile", "local", "Runtime profile")

	root.AddCommand(markSkipBootstrap(newVersionCommand(opts.version)))
	root.AddCommand(markSkipBootstrap(cliCompletion.NewCommand(root)))

	if includeDaemonGroup {
		root.AddCommand(newInitCommand(&flags))
		root.AddCommand(newDoctorCommand(&flags))
		root.AddCommand(newConfigCommand(&flags))
		root.AddCommand(newChatCommand(&flags, func() bootstrap.App { return app }, &sessionCtx, &llmProvider))
		root.AddCommand(newCapabilitiesCommand(&flags))
		root.AddCommand(newStatusCommand(&flags, func() bootstrap.App { return app }))
		root.AddCommand(newCockpitCommand(&flags, func() bootstrap.App { return app }))
		root.AddCommand(newPlanGroupCommand(&flags, func() bootstrap.App { return app }))
		root.AddCommand(newTaskGroupCommand(&flags, func() bootstrap.App { return app }))
		root.AddCommand(newIdentityGroupCommand(&flags, func() bootstrap.App { return app }))
		root.AddCommand(newPolicyGroupCommand(&flags, func() bootstrap.App { return app }))
		root.AddCommand(newAuditGroupCommand(&flags, func() bootstrap.App { return app }))
		root.AddCommand(newEmergencyGroupCommand(&flags, func() bootstrap.App { return app }))
		root.AddCommand(newAutonomyGroupCommand(&flags, func() bootstrap.App { return app }))
		root.AddCommand(newMonitorGroupCommand(&flags, func() bootstrap.App { return app }))
		root.AddCommand(newTriggerGroupCommand(&flags, func() bootstrap.App { return app }))
		root.AddCommand(newRecoveryGroupCommand(&flags, func() bootstrap.App { return app }))
		root.AddCommand(newHandoffGroupCommand(&flags, func() bootstrap.App { return app }))
		root.AddCommand(newCampaignGroupCommand(&flags, func() bootstrap.App { return app }))
		root.AddCommand(newRepoGroupCommand(&flags, func() bootstrap.App { return app }))
		root.AddCommand(newCollabGroupCommand(&flags, func() bootstrap.App { return app }))
		root.AddCommand(newReleaseGroupCommand(&flags, func() bootstrap.App { return app }))
		root.AddCommand(newAPIGroupCommand(&flags, func() bootstrap.App { return app }))
		root.AddCommand(newExportGroupCommand(&flags, func() bootstrap.App { return app }))
		root.AddCommand(newDaemonGroupCommand(func() bootstrap.App { return app }, opts.version))
	} else {
		root.AddCommand(newDaemonRunCommand(func() bootstrap.App { return app }, opts.version, version.DaemonName))
	}

	return root
}

func buildBootstrapOptions(flags runtimeOptions, currentVersion string, flagChanged func(string) bool) bootstrap.Options {
	if flagChanged == nil {
		flagChanged = func(string) bool { return false }
	}

	return bootstrap.Options{
		ConfigFile:  flags.configFile,
		Output:      flags.output,
		OutputSet:   flagChanged("output"),
		LogLevel:    flags.logLevel,
		LogLevelSet: flagChanged("log-level"),
		Profile:     flags.profile,
		ProfileSet:  flagChanged("profile"),
		Version:     currentVersion,
	}
}

func commandFlagChanged(cmd *cobra.Command, name string) bool {
	if cmd == nil {
		return false
	}

	flag := cmd.Flags().Lookup(name)
	return flag != nil && flag.Changed
}

func newVersionCommand(currentVersion string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the current starter version",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\n", currentVersion)
			return err
		},
	}
}

func newDaemonGroupCommand(appFn func() bootstrap.App, currentVersion string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "daemon",
		Short: "Run local daemon-oriented starter commands",
	}
	cmd.AddCommand(newDaemonRunCommand(appFn, currentVersion, version.CLIName))
	return cmd
}

func launchInteractiveTUI(cmd *cobra.Command, appFn func() bootstrap.App, _ *runtimeOptions) error {
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return cmd.Help()
	}

	a := appFn()
	m := tuiapp.New()
	m.SetBootstrapApp(a)

	summary := loadCockpitSummary(cmd, a, a.Config.Paths.RepositoryRoot, "", "")
	if summary != nil {
		m.SetSummary(summary)
	}

	p := tea.NewProgram(m)
	_, err := p.Run()
	return err
}

func newDaemonRunCommand(appFn func() bootstrap.App, currentVersion, binaryName string) *cobra.Command {
	var cruise bool
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Start the Gitdex daemon (HTTP control plane)",
		RunE: func(cmd *cobra.Command, args []string) error {
			app := appFn()
			healthAddress := app.Config.Daemon.HealthAddress
			if healthAddress == "" {
				healthAddress = "127.0.0.1:7777"
			}
			provider := app.StorageProvider

			if app.Config.Storage.AutoMigrate {
				if err := provider.Migrate(context.Background()); err != nil {
					_ = provider.Close()
					return err
				}
			}

			runCtx, cancel := context.WithCancel(context.Background())
			scheduler, err := startDaemonAutomation(runCtx, cmd, app)
			if err != nil {
				cancel()
				_ = provider.Close()
				return err
			}

			var cruiseEngine *autonomy.CruiseEngine
			if cruise {
				eng, cerr := autonomyexec.NewCruiseEngineForDaemon(app)
				if cerr != nil {
					cancel()
					if scheduler != nil {
						scheduler.Stop()
					}
					_ = provider.Close()
					return cerr
				}
				cruiseEngine = eng
				if serr := cruiseEngine.Start(runCtx); serr != nil {
					cancel()
					if scheduler != nil {
						scheduler.Stop()
					}
					_ = provider.Close()
					return serr
				}
			}

			cfg := service.Config{
				Address:         healthAddress,
				ReadTimeout:     service.DefaultConfig().ReadTimeout,
				WriteTimeout:    service.DefaultConfig().WriteTimeout,
				ShutdownTimeout: service.DefaultConfig().ShutdownTimeout,
				OnGitHubWebhook: func(ctx context.Context, cfg *autonomy.TriggerConfig, repoFullName string, ev *autonomy.TriggerEvent) error {
					return executeDaemonTrigger(ctx, cmd, app, cfg, repoFullName, ev)
				},
				Cruise: cruiseEngine,
			}
			runErr := service.Run(runCtx, cfg, provider)
			cancel()
			if cruiseEngine != nil {
				cruiseEngine.Stop()
			}
			if scheduler != nil {
				scheduler.Stop()
			}
			closeErr := provider.Close()
			if runErr != nil {
				return runErr
			}
			return closeErr
		},
	}
	cmd.Flags().BoolVar(&cruise, "cruise", false, "Run the autonomous cruise engine alongside the daemon")
	return cmd
}
