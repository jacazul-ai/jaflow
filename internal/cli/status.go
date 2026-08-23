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

// StatusCommand displays project task state.
type StatusCommand struct {
	PendingOnly bool `long:"pending" description:"Show pending tasks only"`
	Force       bool `long:"force" description:"Bypass the status cache"`
	appOpts     *config.AppOptions
}

// SetAppOptions supplies project-scoped runtime options to the command.
func (cmd *StatusCommand) SetAppOptions(opts *config.AppOptions) {
	cmd.appOpts = opts
}

// Execute prints tasks, optionally filtered by initiative name.
func (cmd *StatusCommand) Execute(args []string) error {
	if len(args) > 1 {
		return fmt.Errorf("status accepts at most one initiative name")
	}

	initiativeName := ""
	if len(args) == 1 {
		initiativeName = args[0]
	}
	store, err := sqlite.Open(context.Background(), cmd.appOpts.DatabasePath)
	if err != nil {
		return fmt.Errorf("open project database: %w", err)
	}
	defer store.Close()

	cacheKey := "status"
	if initiativeName != "" {
		cacheKey += "_" + initiativeName
	}
	if !cmd.Force && !cmd.PendingOnly {
		_, found, err := store.GetCache(
			context.Background(),
			cmd.appOpts.ProjectID,
			cmd.appOpts.SessionID,
			cacheKey,
			time.Now().UTC(),
		)
		if err != nil {
			return err
		}
		if found {
			fmt.Println("🐊 [cached] Status unchanged. Use --force to refresh.")
			return nil
		}
	}

	tasks, err := store.ListTasks(context.Background(), cmd.appOpts.ProjectID, initiativeName)
	if err != nil {
		return err
	}
	output := renderStatus(tasks, cmd.PendingOnly)
	fmt.Print(output)
	if !cmd.PendingOnly {
		if err := store.SetCache(
			context.Background(),
			cmd.appOpts.ProjectID,
			cmd.appOpts.SessionID,
			cacheKey,
			output,
			time.Now().UTC().Add(30*time.Second),
		); err != nil {
			return err
		}
	}
	return nil
}

func renderStatus(tasks []task.Task, pendingOnly bool) string {
	if len(tasks) == 0 {
		return "No tasks found.\n"
	}

	var output strings.Builder
	pending := 0
	completed := 0
	for _, current := range tasks {
		switch current.Status {
		case task.Pending, task.Active:
			pending++
		case task.Completed:
			completed++
		}
	}
	if pending > 0 {
		output.WriteString("PENDING:\n")
		for _, current := range tasks {
			if current.Status == task.Pending || current.Status == task.Active {
				fmt.Fprintf(&output, "- [%s] %s\n", shortID(current.ID), current.Description)
			}
		}
	}
	if !pendingOnly && completed > 0 {
		output.WriteString("COMPLETED:\n")
		for _, current := range tasks {
			if current.Status == task.Completed {
				fmt.Fprintf(&output, "- [%s] %s\n", shortID(current.ID), current.Description)
			}
		}
	}
	if output.Len() == 0 {
		return "No tasks found.\n"
	}
	return output.String()
}
