package sqlite_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/jacazul-ai/jaflow/internal/storage/sqlite"
	"github.com/jacazul-ai/jaflow/internal/task"
)

func TestStorePersistsInitiativesTasksAndDependencies(t *testing.T) {
	ctx := context.Background()
	store := openStore(t, filepath.Join(t.TempDir(), "alpha", "jaflow.sqlite3"))

	initiative, err := store.GetOrCreateInitiative(ctx, task.CreateInitiativeInput{
		ProjectID: "project-alpha",
		Name:      "parity",
	})
	if err != nil {
		t.Fatalf("create initiative: %v", err)
	}
	if initiative.ID == "" {
		t.Fatal("initiative ID is empty")
	}

	first, err := store.CreateTask(ctx, task.CreateTaskInput{
		InitiativeID: initiative.ID,
		Description:  "Define the schema",
		Mode:         "DESIGN",
	})
	if err != nil {
		t.Fatalf("create first task: %v", err)
	}
	second, err := store.CreateTask(ctx, task.CreateTaskInput{
		InitiativeID: initiative.ID,
		Description:  "Implement the store",
		Mode:         "EXECUTE",
		Dependencies: []string{first.ID},
	})
	if err != nil {
		t.Fatalf("create second task: %v", err)
	}

	tasks, err := store.ListTasks(ctx, "project-alpha", "parity")
	if err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("task count = %d, want 2", len(tasks))
	}
	if tasks[1].ID != second.ID {
		t.Fatalf("second task ID = %q, want %q", tasks[1].ID, second.ID)
	}
	if tasks[1].InitiativeID != initiative.ID {
		t.Fatalf("second task initiative = %q, want %q", tasks[1].InitiativeID, initiative.ID)
	}
	if len(tasks[1].Dependencies) != 1 || tasks[1].Dependencies[0] != first.ID {
		t.Fatalf("second task dependencies = %#v, want [%q]", tasks[1].Dependencies, first.ID)
	}
	if tasks[1].Status != task.Pending {
		t.Fatalf("second task status = %q, want %q", tasks[1].Status, task.Pending)
	}
}

func TestGooseAdoptsLegacyV1Schema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy-jaflow.sqlite3")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open legacy database: %v", err)
	}
	for _, statement := range []string{
		`CREATE TABLE initiatives (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL,
			name TEXT NOT NULL,
			status TEXT NOT NULL,
			external_ticket TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			UNIQUE (project_id, name)
		)`,
		`CREATE TABLE tasks (
			id TEXT PRIMARY KEY,
			initiative_id TEXT NOT NULL REFERENCES initiatives(id) ON DELETE CASCADE,
			description TEXT NOT NULL,
			mode TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL,
			outcome TEXT NOT NULL DEFAULT '',
			external_ticket TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at TEXT NOT NULL
		)`,
		`INSERT INTO schema_migrations (version, applied_at)
			VALUES (1, '2026-01-01T00:00:00Z')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			db.Close()
			t.Fatalf("create legacy schema: %v", err)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close legacy database: %v", err)
	}

	store, err := sqlite.Open(context.Background(), path)
	if err != nil {
		t.Fatalf("open legacy database with Goose: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close adopted database: %v", err)
	}

	adopted, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("reopen adopted database: %v", err)
	}
	defer adopted.Close()
	var version int
	if err := adopted.QueryRow(
		"SELECT MAX(version_id) FROM goose_db_version WHERE is_applied = 1",
	).Scan(&version); err != nil {
		t.Fatalf("read adopted Goose version: %v", err)
	}
	if version != 4 {
		t.Fatalf("adopted Goose version = %d, want 4", version)
	}
	for _, column := range []string{"started_at", "completed_at", "disposition"} {
		var count int
		if err := adopted.QueryRow(
			"SELECT COUNT(*) FROM pragma_table_info('tasks') WHERE name = ?", column,
		).Scan(&count); err != nil {
			t.Fatalf("check legacy column %s: %v", column, err)
		}
		if count != 1 {
			t.Fatalf("legacy column %s was not added", column)
		}
	}
}

func TestOpenAppliesAllGooseMigrations(t *testing.T) {
	path := filepath.Join(t.TempDir(), "migrations", "jaflow.sqlite3")
	store := openStore(t, path)
	if err := store.Close(); err != nil {
		t.Fatalf("close migrated store: %v", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open migrated database: %v", err)
	}
	defer db.Close()

	var version int
	if err := db.QueryRow(
		"SELECT MAX(version_id) FROM goose_db_version WHERE is_applied = 1",
	).Scan(&version); err != nil {
		t.Fatalf("read goose version: %v", err)
	}
	if version != 4 {
		t.Fatalf("goose version = %d, want 4", version)
	}
	var sessionNotes int
	if err := db.QueryRow(
		"SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'session_notes'",
	).Scan(&sessionNotes); err != nil {
		t.Fatalf("check session notes table: %v", err)
	}
	if sessionNotes != 1 {
		t.Fatal("session_notes table was not created")
	}
}

func TestStoreSeparatesProjectDatabases(t *testing.T) {
	ctx := context.Background()
	first := openStore(t, filepath.Join(t.TempDir(), "first", "jaflow.sqlite3"))
	second := openStore(t, filepath.Join(t.TempDir(), "second", "jaflow.sqlite3"))

	initiative, err := first.GetOrCreateInitiative(ctx, task.CreateInitiativeInput{
		ProjectID: "project-alpha",
		Name:      "private",
	})
	if err != nil {
		t.Fatalf("create first initiative: %v", err)
	}
	if _, err := first.CreateTask(ctx, task.CreateTaskInput{
		InitiativeID: initiative.ID,
		Description:  "Alpha-only task",
	}); err != nil {
		t.Fatalf("create first task: %v", err)
	}

	visible, err := second.ListTasks(ctx, "project-alpha", "private")
	if err != nil {
		t.Fatalf("list second project database: %v", err)
	}
	if len(visible) != 0 {
		t.Fatalf("second database observed %d tasks from first database", len(visible))
	}
}

func openStore(t *testing.T, path string) *sqlite.Store {
	t.Helper()

	store, err := sqlite.Open(context.Background(), path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Errorf("close store: %v", err)
		}
	})
	return store
}
