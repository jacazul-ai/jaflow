package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jessevdk/go-flags"
)

// HelpCommand renders agent-facing workflow guidance.
type HelpCommand struct {
	parser *flags.Parser
}

// NewHelpCommand creates a help command backed by the root parser.
func NewHelpCommand(parser *flags.Parser) *HelpCommand {
	return &HelpCommand{parser: parser}
}

// Execute prints root help or the operational brief for one command.
func (cmd *HelpCommand) Execute(args []string) error {
	if len(args) > 1 {
		return fmt.Errorf("help accepts at most one command name")
	}

	command := ""
	if len(args) == 1 {
		command = args[0]
	}
	if command != "" && command != "help" {
		entry, ok := findHelpEntry(command)
		if !ok {
			return fmt.Errorf("unknown help topic %q; use 'jaflow help' to list commands", command)
		}
		writeCommandHelp(os.Stdout, entry)
		return nil
	}

	writeRootHelp(os.Stdout)
	if cmd.parser != nil {
		fmt.Fprintln(os.Stdout, "Parser options:")
		cmd.parser.WriteHelp(os.Stdout)
	}
	return nil
}

type helpEntry struct {
	name          string
	summary       string
	usage         string
	role          string
	preconditions []string
	effects       []string
	examples      []string
	next          string
}

var helpEntries = []helpEntry{
	{
		name:    "help",
		summary: "Show the agent workflow briefing",
		usage:   "jaflow help [command]",
		role:    "Use this before acting when the command contract or next step is unclear.",
		effects: []string{
			"Reads command metadata and renders guidance; it does not change workflow state.",
		},
		examples: []string{"jaflow help", "jaflow help plan"},
		next:     "Run 'jaflow status' to inspect the current project state.",
	},
	{
		name:    "plan",
		summary: "Create an initiative and chained tasks",
		usage:   "jaflow plan <initiative> <task> [<task>...]",
		role:    "Use this to create a first-class initiative and its ordered work chain.",
		preconditions: []string{
			"A project identity must resolve from --project-id or PROJECT_ID.",
			"Provide an initiative name and at least one non-empty task description.",
		},
		effects: []string{
			"Creates the initiative in the project's SQLite database when absent.",
			"Creates one task per description; each task after the first depends on the previous task.",
			"Prints short UUIDs for the created tasks; full UUIDs remain the persisted identity.",
		},
		examples: []string{
			"jaflow plan parity 'Define schema' 'Implement store' 'Add tests'",
			"jaflow plan parity --database-path /tmp/parity.sqlite 'Define schema'",
		},
		next: "Run 'jaflow status <initiative>' to inspect pending work, then execute the first ready task.",
	},
	{
		name:    "status",
		summary: "Show pending tasks for the current project",
		usage:   "jaflow status [initiative]",
		role:    "Use this as the hands-on view before switching focus or starting work.",
		preconditions: []string{
			"The project database is selected from --database-path or PROJECT_ID.",
		},
		effects: []string{
			"Reads only the selected project's task state; another project's database is not queried.",
			"An empty database returns a quiet 'No tasks found.' result.",
			"Pending tasks are printed with short UUIDs and descriptions.",
		},
		examples: []string{"jaflow status", "jaflow status parity"},
		next:     "Choose the first ready task in the initiative chain; do not skip a blocking dependency.",
	},
	{
		name:    "execute",
		summary: "Start a ready task",
		usage:   "jaflow execute <task-uuid>",
		role:    "Use this to begin work on the next task whose dependencies are complete.",
		preconditions: []string{
			"The task must exist in the selected project database.",
			"Every dependency must be completed; blocked tasks fail with an ACTION.",
		},
		effects: []string{
			"Moves the task from pending to active and records its start time.",
			"Does not silently switch to another initiative.",
		},
		examples: []string{"jaflow execute 57c3fc80"},
		next:     "Do the work, then run 'jaflow outcome <uuid> <message>' before done.",
	},
	{
		name:    "outcome",
		summary: "Record the result required for completion",
		usage:   "jaflow outcome <task-uuid> <message...>",
		role:    "Use this to persist the result and handoff context before closing a task.",
		effects: []string{
			"Stores the outcome on the task and adds an OUTCOME annotation.",
			"Does not complete the task by itself.",
		},
		examples: []string{"jaflow outcome 57c3fc80 'Schema contract implemented'"},
		next:     "Run 'jaflow done <uuid>' after the outcome is recorded.",
	},
	{
		name:    "done",
		summary: "Complete a task and expose ready work",
		usage:   "jaflow done <task-uuid>",
		role:    "Use this only after the task outcome is recorded.",
		preconditions: []string{
			"The task must not already be completed.",
			"An OUTCOME record is mandatory.",
		},
		effects: []string{
			"Marks the task completed and reports tasks newly unblocked in the same initiative.",
			"Preserves the dependency chain for the next focus switch.",
		},
		examples: []string{"jaflow done 57c3fc80"},
		next:     "Switch focus to the reported ready task, then run 'jaflow execute <uuid>'.",
	},
	{
		name:    "reopen",
		summary: "Return a completed task to pending",
		usage:   "jaflow reopen <task-uuid>",
		role:    "Use this when more work is required on a completed task.",
		effects: []string{
			"Moves the task back to pending and clears completion disposition.",
			"Keeps the recorded outcome as historical context.",
		},
		examples: []string{"jaflow reopen 57c3fc80"},
		next:     "Run 'jaflow execute <uuid>' after its dependencies are ready.",
	},
	{
		name:    "discard",
		summary: "Discard a task with an audit outcome",
		usage:   "jaflow discard <task-uuid>",
		role:    "Use this instead of manually deleting or marking a task discarded.",
		effects: []string{
			"Completes the task with a discarded disposition.",
			"Adds an auditable OUTCOME record explaining the discard.",
		},
		examples: []string{"jaflow discard 57c3fc80"},
		next:     "Run 'jaflow status' to inspect the remaining initiative chain.",
	},
}

func findHelpEntry(name string) (helpEntry, bool) {
	for _, entry := range helpEntries {
		if entry.name == name {
			return entry, true
		}
	}
	return helpEntry{}, false
}

func writeRootHelp(writer io.Writer) {
	fmt.Fprintln(writer, "jaflow — local project workflow engine")
	fmt.Fprintln(writer, "")
	fmt.Fprintln(writer, "ROLE")
	fmt.Fprintln(writer, "  Preserve initiatives, chained tasks, dependencies, focus, and context.")
	fmt.Fprintln(writer, "  The current command operates on one project database at a time.")
	fmt.Fprintln(writer, "")
	fmt.Fprintln(writer, "WORKFLOW")
	fmt.Fprintln(writer, "  orient → plan → test → execute → record outcome → done → switch focus")
	fmt.Fprintln(writer, "  A task with an unfinished dependency is blocked; completion exposes the next ready task.")
	fmt.Fprintln(writer, "")
	fmt.Fprintln(writer, "COMMANDS")
	for _, entry := range helpEntries {
		fmt.Fprintf(writer, "  %-8s %s\n", entry.name, entry.summary)
	}
	fmt.Fprintln(writer, "")
	fmt.Fprintln(writer, "AGENT RULES")
	fmt.Fprintln(writer, "  Use short UUIDs for display, but resolve and persist full UUIDs.")
	fmt.Fprintln(writer, "  Read status before acting. Treat errors and ACTION lines as workflow guidance.")
	fmt.Fprintln(writer, "  Use 'jaflow help <command>' for prerequisites, effects, examples, and next action.")
	fmt.Fprintln(writer, "")
	fmt.Fprintln(writer, "NEXT")
	fmt.Fprintln(writer, "  Run 'jaflow status' to see the current project's pending work.")
}

func writeCommandHelp(writer io.Writer, entry helpEntry) {
	fmt.Fprintf(writer, "jaflow %s — %s\n\n", entry.name, entry.summary)
	fmt.Fprintf(writer, "USAGE\n  %s\n\n", entry.usage)
	fmt.Fprintf(writer, "ROLE\n  %s\n\n", entry.role)
	writeList(writer, "PREREQUISITES", entry.preconditions)
	writeList(writer, "SIDE EFFECTS AND OUTPUT", entry.effects)
	writeList(writer, "EXAMPLES", entry.examples)
	fmt.Fprintf(writer, "NEXT ACTION\n  %s\n", entry.next)
}

func writeList(writer io.Writer, heading string, values []string) {
	if len(values) == 0 {
		return
	}
	fmt.Fprintf(writer, "%s\n", heading)
	for _, value := range values {
		fmt.Fprintf(writer, "  - %s\n", strings.TrimSpace(value))
	}
	fmt.Fprintln(writer)
}
