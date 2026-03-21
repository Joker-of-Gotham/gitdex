package storage

import (
	"path/filepath"
	"strings"
)

const (
	defaultSQLiteFile = "gitdex.sqlite"
	defaultBBoltFile  = "gitdex.db"
)

// DefaultDSN returns the default on-disk path for file-backed storage engines.
func DefaultDSN(baseDir string, backend BackendType) string {
	baseDir = strings.TrimSpace(baseDir)
	if baseDir == "" {
		baseDir = "."
	}
	baseDir = filepath.Clean(baseDir)

	switch backend {
	case BackendSQLite:
		return filepath.Join(baseDir, defaultSQLiteFile)
	case BackendBBolt:
		return filepath.Join(baseDir, defaultBBoltFile)
	default:
		return ""
	}
}

// Normalized fills in implicit defaults needed to create a working provider.
func (c Config) Normalized(baseDir string) Config {
	normalized := c
	if normalized.Type == "" {
		normalized.Type = BackendMemory
	}
	if strings.TrimSpace(normalized.DSN) != "" {
		return normalized
	}

	switch normalized.Type {
	case BackendSQLite, BackendBBolt:
		normalized.DSN = DefaultDSN(baseDir, normalized.Type)
	}
	return normalized
}
