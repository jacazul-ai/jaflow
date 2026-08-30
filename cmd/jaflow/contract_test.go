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

func TestFocusSwitchesTaskStack(t *testing.T) {
	binary := buildJaflow(t)
	harness := testharness.NewHarness(t, "project", "session")

	planOutput, err := runJaflow(t, binary, harness, "plan", "focus", "First", "Second")
	if err != nil {
		t.Fatalf("create focus plan: %v\n%s", err, planOutput)
	}
	matches := regexp.MustCompile(`Created task ([0-9a-f]{8})`).FindAllStringSubmatch(planOutput, -1)
	if len(matches) != 2 {
		t.Fatalf("plan output = %q, want two task UUIDs", planOutput)
	}
	first, second := matches[0][1], matches[1][1]

	for _, taskID := range []string{first, second} {
		if output, err := runJaflow(t, binary, harness, "focus", "task", taskID); err != nil {
			t.Fatalf("focus task %s: %v\n%s", taskID, err, output)
		}
	}
	output, err := runJaflow(t, binary, harness, "focus", "show")
	if err != nil || !strings.Contains(output, "Task: "+second) {
		t.Fatalf("focus show = %q, err %v; want second task", output, err)
	}
	output, err = runJaflow(t, binary, harness, "focus", "pop")
	if err != nil || !strings.Contains(output, first) {
		t.Fatalf("focus pop = %q, err %v; want first task", output, err)
	}
	if output, err := runJaflow(t, binary, harness, "focus", "clear"); err != nil {
		t.Fatalf("focus clear: %v\n%s", err, output)
	}
}

func TestSessionListShowsCurrentAnchor(t *testing.T) {
	binary := buildJaflow(t)
	harness := testharness.NewHarness(t, "project", "session")

	planOutput, err := runJaflow(t, binary, harness, "plan", "sessions", "First")
	if err != nil {
		t.Fatalf("create session plan: %v\n%s", err, planOutput)
	}
	match := regexp.MustCompile(`Created task ([0-9a-f]{8})`).FindStringSubmatch(planOutput)
	if len(match) != 2 {
		t.Fatalf("plan output = %q, want one task UUID", planOutput)
	}
	if output, err := runJaflow(t, binary, harness, "focus", "task", match[1]); err != nil {
		t.Fatalf("focus session task: %v\n%s", err, output)
	}
	output, err := runJaflow(t, binary, harness, "session", "list")
	if err != nil || !strings.Contains(output, "* session") || !strings.Contains(output, match[1]) {
		t.Fatalf("session list = %q, err %v; want current anchor", output, err)
	}
}

func TestDashboardShowsBlockedWorkAndBacklogLifecycle(t *testing.T) {
	binary := buildJaflow(t)
	harness := testharness.NewHarness(t, "project", "session")

	output, err := runJaflow(t, binary, harness, "plan", "dashboard", "First", "Second")
	if err != nil {
		t.Fatalf("create dashboard plan: %v\n%s", err, output)
	}
	output, err = runJaflow(t, binary, harness, "ponder")
	if err != nil || !strings.Contains(output, "dashboard") || !strings.Contains(output, "blocked:1") {
		t.Fatalf("ponder = %q, err %v; want blocked dashboard count", output, err)
	}
	output, err = runJaflow(t, binary, harness, "tree", "dashboard")
	if err != nil || !strings.Contains(output, "READY") || !strings.Contains(output, "BLOCKED") {
		t.Fatalf("tree = %q, err %v; want dependency markers", output, err)
	}
	if output, err := runJaflow(t, binary, harness, "backlog", "dashboard"); err != nil {
		t.Fatalf("backlog dashboard: %v\n%s", err, output)
	}
	output, err = runJaflow(t, binary, harness, "plans")
	if err != nil || strings.Contains(output, "dashboard") {
		t.Fatalf("plans after backlog = %q, err %v; want hidden initiative", output, err)
	}
	output, err = runJaflow(t, binary, harness, "plans", "--with-backlog")
	if err != nil || !strings.Contains(output, "dashboard") {
		t.Fatalf("plans with backlog = %q, err %v; want initiative", output, err)
	}
}

func TestStatusUsesPromptAsAdCacheSignal(t *testing.T) {
	binary := buildJaflow(t)
	harness := testharness.NewHarness(t, "project", "session")
	if output, err := runJaflow(t, binary, harness, "plan", "cache", "First"); err != nil {
		t.Fatalf("create cache plan: %v\n%s", err, output)
	}
	if output, err := runJaflow(t, binary, harness, "status"); err != nil {
		t.Fatalf("prime status cache: %v\n%s", err, output)
	}
	output, err := runJaflow(t, binary, harness, "status")
	if err != nil || !strings.Contains(output, "[cached]") {
		t.Fatalf("cached status = %q, err %v; want Prompt as Ad signal", output, err)
	}
}

func TestRoadmapInitializationHasDuplicateGuard(t *testing.T) {
	binary := buildJaflow(t)
	harness := testharness.NewHarness(t, "project", "session")
	if output, err := runJaflow(t, binary, harness, "plan", "roadmap", "First"); err != nil {
		t.Fatalf("create roadmap initiative: %v\n%s", err, output)
	}
	if output, err := runJaflow(t, binary, harness, "roadmap", "init"); err != nil {
		t.Fatalf("initialize roadmap: %v\n%s", err, output)
	}
	output, err := runJaflow(t, binary, harness, "roadmap", "init")
	if err == nil || !strings.Contains(output, "already initialized") || !strings.Contains(output, "ACTION:") {
		t.Fatalf("duplicate roadmap init = %q, err %v; want actionable guard", output, err)
	}
	output, err = runJaflow(t, binary, harness, "roadmap", "show")
	if err != nil || !strings.Contains(output, "ROADMAP") || !strings.Contains(output, "roadmap") {
		t.Fatalf("roadmap show = %q, err %v; want ledger entry", output, err)
	}
}

func TestExternalTicketInheritanceContracts(t *testing.T) {
	binary := buildJaflow(t)
	harness := testharness.NewHarness(t, "project", "session")

	output, err := runJaflow(t, binary, harness, "plan", "ticket", "Root", "Child")
	if err != nil {
		t.Fatalf("create ticket plan: %v\n%s", err, output)
	}
	matches := regexp.MustCompile(`Created task ([0-9a-f]{8})`).FindAllStringSubmatch(output, -1)
	if len(matches) != 2 {
		t.Fatalf("ticket plan output = %q, want two task UUIDs", output)
	}
	root, child := matches[0][1], matches[1][1]

	if output, err := runJaflow(t, binary, harness, "ticket", root, "#PARENT-123"); err != nil {
		t.Fatalf("set parent ticket: %v\n%s", err, output)
	}
	if output, err := runJaflow(t, binary, harness, "focus", "task", child); err != nil {
		t.Fatalf("focus child task: %v\n%s", err, output)
	}
	output, err = runJaflow(t, binary, harness, "status", "ticket")
	for _, expected := range []string{
		"[#PARENT-123] Root",
		"[#PARENT-123] Child",
		"Inherited ticket detected (#PARENT-123)",
	} {
		if err != nil || !strings.Contains(output, expected) {
			t.Fatalf("inherited ticket status = %q, err %v; want %q", output, err, expected)
		}
	}
	if output, err := runJaflow(t, binary, harness, "commit"); err != nil || !strings.Contains(output, "Refs: #PARENT-123") {
		t.Fatalf("inherited ticket commit = %q, err %v; want parent reference", output, err)
	}

	if output, err := runJaflow(t, binary, harness, "ticket", child, "#CHILD-456"); err != nil {
		t.Fatalf("set child ticket: %v\n%s", err, output)
	}
	output, err = runJaflow(t, binary, harness, "status", "ticket", "--force")
	if err != nil || !strings.Contains(output, "[#CHILD-456] Child") {
		t.Fatalf("direct ticket status = %q, err %v; want child ticket", output, err)
	}
}

func TestNoteAndContextContracts(t *testing.T) {
	binary := buildJaflow(t)
	harness := testharness.NewHarness(t, "project", "session")

	output, err := runJaflow(t, binary, harness, "plan", "context", "Root", "Middle", "Leaf")
	if err != nil {
		t.Fatalf("create context plan: %v\n%s", err, output)
	}
	matches := regexp.MustCompile(`Created task ([0-9a-f]{8})`).FindAllStringSubmatch(output, -1)
	if len(matches) != 3 {
		t.Fatalf("context plan output = %q, want three task UUIDs", output)
	}
	root, middle, leaf := matches[0][1], matches[1][1], matches[2][1]

	if output, err := runJaflow(t, binary, harness, "note", root, "decision", "Root decision"); err != nil {
		t.Fatalf("add decision note: %v\n%s", err, output)
	}
	if output, err := runJaflow(t, binary, harness, "note", middle, "research", "Middle research"); err != nil {
		t.Fatalf("add research note: %v\n%s", err, output)
	}
	if output, err := runJaflow(t, binary, harness, "note", root, "question", "Why root?"); err != nil {
		t.Fatalf("add question note: %v\n%s", err, output)
	}
	if output, err := runJaflow(t, binary, harness, "note", root, "hypothesis", "Root may explain it"); err != nil {
		t.Fatalf("add hypothesis note: %v\n%s", err, output)
	}

	notes, err := runJaflow(t, binary, harness, "notes", root)
	if err != nil || !strings.Contains(notes, "DECISION: Root decision") {
		t.Fatalf("notes output = %q, err %v; want decision annotation", notes, err)
	}
	timestampMatch := regexp.MustCompile(`\[([^]]+)\] DECISION: Root decision`).FindStringSubmatch(notes)
	if len(timestampMatch) != 2 {
		t.Fatalf("notes output = %q, want timestamped decision", notes)
	}

	contextOutput, err := runJaflow(t, binary, harness, "context", leaf)
	if err != nil || !strings.Contains(contextOutput, "Root") {
		t.Fatalf("context output = %q, err %v; want target context", contextOutput, err)
	}
	if output, err := runJaflow(t, binary, harness, "focus", "task", leaf); err != nil {
		t.Fatalf("focus leaf task: %v\n%s", err, output)
	}
	statusOutput, err := runJaflow(t, binary, harness, "status", "context")
	for _, expected := range []string{
		"INHERITED CONTEXT",
		"DECISION: Root decision",
		"RESEARCH: Middle research",
		"QUESTION: Why root?",
		"HYPOTHESIS: Root may explain it",
	} {
		if err != nil || !strings.Contains(statusOutput, expected) {
			t.Fatalf("status output = %q, err %v; want %q", statusOutput, err, expected)
		}
	}

	if output, err := runJaflow(t, binary, harness, "note", root, "delete", timestampMatch[1]); err != nil {
		t.Fatalf("delete annotation: %v\n%s", err, output)
	}
	remaining, err := runJaflow(t, binary, harness, "notes", root)
	if err != nil || strings.Contains(remaining, "Root decision") {
		t.Fatalf("remaining notes = %q, err %v; deleted annotation remains", remaining, err)
	}

	if output, err := runJaflow(t, binary, harness, "note", leaf, "note", "Direct context"); err != nil {
		t.Fatalf("add direct note: %v\n%s", err, output)
	}
	contextOutput, err = runJaflow(t, binary, harness, "context", leaf)
	if err != nil || !strings.Contains(contextOutput, "NOTE: Direct context") {
		t.Fatalf("direct context output = %q, err %v; want direct note", contextOutput, err)
	}
}

func TestNotesAllowCompletedTasksAndRejectInvalidKinds(t *testing.T) {
	binary := buildJaflow(t)
	harness := testharness.NewHarness(t, "project", "session")

	output, err := runJaflow(t, binary, harness, "plan", "completed-notes", "Task")
	if err != nil {
		t.Fatalf("create completed-notes plan: %v\n%s", err, output)
	}
	match := regexp.MustCompile(`Created task ([0-9a-f]{8})`).FindStringSubmatch(output)
	if len(match) != 2 {
		t.Fatalf("plan output = %q, want task UUID", output)
	}
	taskID := match[1]

	if output, err := runJaflow(t, binary, harness, "note", taskID, "invalid", "Message"); err == nil || !strings.Contains(output, "ACTION: Use one of the allowed semantic types") {
		t.Fatalf("invalid note output = %q, err %v; want actionable error", output, err)
	}
	if output, err := runJaflow(t, binary, harness, "execute", taskID); err != nil {
		t.Fatalf("execute task: %v\n%s", err, output)
	}
	if output, err := runJaflow(t, binary, harness, "outcome", taskID, "Completed"); err != nil {
		t.Fatalf("record outcome: %v\n%s", err, output)
	}
	if output, err := runJaflow(t, binary, harness, "done", taskID); err != nil {
		t.Fatalf("complete task: %v\n%s", err, output)
	}
	if output, err := runJaflow(t, binary, harness, "note", taskID, "note", "Post-completion note"); err != nil {
		t.Fatalf("add post-completion note: %v\n%s", err, output)
	}
	if output, err := runJaflow(t, binary, harness, "notes", taskID); err != nil || !strings.Contains(output, "NOTE: Post-completion note") {
		t.Fatalf("completed task notes = %q, err %v; want note", output, err)
	}
	if output, err := runJaflow(t, binary, harness, "ticket", taskID, "#LATE-1"); err == nil || !strings.Contains(output, "already COMPLETED") {
		t.Fatalf("completed task ticket = %q, err %v; want lifecycle protection", output, err)
	}
}

func TestNoteInvalidatesFocusedStatusCache(t *testing.T) {
	binary := buildJaflow(t)
	harness := testharness.NewHarness(t, "project", "session")

	output, err := runJaflow(t, binary, harness, "plan", "cache-notes", "Task")
	if err != nil {
		t.Fatalf("create cache-notes plan: %v\n%s", err, output)
	}
	match := regexp.MustCompile(`Created task ([0-9a-f]{8})`).FindStringSubmatch(output)
	if len(match) != 2 {
		t.Fatalf("plan output = %q, want task UUID", output)
	}
	taskID := match[1]
	if output, err := runJaflow(t, binary, harness, "focus", "task", taskID); err != nil {
		t.Fatalf("focus cache task: %v\n%s", err, output)
	}
	if output, err := runJaflow(t, binary, harness, "status", "cache-notes"); err != nil {
		t.Fatalf("prime status cache: %v\n%s", err, output)
	}
	if output, err := runJaflow(t, binary, harness, "note", taskID, "decision", "Cache changed"); err != nil {
		t.Fatalf("add cache note: %v\n%s", err, output)
	}
	output, err = runJaflow(t, binary, harness, "status", "cache-notes")
	if err != nil || strings.Contains(output, "[cached]") || !strings.Contains(output, "DECISION: Cache changed") {
		t.Fatalf("status after note = %q, err %v; want refreshed context", output, err)
	}
}
