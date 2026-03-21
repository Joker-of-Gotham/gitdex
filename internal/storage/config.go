package storage

import "fmt"

// BackendType identifies the storage backend to use.
type BackendType string

const (
	BackendMemory   BackendType = "memory"
	BackendPostgres BackendType = "postgres"
	BackendSQLite   BackendType = "sqlite"
	BackendBBolt    BackendType = "bbolt"
)

// Config holds the settings needed to open any supported storage backend.
type Config struct {
	// Type selects the backend: "memory", "postgres", "sqlite", "bbolt".
	Type BackendType `json:"type" yaml:"type"`

	// DSN is the data-source name. Interpretation depends on Type:
	//   postgres → "postgres://user:pass@host:5432/dbname?sslmode=disable"
	//   sqlite   → file path or ":memory:"
	//   bbolt    → file path (e.g. "~/.gitdex/gitdex.db")
	//   memory   → ignored
	DSN string `json:"dsn,omitempty" yaml:"dsn,omitempty"`

	// MaxOpenConns limits the connection pool (postgres/sqlite only).
	MaxOpenConns int `json:"max_open_conns,omitempty" yaml:"max_open_conns,omitempty"`

	// MaxIdleConns sets the idle pool size (postgres/sqlite only).
	MaxIdleConns int `json:"max_idle_conns,omitempty" yaml:"max_idle_conns,omitempty"`

	// AutoMigrate runs pending migrations on startup when true.
	AutoMigrate bool `json:"auto_migrate,omitempty" yaml:"auto_migrate,omitempty"`
}

// Validate returns an error if the configuration is incomplete.
func (c Config) Validate() error {
	switch c.Type {
	case BackendMemory:
		return nil
	case BackendPostgres, BackendSQLite, BackendBBolt:
		if c.DSN == "" {
			return fmt.Errorf("storage: dsn is required for backend %q", c.Type)
		}
		return nil
	case "":
		return fmt.Errorf("storage: type is required (memory, postgres, sqlite, bbolt)")
	default:
		return fmt.Errorf("storage: unsupported backend type %q", c.Type)
	}
}
