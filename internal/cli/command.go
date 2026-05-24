package cli

import (
	"fmt"
	"os"

	"github.com/jessevdk/go-flags"
)

type HelpCommand struct {
	parser *flags.Parser
}

func NewHelpCommand(parser *flags.Parser) *HelpCommand {
	return &HelpCommand{parser: parser}
}

func (cmd *HelpCommand) Execute(args []string) error {
	fmt.Printf("%v\n\n", args)
	cmd.parser.WriteHelp(os.Stdout)
	return nil
}
