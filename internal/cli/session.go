package cli

import (
	"context"
	"fmt"

	"github.com/jacazul-ai/jaflow/internal/config"
)

// SessionCommand groups session inspection commands.
type SessionCommand struct {
	List SessionListCommand `command:"list" description:"List project sessions"`
	Show SessionShowCommand `command:"show" description:"Show the current session"`
}

// Execute requires a session subcommand.
func (cmd *SessionCommand) Execute(args []string) error {
	return fmt.Errorf("session requires a subcommand\nACTION: Run 'jaflow help session'.")
}

// SessionListCommand lists persisted sessions for the current project.
type SessionListCommand struct {
	appOpts *config.AppOptions
}

// SetAppOptions supplies project options to the command.
func (cmd *SessionListCommand) SetAppOptions(opts *config.AppOptions) {
	cmd.appOpts = opts
}

// Execute lists sessions and their anchors.
func (cmd *SessionListCommand) Execute(args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("session list accepts no arguments")
	}
	store, err := openStore(cmd.appOpts)
	if err != nil {
		return err
	}
	defer store.Close()
	sessions, err := store.ListSessions(context.Background(), cmd.appOpts.ProjectID)
	if err != nil {
		return err
	}
	if len(sessions) == 0 {
		fmt.Println("No sessions found.")
		return nil
	}
	fmt.Println("SESSIONS:")
	for _, session := range sessions {
		marker := " "
		if session.SessionID == cmd.appOpts.SessionID {
			marker = "*"
		}
		fmt.Printf("%s %s task:%s initiative:%s updated:%s\n",
			marker,
			session.SessionID,
			displayTaskID(session.FocusedTaskID),
			displayTaskID(session.InitiativeID),
			session.UpdatedAt,
		)
	}
	return nil
}

// SessionShowCommand shows the current session focus.
type SessionShowCommand struct {
	appOpts *config.AppOptions
}

// SetAppOptions supplies project options to the command.
func (cmd *SessionShowCommand) SetAppOptions(opts *config.AppOptions) {
	cmd.appOpts = opts
}

// Execute displays the current session focus.
func (cmd *SessionShowCommand) Execute(args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("session show accepts no arguments")
	}
	return (&FocusShowCommand{appOpts: cmd.appOpts}).Execute(nil)
}
