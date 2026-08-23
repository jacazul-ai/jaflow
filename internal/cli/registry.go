package cli

import "github.com/jessevdk/go-flags"

// RegisterCommands registers the CLI command tree from the help catalog.
func RegisterCommands(parser *flags.Parser) {
	registerCommand(parser, "help", NewHelpCommand(parser))
	registerCommand(parser, "plan", &PlanCommand{})
	registerCommand(parser, "status", &StatusCommand{})
}

func registerCommand(parser *flags.Parser, name string, command flags.Commander) {
	entry, ok := findHelpEntry(name)
	if !ok {
		panic("command is missing from help catalog: " + name)
	}
	parser.AddCommand(name, entry.summary, entry.role, command)
}
