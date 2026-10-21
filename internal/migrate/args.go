// Package migrate parses the arguments for the cmd/migrate CLI, kept
// separate from main so the parsing logic is unit-testable without a
// database.
package migrate

import (
	"errors"
	"flag"
)

// ErrMissingCommand is returned when no goose command (up, down, status, ...)
// was given.
var ErrMissingCommand = errors.New("migrate: missing command")

// Args is the parsed form of the CLI's arguments.
type Args struct {
	ConfigPath  string
	Dir         string
	Command     string
	CommandArgs []string
}

// ParseArgs parses args (typically os.Args[1:]) into Args.
func ParseArgs(args []string) (Args, error) {
	fs := flag.NewFlagSet("migrate", flag.ContinueOnError)
	configPath := fs.String("config", "config.yaml", "path to config file")
	dir := fs.String("dir", "migrations", "migrations directory")

	if err := fs.Parse(args); err != nil {
		return Args{}, err
	}

	rest := fs.Args()
	if len(rest) == 0 {
		return Args{}, ErrMissingCommand
	}

	return Args{
		ConfigPath:  *configPath,
		Dir:         *dir,
		Command:     rest[0],
		CommandArgs: rest[1:],
	}, nil
}
