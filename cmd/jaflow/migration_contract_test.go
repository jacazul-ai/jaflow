package main

import (
	"os"
	"strings"
	"testing"

	"github.com/jacazul-ai/jaflow/internal/testharness"
)

func TestMigrationDryRunDoesNotCreateTargetDatabase(t *testing.T) {
	binary := buildJaflow(t)
	harness := testharness.NewHarness(t, "project-alpha", "session")
	source := harness.WriteFile(t, "source.json", []byte(`[
  {
    "id": 1,
    "uuid": "11111111-1111-4111-8111-111111111111",
    "project": "parity",
    "description": "[EXECUTE] Import task",
    "status": "pending"
  }
]`))

	output, err := runJaflow(t, binary, harness, "migrate", "taskwarrior", "--source", source)
	if err != nil {
		t.Fatalf("migration dry-run: %v\n%s", err, output)
	}
	if !strings.Contains(output, "Migration dry-run: no changes written") {
		t.Fatalf("dry-run output = %q, want no-write confirmation", output)
	}
	if _, err := os.Stat(harness.DatabasePath); !os.IsNotExist(err) {
		t.Fatalf("dry-run created target database: %v", err)
	}
}

func TestMigrationApplyIsRepeatableAndPreservesNativeState(t *testing.T) {
	binary := buildJaflow(t)
	harness := testharness.NewHarness(t, "project-alpha", "session")
	source := harness.WriteFile(t, "source.json", []byte(`[
  {
    "id": 1,
    "uuid": "11111111-1111-4111-8111-111111111111",
    "project": "parity",
    "description": "[EXECUTE] Import task",
    "status": "pending",
    "externalid": "#JAF-123",
    "annotations": [
      {"entry": "20260830T120000Z", "description": "DECISION: Keep UUID"}
    ]
  }
]`))

	for _, args := range [][]string{
		{"migrate", "taskwarrior", "--source", source, "--apply"},
		{"migrate", "taskwarrior", "--source", source, "--apply"},
	} {
		output, err := runJaflow(t, binary, harness, args...)
		if err != nil {
			t.Fatalf("run %v: %v\n%s", args, err, output)
		}
		if !strings.Contains(output, "Migration applied") {
			t.Fatalf("apply output = %q, want confirmation", output)
		}
	}

	output, err := runJaflow(t, binary, harness, "status", "parity", "--force")
	if err != nil {
		t.Fatalf("read imported state: %v\n%s", err, output)
	}
	for _, expected := range []string{"Import task", "#JAF-123"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("imported status = %q, want %s", output, expected)
		}
	}
	output, err = runJaflow(t, binary, harness, "notes", "11111111-1111-4111-8111-111111111111")
	if err != nil || !strings.Contains(output, "DECISION: Keep UUID") {
		t.Fatalf("imported notes = %q, err %v; want one decision", output, err)
	}
}

func TestMigrationRejectsConflictingModes(t *testing.T) {
	binary := buildJaflow(t)
	harness := testharness.NewHarness(t, "project-alpha", "session")
	source := harness.WriteFile(t, "source.json", []byte("[]"))

	output, err := runJaflow(t, binary, harness, "migrate", "taskwarrior", "--source", source, "--apply", "--dry-run")
	if err == nil || !strings.Contains(output, "cannot be combined") || !strings.Contains(output, "ACTION:") {
		t.Fatalf("conflicting migration flags = %q, err %v; want actionable error", output, err)
	}
}
