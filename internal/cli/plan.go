package cli

import (
	"context"
	"fmt"

	"github.com/jacazul-ai/jaflow/internal/config"
	"github.com/jacazul-ai/jaflow/internal/storage/sqlite"
	"github.com/jacazul-ai/jaflow/internal/task"
)

// PlanCommand creates a plan and its initial tasks.
type PlanCommand struct {
	appOpts *config.AppOptions
}

// SetAppOptions supplies project-scoped runtime options to the command.
func (cmd *PlanCommand) SetAppOptions(opts *config.AppOptions) {
	cmd.appOpts = opts
}

// Execute creates one pending task for each description after the plan name.
func (cmd *PlanCommand) Execute(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("plan requires a name and at least one task")
	}

	store, err := sqlite.Open(context.Background(), cmd.appOpts.DatabasePath)
	if err != nil {
		return fmt.Errorf("open project database: %w", err)
	}
	defer store.Close()

	initiative, err := store.GetOrCreateInitiative(context.Background(), task.CreateInitiativeInput{
		ProjectID: cmd.appOpts.ProjectID,
		Name:      args[0],
	})
	if err != nil {
		return err
	}

	var previous string
	for _, description := range args[1:] {
		var dependencies []string
		if previous != "" {
			dependencies = []string{previous}
		}
		created, err := store.CreateTask(context.Background(), task.CreateTaskInput{
			InitiativeID: initiative.ID,
			Description:  description,
			Dependencies: dependencies,
		})
		if err != nil {
			return err
		}
		fmt.Printf("Created task %s: %s\n", created.ID[:8], created.Description)
		previous = created.ID
	}
	return nil
}
