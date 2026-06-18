package migrate

import (
	"database/sql"
	_ "embed"
	"fmt"
	"strings"
)

//go:embed migrations/001_init.sql
var initSQL string

// Run executes embedded SQL migrations. Statements are split on ";" since the MySQL driver doesn't support multi-statement exec.
func Run(db *sql.DB) error {
	for _, stmt := range strings.Split(initSQL, ";") {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}
	return nil
}
