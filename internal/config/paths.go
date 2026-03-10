package config

import (
	"os"
	"path/filepath"
	"runtime"
)

const (
	AppName             = "gitdex"
	LegacyAppName       = "gitmanual"
	ProjectConfigName   = ".gitdexrc"
	LegacyProjectConfig = ".gitmanualrc"
)

func GlobalConfigDir() (string, error) {
	return configDirForApp(AppName)
}

func LegacyGlobalConfigDir() (string, error) {
	return configDirForApp(LegacyAppName)
}

func GlobalConfigPath() (string, error) {
	dir, err := GlobalConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

func LegacyGlobalConfigPath() (string, error) {
	dir, err := LegacyGlobalConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

func configDirForApp(app string) (string, error) {
	if runtime.GOOS == "windows" {
		cfg, err := os.UserConfigDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(cfg, app), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", app), nil
}

func existingProjectConfigFiles() []string {
	paths := []string{LegacyProjectConfig, ProjectConfigName}
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			out = append(out, path)
		}
	}
	return out
}
