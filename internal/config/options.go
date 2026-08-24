package config

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/jessevdk/go-flags"
)

var ErrVersionRequired = errors.New("version required")

type AppOptions struct {
	Verbose      bool   `short:"v" long:"verbose" description:"Enable verbose mode"`
	Version      bool   `short:"V" long:"version" description:"Show version"`
	ProjectID    string `long:"project-id" description:"Project identity"`
	TaskData     string `long:"taskdata" description:"Legacy Taskwarrior data directory"`
	DatabasePath string `long:"database-path" env:"JAFLOW_DATABASE_PATH" description:"Project SQLite database path"`
	SessionID    string `long:"session-id" env:"JACAZUL_SESSION_ID" description:"Workflow session identity"`
}

type AppOptionsAware interface {
	SetAppOptions(opts *AppOptions)
}

type AppOptionsFunc func(opts *AppOptions) error

func WithAppOptions(opts *AppOptions, fns ...AppOptionsFunc) func(
	cmd flags.Commander, args []string) error {
	return func(cmd flags.Commander, args []string) error {
		if opts.Version {
			return ErrVersionRequired
		}
		if err := Resolve(opts); err != nil {
			return err
		}

		if len(fns) > 0 {
			for _, fn := range fns {
				if err := fn(opts); err != nil {
					return err
				}
			}
		}

		if aware, ok := cmd.(AppOptionsAware); ok == true {
			aware.SetAppOptions(opts)
		}
		return cmd.Execute(args)
	}
}

// Resolve fills project-scoped options from the runtime environment.
func Resolve(opts *AppOptions) error {
	if opts.ProjectID == "" {
		opts.ProjectID = os.Getenv("PROJECT_ID")
	}
	if opts.ProjectID == "" {
		opts.ProjectID = "global"
	}
	if opts.TaskData == "" {
		opts.TaskData = os.Getenv("TASKDATA")
	}
	home, err := runtimeHome()
	if err != nil {
		return err
	}
	if opts.TaskData == "" {
		opts.TaskData = filepath.Join(home, ".task", opts.ProjectID)
	}
	if opts.SessionID == "" {
		opts.SessionID = os.Getenv("JACAZUL_SESSION_ID")
	}
	if opts.SessionID == "" {
		opts.SessionID = "global"
	}
	if opts.DatabasePath == "" {
		opts.DatabasePath = filepath.Join(
			home,
			"jaflow",
			opts.ProjectID,
			"jaflow.sqlite3",
		)
	}
	return nil
}

func runtimeHome() (string, error) {
	if home := os.Getenv("JACAZUL_HOME"); home != "" {
		return home, nil
	}
	return os.UserHomeDir()
}
