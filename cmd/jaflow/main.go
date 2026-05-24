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

	parser.AddCommand(
		"help",
		"Display help",
		"Display help infomation about jaflow",
		cli.NewHelpCommand(parser),
	)

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
		fmt.Fprintln(os.Stderr, err)
		parser.WriteHelp(os.Stdout)
		os.Exit(0)
	}
}
