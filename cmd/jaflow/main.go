package main

import (
	"errors"
	"fmt"
	"os"
	"runtime/debug"
	"strings"

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
	os.Args = append([]string{os.Args[0]}, normalizeFocusAlias(os.Args[1:])...)

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

func normalizeFocusAlias(args []string) []string {
	result := append([]string(nil), args...)
	index := 0
	for index < len(result) {
		if strings.HasPrefix(result[index], "--project-id=") ||
			strings.HasPrefix(result[index], "--taskdata=") ||
			strings.HasPrefix(result[index], "--database-path=") ||
			strings.HasPrefix(result[index], "--session-id=") {
			index++
			continue
		}
		switch result[index] {
		case "-v", "--verbose", "-V", "--version":
			index++
		case "--project-id", "--taskdata", "--database-path", "--session-id":
			index += 2
		default:
			if result[index] != "focus" {
				return result
			}
			if index+1 == len(result) {
				return append(result, "show")
			}
			switch result[index+1] {
			case "show", "plan", "ini", "task", "pop", "clear", "back", "ind", "interest":
				return result
			default:
				if result[index+1] == "" || result[index+1][0] == '-' {
					return result
				}
				return append(result[:index+1], append([]string{"plan"}, result[index+1:]...)...)
			}
		}
	}
	return result
}
