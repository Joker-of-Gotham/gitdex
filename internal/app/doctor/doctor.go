package doctor

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/your-org/gitdex/internal/platform/config"
	"github.com/your-org/gitdex/internal/platform/identity"
)

const envGitHubAPIURL = "GITDEX_GITHUB_API_URL"

type Status string

const (
	StatusPass          Status = "pass"
	StatusNotConfigured Status = "not_configured"
	StatusIncomplete    Status = "incomplete"
	StatusFail          Status = "fail"
)

type Check struct {
	ID      string `json:"id" yaml:"id"`
	Status  Status `json:"status" yaml:"status"`
	Summary string `json:"summary" yaml:"summary"`
	Detail  string `json:"detail" yaml:"detail"`
	Fix     string `json:"fix" yaml:"fix"`
	Source  string `json:"source" yaml:"source"`
}

type Report struct {
	Status string             `json:"status" yaml:"status"`
	Paths  config.ConfigPaths `json:"paths" yaml:"paths"`
	Checks []Check            `json:"checks" yaml:"checks"`
}

type Options struct {
	ConfigOptions config.Options
	HTTPClient    *http.Client
	LookPath      func(string) (string, error)
}

func Run(opts Options) (Report, error) {
	paths, err := config.ResolvePaths(opts.ConfigOptions)
	if err != nil {
		return Report{}, err
	}

	cfg, loadErr := config.Load(opts.ConfigOptions)
	if loadErr == nil {
		paths = cfg.Paths
	}
	checks := []Check{
		configCheck(paths, loadErr),
		repositoryCheck(paths),
		identityCheck(cfg, loadErr),
		connectivityCheck(cfg, loadErr, opts.HTTPClient),
		gitToolCheck(opts.LookPath),
	}

	report := Report{
		Status: overallStatus(checks),
		Paths:  paths,
		Checks: checks,
	}
	return report, nil
}

func configCheck(paths config.ConfigPaths, loadErr error) Check {
	source := strings.Join(paths.ActiveFiles, ", ")
	if source == "" {
		source = firstNonEmpty(paths.ExplicitConfig, paths.GlobalConfig, paths.RepoConfig, paths.WorkingDir)
	}

	if loadErr != nil {
		return Check{
			ID:      "config.load",
			Status:  StatusFail,
			Summary: "Gitdex configuration could not be loaded",
			Detail:  loadErr.Error(),
			Fix:     "Fix or remove the broken config file, then rerun `gitdex doctor`.",
			Source:  source,
		}
	}

	if len(paths.ActiveFiles) == 0 {
		return Check{
			ID:      "config.load",
			Status:  StatusNotConfigured,
			Summary: "No Gitdex config files were found",
			Detail:  "Gitdex is currently running on built-in defaults only.",
			Fix:     "Run `gitdex init` to create a global config, or set `GITDEX_CONFIG` to an explicit file.",
			Source:  source,
		}
	}

	return Check{
		ID:      "config.load",
		Status:  StatusPass,
		Summary: "Gitdex configuration loaded successfully",
		Detail:  fmt.Sprintf("Active config files: %s", strings.Join(paths.ActiveFiles, ", ")),
		Fix:     "No action required.",
		Source:  source,
	}
}

func repositoryCheck(paths config.ConfigPaths) Check {
	if !paths.RepositoryDetected {
		return Check{
			ID:      "repository.context",
			Status:  StatusNotConfigured,
			Summary: "No Git repository context detected",
			Detail:  fmt.Sprintf("Working directory %s is not inside a Git repository.", paths.WorkingDir),
			Fix:     "Run the command inside a Git repository if you want repo-local config and repository diagnostics.",
			Source:  paths.WorkingDir,
		}
	}

	return Check{
		ID:      "repository.context",
		Status:  StatusPass,
		Summary: "Git repository context detected",
		Detail:  fmt.Sprintf("Repository root: %s", paths.RepositoryRoot),
		Fix:     "No action required.",
		Source:  paths.RepositoryRoot,
	}
}

func identityCheck(cfg config.Config, loadErr error) Check {
	if loadErr != nil {
		return Check{
			ID:      "identity.github",
			Status:  StatusIncomplete,
			Summary: "GitHub identity could not be fully evaluated",
			Detail:  "Configuration failed to load before identity checks could complete.",
			Fix:     "Fix the config loading error first, then rerun `gitdex doctor`.",
			Source:  "identity",
		}
	}

	switch cfg.Identity.Mode {
	case "", "none":
		if resolved, err := identity.ResolveTransport(cfg.Identity, nil); err == nil {
			return Check{
				ID:      "identity.github",
				Status:  StatusPass,
				Summary: "GitHub identity is available from runtime fallback",
				Detail:  fmt.Sprintf("Gitdex can authenticate through %s against %s even though config mode is disabled.", resolved.Source, resolved.Host),
				Fix:     "No action required, but you may persist credentials explicitly in Settings if you want deterministic config.",
				Source:  resolved.Source,
			}
		}
		return Check{
			ID:      "identity.github",
			Status:  StatusNotConfigured,
			Summary: "GitHub identity is disabled",
			Detail:  "Remote GitHub operations will remain unavailable until an identity mode is configured or a runtime fallback token is present.",
			Fix:     "Set `identity.mode` to `token` or `github-app`, provide matching credentials, or log in with `gh auth login`.",
			Source:  "identity.mode",
		}
	case "token":
		if strings.TrimSpace(cfg.Identity.GitHubPAT) == "" {
			if resolved, err := identity.ResolveTransport(cfg.Identity, nil); err == nil {
				return Check{
					ID:      "identity.github",
					Status:  StatusPass,
					Summary: "GitHub token identity is available from runtime fallback",
					Detail:  fmt.Sprintf("No PAT is stored in config, but Gitdex can authenticate through %s against %s.", resolved.Source, resolved.Host),
					Fix:     "No action required, but persisting a token or keeping `gh` logged in will make behavior more predictable.",
					Source:  resolved.Source,
				}
			}
			return Check{
				ID:      "identity.github",
				Status:  StatusIncomplete,
				Summary: "GitHub token mode is selected but PAT is missing",
				Detail:  "Set `identity.github_pat` to enable authenticated GitHub reads and writes, or make sure `gh auth login` / `GITHUB_TOKEN` is available at runtime.",
				Fix:     "Add a valid personal access token in config or Settings, or log in with `gh auth login`, then rerun `gitdex doctor`.",
				Source:  "identity.github_pat",
			}
		}
		return Check{
			ID:      "identity.github",
			Status:  StatusPass,
			Summary: "GitHub token identity is configured",
			Detail:  "PAT-based authentication is available for GitHub operations.",
			Fix:     "No action required.",
			Source:  "identity.github_pat",
		}
	case "github-app":
		return githubAppIdentityCheck(cfg)
	default:
		if resolved, err := identity.ResolveTransport(cfg.Identity, nil); err == nil {
			return Check{
				ID:      "identity.github",
				Status:  StatusPass,
				Summary: "GitHub identity is available from runtime fallback",
				Detail:  fmt.Sprintf("Configured mode %q is invalid, but Gitdex can still authenticate through %s against %s.", cfg.Identity.Mode, resolved.Source, resolved.Host),
				Fix:     "Clean up `identity.mode` in config to avoid ambiguity.",
				Source:  resolved.Source,
			}
		}
		return Check{
			ID:      "identity.github",
			Status:  StatusIncomplete,
			Summary: "Unsupported identity mode configured",
			Detail:  fmt.Sprintf("Configured mode %q is invalid. Supported modes: none, token, github-app.", cfg.Identity.Mode),
			Fix:     "Set `identity.mode` to `token` or `github-app`, then rerun setup.",
			Source:  "identity.mode",
		}
	}
}

func githubAppIdentityCheck(cfg config.Config) Check {
	missing := []string{}
	if strings.TrimSpace(cfg.Identity.GitHubApp.AppID) == "" {
		missing = append(missing, "identity.github_app.app_id")
	}
	if strings.TrimSpace(cfg.Identity.GitHubApp.InstallationID) == "" {
		missing = append(missing, "identity.github_app.installation_id")
	}
	if strings.TrimSpace(cfg.Identity.GitHubApp.PrivateKeyPath) == "" {
		missing = append(missing, "identity.github_app.private_key_path")
	}

	if len(missing) == 3 {
		return Check{
			ID:      "identity.github",
			Status:  StatusNotConfigured,
			Summary: "GitHub App identity is not configured yet",
			Detail:  "App ID, installation ID, and private key path are all empty.",
			Fix:     "Run `gitdex init` and provide your GitHub App settings.",
			Source:  "identity.github_app",
		}
	}
	if len(missing) > 0 {
		return Check{
			ID:      "identity.github",
			Status:  StatusIncomplete,
			Summary: "GitHub App identity is incomplete",
			Detail:  fmt.Sprintf("Missing fields: %s", strings.Join(missing, ", ")),
			Fix:     "Fill the missing GitHub App fields in config and rerun `gitdex doctor`.",
			Source:  "identity.github_app",
		}
	}

	if _, err := os.Stat(cfg.Identity.GitHubApp.PrivateKeyPath); err != nil {
		return Check{
			ID:      "identity.github",
			Status:  StatusFail,
			Summary: "GitHub App private key path is not readable",
			Detail:  err.Error(),
			Fix:     "Point `identity.github_app.private_key_path` to a readable private key file.",
			Source:  cfg.Identity.GitHubApp.PrivateKeyPath,
		}
	}

	return Check{
		ID:      "identity.github",
		Status:  StatusPass,
		Summary: "GitHub App identity fields are present",
		Detail:  "App ID, installation ID, and private key path are configured.",
		Fix:     "No action required.",
		Source:  "identity.github_app",
	}
}

func connectivityCheck(cfg config.Config, loadErr error, client *http.Client) Check {
	endpoint := os.Getenv(envGitHubAPIURL)
	if endpoint == "" {
		host := "github.com"
		if loadErr == nil {
			if resolved, err := identity.ResolveTransport(cfg.Identity, nil); err == nil {
				host = resolved.Host
			} else if !errors.Is(err, identity.ErrNoIdentity) && strings.TrimSpace(cfg.Identity.GitHubApp.Host) != "" {
				host = strings.TrimSpace(cfg.Identity.GitHubApp.Host)
			} else if strings.TrimSpace(cfg.Identity.GitHubApp.Host) != "" {
				host = strings.TrimSpace(cfg.Identity.GitHubApp.Host)
			}
		}
		endpoint = githubAPIURL(host)
	}

	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}

	resp, err := client.Get(endpoint)
	if err != nil {
		return Check{
			ID:      "connectivity.github_api",
			Status:  StatusFail,
			Summary: "GitHub connectivity check failed",
			Detail:  err.Error(),
			Fix:     "Check network connectivity, proxy settings, or the configured GitHub host.",
			Source:  endpoint,
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return Check{
			ID:      "connectivity.github_api",
			Status:  StatusFail,
			Summary: "GitHub connectivity check returned an unexpected status",
			Detail:  fmt.Sprintf("HTTP %d from %s", resp.StatusCode, endpoint),
			Fix:     "Verify the GitHub host is correct and reachable from this machine.",
			Source:  endpoint,
		}
	}

	return Check{
		ID:      "connectivity.github_api",
		Status:  StatusPass,
		Summary: "GitHub connectivity check passed",
		Detail:  fmt.Sprintf("Received HTTP %d from %s", resp.StatusCode, endpoint),
		Fix:     "No action required.",
		Source:  endpoint,
	}
}

func gitToolCheck(lookPath func(string) (string, error)) Check {
	if lookPath == nil {
		lookPath = exec.LookPath
	}

	path, err := lookPath("git")
	if err != nil {
		return Check{
			ID:      "tool.git",
			Status:  StatusFail,
			Summary: "Required local tool `git` was not found",
			Detail:  err.Error(),
			Fix:     "Install Git and ensure it is available on PATH.",
			Source:  "git",
		}
	}

	return Check{
		ID:      "tool.git",
		Status:  StatusPass,
		Summary: "Required local tool `git` is available",
		Detail:  fmt.Sprintf("Resolved git binary: %s", path),
		Fix:     "No action required.",
		Source:  path,
	}
}

func githubAPIURL(host string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return "https://api.github.com"
	}
	if strings.HasPrefix(host, "http://") || strings.HasPrefix(host, "https://") {
		return host
	}
	if host == "github.com" {
		return "https://api.github.com"
	}
	return fmt.Sprintf("https://%s/api/v3", host)
}

func overallStatus(checks []Check) string {
	for _, check := range checks {
		if check.Status != StatusPass {
			return "needs_attention"
		}
	}
	return "pass"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
