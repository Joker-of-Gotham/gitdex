package bootstrap

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/your-org/gitdex/internal/platform/config"
	"github.com/your-org/gitdex/internal/storage"
)

type App struct {
	Config          config.Config
	RepoRoot        string
	Version         string
	StorageProvider storage.StorageProvider
}

type Options struct {
	RepoRoot      string
	WorkingDir    string
	UserConfigDir string
	ConfigFile    string
	Output        string
	OutputSet     bool
	LogLevel      string
	LogLevelSet   bool
	Profile       string
	ProfileSet    bool
	Version       string
}

func Load(opts Options) (App, error) {
	repoRoot := strings.TrimSpace(opts.RepoRoot)
	if repoRoot != "" {
		if abs, err := filepath.Abs(repoRoot); err == nil {
			repoRoot = abs
		}
	} else {
		searchRoot := opts.WorkingDir
		if root, err := config.ResolveRepositoryRoot(searchRoot); err == nil {
			repoRoot = root
		}
	}

	cfg, err := config.Load(config.Options{
		RepoRoot:      repoRoot,
		WorkingDir:    opts.WorkingDir,
		UserConfigDir: opts.UserConfigDir,
		ConfigFile:    opts.ConfigFile,
		Output:        opts.Output,
		OutputSet:     opts.OutputSet,
		LogLevel:      opts.LogLevel,
		LogLevelSet:   opts.LogLevelSet,
		Profile:       opts.Profile,
		ProfileSet:    opts.ProfileSet,
	})
	if err != nil {
		return App{}, err
	}

	storageCfg := storage.Config{
		Type:         storage.BackendType(cfg.Storage.Type),
		DSN:          cfg.Storage.DSN,
		MaxOpenConns: cfg.Storage.MaxOpenConns,
		MaxIdleConns: cfg.Storage.MaxIdleConns,
		AutoMigrate:  cfg.Storage.AutoMigrate,
	}
	storageCfg = storageCfg.Normalized(resolveStorageBaseDir(cfg))
	cfg.Storage.Type = string(storageCfg.Type)
	cfg.Storage.DSN = storageCfg.DSN
	provider, err := storage.NewProvider(storageCfg)
	if err != nil {
		return App{}, fmt.Errorf("initialize storage: %w", err)
	}
	if storageCfg.AutoMigrate {
		if err := provider.Migrate(context.Background()); err != nil {
			_ = provider.Close()
			return App{}, fmt.Errorf("run storage migrations: %w", err)
		}
	}

	return App{
		Config:          cfg,
		RepoRoot:        repoRoot,
		Version:         opts.Version,
		StorageProvider: provider,
	}, nil
}

func resolveStorageBaseDir(cfg config.Config) string {
	candidates := []string{
		cfg.ConfigFile,
		cfg.Paths.GlobalConfig,
		cfg.Paths.WorkingDir,
	}
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if filepath.Ext(candidate) == "" {
			return candidate
		}
		return filepath.Dir(candidate)
	}
	return "."
}
