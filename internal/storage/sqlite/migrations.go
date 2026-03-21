package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"sort"
	"strings"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// runMigrations executes all .up.sql files in order.
func runMigrations(ctx context.Context, db *sql.DB) error {
	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return err
	}

	var upFiles []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".up.sql") {
			upFiles = append(upFiles, name)
		}
	}
	sort.Strings(upFiles)

	for _, name := range upFiles {
		sqlText, err := migrationFS.ReadFile("migrations/" + name)
		if err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, string(sqlText)); err != nil {
			return err
		}
	}
	return nil
}
