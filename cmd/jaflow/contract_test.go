package main

import (
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/jacazul-ai/jaflow/internal/testharness"
)

func buildJaflow(t *testing.T) string {
	t.Helper()

	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve module root: %v", err)
	}

	binary := filepath.Join(t.TempDir(), "jaflow")
	command := exec.Command("go", "build", "-o", binary, "./cmd/jaflow")
	command.Dir = root
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build jaflow: %v\n%s", err, output)
	}
	return binary
}

func runJaflow(t *testing.T, binary string, harness *testharness.Harness, args ...string) (string, error) {
	t.Helper()

	command := exec.Command(binary, args...)
	command.Dir = harness.Root
	command.Env = harness.Environment
	output, err := command.CombinedOutput()
	return string(output), err
}

func TestPlanStateIsIsolatedByProject(t *testing.T) {
	binary := buildJaflow(t)
	first := testharness.NewHarness(t, "project-alpha", "session-alpha")
	second := testharness.NewHarness(t, "project-beta", "session-beta")

	firstOutput, err := runJaflow(t, binary, first, "plan", "alpha", "Alpha task")
	if err != nil {
		t.Fatalf("create alpha plan: %v\n%s", err, firstOutput)
	}
	if !strings.Contains(firstOutput, "Alpha task") {
		t.Fatalf("alpha output = %q, want task description", firstOutput)
	}

	secondOutput, err := runJaflow(t, binary, second, "status")
	if err != nil {
		t.Fatalf("read beta status: %v\n%s", err, secondOutput)
	}
	if strings.Contains(secondOutput, "Alpha task") {
		t.Fatalf("project beta observed project alpha state: %q", secondOutput)
	}
}

func TestPlanCreationReturnsShortUUID(t *testing.T) {
	binary := buildJaflow(t)
	harness := testharness.NewHarness(t, "project", "session")

	output, err := runJaflow(t, binary, harness, "plan", "parity", "A task")
	if err != nil {
		t.Fatalf("create plan: %v\n%s", err, output)
	}
	if !regexp.MustCompile(`Created task [0-9a-f]{8}`).MatchString(output) {
		t.Fatalf("plan output = %q, want short UUID", output)
	}
}

func TestUnknownCommandReturnsActionableError(t *testing.T) {
	binary := buildJaflow(t)
	harness := testharness.NewHarness(t, "project", "session")

	output, err := runJaflow(t, binary, harness, "not-a-command")
	if err == nil {
		t.Fatalf("unknown command succeeded with output %q", output)
	}
	if !strings.Contains(strings.ToLower(output), "unknown") {
		t.Fatalf("unknown command output = %q, want actionable error", output)
	}
	if !strings.Contains(output, "ACTION:") {
		t.Fatalf("unknown command output = %q, want an ACTION prompt", output)
	}
}

func TestHelpProvidesAgentWorkflowBriefing(t *testing.T) {
	binary := buildJaflow(t)
	harness := testharness.NewHarness(t, "project", "session")

	rootOutput, err := runJaflow(t, binary, harness, "help")
	if err != nil {
		t.Fatalf("root help failed: %v\n%s", err, rootOutput)
	}
	for _, section := range []string{"ROLE", "WORKFLOW", "COMMANDS", "AGENT RULES", "NEXT"} {
		if !strings.Contains(rootOutput, section) {
			t.Fatalf("root help = %q, want %q section", rootOutput, section)
		}
	}

	planOutput, err := runJaflow(t, binary, harness, "help", "plan")
	if err != nil {
		t.Fatalf("plan help failed: %v\n%s", err, planOutput)
	}
	for _, section := range []string{"PREREQUISITES", "SIDE EFFECTS AND OUTPUT", "EXAMPLES", "NEXT ACTION"} {
		if !strings.Contains(planOutput, section) {
			t.Fatalf("plan help = %q, want %q section", planOutput, section)
		}
	}
}

func TestTaskLifecycleEnforcesOutcomeAndUnblocks(t *testing.T) {
	binary := buildJaflow(t)
	harness := testharness.NewHarness(t, "project", "session")

	planOutput, err := runJaflow(t, binary, harness, "plan", "parity", "First", "Second")
	if err != nil {
		t.Fatalf("create plan: %v\n%s", err, planOutput)
	}
	matches := regexp.MustCompile(`Created task ([0-9a-f]{8})`).FindAllStringSubmatch(planOutput, -1)
	if len(matches) != 2 {
		t.Fatalf("plan output = %q, want two task UUIDs", planOutput)
	}
	first, second := matches[0][1], matches[1][1]

	if output, err := runJaflow(t, binary, harness, "execute", first); err != nil {
		t.Fatalf("execute first task: %v\n%s", err, output)
	}
	output, err := runJaflow(t, binary, harness, "done", first)
	if err == nil || !strings.Contains(output, "OUTCOME") {
		t.Fatalf("done without outcome = %q, err %v; want OUTCOME gate", output, err)
	}
	if output, err := runJaflow(t, binary, harness, "outcome", first, "First is complete"); err != nil {
		t.Fatalf("record first outcome: %v\n%s", err, output)
	}
	output, err = runJaflow(t, binary, harness, "done", first)
	if err != nil || !strings.Contains(output, "Ready task "+second) {
		t.Fatalf("complete first task = %q, err %v; want second ready", output, err)
	}
	if output, err := runJaflow(t, binary, harness, "execute", second); err != nil {
		t.Fatalf("execute second task: %v\n%s", err, output)
	}
}
