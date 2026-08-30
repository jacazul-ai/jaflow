package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/jacazul-ai/jaflow/internal/config"
	"github.com/jacazul-ai/jaflow/internal/task"
)

// FocusCommand groups focus navigation commands.
type FocusCommand struct {
	Show    FocusShowCommand        `command:"show" description:"Show the current focus"`
	Plan    FocusPlanCommand        `command:"plan" description:"Focus an initiative"`
	Task    FocusTaskCommand        `command:"task" description:"Focus a task"`
	Pop     FocusPopCommand         `command:"pop" description:"Pop the current task focus"`
	Clear   FocusClearCommand       `command:"clear" description:"Clear the session focus"`
	Back    FocusBackCommand        `command:"back" description:"Return to global focus"`
	Ind     FocusIndependentCommand `command:"ind" description:"Use independent session focus"`
	appOpts *config.AppOptions
}

// SetAppOptions supplies project options to all focus subcommands.
func (cmd *FocusCommand) SetAppOptions(opts *config.AppOptions) {
	cmd.appOpts = opts
	cmd.Show.SetAppOptions(opts)
	cmd.Plan.SetAppOptions(opts)
	cmd.Task.SetAppOptions(opts)
	cmd.Pop.SetAppOptions(opts)
	cmd.Clear.SetAppOptions(opts)
	cmd.Back.SetAppOptions(opts)
	cmd.Ind.SetAppOptions(opts)
}

// Execute requires a focus subcommand so the agent intent is explicit.
func (cmd *FocusCommand) Execute(args []string) error {
	return fmt.Errorf("focus requires a subcommand\nACTION: Run 'jaflow help focus'.")
}

// FocusIndependentCommand provides focus operations for an isolated session.
type FocusIndependentCommand struct {
	Plan    FocusPlanCommand `command:"plan" description:"Focus an initiative independently"`
	Task    FocusTaskCommand `command:"task" description:"Focus a task independently"`
	Args    []string         `positional-args:"yes"`
	appOpts *config.AppOptions
}

// SetAppOptions supplies project options to independent focus commands.
func (cmd *FocusIndependentCommand) SetAppOptions(opts *config.AppOptions) {
	cmd.appOpts = opts
	cmd.Plan.SetAppOptions(opts)
	cmd.Task.SetAppOptions(opts)
}

// Execute supports smart independent focus by initiative name.
func (cmd *FocusIndependentCommand) Execute(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("focus ind requires plan or task\nACTION: Run 'jaflow focus ind plan <name>' or 'jaflow focus ind task <uuid>'.")
	}
	return cmd.Plan.execute(args, true)
}

// FocusShowCommand displays the current session anchor.
type FocusShowCommand struct {
	appOpts *config.AppOptions
}

// SetAppOptions supplies project options to the command.
func (cmd *FocusShowCommand) SetAppOptions(opts *config.AppOptions) {
	cmd.appOpts = opts
}

// Execute renders current focus state.
func (cmd *FocusShowCommand) Execute(args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("focus show accepts no arguments")
	}
	store, err := openStore(cmd.appOpts)
	if err != nil {
		return err
	}
	defer store.Close()
	state, err := store.LoadFocus(context.Background(), cmd.appOpts.ProjectID, cmd.appOpts.SessionID)
	if err != nil {
		return err
	}
	printFocus(state)
	return nil
}

// FocusPlanCommand anchors a session to an initiative and its next ready task.
type FocusPlanCommand struct {
	appOpts *config.AppOptions
}

// SetAppOptions supplies project options to the command.
func (cmd *FocusPlanCommand) SetAppOptions(opts *config.AppOptions) {
	cmd.appOpts = opts
}

// Execute focuses an initiative and pushes its next ready task.
func (cmd *FocusPlanCommand) Execute(args []string) error {
	return cmd.execute(args, false)
}

func (cmd *FocusPlanCommand) execute(args []string, independent bool) error {
	if len(args) != 1 {
		return fmt.Errorf("focus plan requires one initiative name\nACTION: Run 'jaflow help focus'.")
	}
	store, err := openStore(cmd.appOpts)
	if err != nil {
		return err
	}
	defer store.Close()
	initiative, err := store.FindInitiative(context.Background(), cmd.appOpts.ProjectID, args[0])
	if err != nil {
		return err
	}
	state, err := store.LoadFocus(context.Background(), cmd.appOpts.ProjectID, cmd.appOpts.SessionID)
	if err != nil {
		return err
	}
	state.InitiativeID = initiative.ID
	state.FocusedTaskID = ""
	state.TaskStack = []task.FocusEntry{}
	ready, err := store.ReadyTasks(context.Background(), cmd.appOpts.ProjectID, initiative.Name)
	if err != nil {
		return err
	}
	if len(ready) > 0 {
		state.FocusedTaskID = ready[0].ID
		state.TaskStack = []task.FocusEntry{{
			TaskID:       ready[0].ID,
			InitiativeID: initiative.ID,
		}}
	}
	if err := store.SaveFocus(context.Background(), state); err != nil {
		return err
	}
	if err := clearTaskCaches(store, cmd.appOpts, task.Task{InitiativeName: initiative.Name}); err != nil {
		return err
	}
	if independent {
		fmt.Printf("Independent focus anchored to plan: %s\n", initiative.Name)
	} else {
		fmt.Printf("Focused initiative %s\n", initiative.Name)
	}
	if len(ready) > 0 {
		fmt.Printf("Next task %s: %s\n", shortID(ready[0].ID), ready[0].Description)
	}
	return nil
}

// FocusTaskCommand anchors a session to one task.
type FocusTaskCommand struct {
	appOpts *config.AppOptions
}

// SetAppOptions supplies project options to the command.
func (cmd *FocusTaskCommand) SetAppOptions(opts *config.AppOptions) {
	cmd.appOpts = opts
}

// Execute focuses a task and preserves its initiative anchor.
func (cmd *FocusTaskCommand) Execute(args []string) error {
	return cmd.execute(args, false)
}

func (cmd *FocusTaskCommand) execute(args []string, independent bool) error {
	taskID, err := oneTaskID("focus task", args)
	if err != nil {
		return err
	}
	store, err := openStore(cmd.appOpts)
	if err != nil {
		return err
	}
	defer store.Close()
	current, err := store.GetTask(context.Background(), taskID)
	if err != nil {
		return err
	}
	state, err := store.LoadFocus(context.Background(), cmd.appOpts.ProjectID, cmd.appOpts.SessionID)
	if err != nil {
		return err
	}
	state.InitiativeID = current.InitiativeID
	state.FocusedTaskID = current.ID
	state.TaskStack = pushFocus(state.TaskStack, task.FocusEntry{
		TaskID:       current.ID,
		InitiativeID: current.InitiativeID,
	})
	if err := store.SaveFocus(context.Background(), state); err != nil {
		return err
	}
	if err := clearTaskCaches(store, cmd.appOpts, current); err != nil {
		return err
	}
	fmt.Printf("Focused task %s: %s\n", shortID(current.ID), current.Description)
	return nil
}

// FocusPopCommand returns to the previous task in the session stack.
type FocusPopCommand struct {
	appOpts *config.AppOptions
}

// SetAppOptions supplies project options to the command.
func (cmd *FocusPopCommand) SetAppOptions(opts *config.AppOptions) {
	cmd.appOpts = opts
}

// Execute pops the current task anchor.
func (cmd *FocusPopCommand) Execute(args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("focus pop accepts no arguments")
	}
	store, err := openStore(cmd.appOpts)
	if err != nil {
		return err
	}
	defer store.Close()
	state, err := store.LoadFocus(context.Background(), cmd.appOpts.ProjectID, cmd.appOpts.SessionID)
	if err != nil {
		return err
	}
	if len(state.TaskStack) == 0 {
		return fmt.Errorf("focus stack is empty\nACTION: Run 'jaflow focus task <uuid>' or 'jaflow focus plan <name>'.")
	}
	state.TaskStack = state.TaskStack[1:]
	state.FocusedTaskID = ""
	if len(state.TaskStack) > 0 {
		state.FocusedTaskID = state.TaskStack[0].TaskID
		state.InitiativeID = state.TaskStack[0].InitiativeID
	}
	if err := store.SaveFocus(context.Background(), state); err != nil {
		return err
	}
	if state.FocusedTaskID == "" {
		if err := store.ClearCache(context.Background(), cmd.appOpts.ProjectID, cmd.appOpts.SessionID, ""); err != nil {
			return err
		}
	} else {
		current, err := store.GetTask(context.Background(), state.FocusedTaskID)
		if err != nil {
			return err
		}
		if err := clearTaskCaches(store, cmd.appOpts, current); err != nil {
			return err
		}
	}
	fmt.Printf("Focus popped; current task: %s\n", displayTaskID(state.FocusedTaskID))
	return nil
}

// FocusBackCommand exits the current independent session.
type FocusBackCommand struct {
	appOpts *config.AppOptions
}

// SetAppOptions supplies project options to the command.
func (cmd *FocusBackCommand) SetAppOptions(opts *config.AppOptions) {
	cmd.appOpts = opts
}

// Execute deletes the current non-global session and falls back to global focus.
func (cmd *FocusBackCommand) Execute(args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("focus back accepts no arguments")
	}
	if cmd.appOpts.SessionID == "global" {
		return fmt.Errorf("cannot leave the global focus\nACTION: Set JACAZUL_SESSION_ID before using 'jaflow focus back'.")
	}
	store, err := openStore(cmd.appOpts)
	if err != nil {
		return err
	}
	defer store.Close()
	if err := store.DeleteSession(context.Background(), cmd.appOpts.ProjectID, cmd.appOpts.SessionID); err != nil {
		return err
	}
	fmt.Println("Switched back to global focus")
	return nil
}

// FocusClearCommand clears the current session anchor.
type FocusClearCommand struct {
	appOpts *config.AppOptions
}

// SetAppOptions supplies project options to the command.
func (cmd *FocusClearCommand) SetAppOptions(opts *config.AppOptions) {
	cmd.appOpts = opts
}

// Execute clears the session focus without deleting the session.
func (cmd *FocusClearCommand) Execute(args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("focus clear accepts no arguments")
	}
	store, err := openStore(cmd.appOpts)
	if err != nil {
		return err
	}
	defer store.Close()
	state := task.FocusState{
		ProjectID: cmd.appOpts.ProjectID,
		SessionID: cmd.appOpts.SessionID,
		TaskStack: []task.FocusEntry{},
	}
	if err := store.SaveFocus(context.Background(), state); err != nil {
		return err
	}
	if err := store.ClearCache(context.Background(), cmd.appOpts.ProjectID, cmd.appOpts.SessionID, ""); err != nil {
		return err
	}
	fmt.Println("Focus cleared")
	return nil
}

func pushFocus(stack []task.FocusEntry, entry task.FocusEntry) []task.FocusEntry {
	filtered := make([]task.FocusEntry, 0, len(stack)+1)
	for _, current := range stack {
		if current.TaskID != entry.TaskID {
			filtered = append(filtered, current)
		}
	}
	return append([]task.FocusEntry{entry}, filtered...)
}

func printFocus(state task.FocusState) {
	fmt.Println("FOCUS")
	fmt.Printf("Project: %s\n", state.ProjectID)
	fmt.Printf("Session: %s\n", state.SessionID)
	fmt.Printf("Initiative: %s\n", displayTaskID(state.InitiativeID))
	fmt.Printf("Task: %s\n", displayTaskID(state.FocusedTaskID))
	fmt.Println("STACK:")
	for _, entry := range state.TaskStack {
		fmt.Printf("- %s\n", shortID(entry.TaskID))
	}
}

func displayTaskID(id string) string {
	if strings.TrimSpace(id) == "" {
		return "none"
	}
	return shortID(id)
}
