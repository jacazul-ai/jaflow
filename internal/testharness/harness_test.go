package testharness

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestNewHarnessProvidesDistinctFixtureRoots(t *testing.T) {
	first := NewHarness(t, "project-alpha", "session-one")
	second := NewHarness(t, "project-beta", "session-two")

	firstFile := first.WriteFile(t, "state/focus.json", []byte("alpha"))
	secondFile := second.WriteFile(t, "state/focus.json", []byte("beta"))

	if first.Root == second.Root {
		t.Fatal("harnesses must use different roots")
	}
	if first.TaskData == second.TaskData {
		t.Fatal("harnesses must use different task data directories")
	}

	firstData, err := os.ReadFile(firstFile)
	if err != nil {
		t.Fatalf("read first fixture state: %v", err)
	}
	if string(firstData) != "alpha" {
		t.Fatalf("first fixture state = %q, want %q", firstData, "alpha")
	}

	secondData, err := os.ReadFile(secondFile)
	if err != nil {
		t.Fatalf("read second fixture state: %v", err)
	}
	if string(secondData) != "beta" {
		t.Fatalf("second fixture state = %q, want %q", secondData, "beta")
	}

	if _, err := os.Stat(filepath.Join(first.Root, "state", "focus.json")); err != nil {
		t.Fatalf("first fixture state is missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(second.Root, "state", "focus.json")); err != nil {
		t.Fatalf("second fixture state is missing: %v", err)
	}
}

func TestFakeCommandUsesIsolatedPath(t *testing.T) {
	harness := NewHarness(t, "project", "session")
	harness.FakeCommand(t, "fake-task", "#!/bin/sh\nprintf 'task:%s\\n' \"$1\"\n")

	output, err := harness.RunCommand(context.Background(), "fake-task", "export")
	if err != nil {
		t.Fatalf("run fake task: %v", err)
	}
	if output != "task:export\n" {
		t.Fatalf("fake task output = %q, want %q", output, "task:export\\n")
	}
}
