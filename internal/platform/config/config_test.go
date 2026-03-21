package config_test

import (
	"os"
	"path/filepath"
	"testing"

	configpkg "github.com/your-org/gitdex/internal/platform/config"
)

func TestLoadDefaultsResolvesGlobalPathWithoutRepositoryContext(t *testing.T) {
	projectRoot := projectRoot(t)
	userConfigDir := t.TempDir()
	workingDir := t.TempDir()

	cfg, err := configpkg.Load(configpkg.Options{
		RepoRoot:      projectRoot,
		WorkingDir:    workingDir,
		UserConfigDir: userConfigDir,
	})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	expectedExamplePath := filepath.Join(projectRoot, "configs", "gitdex.example.yaml")
	if cfg.ExampleConfigPath != expectedExamplePath {
		t.Fatalf("ExampleConfigPath = %q, want %q", cfg.ExampleConfigPath, expectedExamplePath)
	}

	expectedGlobalPath := filepath.Join(userConfigDir, "gitdex", "config.yaml")
	if cfg.Paths.GlobalConfig != expectedGlobalPath {
		t.Fatalf("Paths.GlobalConfig = %q, want %q", cfg.Paths.GlobalConfig, expectedGlobalPath)
	}

	if cfg.Paths.RepositoryDetected {
		t.Fatal("expected no repository context to be detected")
	}

	if cfg.Output != "text" {
		t.Fatalf("Output = %q, want %q", cfg.Output, "text")
	}
	if cfg.LogLevel != "info" {
		t.Fatalf("LogLevel = %q, want %q", cfg.LogLevel, "info")
	}
	if cfg.Profile != "local" {
		t.Fatalf("Profile = %q, want %q", cfg.Profile, "local")
	}
	if cfg.Identity.Mode != "github-app" {
		t.Fatalf("Identity.Mode = %q, want %q", cfg.Identity.Mode, "github-app")
	}
	if cfg.Identity.GitHubApp.Host != "github.com" {
		t.Fatalf("Identity.GitHubApp.Host = %q, want %q", cfg.Identity.GitHubApp.Host, "github.com")
	}
}

func TestLoadMergesGlobalRepoEnvAndFlagLayersInOrder(t *testing.T) {
	projectRoot := projectRoot(t)
	userConfigDir := t.TempDir()
	repoRoot, nestedDir := createRepository(t)

	globalConfigPath := filepath.Join(userConfigDir, "gitdex", "config.yaml")
	writeConfigFile(t, globalConfigPath, `output: yaml
log_level: warn
profile: global
daemon:
  health_address: 127.0.0.1:9999
identity:
  github_app:
    host: ghe.example.test
`)
	writeConfigFile(t, filepath.Join(repoRoot, ".gitdex", "config.yaml"), `profile: repo
identity:
  github_app:
    app_id: repo-app
`)

	t.Setenv("GITDEX_LOG_LEVEL", "error")
	t.Setenv("GITDEX_IDENTITY_GITHUB_APP_INSTALLATION_ID", "12345")

	cfg, err := configpkg.Load(configpkg.Options{
		RepoRoot:      projectRoot,
		WorkingDir:    nestedDir,
		UserConfigDir: userConfigDir,
		Output:        "json",
		OutputSet:     true,
	})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Output != "json" {
		t.Fatalf("Output = %q, want %q", cfg.Output, "json")
	}
	if cfg.LogLevel != "error" {
		t.Fatalf("LogLevel = %q, want %q", cfg.LogLevel, "error")
	}
	if cfg.Profile != "repo" {
		t.Fatalf("Profile = %q, want %q", cfg.Profile, "repo")
	}
	if cfg.Daemon.HealthAddress != "127.0.0.1:9999" {
		t.Fatalf("Daemon.HealthAddress = %q, want %q", cfg.Daemon.HealthAddress, "127.0.0.1:9999")
	}
	if cfg.Identity.GitHubApp.Host != "ghe.example.test" {
		t.Fatalf("Identity.GitHubApp.Host = %q, want %q", cfg.Identity.GitHubApp.Host, "ghe.example.test")
	}
	if cfg.Identity.GitHubApp.AppID != "repo-app" {
		t.Fatalf("Identity.GitHubApp.AppID = %q, want %q", cfg.Identity.GitHubApp.AppID, "repo-app")
	}
	if cfg.Identity.GitHubApp.InstallationID != "12345" {
		t.Fatalf("Identity.GitHubApp.InstallationID = %q, want %q", cfg.Identity.GitHubApp.InstallationID, "12345")
	}

	if cfg.Sources["output"] != configpkg.SourceFlag {
		t.Fatalf("output source = %q, want %q", cfg.Sources["output"], configpkg.SourceFlag)
	}
	if cfg.Sources["log_level"] != configpkg.SourceEnv {
		t.Fatalf("log_level source = %q, want %q", cfg.Sources["log_level"], configpkg.SourceEnv)
	}
	if cfg.Sources["profile"] != configpkg.SourceRepo {
		t.Fatalf("profile source = %q, want %q", cfg.Sources["profile"], configpkg.SourceRepo)
	}
	if cfg.Sources["daemon.health_address"] != configpkg.SourceGlobal {
		t.Fatalf("daemon.health_address source = %q, want %q", cfg.Sources["daemon.health_address"], configpkg.SourceGlobal)
	}

	if !cfg.Paths.RepositoryDetected {
		t.Fatal("expected repository context to be detected")
	}
	if cfg.Paths.RepositoryRoot != repoRoot {
		t.Fatalf("RepositoryRoot = %q, want %q", cfg.Paths.RepositoryRoot, repoRoot)
	}
	if len(cfg.Paths.ActiveFiles) != 2 {
		t.Fatalf("len(Paths.ActiveFiles) = %d, want %d", len(cfg.Paths.ActiveFiles), 2)
	}
}

func TestLoadUsesExplicitConfigFileInsteadOfDiscoveredLayers(t *testing.T) {
	projectRoot := projectRoot(t)
	userConfigDir := t.TempDir()
	repoRoot, nestedDir := createRepository(t)

	writeConfigFile(t, filepath.Join(userConfigDir, "gitdex", "config.yaml"), `output: yaml`)
	writeConfigFile(t, filepath.Join(repoRoot, ".gitdex", "config.yaml"), `profile: repo`)
	explicitConfig := filepath.Join(t.TempDir(), "explicit.yaml")
	writeConfigFile(t, explicitConfig, `output: json
log_level: trace
profile: explicit
identity:
  github_app:
    app_id: explicit-app
`)

	cfg, err := configpkg.Load(configpkg.Options{
		RepoRoot:      projectRoot,
		WorkingDir:    nestedDir,
		UserConfigDir: userConfigDir,
		ConfigFile:    explicitConfig,
	})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Output != "json" {
		t.Fatalf("Output = %q, want %q", cfg.Output, "json")
	}
	if cfg.Profile != "explicit" {
		t.Fatalf("Profile = %q, want %q", cfg.Profile, "explicit")
	}
	if cfg.Identity.GitHubApp.AppID != "explicit-app" {
		t.Fatalf("Identity.GitHubApp.AppID = %q, want %q", cfg.Identity.GitHubApp.AppID, "explicit-app")
	}
	if cfg.Sources["output"] != configpkg.SourceExplicitConfig {
		t.Fatalf("output source = %q, want %q", cfg.Sources["output"], configpkg.SourceExplicitConfig)
	}
	if len(cfg.Paths.ActiveFiles) != 1 || cfg.Paths.ActiveFiles[0] != explicitConfig {
		t.Fatalf("unexpected active files: %#v", cfg.Paths.ActiveFiles)
	}
}

func TestLoadTracksGitAndTokenSources(t *testing.T) {
	projectRoot := projectRoot(t)
	userConfigDir := t.TempDir()
	repoRoot, nestedDir := createRepository(t)

	globalConfigPath := filepath.Join(userConfigDir, "gitdex", "config.yaml")
	writeConfigFile(t, globalConfigPath, `identity:
  mode: token
  github_pat: from-file
git:
  user_name: File User
  user_email: file@example.com
  workspace_roots:
    - D:/Code
    - E:/Repos
`)
	writeConfigFile(t, filepath.Join(repoRoot, ".gitdex", "config.yaml"), `git:
  ssh_key_path: /repo/.ssh/id_ed25519
`)
	t.Setenv("GITDEX_IDENTITY_GITHUB_PAT", "from-env")
	t.Setenv("GITDEX_GIT_USER_EMAIL", "env@example.com")

	cfg, err := configpkg.Load(configpkg.Options{
		RepoRoot:      projectRoot,
		WorkingDir:    nestedDir,
		UserConfigDir: userConfigDir,
	})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Identity.GitHubPAT != "from-env" {
		t.Fatalf("Identity.GitHubPAT = %q, want from-env", cfg.Identity.GitHubPAT)
	}
	if cfg.Git.UserName != "File User" {
		t.Fatalf("Git.UserName = %q, want File User", cfg.Git.UserName)
	}
	if cfg.Git.UserEmail != "env@example.com" {
		t.Fatalf("Git.UserEmail = %q, want env@example.com", cfg.Git.UserEmail)
	}
	if cfg.Git.SSHKeyPath != "/repo/.ssh/id_ed25519" {
		t.Fatalf("Git.SSHKeyPath = %q, want /repo/.ssh/id_ed25519", cfg.Git.SSHKeyPath)
	}
	if len(cfg.Git.WorkspaceRoots) != 2 {
		t.Fatalf("Git.WorkspaceRoots len = %d, want 2", len(cfg.Git.WorkspaceRoots))
	}
	if cfg.Git.WorkspaceRoots[0] != "D:/Code" || cfg.Git.WorkspaceRoots[1] != "E:/Repos" {
		t.Fatalf("Git.WorkspaceRoots = %#v, want configured roots", cfg.Git.WorkspaceRoots)
	}

	if cfg.Sources["identity.github_pat"] != configpkg.SourceEnv {
		t.Fatalf("identity.github_pat source = %q, want %q", cfg.Sources["identity.github_pat"], configpkg.SourceEnv)
	}
	if cfg.Sources["git.user_name"] != configpkg.SourceGlobal {
		t.Fatalf("git.user_name source = %q, want %q", cfg.Sources["git.user_name"], configpkg.SourceGlobal)
	}
	if cfg.Sources["git.user_email"] != configpkg.SourceEnv {
		t.Fatalf("git.user_email source = %q, want %q", cfg.Sources["git.user_email"], configpkg.SourceEnv)
	}
	if cfg.Sources["git.ssh_key_path"] != configpkg.SourceRepo {
		t.Fatalf("git.ssh_key_path source = %q, want %q", cfg.Sources["git.ssh_key_path"], configpkg.SourceRepo)
	}
	if cfg.Sources["git.workspace_roots"] != configpkg.SourceGlobal {
		t.Fatalf("git.workspace_roots source = %q, want %q", cfg.Sources["git.workspace_roots"], configpkg.SourceGlobal)
	}
}

func TestLoadParsesWorkspaceRootsFromEnvironmentPathList(t *testing.T) {
	projectRoot := projectRoot(t)
	userConfigDir := t.TempDir()
	workingDir := t.TempDir()

	t.Setenv("GITDEX_GIT_WORKSPACE_ROOTS", "D:\\Code;E:\\Repos,/srv/git")

	cfg, err := configpkg.Load(configpkg.Options{
		RepoRoot:      projectRoot,
		WorkingDir:    workingDir,
		UserConfigDir: userConfigDir,
	})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	want := []string{"D:\\Code", "E:\\Repos", "/srv/git"}
	if len(cfg.Git.WorkspaceRoots) != len(want) {
		t.Fatalf("Git.WorkspaceRoots len = %d, want %d (%#v)", len(cfg.Git.WorkspaceRoots), len(want), cfg.Git.WorkspaceRoots)
	}
	for i := range want {
		if cfg.Git.WorkspaceRoots[i] != want[i] {
			t.Fatalf("Git.WorkspaceRoots[%d] = %q, want %q", i, cfg.Git.WorkspaceRoots[i], want[i])
		}
	}
	if cfg.Sources["git.workspace_roots"] != configpkg.SourceEnv {
		t.Fatalf("git.workspace_roots source = %q, want %q", cfg.Sources["git.workspace_roots"], configpkg.SourceEnv)
	}
}

func TestLoadUsesConfigPathFromEnvironmentWhenExplicitPathIsUnset(t *testing.T) {
	projectRoot := projectRoot(t)
	workingDir := t.TempDir()
	configFile := filepath.Join(t.TempDir(), "env-config.yaml")
	writeConfigFile(t, configFile, `output: yaml
log_level: trace
profile: env-file
`)
	t.Setenv("GITDEX_CONFIG", configFile)

	cfg, err := configpkg.Load(configpkg.Options{
		RepoRoot:      projectRoot,
		WorkingDir:    workingDir,
		UserConfigDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.ConfigFile != configFile {
		t.Fatalf("ConfigFile = %q, want %q", cfg.ConfigFile, configFile)
	}
	if cfg.Output != "yaml" {
		t.Fatalf("Output = %q, want %q", cfg.Output, "yaml")
	}
	if cfg.LogLevel != "trace" {
		t.Fatalf("LogLevel = %q, want %q", cfg.LogLevel, "trace")
	}
	if cfg.Profile != "env-file" {
		t.Fatalf("Profile = %q, want %q", cfg.Profile, "env-file")
	}
}

func TestLoadReturnsErrorForMissingExplicitConfigFile(t *testing.T) {
	_, err := configpkg.Load(configpkg.Options{
		WorkingDir:    t.TempDir(),
		UserConfigDir: t.TempDir(),
		ConfigFile:    filepath.Join(t.TempDir(), "missing.yaml"),
	})
	if err == nil {
		t.Fatal("expected error for missing config file")
	}
}

func TestWriteFilePersistsConfigThatCanBeReloaded(t *testing.T) {
	projectRoot := projectRoot(t)
	userConfigDir := t.TempDir()
	repoRoot, nestedDir := createRepository(t)
	globalPath := filepath.Join(userConfigDir, "gitdex", "config.yaml")
	repoPath := filepath.Join(repoRoot, ".gitdex", "config.yaml")

	if err := configpkg.WriteFile(globalPath, configpkg.FileConfig{
		Output:   "json",
		LogLevel: "debug",
		Profile:  "global",
		Identity: configpkg.IdentityConfig{
			Mode: "github-app",
			GitHubApp: configpkg.GitHubAppConfig{
				Host:  "github.com",
				AppID: "1",
			},
		},
	}); err != nil {
		t.Fatalf("WriteFile(global) returned error: %v", err)
	}
	if err := configpkg.WriteFile(repoPath, configpkg.FileConfig{
		Profile: "repo",
	}); err != nil {
		t.Fatalf("WriteFile(repo) returned error: %v", err)
	}

	cfg, err := configpkg.Load(configpkg.Options{
		RepoRoot:      projectRoot,
		WorkingDir:    nestedDir,
		UserConfigDir: userConfigDir,
	})
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Output != "json" {
		t.Fatalf("Output = %q, want %q", cfg.Output, "json")
	}
	if cfg.Profile != "repo" {
		t.Fatalf("Profile = %q, want %q", cfg.Profile, "repo")
	}
	if cfg.Identity.GitHubApp.AppID != "1" {
		t.Fatalf("Identity.GitHubApp.AppID = %q, want %q", cfg.Identity.GitHubApp.AppID, "1")
	}
}

func TestSnapshotRedactsSecrets(t *testing.T) {
	cfg := configpkg.Config{
		FileConfig: configpkg.FileConfig{
			Identity: configpkg.IdentityConfig{
				Mode:      "token",
				GitHubPAT: "ghp-secret",
			},
			LLM: configpkg.LLMConfig{
				APIKey: "sk-secret",
			},
			Storage: configpkg.StorageConfig{
				DSN: "postgres://secret",
			},
		},
		Sources: map[string]configpkg.ValueSource{},
	}

	snapshot := cfg.Snapshot()
	if snapshot.Config.Identity.GitHubPAT != "***" {
		t.Fatalf("Snapshot GitHubPAT = %q, want ***", snapshot.Config.Identity.GitHubPAT)
	}
	if snapshot.Config.LLM.APIKey != "***" {
		t.Fatalf("Snapshot APIKey = %q, want ***", snapshot.Config.LLM.APIKey)
	}
	if snapshot.Config.Storage.DSN != "***" {
		t.Fatalf("Snapshot DSN = %q, want ***", snapshot.Config.Storage.DSN)
	}
}

func TestResolveRepoRootReturnsErrorOutsideProject(t *testing.T) {
	_, err := configpkg.ResolveRepoRoot(t.TempDir())
	if err == nil {
		t.Fatal("expected error outside repository")
	}
}

func TestResolveRepositoryRootFindsAncestorGitDirectory(t *testing.T) {
	repoRoot, nestedDir := createRepository(t)

	got, err := configpkg.ResolveRepositoryRoot(nestedDir)
	if err != nil {
		t.Fatalf("ResolveRepositoryRoot returned error: %v", err)
	}
	if got != repoRoot {
		t.Fatalf("ResolveRepositoryRoot = %q, want %q", got, repoRoot)
	}
}

func projectRoot(t *testing.T) string {
	t.Helper()

	root, err := filepath.Abs("../../../")
	if err != nil {
		t.Fatalf("filepath.Abs failed: %v", err)
	}
	return root
}

func createRepository(t *testing.T) (string, string) {
	t.Helper()

	repoRoot := t.TempDir()
	if err := os.Mkdir(filepath.Join(repoRoot, ".git"), 0o755); err != nil {
		t.Fatalf("os.Mkdir(.git) failed: %v", err)
	}

	nestedDir := filepath.Join(repoRoot, "nested", "deeper")
	if err := os.MkdirAll(nestedDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll failed: %v", err)
	}

	return repoRoot, nestedDir
}

func writeConfigFile(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("os.MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("os.WriteFile failed: %v", err)
	}
}
