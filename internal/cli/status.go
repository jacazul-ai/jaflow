package cli

import (
	"context"
	"fmt"

	"github.com/jacazul-ai/jaflow/internal/config"
	"github.com/jacazul-ai/jaflow/internal/storage/sqlite"
)

// StatusCommand displays pending tasks for the current project.
type StatusCommand struct {
	appOpts *config.AppOptions
}

// SetAppOptions supplies project-scoped runtime options to the command.
func (cmd *StatusCommand) SetAppOptions(opts *config.AppOptions) {
	cmd.appOpts = opts
}

// Execute prints tasks, optionally filtered by plan name.
func (cmd *StatusCommand) Execute(args []string) error {
	if len(args) > 1 {
		return fmt.Errorf("status accepts at most one plan name")
	}

	plan := ""
	if len(args) == 1 {
		plan = args[0]
	}
	store, err := sqlite.Open(context.Background(), cmd.appOpts.DatabasePath)
	if err != nil {
		return fmt.Errorf("open project database: %w", err)
	}
	defer store.Close()

	tasks, err := store.ListTasks(context.Background(), cmd.appOpts.ProjectID, plan)
	if err != nil {
		return err
	}
	if len(tasks) == 0 {
		fmt.Println("No tasks found.")
		return nil
	}

	fmt.Println("PENDING:")
	for _, current := range tasks {
		if current.Status != "pending" {
			continue
		}
		fmt.Printf("- [%s] %s\n", current.ID[:8], current.Description)
	}
	return nil
}
