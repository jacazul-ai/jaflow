package config_test

import (
	"path/filepath"
	"testing"

	"github.com/jacazul-ai/jaflow/internal/config"
)

func TestLegacyTaskDataDoesNotReplaceNativeDatabase(t *testing.T) {
	home := t.TempDir()
	t.Setenv("JACAZUL_HOME", home)
	t.Setenv("TASKDATA", filepath.Join(t.TempDir(), "legacy-taskdata"))

	opts := config.AppOptions{ProjectID: "project-alpha"}
	if err := config.Resolve(&opts); err != nil {
		t.Fatalf("resolve options: %v", err)
	}
	want := filepath.Join(home, "jaflow", "project-alpha", "jaflow.sqlite3")
	if opts.DatabasePath != want {
		t.Fatalf("database path = %q, want native path %q", opts.DatabasePath, want)
	}
	if opts.TaskData == "" {
		t.Fatal("legacy taskdata context was unexpectedly discarded")
	}
}
