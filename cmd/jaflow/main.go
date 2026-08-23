package main

import (
	"errors"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/jacazul-ai/jaflow/internal/cli"
	"github.com/jacazul-ai/jaflow/internal/config"
	"github.com/jessevdk/go-flags"
)

func getVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
		return info.Main.Version
	}
	return "dev"
}

func main() {
	var opts config.AppOptions

	parser := flags.NewParser(&opts, flags.Default&^flags.PrintErrors)
	parser.Usage = "[Options] command"

	parser.CommandHandler = config.WithAppOptions(&opts)

	cli.RegisterCommands(parser)

	_, err := parser.Parse()
	if err != nil {
		if errors.Is(err, config.ErrVersionRequired) {
			fmt.Println(getVersion())
			os.Exit(0)
		}
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			parser.WriteHelp(os.Stdout)
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		fmt.Fprintln(os.Stderr, "ACTION: Review the command syntax or run 'jaflow help'.")
		parser.WriteHelp(os.Stderr)
		os.Exit(1)
	}
}
