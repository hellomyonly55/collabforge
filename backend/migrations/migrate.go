// Package migrations runs embedded, versioned SQL on one connection under a MySQL advisory lock.
package migrations

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

//go:embed *.sql
var scripts embed.FS

const LatestVersion = 12

func Run(ctx context.Context, db *sql.DB) error {
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	var locked int
	if err = conn.QueryRowContext(ctx, "SELECT GET_LOCK(CONCAT(DATABASE(), ':taskflow:migrate'), 15)").Scan(&locked); err != nil {
		return err
	}
	if locked != 1 {
		return fmt.Errorf("migration lock unavailable")
	}
	defer func() {
		release, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_, _ = conn.ExecContext(release, "SELECT RELEASE_LOCK(CONCAT(DATABASE(), ':taskflow:migrate'))")
	}()
	entries, err := scripts.ReadDir(".")
	if err != nil {
		return err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	// Bootstrap tracking table before checking applied versions. DDL is idempotent because MySQL DDL is not transactional.
	bootstrap, err := scripts.ReadFile("000001_schema_migrations.sql")
	if err != nil {
		return err
	}
	if _, err = conn.ExecContext(ctx, string(bootstrap)); err != nil {
		return err
	}
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}
		version, err := strconv.Atoi(strings.SplitN(name, "_", 2)[0])
		if err != nil {
			return err
		}
		var applied int
		if err = conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations WHERE version = ?", version).Scan(&applied); err != nil {
			return err
		}
		if applied > 0 {
			continue
		}
		script, err := scripts.ReadFile(name)
		if err != nil {
			return err
		}
		// Each migration file is exactly one statement; future multi-step changes use separate numbered files.
		appliedDDL, err := ddlAlreadyApplied(ctx, conn, name)
		if err != nil {
			return err
		}
		if !appliedDDL {
			if _, err = conn.ExecContext(ctx, string(script)); err != nil {
				return fmt.Errorf("migration %s failed: %w", name, err)
			}
		}
		if _, err = conn.ExecContext(ctx, "INSERT INTO schema_migrations(version,name) VALUES (?,?)", version, name); err != nil {
			return err
		}
	}
	return nil
}

// CREATE migrations use IF NOT EXISTS. ALTER migrations declare a precise schema postcondition.
func ddlAlreadyApplied(ctx context.Context, conn *sql.Conn, name string) (bool, error) {
	if name == "000010_project_task_revision.sql" {
		var count int
		err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'projects' AND COLUMN_NAME = 'task_revision' AND COLUMN_TYPE = 'bigint unsigned' AND IS_NULLABLE = 'NO' AND COLUMN_DEFAULT = '0'`).Scan(&count)
		return count == 1, err
	}
	if name != "000008_activity_project_fk.sql" {
		return false, nil
	}
	var columns int
	err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.KEY_COLUMN_USAGE WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'activity_logs' AND CONSTRAINT_NAME = 'fk_activity_project_team' AND REFERENCED_TABLE_NAME = 'projects' AND ((COLUMN_NAME = 'team_id' AND REFERENCED_COLUMN_NAME = 'team_id') OR (COLUMN_NAME = 'project_id' AND REFERENCED_COLUMN_NAME = 'id'))`).Scan(&columns)
	return columns == 2, err
}
