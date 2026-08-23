package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jacazul-ai/jaflow/internal/config"
	"github.com/jacazul-ai/jaflow/internal/storage/sqlite"
	"github.com/jacazul-ai/jaflow/internal/task"
)

// PonderCommand renders the project-wide initiative dashboard.
type PonderCommand struct {
	All         bool `long:"all" description:"Include completed initiatives"`
	WithBacklog bool `long:"with-backlog" description:"Include backlog initiatives"`
	Force       bool `long:"force" description:"Bypass the dashboard cache"`
	appOpts     *config.AppOptions
}

// SetAppOptions supplies project options to the command.
func (cmd *PonderCommand) SetAppOptions(opts *config.AppOptions) {
	cmd.appOpts = opts
}

// Execute renders initiative counts for the current project.
func (cmd *PonderCommand) Execute(args []string) error {
	if len(args) > 1 {
		return fmt.Errorf("ponder accepts at most one project filter")
	}
	store, err := openStore(cmd.appOpts)
	if err != nil {
		return err
	}
	defer store.Close()
	return renderPonder(store, cmd.appOpts, cmd.All, cmd.WithBacklog, cmd.Force)
}

// PlansCommand lists initiatives using the same dashboard model as ponder.
type PlansCommand struct {
	All         bool `long:"all" description:"Include completed initiatives"`
	WithBacklog bool `long:"with-backlog" description:"Include backlog initiatives"`
	Force       bool `long:"force" description:"Bypass the dashboard cache"`
	appOpts     *config.AppOptions
}

// SetAppOptions supplies project options to the command.
func (cmd *PlansCommand) SetAppOptions(opts *config.AppOptions) {
	cmd.appOpts = opts
}

// Execute lists initiative summaries.
func (cmd *PlansCommand) Execute(args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("plans accepts no arguments")
	}
	store, err := openStore(cmd.appOpts)
	if err != nil {
		return err
	}
	defer store.Close()
	return renderPonder(store, cmd.appOpts, cmd.All, cmd.WithBacklog, cmd.Force)
}

// BacklogCommand hides an initiative from default dashboards.
type BacklogCommand struct {
	appOpts *config.AppOptions
}

// SetAppOptions supplies project options to the command.
func (cmd *BacklogCommand) SetAppOptions(opts *config.AppOptions) {
	cmd.appOpts = opts
}

// Execute moves an initiative to backlog.
func (cmd *BacklogCommand) Execute(args []string) error {
	return setInitiativeState(cmd.appOpts, args, task.InitiativeBacklog, "backlog")
}

// ActivateCommand restores an initiative from backlog.
type ActivateCommand struct {
	appOpts *config.AppOptions
}

// SetAppOptions supplies project options to the command.
func (cmd *ActivateCommand) SetAppOptions(opts *config.AppOptions) {
	cmd.appOpts = opts
}

// Execute activates an initiative.
func (cmd *ActivateCommand) Execute(args []string) error {
	return setInitiativeState(cmd.appOpts, args, task.InitiativeActive, "activate")
}

// TreeCommand renders dependency markers for an initiative.
type TreeCommand struct {
	appOpts *config.AppOptions
}

// SetAppOptions supplies project options to the command.
func (cmd *TreeCommand) SetAppOptions(opts *config.AppOptions) {
	cmd.appOpts = opts
}

// Execute renders a compact dependency tree view.
func (cmd *TreeCommand) Execute(args []string) error {
	if len(args) > 1 {
		return fmt.Errorf("tree accepts at most one initiative name")
	}
	initiative := ""
	if len(args) == 1 {
		initiative = args[0]
	}
	store, err := openStore(cmd.appOpts)
	if err != nil {
		return err
	}
	defer store.Close()
	tasks, err := store.ListTasks(context.Background(), cmd.appOpts.ProjectID, initiative)
	if err != nil {
		return err
	}
	if len(tasks) == 0 {
		fmt.Println("No tasks found.")
		return nil
	}
	states := make(map[string]task.Status, len(tasks))
	for _, current := range tasks {
		states[current.ID] = current.Status
	}
	for _, current := range tasks {
		marker := "READY"
		if current.Status == task.Completed {
			marker = "DONE"
		} else if current.Status == task.Active {
			marker = "ACTIVE"
		} else if !dependenciesReady(current, states) {
			marker = "BLOCKED"
		}
		fmt.Printf("[%s] %s %s\n", marker, shortID(current.ID), current.Description)
	}
	return nil
}

func renderPonder(store *sqlite.Store, opts *config.AppOptions, all bool, withBacklog bool, force bool) error {
	cacheKey := "ponder"
	if all {
		cacheKey += "_all"
	}
	if withBacklog {
		cacheKey += "_backlog"
	}
	if !force {
		_, found, err := store.GetCache(context.Background(), opts.ProjectID, opts.SessionID, cacheKey, time.Now().UTC())
		if err != nil {
			return err
		}
		if found {
			fmt.Println("🐊 [cached] Ponder unchanged. Use --force to refresh.")
			return nil
		}
	}
	summaries, err := store.ListInitiatives(context.Background(), opts.ProjectID, withBacklog, all)
	if err != nil {
		return err
	}
	output := renderInitiatives(opts.ProjectID, summaries)
	fmt.Print(output)
	return store.SetCache(context.Background(), opts.ProjectID, opts.SessionID, cacheKey, output, time.Now().UTC().Add(5*time.Minute))
}

func renderInitiatives(projectID string, summaries []task.InitiativeSummary) string {
	var output strings.Builder
	fmt.Fprintf(&output, "PROJECT: %s\n", projectID)
	if len(summaries) == 0 {
		output.WriteString("No initiatives found.\n")
		return output.String()
	}
	output.WriteString("INITIATIVES:\n")
	for _, summary := range summaries {
		if summary.Initiative.Status == task.InitiativeCompleted {
			continue
		}
		fmt.Fprintf(&output, "- [%s] %s pending:%d active:%d completed:%d blocked:%d\n",
			strings.ToUpper(string(summary.Initiative.Status)),
			summary.Initiative.Name,
			summary.Pending,
			summary.Active,
			summary.Completed,
			summary.Blocked,
		)
	}
	closedHeader := false
	for _, summary := range summaries {
		if summary.Initiative.Status != task.InitiativeCompleted {
			continue
		}
		if !closedHeader {
			output.WriteString("RECENTLY CLOSED:\n")
			closedHeader = true
		}
		fmt.Fprintf(&output, "- %s completed:%d\n", summary.Initiative.Name, summary.Completed)
	}
	return output.String()
}

func setInitiativeState(opts *config.AppOptions, args []string, state task.InitiativeStatus, command string) error {
	if len(args) != 1 {
		return fmt.Errorf("%s requires one initiative name\nACTION: Run 'jaflow help %s'.", command, command)
	}
	store, err := openStore(opts)
	if err != nil {
		return err
	}
	defer store.Close()
	if err := store.SetInitiativeStatus(context.Background(), opts.ProjectID, args[0], state); err != nil {
		return err
	}
	if err := store.ClearCache(context.Background(), opts.ProjectID, opts.SessionID, ""); err != nil {
		return err
	}
	fmt.Printf("Initiative %s: %s\n", args[0], command)
	return nil
}

func dependenciesReady(current task.Task, states map[string]task.Status) bool {
	for _, dependencyID := range current.Dependencies {
		if states[dependencyID] != task.Completed {
			return false
		}
	}
	return true
}
