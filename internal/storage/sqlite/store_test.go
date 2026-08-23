package sqlite_test

import (
	"context"
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
