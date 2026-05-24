package config

import (
	"errors"

	"github.com/jessevdk/go-flags"
)

var ErrVersionRequired = errors.New("version required")

type AppOptions struct {
	Verbose bool `short:"v" long:"verbose" description:"Enable verbose mode"`
	Version bool `short:"V" long:"version" description:"Show version"`
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
