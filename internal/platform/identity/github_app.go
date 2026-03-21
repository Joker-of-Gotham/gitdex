package identity

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/your-org/gitdex/internal/platform/config"
)

var (
	ErrNoIdentity     = errors.New("identity: no GitHub App configuration provided")
	ErrMissingField   = errors.New("identity: required configuration field is missing")
	ErrInvalidKeyFile = errors.New("identity: private key file error")
)

type TransportResult struct {
	Transport http.RoundTripper
	Host      string
	Source    string
}

const (
	SourceGitHubAppConfig = "config.github-app"
	SourcePATConfig       = "config.github_pat"
	SourceEnvToken        = "env.github_token"
	SourceGitHubCLI       = "gh.auth"
)

var ghTokenRunner = func(host string) (string, error) {
	args := []string{"auth", "token"}
	if strings.TrimSpace(host) != "" {
		args = append(args, "--hostname", host)
	}
	out, err := exec.Command("gh", args...).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func NewGitHubAppTransport(cfg config.IdentityConfig, base http.RoundTripper) (*TransportResult, error) {
	if cfg.Mode != "github-app" {
		return nil, fmt.Errorf("%w: identity mode is %q, expected \"github-app\"", ErrNoIdentity, cfg.Mode)
	}

	ghCfg := cfg.GitHubApp
	if ghCfg.AppID == "" {
		return nil, fmt.Errorf("%w: app_id", ErrMissingField)
	}
	if ghCfg.InstallationID == "" {
		return nil, fmt.Errorf("%w: installation_id", ErrMissingField)
	}
	if ghCfg.PrivateKeyPath == "" {
		return nil, fmt.Errorf("%w: private_key_path", ErrMissingField)
	}

	appID, err := strconv.ParseInt(ghCfg.AppID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("%w: app_id %q is not a valid integer: %v", ErrMissingField, ghCfg.AppID, err)
	}

	instID, err := strconv.ParseInt(ghCfg.InstallationID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("%w: installation_id %q is not a valid integer: %v", ErrMissingField, ghCfg.InstallationID, err)
	}

	if _, err := os.Stat(ghCfg.PrivateKeyPath); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidKeyFile, err)
	}

	if base == nil {
		base = http.DefaultTransport
	}

	itr, err := ghinstallation.NewKeyFromFile(base, appID, instID, ghCfg.PrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to create transport: %v", ErrInvalidKeyFile, err)
	}

	host := ghCfg.Host
	if host == "" {
		host = "github.com"
	}
	if host != "github.com" {
		itr.BaseURL = fmt.Sprintf("https://%s/api/v3", host)
	}

	return &TransportResult{
		Transport: itr,
		Host:      host,
		Source:    SourceGitHubAppConfig,
	}, nil
}

func IsIdentityConfigured(cfg config.IdentityConfig) bool {
	switch cfg.Mode {
	case "github-app":
		return cfg.GitHubApp.AppID != "" &&
			cfg.GitHubApp.InstallationID != "" &&
			cfg.GitHubApp.PrivateKeyPath != ""
	case "token", "pat":
		return cfg.GitHubPAT != ""
	}
	return false
}

type patTransport struct {
	token string
	base  http.RoundTripper
}

func (t *patTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req2 := req.Clone(req.Context())
	req2.Header.Set("Authorization", "token "+t.token)
	return t.base.RoundTrip(req2)
}

func NewPATTransport(cfg config.IdentityConfig, base http.RoundTripper) (*TransportResult, error) {
	if cfg.GitHubPAT == "" {
		return nil, fmt.Errorf("%w: github_pat is empty", ErrMissingField)
	}
	if base == nil {
		base = http.DefaultTransport
	}
	host := cfg.GitHubApp.Host
	if host == "" {
		host = "github.com"
	}
	return &TransportResult{
		Transport: &patTransport{token: cfg.GitHubPAT, base: base},
		Host:      host,
		Source:    SourcePATConfig,
	}, nil
}

func ResolveTransport(cfg config.IdentityConfig, base http.RoundTripper) (*TransportResult, error) {
	if base == nil {
		base = http.DefaultTransport
	}

	if transport, err := NewTransport(cfg, base); err == nil {
		return transport, nil
	}
	if !runtimeFallbackEnabled() {
		return nil, fmt.Errorf("%w: no configured identity was available and runtime auth fallback is disabled", ErrNoIdentity)
	}

	for _, host := range candidateHosts(cfg.GitHubApp.Host) {
		if token := tokenFromEnv(host); token != "" {
			return &TransportResult{
				Transport: &patTransport{token: token, base: base},
				Host:      host,
				Source:    SourceEnvToken,
			}, nil
		}
		token, err := ghTokenRunner(host)
		if err == nil && strings.TrimSpace(token) != "" {
			return &TransportResult{
				Transport: &patTransport{token: token, base: base},
				Host:      host,
				Source:    SourceGitHubCLI,
			}, nil
		}
	}

	return nil, fmt.Errorf("%w: no configured identity, environment token, or gh auth session was available", ErrNoIdentity)
}

func runtimeFallbackEnabled() bool {
	raw := strings.TrimSpace(os.Getenv("GITDEX_DISABLE_RUNTIME_GITHUB_AUTH"))
	switch strings.ToLower(raw) {
	case "1", "true", "yes", "on":
		return false
	default:
		return true
	}
}

func candidateHosts(configuredHost string) []string {
	host := normalizeHost(configuredHost)
	candidates := make([]string, 0, 2)
	seen := map[string]bool{}
	add := func(value string) {
		value = normalizeHost(value)
		if value == "" {
			return
		}
		key := strings.ToLower(value)
		if seen[key] {
			return
		}
		seen[key] = true
		candidates = append(candidates, value)
	}

	add(host)
	add("github.com")
	return candidates
}

func normalizeHost(host string) string {
	host = strings.TrimSpace(host)
	host = strings.TrimPrefix(host, "https://")
	host = strings.TrimPrefix(host, "http://")
	if idx := strings.Index(host, "/"); idx >= 0 {
		host = host[:idx]
	}
	host = strings.TrimSpace(host)
	if host == "" {
		return "github.com"
	}
	return host
}

func tokenFromEnv(host string) string {
	lookup := func(keys ...string) string {
		for _, key := range keys {
			if value, ok := os.LookupEnv(key); ok && strings.TrimSpace(value) != "" {
				return strings.TrimSpace(value)
			}
		}
		return ""
	}

	host = normalizeHost(host)
	if strings.EqualFold(host, "github.com") {
		return lookup("GH_TOKEN", "GITHUB_TOKEN")
	}
	return lookup("GH_ENTERPRISE_TOKEN", "GITHUB_ENTERPRISE_TOKEN")
}

func NewTransport(cfg config.IdentityConfig, base http.RoundTripper) (*TransportResult, error) {
	switch cfg.Mode {
	case "github-app":
		return NewGitHubAppTransport(cfg, base)
	case "token", "pat":
		return NewPATTransport(cfg, base)
	default:
		if cfg.GitHubPAT != "" {
			return NewPATTransport(cfg, base)
		}
		return nil, fmt.Errorf("%w: unsupported identity mode %q", ErrNoIdentity, cfg.Mode)
	}
}
