package runtime

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Joker-of-Gotham/gitdex/internal/config"
	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
	"github.com/Joker-of-Gotham/gitdex/internal/platform"
	"github.com/Joker-of-Gotham/gitdex/internal/platform/bitbucket"
	"github.com/Joker-of-Gotham/gitdex/internal/platform/github"
	"github.com/Joker-of-Gotham/gitdex/internal/platform/gitlab"
)

type bundleAdapter interface {
	Kind() platform.AdapterKind
	Resolve(ctx context.Context, remoteURL, owner, repo string, platformCfg config.PlatformConfig, adapterCfg config.AdapterConfig) (*Bundle, error)
}

type Bundle struct {
	Platform        platform.Platform
	RemoteURL       string
	Executors       map[string]platform.AdminExecutor
	Adapter         platform.AdapterKind
	ExecutorAdapter platform.AdapterExecutor
}

type resolver func(remoteURL string, platformCfg config.PlatformConfig, adapterCfg config.AdapterConfig) (*Bundle, error)

type routeAttempt struct {
	label string
	err   error
}

var resolvers = map[platform.Platform]resolver{
	platform.PlatformGitHub:    resolveGitHub,
	platform.PlatformGitLab:    resolveGitLab,
	platform.PlatformBitbucket: resolveBitbucket,
}

func ResolveAdminBundle(state *status.GitState, platformCfg config.PlatformConfig, adapterCfg config.AdapterConfig) (*Bundle, error) {
	if state == nil {
		return nil, fmt.Errorf("git state unavailable")
	}
	remoteURL := strings.TrimSpace(platform.PreferredRemoteURL(state.RemoteInfos))
	if remoteURL == "" {
		return nil, fmt.Errorf("repository remote URL unavailable")
	}
	detected := platform.DetectPlatform(remoteURL)
	resolve := resolvers[detected]
	if resolve == nil {
		return nil, fmt.Errorf("platform %s admin executors are unavailable", detected.String())
	}
	return resolve(remoteURL, platformCfg, adapterCfg)
}

func resolveGitHub(remoteURL string, platformCfg config.PlatformConfig, adapterCfg config.AdapterConfig) (*Bundle, error) {
	owner, repo, err := platform.GitHubOwnerRepoFromRemote(remoteURL)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	adapters := []bundleAdapter{
		gitHubAPIAdapter{},
		gitHubGHAdapter{},
		gitHubBrowserAdapter{},
	}
	attempts := make([]routeAttempt, 0, len(adapters))
	for _, candidate := range adapters {
		bundle, err := candidate.Resolve(ctx, remoteURL, owner, repo, platformCfg, adapterCfg)
		if err == nil && bundle != nil {
			return bundle, nil
		}
		attempts = append(attempts, routeAttempt{label: string(candidate.Kind()), err: err})
	}
	return nil, explicitRouteFailure("GitHub", []string{"api", "gh", "browser"}, attempts)
}

func resolveGitLab(remoteURL string, platformCfg config.PlatformConfig, adapterCfg config.AdapterConfig) (*Bundle, error) {
	projectPath, err := platform.GitLabProjectPathFromRemote(remoteURL)
	if err != nil {
		return nil, err
	}
	token := strings.TrimSpace(platformCfg.GitLabToken)
	if token != "" {
		client := gitlab.New(token, projectPath)
		return &Bundle{
			Platform:        platform.PlatformGitLab,
			RemoteURL:       remoteURL,
			Executors:       client.AdminExecutors(),
			Adapter:         platform.AdapterAPI,
			ExecutorAdapter: platform.NewAPIAdapterExecutor(),
		}, nil
	}
	bundle, browserErr := browserFallbackBundle(platform.PlatformGitLab, remoteURL, adapterCfg.GitLab.Browser.Enabled, adapterCfg.GitLab.Browser.Driver)
	if browserErr == nil {
		return bundle, nil
	}
	return nil, explicitRouteFailure("GitLab", []string{"api", "browser"}, []routeAttempt{
		{label: "api", err: fmt.Errorf("GitLab token is not configured")},
		{label: "browser", err: browserErr},
	})
}

func resolveBitbucket(remoteURL string, platformCfg config.PlatformConfig, adapterCfg config.AdapterConfig) (*Bundle, error) {
	workspace, repo, err := platform.BitbucketWorkspaceRepoFromRemote(remoteURL)
	if err != nil {
		return nil, err
	}
	token := strings.TrimSpace(platformCfg.BitbucketToken)
	if token != "" {
		client := bitbucket.New(token, workspace, repo)
		return &Bundle{
			Platform:        platform.PlatformBitbucket,
			RemoteURL:       remoteURL,
			Executors:       client.AdminExecutors(),
			Adapter:         platform.AdapterAPI,
			ExecutorAdapter: platform.NewAPIAdapterExecutor(),
		}, nil
	}
	bundle, browserErr := browserFallbackBundle(platform.PlatformBitbucket, remoteURL, adapterCfg.Bitbucket.Browser.Enabled, adapterCfg.Bitbucket.Browser.Driver)
	if browserErr == nil {
		return bundle, nil
	}
	return nil, explicitRouteFailure("Bitbucket", []string{"api", "browser"}, []routeAttempt{
		{label: "api", err: fmt.Errorf("Bitbucket token is not configured")},
		{label: "browser", err: browserErr},
	})
}

func browserFallbackBundle(platformID platform.Platform, remoteURL string, enabled bool, driver string) (*Bundle, error) {
	if !enabled {
		return nil, fmt.Errorf("browser adapter disabled")
	}
	driver = strings.TrimSpace(driver)
	if driver == "" {
		driver = selectBrowserDriver(platformID)
	}
	return &Bundle{
		Platform:        platformID,
		RemoteURL:       remoteURL,
		Executors:       platform.NewStubAdminExecutors(platform.CapabilityIDs(platformID)),
		Adapter:         platform.AdapterBrowser,
		ExecutorAdapter: platform.NewBrowserStubAdapterExecutor(driver),
	}, nil
}

func selectBrowserDriver(platformID platform.Platform) string {
	switch platformID {
	case platform.PlatformGitLab:
		return "gitlab-browser"
	case platform.PlatformBitbucket:
		return "bitbucket-browser"
	default:
		return "default"
	}
}

func ghAuthToken(binary string) (string, error) {
	binary = strings.TrimSpace(binary)
	if binary == "" {
		binary = "gh"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, binary, "auth", "token").Output()
	if err != nil {
		ext := strings.ToLower(filepath.Ext(binary))
		if ext == ".cmd" || ext == ".bat" {
			out, cmdErr := exec.CommandContext(ctx, "cmd", "/c", binary, "auth", "token").Output()
			if cmdErr == nil {
				return strings.TrimSpace(string(out)), nil
			}
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

type gitHubAPIAdapter struct{}

func (gitHubAPIAdapter) Kind() platform.AdapterKind { return platform.AdapterAPI }

func (gitHubAPIAdapter) Resolve(_ context.Context, remoteURL, owner, repo string, platformCfg config.PlatformConfig, _ config.AdapterConfig) (*Bundle, error) {
	token := strings.TrimSpace(platformCfg.GitHubToken)
	if token == "" {
		return nil, fmt.Errorf("GitHub token is not configured")
	}
	client := github.New(token, owner, repo)
	return &Bundle{
		Platform:        platform.PlatformGitHub,
		RemoteURL:       remoteURL,
		Executors:       client.AdminExecutors(),
		Adapter:         platform.AdapterAPI,
		ExecutorAdapter: platform.NewAPIAdapterExecutor(),
	}, nil
}

type gitHubGHAdapter struct{}

func (gitHubGHAdapter) Kind() platform.AdapterKind { return platform.AdapterCLI }

func (gitHubGHAdapter) Resolve(_ context.Context, remoteURL, owner, repo string, _ config.PlatformConfig, adapterCfg config.AdapterConfig) (*Bundle, error) {
	if !adapterCfg.GitHub.GH.Enabled {
		return nil, fmt.Errorf("gh adapter disabled")
	}
	token, err := ghAuthToken(adapterCfg.GitHub.GH.Binary)
	if err != nil || strings.TrimSpace(token) == "" {
		if err == nil {
			err = fmt.Errorf("gh adapter returned an empty token")
		}
		return nil, err
	}
	client := github.NewCLI(adapterCfg.GitHub.GH.Binary, owner, repo)
	return &Bundle{
		Platform:        platform.PlatformGitHub,
		RemoteURL:       remoteURL,
		Executors:       client.AdminExecutors(),
		Adapter:         platform.AdapterCLI,
		ExecutorAdapter: platform.NewCLIAdapterExecutor(adapterCfg.GitHub.GH.Binary),
	}, nil
}

type gitHubBrowserAdapter struct{}

func (gitHubBrowserAdapter) Kind() platform.AdapterKind { return platform.AdapterBrowser }

func (gitHubBrowserAdapter) Resolve(_ context.Context, remoteURL, _, _ string, _ config.PlatformConfig, adapterCfg config.AdapterConfig) (*Bundle, error) {
	if !adapterCfg.GitHub.Browser.Enabled {
		return nil, fmt.Errorf("browser adapter disabled")
	}
	return &Bundle{
		Platform:        platform.PlatformGitHub,
		RemoteURL:       remoteURL,
		Executors:       platform.NewStubAdminExecutors(platform.CapabilityIDs(platform.PlatformGitHub)),
		Adapter:         platform.AdapterBrowser,
		ExecutorAdapter: platform.NewBrowserStubAdapterExecutor(adapterCfg.GitHub.Browser.Driver),
	}, nil
}

func explicitRouteFailure(platformLabel string, route []string, attempts []routeAttempt) error {
	parts := make([]string, 0, len(attempts))
	for _, attempt := range attempts {
		label := strings.TrimSpace(attempt.label)
		if label == "" {
			continue
		}
		if attempt.err == nil {
			parts = append(parts, label+": unavailable")
			continue
		}
		parts = append(parts, label+": "+strings.TrimSpace(attempt.err.Error()))
	}
	routeLabel := strings.Join(route, " -> ")
	if len(parts) == 0 {
		return fmt.Errorf("%s adapter routing failed (%s)", platformLabel, routeLabel)
	}
	return fmt.Errorf("%s adapter routing failed (%s): %s", platformLabel, routeLabel, strings.Join(parts, "; "))
}
