package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
	"go.yaml.in/yaml/v3"
)

const envPrefix = "GITDEX"

const (
	envConfigFile    = envPrefix + "_CONFIG"
	envGlobalConfig  = envPrefix + "_GLOBAL_CONFIG"
	envRepoConfig    = envPrefix + "_REPO_CONFIG"
	envUserConfigDir = envPrefix + "_USER_CONFIG_DIR"
	envOutput        = envPrefix + "_OUTPUT"
	envLogLevel      = envPrefix + "_LOG_LEVEL"
	envProfile       = envPrefix + "_PROFILE"
	envHealthAddress = envPrefix + "_DAEMON_HEALTH_ADDRESS"
	envIdentityMode  = envPrefix + "_IDENTITY_MODE"
	envGitHubPAT     = envPrefix + "_IDENTITY_GITHUB_PAT"
	envGitHubHost    = envPrefix + "_IDENTITY_GITHUB_APP_HOST"
	envGitHubAppID   = envPrefix + "_IDENTITY_GITHUB_APP_APP_ID"
	envGitHubInstID  = envPrefix + "_IDENTITY_GITHUB_APP_INSTALLATION_ID"
	envGitHubKeyPath = envPrefix + "_IDENTITY_GITHUB_APP_PRIVATE_KEY_PATH"
	envGitUserName   = envPrefix + "_GIT_USER_NAME"
	envGitUserEmail  = envPrefix + "_GIT_USER_EMAIL"
	envGitSSHKeyPath = envPrefix + "_GIT_SSH_KEY_PATH"
	envGitRoots      = envPrefix + "_GIT_WORKSPACE_ROOTS"
	envLLMProvider   = envPrefix + "_LLM_PROVIDER"
	envLLMModel      = envPrefix + "_LLM_MODEL"
	envLLMAPIKey     = envPrefix + "_LLM_API_KEY"
	envLLMEndpoint   = envPrefix + "_LLM_ENDPOINT"
	envStorageType   = envPrefix + "_STORAGE_TYPE"
	envStorageDSN    = envPrefix + "_STORAGE_DSN"
)

type ValueSource string

const (
	SourceDefault        ValueSource = "default"
	SourceGlobal         ValueSource = "global"
	SourceRepo           ValueSource = "repo"
	SourceEnv            ValueSource = "env"
	SourceFlag           ValueSource = "flag"
	SourceExplicitConfig ValueSource = "config-file"
)

type LLMConfig struct {
	Provider string `json:"provider,omitempty" yaml:"provider,omitempty"`
	Model    string `json:"model,omitempty" yaml:"model,omitempty"`
	APIKey   string `json:"api_key,omitempty" yaml:"api_key,omitempty"`
	Endpoint string `json:"endpoint,omitempty" yaml:"endpoint,omitempty"`
}

type StorageConfig struct {
	Type         string `json:"type,omitempty" yaml:"type,omitempty"`
	DSN          string `json:"dsn,omitempty" yaml:"dsn,omitempty"`
	MaxOpenConns int    `json:"max_open_conns,omitempty" yaml:"max_open_conns,omitempty"`
	MaxIdleConns int    `json:"max_idle_conns,omitempty" yaml:"max_idle_conns,omitempty"`
	AutoMigrate  bool   `json:"auto_migrate,omitempty" yaml:"auto_migrate,omitempty"`
}

type FileConfig struct {
	Output   string         `json:"output,omitempty" yaml:"output,omitempty"`
	LogLevel string         `json:"log_level,omitempty" yaml:"log_level,omitempty"`
	Profile  string         `json:"profile,omitempty" yaml:"profile,omitempty"`
	Identity IdentityConfig `json:"identity,omitempty" yaml:"identity,omitempty"`
	Daemon   DaemonConfig   `json:"daemon,omitempty" yaml:"daemon,omitempty"`
	LLM      LLMConfig      `json:"llm,omitempty" yaml:"llm,omitempty"`
	Storage  StorageConfig  `json:"storage,omitempty" yaml:"storage,omitempty"`
	Git      GitConfig      `json:"git,omitempty" yaml:"git,omitempty"`
}

type Config struct {
	FileConfig
	ExampleConfigPath string                 `json:"-" yaml:"-"`
	ConfigFile        string                 `json:"-" yaml:"-"`
	Paths             ConfigPaths            `json:"-" yaml:"-"`
	Sources           map[string]ValueSource `json:"-" yaml:"-"`
}

type Snapshot struct {
	Config  FileConfig             `json:"config" yaml:"config"`
	Paths   ConfigPaths            `json:"paths" yaml:"paths"`
	Sources map[string]ValueSource `json:"sources" yaml:"sources"`
}

type ConfigPaths struct {
	WorkingDir         string   `json:"working_dir" yaml:"working_dir"`
	ProjectRoot        string   `json:"project_root,omitempty" yaml:"project_root,omitempty"`
	GlobalConfig       string   `json:"global_config" yaml:"global_config"`
	RepoConfig         string   `json:"repo_config,omitempty" yaml:"repo_config,omitempty"`
	ExplicitConfig     string   `json:"explicit_config,omitempty" yaml:"explicit_config,omitempty"`
	ActiveFiles        []string `json:"active_files" yaml:"active_files"`
	RepositoryRoot     string   `json:"repository_root,omitempty" yaml:"repository_root,omitempty"`
	RepositoryDetected bool     `json:"repository_detected" yaml:"repository_detected"`
}

type IdentityConfig struct {
	Mode      string          `json:"mode,omitempty" yaml:"mode,omitempty"`
	GitHubPAT string          `json:"github_pat,omitempty" yaml:"github_pat,omitempty"`
	GitHubApp GitHubAppConfig `json:"github_app,omitempty" yaml:"github_app,omitempty"`
}

type GitConfig struct {
	UserName       string   `json:"user_name,omitempty" yaml:"user_name,omitempty"`
	UserEmail      string   `json:"user_email,omitempty" yaml:"user_email,omitempty"`
	SSHKeyPath     string   `json:"ssh_key_path,omitempty" yaml:"ssh_key_path,omitempty"`
	WorkspaceRoots []string `json:"workspace_roots,omitempty" yaml:"workspace_roots,omitempty"`
}

type GitHubAppConfig struct {
	Host           string `json:"host,omitempty" yaml:"host,omitempty"`
	AppID          string `json:"app_id,omitempty" yaml:"app_id,omitempty"`
	InstallationID string `json:"installation_id,omitempty" yaml:"installation_id,omitempty"`
	PrivateKeyPath string `json:"private_key_path,omitempty" yaml:"private_key_path,omitempty"`
}

type DaemonConfig struct {
	HealthAddress string `json:"health_address,omitempty" yaml:"health_address,omitempty"`
}

type Options struct {
	RepoRoot         string
	WorkingDir       string
	ConfigFile       string
	GlobalConfigPath string
	RepoConfigPath   string
	UserConfigDir    string
	Output           string
	OutputSet        bool
	LogLevel         string
	LogLevelSet      bool
	Profile          string
	ProfileSet       bool
}

func ResolveRepoRoot(start string) (string, error) {
	current, err := resolveWorkingDir(start)
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(current, "go.mod")); err == nil {
			return current, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("go.mod not found from %q upward", start)
		}
		current = parent
	}
}

func ResolveRepositoryRoot(start string) (string, error) {
	current, err := resolveWorkingDir(start)
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(current, ".git")); err == nil {
			return current, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf(".git not found from %q upward", start)
		}
		current = parent
	}
}

func ResolveGlobalConfigPath(userConfigDir string) (string, error) {
	if userConfigDir == "" {
		userConfigDir = os.Getenv(envUserConfigDir)
	}
	if userConfigDir == "" {
		resolved, err := os.UserConfigDir()
		if err != nil {
			return "", fmt.Errorf("resolve user config dir: %w", err)
		}
		userConfigDir = resolved
	}

	resolved, err := filepath.Abs(filepath.Join(userConfigDir, "gitdex", "config.yaml"))
	if err != nil {
		return "", fmt.Errorf("resolve global config path: %w", err)
	}
	return resolved, nil
}

func ResolveRepoConfigPath(repoRoot string) string {
	if repoRoot == "" {
		return ""
	}
	return filepath.Join(repoRoot, ".gitdex", "config.yaml")
}

func ResolvePaths(opts Options) (ConfigPaths, error) {
	workingDir, err := resolveWorkingDir(opts.WorkingDir)
	if err != nil {
		return ConfigPaths{}, err
	}

	projectRoot := ""
	projectSearchStart := opts.RepoRoot
	if projectSearchStart == "" {
		projectSearchStart = workingDir
	}
	if root, err := ResolveRepoRoot(projectSearchStart); err == nil {
		projectRoot = root
	}

	globalConfigPath := opts.GlobalConfigPath
	if globalConfigPath == "" {
		globalConfigPath = os.Getenv(envGlobalConfig)
	}
	if globalConfigPath == "" {
		globalConfigPath, err = ResolveGlobalConfigPath(opts.UserConfigDir)
		if err != nil {
			return ConfigPaths{}, err
		}
	} else {
		globalConfigPath, err = filepath.Abs(globalConfigPath)
		if err != nil {
			return ConfigPaths{}, fmt.Errorf("resolve global config path: %w", err)
		}
	}

	repositoryRoot := ""
	if root, err := ResolveRepositoryRoot(workingDir); err == nil {
		repositoryRoot = root
	}

	repoConfigPath := opts.RepoConfigPath
	if repoConfigPath == "" {
		repoConfigPath = os.Getenv(envRepoConfig)
	}
	if repoConfigPath == "" {
		repoConfigPath = ResolveRepoConfigPath(repositoryRoot)
	}
	if repoConfigPath != "" {
		repoConfigPath, err = filepath.Abs(repoConfigPath)
		if err != nil {
			return ConfigPaths{}, fmt.Errorf("resolve repo config path: %w", err)
		}
	}

	explicitConfig := opts.ConfigFile
	if explicitConfig == "" {
		explicitConfig = os.Getenv(envConfigFile)
	}
	if explicitConfig != "" {
		explicitConfig, err = filepath.Abs(explicitConfig)
		if err != nil {
			return ConfigPaths{}, fmt.Errorf("resolve explicit config path: %w", err)
		}
	}

	return ConfigPaths{
		WorkingDir:         workingDir,
		ProjectRoot:        projectRoot,
		GlobalConfig:       globalConfigPath,
		RepoConfig:         repoConfigPath,
		ExplicitConfig:     explicitConfig,
		ActiveFiles:        []string{},
		RepositoryRoot:     repositoryRoot,
		RepositoryDetected: repositoryRoot != "",
	}, nil
}

func Load(opts Options) (Config, error) {
	paths, err := ResolvePaths(opts)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		FileConfig: defaultFileConfig(),
		Paths:      paths,
		Sources:    defaultSources(),
	}
	if paths.ProjectRoot != "" {
		cfg.ExampleConfigPath = filepath.Join(paths.ProjectRoot, "configs", "gitdex.example.yaml")
	}

	if paths.ExplicitConfig != "" {
		layer, err := readConfigFile(paths.ExplicitConfig)
		if err != nil {
			return Config{}, err
		}

		applyViperLayer(&cfg, layer, SourceExplicitConfig)
		cfg.Paths.ActiveFiles = append(cfg.Paths.ActiveFiles, paths.ExplicitConfig)
	} else {
		if fileExists(paths.GlobalConfig) {
			layer, err := readConfigFile(paths.GlobalConfig)
			if err != nil {
				return Config{}, err
			}

			applyViperLayer(&cfg, layer, SourceGlobal)
			cfg.Paths.ActiveFiles = append(cfg.Paths.ActiveFiles, paths.GlobalConfig)
		}

		if paths.RepoConfig != "" && fileExists(paths.RepoConfig) {
			layer, err := readConfigFile(paths.RepoConfig)
			if err != nil {
				return Config{}, err
			}

			applyViperLayer(&cfg, layer, SourceRepo)
			cfg.Paths.ActiveFiles = append(cfg.Paths.ActiveFiles, paths.RepoConfig)
		}
	}

	applyEnvLayer(&cfg)
	applyFlagOverrides(&cfg, opts)

	if len(cfg.Paths.ActiveFiles) > 0 {
		cfg.ConfigFile = cfg.Paths.ActiveFiles[len(cfg.Paths.ActiveFiles)-1]
	}

	return cfg, nil
}

func ReadFile(path string) (FileConfig, error) {
	resolved, err := filepath.Abs(path)
	if err != nil {
		return FileConfig{}, fmt.Errorf("resolve config read path: %w", err)
	}
	data, err := os.ReadFile(resolved)
	if err != nil {
		return FileConfig{}, fmt.Errorf("read config file: %w", err)
	}
	var fc FileConfig
	if err := yaml.Unmarshal(data, &fc); err != nil {
		return FileConfig{}, fmt.Errorf("parse config file: %w", err)
	}
	return fc, nil
}

func WriteFile(path string, fileConfig FileConfig) error {
	resolved, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve config write path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(resolved), 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	content, err := yaml.Marshal(fileConfig)
	if err != nil {
		return fmt.Errorf("marshal config file: %w", err)
	}
	if err := os.WriteFile(resolved, content, 0o600); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	return nil
}

const redacted = "***"

func (c Config) Snapshot() Snapshot {
	sources := make(map[string]ValueSource, len(c.Sources))
	for key, value := range c.Sources {
		sources[key] = value
	}

	activeFiles := append([]string(nil), c.Paths.ActiveFiles...)

	fc := c.FileConfig
	if fc.LLM.APIKey != "" {
		fc.LLM.APIKey = redacted
	}
	if fc.Identity.GitHubPAT != "" {
		fc.Identity.GitHubPAT = redacted
	}
	if fc.Storage.DSN != "" {
		fc.Storage.DSN = redacted
	}

	return Snapshot{
		Config: fc,
		Paths: ConfigPaths{
			WorkingDir:         c.Paths.WorkingDir,
			ProjectRoot:        c.Paths.ProjectRoot,
			GlobalConfig:       c.Paths.GlobalConfig,
			RepoConfig:         c.Paths.RepoConfig,
			ExplicitConfig:     c.Paths.ExplicitConfig,
			ActiveFiles:        activeFiles,
			RepositoryRoot:     c.Paths.RepositoryRoot,
			RepositoryDetected: c.Paths.RepositoryDetected,
		},
		Sources: sources,
	}
}

func defaultFileConfig() FileConfig {
	return FileConfig{
		Output:   "text",
		LogLevel: "info",
		Profile:  "local",
		Identity: IdentityConfig{
			Mode: "github-app",
			GitHubApp: GitHubAppConfig{
				Host: "github.com",
			},
		},
		Daemon: DaemonConfig{
			HealthAddress: "127.0.0.1:7777",
		},
		Storage: StorageConfig{
			Type: "memory",
		},
	}
}

func defaultSources() map[string]ValueSource {
	return map[string]ValueSource{
		"output":                               SourceDefault,
		"log_level":                            SourceDefault,
		"profile":                              SourceDefault,
		"daemon.health_address":                SourceDefault,
		"identity.mode":                        SourceDefault,
		"identity.github_pat":                  SourceDefault,
		"identity.github_app.host":             SourceDefault,
		"identity.github_app.app_id":           SourceDefault,
		"identity.github_app.installation_id":  SourceDefault,
		"identity.github_app.private_key_path": SourceDefault,
		"git.user_name":                        SourceDefault,
		"git.user_email":                       SourceDefault,
		"git.ssh_key_path":                     SourceDefault,
		"git.workspace_roots":                  SourceDefault,
		"llm.provider":                         SourceDefault,
		"llm.model":                            SourceDefault,
		"llm.api_key":                          SourceDefault,
		"llm.endpoint":                         SourceDefault,
		"storage.type":                         SourceDefault,
		"storage.dsn":                          SourceDefault,
	}
}

func resolveWorkingDir(start string) (string, error) {
	if start == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("resolve working directory: %w", err)
		}
		start = wd
	}

	current, err := filepath.Abs(start)
	if err != nil {
		return "", fmt.Errorf("resolve working directory from %q: %w", start, err)
	}
	return current, nil
}

func readConfigFile(path string) (*viper.Viper, error) {
	v := viper.New()
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}
	return v, nil
}

func applyViperLayer(cfg *Config, v *viper.Viper, source ValueSource) {
	if v.InConfig("output") {
		cfg.Output = v.GetString("output")
		cfg.Sources["output"] = source
	}
	if v.InConfig("log_level") {
		cfg.LogLevel = v.GetString("log_level")
		cfg.Sources["log_level"] = source
	}
	if v.InConfig("profile") {
		cfg.Profile = v.GetString("profile")
		cfg.Sources["profile"] = source
	}
	if v.InConfig("daemon.health_address") {
		cfg.Daemon.HealthAddress = v.GetString("daemon.health_address")
		cfg.Sources["daemon.health_address"] = source
	}
	if v.InConfig("identity.mode") {
		cfg.Identity.Mode = v.GetString("identity.mode")
		cfg.Sources["identity.mode"] = source
	}
	if v.InConfig("identity.github_pat") {
		cfg.Identity.GitHubPAT = v.GetString("identity.github_pat")
		cfg.Sources["identity.github_pat"] = source
	}
	if v.InConfig("identity.github_app.host") {
		cfg.Identity.GitHubApp.Host = v.GetString("identity.github_app.host")
		cfg.Sources["identity.github_app.host"] = source
	}
	if v.InConfig("identity.github_app.app_id") {
		cfg.Identity.GitHubApp.AppID = v.GetString("identity.github_app.app_id")
		cfg.Sources["identity.github_app.app_id"] = source
	}
	if v.InConfig("identity.github_app.installation_id") {
		cfg.Identity.GitHubApp.InstallationID = v.GetString("identity.github_app.installation_id")
		cfg.Sources["identity.github_app.installation_id"] = source
	}
	if v.InConfig("identity.github_app.private_key_path") {
		cfg.Identity.GitHubApp.PrivateKeyPath = v.GetString("identity.github_app.private_key_path")
		cfg.Sources["identity.github_app.private_key_path"] = source
	}
	if v.InConfig("llm.provider") {
		cfg.LLM.Provider = v.GetString("llm.provider")
		cfg.Sources["llm.provider"] = source
	}
	if v.InConfig("llm.model") {
		cfg.LLM.Model = v.GetString("llm.model")
		cfg.Sources["llm.model"] = source
	}
	if v.InConfig("llm.api_key") {
		cfg.LLM.APIKey = v.GetString("llm.api_key")
		cfg.Sources["llm.api_key"] = source
	}
	if v.InConfig("llm.endpoint") {
		cfg.LLM.Endpoint = v.GetString("llm.endpoint")
		cfg.Sources["llm.endpoint"] = source
	}
	if v.InConfig("storage.type") {
		cfg.Storage.Type = v.GetString("storage.type")
		cfg.Sources["storage.type"] = source
	}
	if v.InConfig("storage.dsn") {
		cfg.Storage.DSN = v.GetString("storage.dsn")
		cfg.Sources["storage.dsn"] = source
	}
	if v.InConfig("git.user_name") {
		cfg.Git.UserName = v.GetString("git.user_name")
		cfg.Sources["git.user_name"] = source
	}
	if v.InConfig("git.user_email") {
		cfg.Git.UserEmail = v.GetString("git.user_email")
		cfg.Sources["git.user_email"] = source
	}
	if v.InConfig("git.ssh_key_path") {
		cfg.Git.SSHKeyPath = v.GetString("git.ssh_key_path")
		cfg.Sources["git.ssh_key_path"] = source
	}
	if v.InConfig("git.workspace_roots") {
		cfg.Git.WorkspaceRoots = readPathList(v, "git.workspace_roots")
		cfg.Sources["git.workspace_roots"] = source
	}
	if v.InConfig("storage.max_open_conns") {
		cfg.Storage.MaxOpenConns = v.GetInt("storage.max_open_conns")
	}
	if v.InConfig("storage.max_idle_conns") {
		cfg.Storage.MaxIdleConns = v.GetInt("storage.max_idle_conns")
	}
	if v.InConfig("storage.auto_migrate") {
		cfg.Storage.AutoMigrate = v.GetBool("storage.auto_migrate")
	}
}

func applyEnvLayer(cfg *Config) {
	applyEnvString(envOutput, func(value string) {
		cfg.Output = value
		cfg.Sources["output"] = SourceEnv
	})
	applyEnvString(envLogLevel, func(value string) {
		cfg.LogLevel = value
		cfg.Sources["log_level"] = SourceEnv
	})
	applyEnvString(envProfile, func(value string) {
		cfg.Profile = value
		cfg.Sources["profile"] = SourceEnv
	})
	applyEnvString(envHealthAddress, func(value string) {
		cfg.Daemon.HealthAddress = value
		cfg.Sources["daemon.health_address"] = SourceEnv
	})
	applyEnvString(envIdentityMode, func(value string) {
		cfg.Identity.Mode = value
		cfg.Sources["identity.mode"] = SourceEnv
	})
	applyEnvString(envGitHubPAT, func(value string) {
		cfg.Identity.GitHubPAT = value
		cfg.Sources["identity.github_pat"] = SourceEnv
	})
	applyEnvString(envGitHubHost, func(value string) {
		cfg.Identity.GitHubApp.Host = value
		cfg.Sources["identity.github_app.host"] = SourceEnv
	})
	applyEnvString(envGitHubAppID, func(value string) {
		cfg.Identity.GitHubApp.AppID = value
		cfg.Sources["identity.github_app.app_id"] = SourceEnv
	})
	applyEnvString(envGitHubInstID, func(value string) {
		cfg.Identity.GitHubApp.InstallationID = value
		cfg.Sources["identity.github_app.installation_id"] = SourceEnv
	})
	applyEnvString(envGitHubKeyPath, func(value string) {
		cfg.Identity.GitHubApp.PrivateKeyPath = value
		cfg.Sources["identity.github_app.private_key_path"] = SourceEnv
	})
	applyEnvString(envLLMProvider, func(value string) {
		cfg.LLM.Provider = value
		cfg.Sources["llm.provider"] = SourceEnv
	})
	applyEnvString(envLLMModel, func(value string) {
		cfg.LLM.Model = value
		cfg.Sources["llm.model"] = SourceEnv
	})
	applyEnvString(envLLMAPIKey, func(value string) {
		cfg.LLM.APIKey = value
		cfg.Sources["llm.api_key"] = SourceEnv
	})
	applyEnvString(envLLMEndpoint, func(value string) {
		cfg.LLM.Endpoint = value
		cfg.Sources["llm.endpoint"] = SourceEnv
	})
	applyEnvString(envStorageType, func(value string) {
		cfg.Storage.Type = value
		cfg.Sources["storage.type"] = SourceEnv
	})
	applyEnvString(envStorageDSN, func(value string) {
		cfg.Storage.DSN = value
		cfg.Sources["storage.dsn"] = SourceEnv
	})
	applyEnvString(envGitUserName, func(value string) {
		cfg.Git.UserName = value
		cfg.Sources["git.user_name"] = SourceEnv
	})
	applyEnvString(envGitUserEmail, func(value string) {
		cfg.Git.UserEmail = value
		cfg.Sources["git.user_email"] = SourceEnv
	})
	applyEnvString(envGitSSHKeyPath, func(value string) {
		cfg.Git.SSHKeyPath = value
		cfg.Sources["git.ssh_key_path"] = SourceEnv
	})
	applyEnvString(envGitRoots, func(value string) {
		cfg.Git.WorkspaceRoots = splitPathList(value)
		cfg.Sources["git.workspace_roots"] = SourceEnv
	})
}

func applyFlagOverrides(cfg *Config, opts Options) {
	if opts.OutputSet {
		cfg.Output = opts.Output
		cfg.Sources["output"] = SourceFlag
	}
	if opts.LogLevelSet {
		cfg.LogLevel = opts.LogLevel
		cfg.Sources["log_level"] = SourceFlag
	}
	if opts.ProfileSet {
		cfg.Profile = opts.Profile
		cfg.Sources["profile"] = SourceFlag
	}
}

func applyEnvString(key string, assign func(string)) {
	if value, ok := os.LookupEnv(key); ok {
		assign(value)
	}
}

func readPathList(v *viper.Viper, key string) []string {
	values := v.GetStringSlice(key)
	if len(values) == 0 {
		values = splitPathList(v.GetString(key))
	}
	return normalizePathList(values)
}

func splitPathList(raw string) []string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	splitter := func(r rune) bool {
		return r == rune(filepath.ListSeparator) || r == ',' || r == '\n' || r == '\r'
	}
	return normalizePathList(strings.FieldsFunc(trimmed, splitter))
}

func normalizePathList(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, value)
	}
	return out
}

func fileExists(path string) bool {
	if path == "" {
		return false
	}

	_, err := os.Stat(path)
	return err == nil
}
