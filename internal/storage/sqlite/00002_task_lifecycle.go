package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddNamedMigrationContext("00002_task_lifecycle.go", upTaskLifecycle, downTaskLifecycle)
}

func upTaskLifecycle(ctx context.Context, tx *sql.Tx) error {
	for _, column := range []struct {
		name       string
		definition string
	}{
		{name: "started_at", definition: "TEXT NOT NULL DEFAULT ''"},
		{name: "completed_at", definition: "TEXT NOT NULL DEFAULT ''"},
		{name: "disposition", definition: "TEXT NOT NULL DEFAULT ''"},
	} {
		exists, err := tableColumnExists(ctx, tx, "tasks", column.name)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		if _, err := tx.ExecContext(ctx, fmt.Sprintf(
			"ALTER TABLE tasks ADD COLUMN %s %s", column.name, column.definition,
		)); err != nil {
			return fmt.Errorf("add tasks.%s: %w", column.name, err)
		}
	}
	return nil
}

func downTaskLifecycle(context.Context, *sql.Tx) error {
	// SQLite cannot safely drop columns without rebuilding the table. The
	// columns remain harmless when an operator rolls back this migration.
	return nil
}

func tableColumnExists(ctx context.Context, tx *sql.Tx, tableName string, columnName string) (bool, error) {
	rows, err := tx.QueryContext(ctx, "PRAGMA table_info("+tableName+")")
	if err != nil {
		return false, fmt.Errorf("inspect %s columns: %w", tableName, err)
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name string
		var columnType string
		var notNull int
		var defaultValue sql.NullString
		var primaryKey int
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			return false, fmt.Errorf("scan %s column: %w", tableName, err)
		}
		if name == columnName {
			return true, nil
		}
	}
	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("iterate %s columns: %w", tableName, err)
	}
	return false, nil
}
