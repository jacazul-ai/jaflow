package main

import (
	"regexp"
	"strings"
	"testing"

	"github.com/jacazul-ai/jaflow/internal/testharness"
)

var shortTaskIDPattern = regexp.MustCompile(`Created task ([0-9a-f]{8})`)

func createdTaskIDs(output string) []string {
	matches := shortTaskIDPattern.FindAllStringSubmatch(output, -1)
	ids := make([]string, 0, len(matches))
	for _, match := range matches {
		ids = append(ids, match[1])
	}
	return ids
}

func TestReferenceCommandSurfaceIsRoutable(t *testing.T) {
	binary := buildJaflow(t)
	harness := testharness.NewHarness(t, "project", "session")

	for _, command := range []string{
		"next",
		"amend",
		"rename",
		"urgent",
		"block",
		"unblock",
		"wait",
		"cache",
	} {
		t.Run(command, func(t *testing.T) {
			output, err := runJaflow(t, binary, harness, "help", command)
			if err != nil {
				t.Fatalf("help %s failed: %v\n%s", command, err, output)
			}
			if !strings.Contains(output, "USAGE") {
				t.Fatalf("help %s = %q, want usage", command, output)
			}
		})
	}
}

func TestReferenceInitiativeAliasesRemainAvailable(t *testing.T) {
	binary := buildJaflow(t)
	harness := testharness.NewHarness(t, "project", "session")

	for _, alias := range []string{"initiative", "ini"} {
		output, err := runJaflow(t, binary, harness, alias, alias+"-plan", "Task")
		if err != nil {
			t.Fatalf("create plan through %s: %v\n%s", alias, err, output)
		}
	}

	output, err := runJaflow(t, binary, harness, "inis", "--all")
	if err != nil {
		t.Fatalf("list plans through inis: %v\n%s", err, output)
	}
	for _, plan := range []string{"initiative-plan", "ini-plan"} {
		if !strings.Contains(output, plan) {
			t.Fatalf("inis output = %q, want %s", output, plan)
		}
	}

	output, err = runJaflow(t, binary, harness, "initiatives", "--all")
	if err != nil {
		t.Fatalf("list plans through initiatives: %v\n%s", err, output)
	}
	if !strings.Contains(output, "initiative-plan") || !strings.Contains(output, "ini-plan") {
		t.Fatalf("initiatives output = %q, want both aliases", output)
	}
}

func TestNextUsesReadyTasksWithinInitiative(t *testing.T) {
	binary := buildJaflow(t)
	harness := testharness.NewHarness(t, "project", "session")

	output, err := runJaflow(t, binary, harness, "plan", "parity", "First", "Second")
	if err != nil {
		t.Fatalf("create parity plan: %v\n%s", err, output)
	}
	ids := createdTaskIDs(output)
	if len(ids) != 2 {
		t.Fatalf("plan output = %q, want two task IDs", output)
	}

	output, err = runJaflow(t, binary, harness, "next", "parity")
	if err != nil {
		t.Fatalf("find first ready task: %v\n%s", err, output)
	}
	if !strings.Contains(output, "First") || strings.Contains(output, "Second") {
		t.Fatalf("first next output = %q, want only First", output)
	}

	for _, args := range [][]string{
		{"execute", ids[0]},
		{"outcome", ids[0], "First is complete"},
		{"done", ids[0]},
	} {
		if output, err := runJaflow(t, binary, harness, args...); err != nil {
			t.Fatalf("run %v: %v\n%s", args, err, output)
		}
	}

	output, err = runJaflow(t, binary, harness, "next", "parity")
	if err != nil {
		t.Fatalf("find second ready task: %v\n%s", err, output)
	}
	if !strings.Contains(output, "Second") {
		t.Fatalf("second next output = %q, want Second", output)
	}
}

func TestAmendAndRenamePreserveInitiativeMetadata(t *testing.T) {
	binary := buildJaflow(t)
	harness := testharness.NewHarness(t, "project", "session")

	output, err := runJaflow(t, binary, harness, "plan", "old-initiative", "Original")
	if err != nil {
		t.Fatalf("create metadata plan: %v\n%s", err, output)
	}
	ids := createdTaskIDs(output)
	if len(ids) != 1 {
		t.Fatalf("plan output = %q, want one task ID", output)
	}

	output, err = runJaflow(
		t,
		binary,
		harness,
		"amend",
		ids[0],
		"description=Updated",
		"ticket=#JAF-123",
	)
	if err != nil {
		t.Fatalf("amend task metadata: %v\n%s", err, output)
	}

	output, err = runJaflow(t, binary, harness, "rename", "old-initiative", "new-initiative")
	if err != nil {
		t.Fatalf("rename initiative: %v\n%s", err, output)
	}

	output, err = runJaflow(t, binary, harness, "status", "new-initiative", "--force")
	if err != nil {
		t.Fatalf("read renamed initiative: %v\n%s", err, output)
	}
	for _, expected := range []string{"Updated", "#JAF-123"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("renamed status = %q, want %s", output, expected)
		}
	}
}

func TestFocusInterestAndCacheControlsAreAvailable(t *testing.T) {
	binary := buildJaflow(t)
	harness := testharness.NewHarness(t, "project", "session")

	if output, err := runJaflow(t, binary, harness, "plan", "cache", "Task"); err != nil {
		t.Fatalf("create cache plan: %v\n%s", err, output)
	}
	if output, err := runJaflow(t, binary, harness, "focus", "plan", "cache"); err != nil {
		t.Fatalf("focus cache plan: %v\n%s", err, output)
	}
	if output, err := runJaflow(t, binary, harness, "focus", "interest", "add", "cache"); err != nil {
		t.Fatalf("add plan interest: %v\n%s", err, output)
	}

	output, err := runJaflow(t, binary, harness, "focus", "interest", "list")
	if err != nil {
		t.Fatalf("list plan interests: %v\n%s", err, output)
	}
	if !strings.Contains(output, "cache") {
		t.Fatalf("interest list = %q, want cache initiative", output)
	}

	if output, err := runJaflow(t, binary, harness, "status", "cache"); err != nil {
		t.Fatalf("prime status cache: %v\n%s", err, output)
	}
	output, err = runJaflow(t, binary, harness, "cache", "info")
	if err != nil || !strings.Contains(output, "file") {
		t.Fatalf("cache info = %q, err %v; want file count", output, err)
	}

	if output, err := runJaflow(t, binary, harness, "cache", "clear", "status"); err != nil {
		t.Fatalf("clear status cache: %v\n%s", err, output)
	}
	output, err = runJaflow(t, binary, harness, "status", "cache")
	if err != nil || strings.Contains(output, "[cached]") {
		t.Fatalf("status after cache clear = %q, err %v; want fresh output", output, err)
	}
}

func TestDependencyCommandsPreserveReadySelection(t *testing.T) {
	binary := buildJaflow(t)
	harness := testharness.NewHarness(t, "project", "session")

	output, err := runJaflow(t, binary, harness, "plan", "dependencies", "First", "Second")
	if err != nil {
		t.Fatalf("create dependency plan: %v\n%s", err, output)
	}
	ids := createdTaskIDs(output)
	if len(ids) != 2 {
		t.Fatalf("plan output = %q, want two task IDs", output)
	}

	if output, err := runJaflow(t, binary, harness, "unblock", ids[1], ids[0]); err != nil {
		t.Fatalf("remove dependency: %v\n%s", err, output)
	}
	output, err = runJaflow(t, binary, harness, "next", "dependencies")
	if err != nil {
		t.Fatalf("list unblocked tasks: %v\n%s", err, output)
	}
	if !strings.Contains(output, "First") || !strings.Contains(output, "Second") {
		t.Fatalf("unblocked next output = %q, want both tasks", output)
	}

	if output, err := runJaflow(t, binary, harness, "block", ids[1], ids[0]); err != nil {
		t.Fatalf("add dependency: %v\n%s", err, output)
	}
	output, err = runJaflow(t, binary, harness, "next", "dependencies")
	if err != nil {
		t.Fatalf("list blocked tasks: %v\n%s", err, output)
	}
	if !strings.Contains(output, "First") || strings.Contains(output, "Second") {
		t.Fatalf("blocked next output = %q, want only First", output)
	}
}
