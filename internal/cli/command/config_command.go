package command

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	clioutput "github.com/your-org/gitdex/internal/cli/output"
	"github.com/your-org/gitdex/internal/platform/config"
	"github.com/your-org/gitdex/internal/platform/identity"
	"github.com/your-org/gitdex/internal/storage"
)

func newConfigCommand(flags *runtimeOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Inspect effective Gitdex configuration",
	}
	cmd.AddCommand(newConfigShowCommand(flags))
	return markSkipBootstrap(cmd)
}

func newConfigShowCommand(flags *runtimeOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show the current effective configuration and its sources",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(buildConfigOptions(*flags, cmd))
			if err != nil {
				return err
			}

			snapshot := normalizeConfigSnapshot(cfg.Snapshot())
			format := effectiveOutputFormat(cmd, *flags, cfg.Output)
			if clioutput.IsStructured(format) {
				return clioutput.WriteValue(cmd.OutOrStdout(), format, snapshot)
			}

			return renderConfigText(cmd.OutOrStdout(), snapshot)
		},
	}

	return markSkipBootstrap(cmd)
}

func renderConfigText(out io.Writer, snapshot config.Snapshot) error {
	authSource := "(none)"
	effectiveHost := normalizeGitHubHost(snapshot.Config.Identity.GitHubApp.Host)
	if tr, err := identity.ResolveTransport(snapshot.Config.Identity, nil); err == nil {
		if strings.TrimSpace(tr.Source) != "" {
			authSource = tr.Source
		}
		if strings.TrimSpace(tr.Host) != "" {
			effectiveHost = strings.TrimSpace(tr.Host)
		}
	}

	if _, err := fmt.Fprintln(out, "Effective Gitdex configuration"); err != nil {
		return err
	}
	if err := renderKeyValueLine(out, "Output", fmt.Sprintf("%s (source: %s)", snapshot.Config.Output, snapshot.Sources["output"])); err != nil {
		return err
	}
	if err := renderKeyValueLine(out, "Log level", fmt.Sprintf("%s (source: %s)", snapshot.Config.LogLevel, snapshot.Sources["log_level"])); err != nil {
		return err
	}
	if err := renderKeyValueLine(out, "Profile", fmt.Sprintf("%s (source: %s)", snapshot.Config.Profile, snapshot.Sources["profile"])); err != nil {
		return err
	}
	if err := renderKeyValueLine(out, "Identity mode", fmt.Sprintf("%s (source: %s)", snapshot.Config.Identity.Mode, snapshot.Sources["identity.mode"])); err != nil {
		return err
	}
	if err := renderKeyValueLine(out, "GitHub host", fmt.Sprintf("%s (source: %s)", snapshot.Config.Identity.GitHubApp.Host, snapshot.Sources["identity.github_app.host"])); err != nil {
		return err
	}
	if err := renderKeyValueLine(out, "GitHub auth", authSource); err != nil {
		return err
	}
	if err := renderKeyValueLine(out, "GitHub effective host", effectiveHost); err != nil {
		return err
	}
	workspaceRoots := "(none)"
	if len(snapshot.Config.Git.WorkspaceRoots) > 0 {
		workspaceRoots = strings.Join(snapshot.Config.Git.WorkspaceRoots, ", ")
	}
	if err := renderKeyValueLine(out, "Workspace roots", workspaceRoots); err != nil {
		return err
	}
	if err := renderKeyValueLine(out, "Storage backend", fmt.Sprintf("%s (source: %s)", snapshot.Config.Storage.Type, snapshot.Sources["storage.type"])); err != nil {
		return err
	}
	if err := renderKeyValueLine(out, "Storage dsn", firstNonEmpty(snapshot.Config.Storage.DSN, "(auto/default)")); err != nil {
		return err
	}
	if err := renderKeyValueLine(out, "Global config", snapshot.Paths.GlobalConfig); err != nil {
		return err
	}
	activeFiles := "(none)"
	if len(snapshot.Paths.ActiveFiles) > 0 {
		activeFiles = strings.Join(snapshot.Paths.ActiveFiles, ", ")
	}
	if err := renderKeyValueLine(out, "Active config files", activeFiles); err != nil {
		return err
	}
	if err := renderKeyValueLine(out, "Repo config", firstNonEmpty(snapshot.Paths.RepoConfig, "(not detected)")); err != nil {
		return err
	}
	if err := renderKeyValueLine(out, "Repository root", firstNonEmpty(snapshot.Paths.RepositoryRoot, "(not detected)")); err != nil {
		return err
	}
	return nil
}

func normalizeConfigSnapshot(snapshot config.Snapshot) config.Snapshot {
	baseDir := "."
	switch {
	case strings.TrimSpace(snapshot.Paths.ExplicitConfig) != "":
		baseDir = filepath.Dir(snapshot.Paths.ExplicitConfig)
	case strings.TrimSpace(snapshot.Paths.RepoConfig) != "":
		baseDir = filepath.Dir(snapshot.Paths.RepoConfig)
	case strings.TrimSpace(snapshot.Paths.GlobalConfig) != "":
		baseDir = filepath.Dir(snapshot.Paths.GlobalConfig)
	case strings.TrimSpace(snapshot.Paths.WorkingDir) != "":
		baseDir = snapshot.Paths.WorkingDir
	}

	normalized := storage.Config{
		Type:         storage.BackendType(snapshot.Config.Storage.Type),
		DSN:          snapshot.Config.Storage.DSN,
		MaxOpenConns: snapshot.Config.Storage.MaxOpenConns,
		MaxIdleConns: snapshot.Config.Storage.MaxIdleConns,
		AutoMigrate:  snapshot.Config.Storage.AutoMigrate,
	}.Normalized(baseDir)
	snapshot.Config.Storage.Type = string(normalized.Type)
	snapshot.Config.Storage.DSN = normalized.DSN
	return snapshot
}

func normalizeGitHubHost(host string) string {
	host = strings.TrimSpace(host)
	host = strings.TrimPrefix(host, "https://")
	host = strings.TrimPrefix(host, "http://")
	if idx := strings.Index(host, "/"); idx >= 0 {
		host = host[:idx]
	}
	if host == "" {
		return "github.com"
	}
	return host
}
