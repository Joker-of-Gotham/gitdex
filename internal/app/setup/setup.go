package setup

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/your-org/gitdex/internal/platform/config"
)

type Options struct {
	In                   io.Reader
	Out                  io.Writer
	WorkingDir           string
	UserConfigDir        string
	ConfigFile           string
	NonInteractive       bool
	DefaultOutput        string
	DefaultLogLevel      string
	DefaultProfile       string
	IdentityMode         string
	GitHubHost           string
	GitHubAppID          string
	GitHubInstallationID string
	GitHubPrivateKeyPath string
	WriteGlobal          bool
	WriteGlobalSet       bool
	WriteRepo            bool
	WriteRepoSet         bool
}

type Result struct {
	WrittenFiles []string        `json:"written_files" yaml:"written_files"`
	Config       config.Snapshot `json:"config" yaml:"config"`
	NextSteps    []string        `json:"next_steps" yaml:"next_steps"`
}

func Run(opts Options) (Result, error) {
	if opts.In == nil {
		opts.In = strings.NewReader("")
	}
	if opts.Out == nil {
		opts.Out = io.Discard
	}

	paths, err := config.ResolvePaths(config.Options{
		WorkingDir:    opts.WorkingDir,
		UserConfigDir: opts.UserConfigDir,
		ConfigFile:    opts.ConfigFile,
	})
	if err != nil {
		return Result{}, err
	}

	selected, writeGlobal, writeRepo, err := collectSelections(opts, paths.RepositoryDetected)
	if err != nil {
		return Result{}, err
	}
	if !writeGlobal && !writeRepo {
		return Result{}, fmt.Errorf("setup must write at least one config file")
	}

	writtenFiles := []string{}
	writeGlobalPath := paths.GlobalConfig
	if paths.ExplicitConfig != "" {
		writeGlobalPath = paths.ExplicitConfig
	}

	if writeGlobal {
		if err := config.WriteFile(writeGlobalPath, selected); err != nil {
			return Result{}, err
		}
		writtenFiles = append(writtenFiles, writeGlobalPath)
	}

	if writeRepo {
		if paths.RepoConfig == "" {
			return Result{}, fmt.Errorf("cannot write repo config without a repository context")
		}

		repoConfig := config.FileConfig{
			Profile: selected.Profile,
		}
		if !writeGlobal {
			repoConfig = selected
		}

		if err := config.WriteFile(paths.RepoConfig, repoConfig); err != nil {
			return Result{}, err
		}
		writtenFiles = append(writtenFiles, paths.RepoConfig)
	}

	loaded, err := config.Load(config.Options{
		WorkingDir:    opts.WorkingDir,
		UserConfigDir: opts.UserConfigDir,
		ConfigFile:    paths.ExplicitConfig,
	})
	if err != nil {
		return Result{}, err
	}

	nextSteps := []string{"gitdex doctor", "gitdex config show"}
	return Result{
		WrittenFiles: writtenFiles,
		Config:       loaded.Snapshot(),
		NextSteps:    nextSteps,
	}, nil
}

func collectSelections(opts Options, repositoryDetected bool) (config.FileConfig, bool, bool, error) {
	selected := config.FileConfig{
		Output:   firstNonEmpty(opts.DefaultOutput, "text"),
		LogLevel: firstNonEmpty(opts.DefaultLogLevel, "info"),
		Profile:  firstNonEmpty(opts.DefaultProfile, "local"),
		Identity: config.IdentityConfig{
			Mode: firstNonEmpty(opts.IdentityMode, "github-app"),
			GitHubApp: config.GitHubAppConfig{
				Host:           firstNonEmpty(opts.GitHubHost, "github.com"),
				AppID:          strings.TrimSpace(opts.GitHubAppID),
				InstallationID: strings.TrimSpace(opts.GitHubInstallationID),
				PrivateKeyPath: strings.TrimSpace(opts.GitHubPrivateKeyPath),
			},
		},
		Daemon: config.DaemonConfig{
			HealthAddress: "127.0.0.1:7777",
		},
	}
	writeGlobal := true
	writeRepo := repositoryDetected

	if opts.NonInteractive {
		if opts.WriteGlobalSet {
			writeGlobal = opts.WriteGlobal
		}
		if opts.WriteRepoSet {
			writeRepo = opts.WriteRepo
		}
		return validateSelections(selected, writeGlobal, writeRepo, repositoryDetected)
	}

	reader := bufio.NewReader(opts.In)
	var err error
	selected.Identity.Mode, err = promptString(reader, opts.Out, "Identity mode", selected.Identity.Mode)
	if err != nil {
		return config.FileConfig{}, false, false, err
	}
	selected.Identity.GitHubApp.Host, err = promptString(reader, opts.Out, "GitHub host", selected.Identity.GitHubApp.Host)
	if err != nil {
		return config.FileConfig{}, false, false, err
	}
	selected.Identity.GitHubApp.AppID, err = promptString(reader, opts.Out, "GitHub App ID", selected.Identity.GitHubApp.AppID)
	if err != nil {
		return config.FileConfig{}, false, false, err
	}
	selected.Identity.GitHubApp.InstallationID, err = promptString(reader, opts.Out, "GitHub installation ID", selected.Identity.GitHubApp.InstallationID)
	if err != nil {
		return config.FileConfig{}, false, false, err
	}
	selected.Identity.GitHubApp.PrivateKeyPath, err = promptString(reader, opts.Out, "GitHub private key path", selected.Identity.GitHubApp.PrivateKeyPath)
	if err != nil {
		return config.FileConfig{}, false, false, err
	}
	selected.Output, err = promptString(reader, opts.Out, "Default output format", selected.Output)
	if err != nil {
		return config.FileConfig{}, false, false, err
	}
	selected.LogLevel, err = promptString(reader, opts.Out, "Default log level", selected.LogLevel)
	if err != nil {
		return config.FileConfig{}, false, false, err
	}
	selected.Profile, err = promptString(reader, opts.Out, "Default profile", selected.Profile)
	if err != nil {
		return config.FileConfig{}, false, false, err
	}
	writeGlobal, err = promptBool(reader, opts.Out, "Write global config", writeGlobal)
	if err != nil {
		return config.FileConfig{}, false, false, err
	}
	if repositoryDetected {
		writeRepo, err = promptBool(reader, opts.Out, "Write repo config", writeRepo)
		if err != nil {
			return config.FileConfig{}, false, false, err
		}
	} else {
		writeRepo = false
	}

	return validateSelections(selected, writeGlobal, writeRepo, repositoryDetected)
}

func validateSelections(selected config.FileConfig, writeGlobal, writeRepo, repositoryDetected bool) (config.FileConfig, bool, bool, error) {
	if selected.Identity.Mode != "github-app" {
		return config.FileConfig{}, false, false, fmt.Errorf("unsupported identity mode %q", selected.Identity.Mode)
	}

	switch selected.Output {
	case "text", "json", "yaml":
	default:
		return config.FileConfig{}, false, false, fmt.Errorf("unsupported output format %q", selected.Output)
	}

	if writeRepo && !repositoryDetected {
		return config.FileConfig{}, false, false, fmt.Errorf("cannot write repo config without a repository context")
	}

	return selected, writeGlobal, writeRepo, nil
}

func promptString(reader *bufio.Reader, out io.Writer, label, defaultValue string) (string, error) {
	if _, err := fmt.Fprintf(out, "%s [%s]: ", label, defaultValue); err != nil {
		return "", err
	}

	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}

	value := strings.TrimSpace(line)
	if value == "" {
		return defaultValue, nil
	}
	return value, nil
}

func promptBool(reader *bufio.Reader, out io.Writer, label string, defaultValue bool) (bool, error) {
	defaultPrompt := "y/N"
	if defaultValue {
		defaultPrompt = "Y/n"
	}
	if _, err := fmt.Fprintf(out, "%s [%s]: ", label, defaultPrompt); err != nil {
		return false, err
	}

	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return false, err
	}

	value := strings.TrimSpace(strings.ToLower(line))
	switch value {
	case "":
		return defaultValue, nil
	case "y", "yes":
		return true, nil
	case "n", "no":
		return false, nil
	default:
		return false, fmt.Errorf("invalid yes/no value %q", value)
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
