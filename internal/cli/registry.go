package cli

import "github.com/jessevdk/go-flags"

// RegisterCommands registers the CLI command tree from the help catalog.
func RegisterCommands(parser *flags.Parser) {
	registerCommand(parser, "help", NewHelpCommand(parser))
	registerCommand(parser, "plan", &PlanCommand{})
	registerCommand(parser, "status", &StatusCommand{})
	registerCommand(parser, "execute", &ExecuteCommand{})
	registerCommand(parser, "outcome", &OutcomeCommand{})
	registerCommand(parser, "done", &DoneCommand{})
	registerCommand(parser, "reopen", &ReopenCommand{})
	registerCommand(parser, "discard", &DiscardCommand{})
	registerCommand(parser, "focus", &FocusCommand{})
	registerCommand(parser, "session", &SessionCommand{})
}

func registerCommand(parser *flags.Parser, name string, command flags.Commander) {
	entry, ok := findHelpEntry(name)
	if !ok {
		panic("command is missing from help catalog: " + name)
	}
	parser.AddCommand(name, entry.summary, entry.role, command)
}
