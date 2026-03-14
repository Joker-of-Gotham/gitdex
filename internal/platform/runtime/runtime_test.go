package runtime

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Joker-of-Gotham/gitdex/internal/config"
	"github.com/Joker-of-Gotham/gitdex/internal/git"
	"github.com/Joker-of-Gotham/gitdex/internal/git/status"
	"github.com/Joker-of-Gotham/gitdex/internal/platform"
)

func TestResolveAdminBundleGitHub(t *testing.T) {
	state := &status.GitState{
		RemoteInfos: []git.RemoteInfo{{
			Name:    "origin",
			PushURL: "git@github.com:Joker-of-Gotham/gitdex.git",
		}},
	}

	bundle, err := ResolveAdminBundle(state, config.PlatformConfig{GitHubToken: "token"}, config.AdapterConfig{})
	if err != nil {
		t.Fatalf("ResolveAdminBundle returned error: %v", err)
	}
	if bundle.Platform != platform.PlatformGitHub {
		t.Fatalf("expected github platform, got %s", bundle.Platform.String())
	}
	if bundle.Executors["pages"] == nil {
		t.Fatalf("expected pages executor to be registered")
	}
	if bundle.Adapter != platform.AdapterAPI {
		t.Fatalf("expected api adapter, got %s", bundle.Adapter)
	}
	if bundle.ExecutorAdapter == nil {
		t.Fatalf("expected adapter executor to be populated")
	}
	if bundle.ExecutorAdapter.Kind() != platform.AdapterAPI {
		t.Fatalf("expected api adapter executor, got %s", bundle.ExecutorAdapter.Kind())
	}
}

func TestResolveAdminBundleRequiresToken(t *testing.T) {
	state := &status.GitState{
		RemoteInfos: []git.RemoteInfo{{
			Name:    "origin",
			PushURL: "git@github.com:Joker-of-Gotham/gitdex.git",
		}},
	}

	if _, err := ResolveAdminBundle(state, config.PlatformConfig{}, config.AdapterConfig{}); err == nil {
		t.Fatalf("expected missing token error")
	} else if got := err.Error(); got == "" || !containsAll(got, "GitHub adapter routing failed", "api -> gh -> browser") {
		t.Fatalf("expected explicit route failure, got %q", got)
	}
}

func TestResolveAdminBundleFallsBackToGHAdapterToken(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "gh.cmd")
	if err := os.WriteFile(script, []byte("@echo gh-token\r\n"), 0o700); err != nil {
		t.Fatalf("write gh stub: %v", err)
	}

	state := &status.GitState{
		RemoteInfos: []git.RemoteInfo{{
			Name:    "origin",
			PushURL: "git@github.com:Joker-of-Gotham/gitdex.git",
		}},
	}

	bundle, err := ResolveAdminBundle(state, config.PlatformConfig{}, config.AdapterConfig{
		GitHub: config.GitHubAdapterConfig{
			GH: config.CommandAdapterConfig{
				Enabled: true,
				Binary:  script,
			},
		},
	})
	if err != nil {
		t.Fatalf("ResolveAdminBundle gh fallback returned error: %v", err)
	}
	if bundle.Adapter != platform.AdapterCLI {
		t.Fatalf("expected gh adapter, got %s", bundle.Adapter)
	}
	if bundle.ExecutorAdapter == nil {
		t.Fatalf("expected gh adapter executor to be populated")
	}
	if bundle.ExecutorAdapter.Kind() != platform.AdapterCLI {
		t.Fatalf("expected gh adapter executor, got %s", bundle.ExecutorAdapter.Kind())
	}
}

func TestResolveAdminBundleGHAdapterExecutesCLIBackedInspect(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "gh.cmd")
	content := "@echo off\r\n" +
		"setlocal EnableDelayedExpansion\r\n" +
		"if \"%1\"==\"auth\" goto auth\r\n" +
		"if \"%1\"==\"api\" goto api\r\n" +
		"echo unsupported %* 1>&2\r\n" +
		"exit /b 1\r\n" +
		":auth\r\n" +
		"if \"%2\"==\"token\" echo gh-token\r\n" +
		"exit /b 0\r\n" +
		":api\r\n" +
		"set endpoint=%2\r\n" +
		"if \"%endpoint%\"==\"repos/Joker-of-Gotham/gitdex/pages\" (\r\n" +
		"  echo {\"url\":\"https://joker-of-gotham.github.io/gitdex\",\"status\":\"built\"}\r\n" +
		"  exit /b 0\r\n" +
		")\r\n" +
		"echo unexpected endpoint %endpoint% 1>&2\r\n" +
		"exit /b 1\r\n"
	if err := os.WriteFile(script, []byte(content), 0o700); err != nil {
		t.Fatalf("write gh stub: %v", err)
	}

	state := &status.GitState{
		RemoteInfos: []git.RemoteInfo{{
			Name:    "origin",
			PushURL: "git@github.com:Joker-of-Gotham/gitdex.git",
		}},
	}

	bundle, err := ResolveAdminBundle(state, config.PlatformConfig{}, config.AdapterConfig{
		GitHub: config.GitHubAdapterConfig{
			GH: config.CommandAdapterConfig{
				Enabled: true,
				Binary:  script,
			},
		},
	})
	if err != nil {
		t.Fatalf("ResolveAdminBundle gh fallback returned error: %v", err)
	}
	snap, err := bundle.Executors["pages"].Inspect(context.Background(), platform.AdminInspectRequest{})
	if err != nil {
		t.Fatalf("cli-backed inspect failed: %v", err)
	}
	if snap == nil || len(snap.State) == 0 {
		t.Fatal("expected snapshot state from cli-backed bundle")
	}
}

func TestGHAdapterFakeExecutorAnnotatesCLIAuditMetadata(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "gh.cmd")
	content := "@echo off\r\n" +
		"setlocal EnableDelayedExpansion\r\n" +
		"if \"%1\"==\"auth\" goto auth\r\n" +
		"if \"%1\"==\"api\" goto api\r\n" +
		"echo unsupported %* 1>&2\r\n" +
		"exit /b 1\r\n" +
		":auth\r\n" +
		"if \"%2\"==\"token\" echo gh-token\r\n" +
		"exit /b 0\r\n" +
		":api\r\n" +
		"echo {\"url\":\"https://joker-of-gotham.github.io/gitdex\",\"status\":\"built\"}\r\n" +
		"exit /b 0\r\n"
	if err := os.WriteFile(script, []byte(content), 0o700); err != nil {
		t.Fatalf("write gh stub: %v", err)
	}

	state := &status.GitState{
		RemoteInfos: []git.RemoteInfo{{
			Name:    "origin",
			PushURL: "git@github.com:Joker-of-Gotham/gitdex.git",
		}},
	}
	bundle, err := ResolveAdminBundle(state, config.PlatformConfig{}, config.AdapterConfig{
		GitHub: config.GitHubAdapterConfig{
			GH: config.CommandAdapterConfig{
				Enabled: true,
				Binary:  script,
			},
		},
	})
	if err != nil {
		t.Fatalf("ResolveAdminBundle gh fallback returned error: %v", err)
	}
	snap, err := bundle.ExecutorAdapter.Inspect(context.Background(), bundle.Executors["pages"], platform.AdminInspectRequest{ResourceID: "github-pages"})
	if err != nil {
		t.Fatalf("cli-backed adapter inspect failed: %v", err)
	}
	if snap.Metadata["adapter_backed"] != string(platform.AdapterCLI) || snap.Metadata["adapter_binary"] != script {
		t.Fatalf("expected cli audit metadata, got %+v", snap.Metadata)
	}
}

func TestResolveAdminBundleFallsBackToBrowserStubAdapter(t *testing.T) {
	state := &status.GitState{
		RemoteInfos: []git.RemoteInfo{{
			Name:    "origin",
			PushURL: "git@github.com:Joker-of-Gotham/gitdex.git",
		}},
	}

	bundle, err := ResolveAdminBundle(state, config.PlatformConfig{}, config.AdapterConfig{
		GitHub: config.GitHubAdapterConfig{
			Browser: config.BrowserAdapterConfig{
				Enabled: true,
				Driver:  "stub-driver",
			},
		},
	})
	if err != nil {
		t.Fatalf("ResolveAdminBundle browser fallback returned error: %v", err)
	}
	if bundle.Adapter != platform.AdapterBrowser {
		t.Fatalf("expected browser adapter, got %s", bundle.Adapter)
	}
	if bundle.ExecutorAdapter == nil {
		t.Fatal("expected browser adapter executor")
	}
	if bundle.ExecutorAdapter.Kind() != platform.AdapterBrowser {
		t.Fatalf("expected browser adapter executor kind, got %s", bundle.ExecutorAdapter.Kind())
	}
	if bundle.Executors["pages"] == nil {
		t.Fatal("expected browser bundle to expose stub executors")
	}
}

func TestResolveAdminBundlePrefersGHOverBrowserWhenAPIMissing(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "gh.cmd")
	if err := os.WriteFile(script, []byte("@echo gh-token\r\n"), 0o700); err != nil {
		t.Fatalf("write gh stub: %v", err)
	}

	state := &status.GitState{
		RemoteInfos: []git.RemoteInfo{{
			Name:    "origin",
			PushURL: "git@github.com:Joker-of-Gotham/gitdex.git",
		}},
	}

	bundle, err := ResolveAdminBundle(state, config.PlatformConfig{}, config.AdapterConfig{
		GitHub: config.GitHubAdapterConfig{
			GH: config.CommandAdapterConfig{
				Enabled: true,
				Binary:  script,
			},
			Browser: config.BrowserAdapterConfig{
				Enabled: true,
				Driver:  "stub-driver",
			},
		},
	})
	if err != nil {
		t.Fatalf("ResolveAdminBundle returned error: %v", err)
	}
	if bundle.Adapter != platform.AdapterCLI {
		t.Fatalf("expected gh adapter to win before browser, got %s", bundle.Adapter)
	}
}

func TestResolveAdminBundleFallsThroughGHToBrowserOnAuthFailure(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "gh.cmd")
	content := "@echo off\r\n" +
		"echo auth failed 1>&2\r\n" +
		"exit /b 1\r\n"
	if err := os.WriteFile(script, []byte(content), 0o700); err != nil {
		t.Fatalf("write gh stub: %v", err)
	}

	state := &status.GitState{
		RemoteInfos: []git.RemoteInfo{{
			Name:    "origin",
			PushURL: "git@github.com:Joker-of-Gotham/gitdex.git",
		}},
	}

	bundle, err := ResolveAdminBundle(state, config.PlatformConfig{}, config.AdapterConfig{
		GitHub: config.GitHubAdapterConfig{
			GH: config.CommandAdapterConfig{
				Enabled: true,
				Binary:  script,
			},
			Browser: config.BrowserAdapterConfig{
				Enabled: true,
				Driver:  "stub-driver",
			},
		},
	})
	if err != nil {
		t.Fatalf("ResolveAdminBundle returned error: %v", err)
	}
	if bundle.Adapter != platform.AdapterBrowser {
		t.Fatalf("expected browser adapter after gh failure, got %s", bundle.Adapter)
	}
}

func TestResolveAdminBundleGitLab(t *testing.T) {
	state := &status.GitState{
		RemoteInfos: []git.RemoteInfo{{
			Name:    "origin",
			PushURL: "git@gitlab.com:group/repo.git",
		}},
	}

	bundle, err := ResolveAdminBundle(state, config.PlatformConfig{GitLabToken: "token"}, config.AdapterConfig{})
	if err != nil {
		t.Fatalf("ResolveAdminBundle returned error: %v", err)
	}
	if bundle.Platform != platform.PlatformGitLab {
		t.Fatalf("expected gitlab platform, got %s", bundle.Platform)
	}
	if bundle.Adapter != platform.AdapterAPI {
		t.Fatalf("expected api adapter, got %s", bundle.Adapter)
	}
	if bundle.Executors["merge_requests"] == nil {
		t.Fatal("expected gitlab merge request executor")
	}
}

func TestResolveAdminBundleBitbucket(t *testing.T) {
	state := &status.GitState{
		RemoteInfos: []git.RemoteInfo{{
			Name:    "origin",
			PushURL: "git@bitbucket.org:team/repo.git",
		}},
	}

	bundle, err := ResolveAdminBundle(state, config.PlatformConfig{BitbucketToken: "token"}, config.AdapterConfig{})
	if err != nil {
		t.Fatalf("ResolveAdminBundle returned error: %v", err)
	}
	if bundle.Platform != platform.PlatformBitbucket {
		t.Fatalf("expected bitbucket platform, got %s", bundle.Platform)
	}
	if bundle.Adapter != platform.AdapterAPI {
		t.Fatalf("expected api adapter, got %s", bundle.Adapter)
	}
	if bundle.Executors["repository_variables"] == nil {
		t.Fatal("expected bitbucket repository variables executor")
	}
}

func TestResolveAdminBundleGitLabFallsBackToBrowserStub(t *testing.T) {
	state := &status.GitState{
		RemoteInfos: []git.RemoteInfo{{
			Name:    "origin",
			PushURL: "git@gitlab.com:group/repo.git",
		}},
	}

	bundle, err := ResolveAdminBundle(state, config.PlatformConfig{}, config.AdapterConfig{
		GitLab: config.BrowserOnlyAdapterConfig{
			Browser: config.BrowserAdapterConfig{
				Enabled: true,
				Driver:  "gitlab-playwright",
			},
		},
	})
	if err != nil {
		t.Fatalf("ResolveAdminBundle returned error: %v", err)
	}
	if bundle.Platform != platform.PlatformGitLab {
		t.Fatalf("expected gitlab platform, got %s", bundle.Platform)
	}
	if bundle.Adapter != platform.AdapterBrowser {
		t.Fatalf("expected browser adapter, got %s", bundle.Adapter)
	}
	if bundle.ExecutorAdapter == nil || bundle.ExecutorAdapter.Kind() != platform.AdapterBrowser {
		t.Fatalf("expected browser executor adapter, got %+v", bundle.ExecutorAdapter)
	}
	if bundle.Executors["merge_requests"] == nil {
		t.Fatal("expected stub merge request executor")
	}
}

func TestBrowserAdapterStubDriverAnnotatesAuditMetadata(t *testing.T) {
	state := &status.GitState{
		RemoteInfos: []git.RemoteInfo{{
			Name:    "origin",
			PushURL: "git@gitlab.com:group/repo.git",
		}},
	}

	bundle, err := ResolveAdminBundle(state, config.PlatformConfig{}, config.AdapterConfig{
		GitLab: config.BrowserOnlyAdapterConfig{
			Browser: config.BrowserAdapterConfig{
				Enabled: true,
				Driver:  "playwright",
			},
		},
	})
	if err != nil {
		t.Fatalf("ResolveAdminBundle returned error: %v", err)
	}
	mutation, err := bundle.ExecutorAdapter.Mutate(context.Background(), bundle.Executors["pages"], platform.AdminMutationRequest{
		Operation:  "verify_domain",
		ResourceID: "docs.example.com",
	})
	if err != nil {
		t.Fatalf("browser-backed adapter mutate failed: %v", err)
	}
	if mutation.Metadata["browser_driver"] != "playwright" || mutation.Metadata["manual_completion_required"] != "true" {
		t.Fatalf("expected browser audit metadata, got %+v", mutation.Metadata)
	}
}

func TestResolveAdminBundleBitbucketFallsBackToBrowserStub(t *testing.T) {
	state := &status.GitState{
		RemoteInfos: []git.RemoteInfo{{
			Name:    "origin",
			PushURL: "git@bitbucket.org:team/repo.git",
		}},
	}

	bundle, err := ResolveAdminBundle(state, config.PlatformConfig{}, config.AdapterConfig{
		Bitbucket: config.BrowserOnlyAdapterConfig{
			Browser: config.BrowserAdapterConfig{
				Enabled: true,
				Driver:  "bitbucket-playwright",
			},
		},
	})
	if err != nil {
		t.Fatalf("ResolveAdminBundle returned error: %v", err)
	}
	if bundle.Platform != platform.PlatformBitbucket {
		t.Fatalf("expected bitbucket platform, got %s", bundle.Platform)
	}
	if bundle.Adapter != platform.AdapterBrowser {
		t.Fatalf("expected browser adapter, got %s", bundle.Adapter)
	}
	if bundle.ExecutorAdapter == nil || bundle.ExecutorAdapter.Kind() != platform.AdapterBrowser {
		t.Fatalf("expected browser executor adapter, got %+v", bundle.ExecutorAdapter)
	}
	if bundle.Executors["repository_variables"] == nil {
		t.Fatal("expected stub repository variables executor")
	}
}

func TestResolveAdminBundleGitLabReturnsExplicitRouteFailure(t *testing.T) {
	state := &status.GitState{
		RemoteInfos: []git.RemoteInfo{{
			Name:    "origin",
			PushURL: "git@gitlab.com:group/repo.git",
		}},
	}

	_, err := ResolveAdminBundle(state, config.PlatformConfig{}, config.AdapterConfig{})
	if err == nil {
		t.Fatal("expected route failure")
	}
	if got := err.Error(); !containsAll(got, "GitLab adapter routing failed", "api -> browser", "GitLab token is not configured") {
		t.Fatalf("expected explicit gitlab route failure, got %q", got)
	}
}

func TestResolveAdminBundleBitbucketReturnsExplicitRouteFailure(t *testing.T) {
	state := &status.GitState{
		RemoteInfos: []git.RemoteInfo{{
			Name:    "origin",
			PushURL: "git@bitbucket.org:team/repo.git",
		}},
	}

	_, err := ResolveAdminBundle(state, config.PlatformConfig{}, config.AdapterConfig{})
	if err == nil {
		t.Fatal("expected route failure")
	}
	if got := err.Error(); !containsAll(got, "Bitbucket adapter routing failed", "api -> browser", "Bitbucket token is not configured") {
		t.Fatalf("expected explicit bitbucket route failure, got %q", got)
	}
}

func containsAll(text string, parts ...string) bool {
	for _, part := range parts {
		if !strings.Contains(text, part) {
			return false
		}
	}
	return true
}
