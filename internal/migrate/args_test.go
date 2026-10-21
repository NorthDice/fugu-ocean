package migrate_test

import (
	"errors"
	"testing"

	"github.com/NorthDice/fugu-ocean/internal/migrate"
)

func TestParseArgs_DefaultsConfigAndDir(t *testing.T) {
	got, err := migrate.ParseArgs([]string{"up"})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v, want nil", err)
	}

	if got.ConfigPath != "config.yaml" || got.Dir != "migrations" || got.Command != "up" || len(got.CommandArgs) != 0 {
		t.Fatalf("ParseArgs() = %+v, want {ConfigPath:config.yaml Dir:migrations Command:up CommandArgs:[]}", got)
	}
}

func TestParseArgs_OverridesConfigAndDir(t *testing.T) {
	got, err := migrate.ParseArgs([]string{"-config", "other.yaml", "-dir", "db/migrations", "status"})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v, want nil", err)
	}

	if got.ConfigPath != "other.yaml" || got.Dir != "db/migrations" || got.Command != "status" || len(got.CommandArgs) != 0 {
		t.Fatalf("ParseArgs() = %+v, want {ConfigPath:other.yaml Dir:db/migrations Command:status CommandArgs:[]}", got)
	}
}

func TestParseArgs_PassesThroughCommandArguments(t *testing.T) {
	got, err := migrate.ParseArgs([]string{"create", "add_users_table", "sql"})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v, want nil", err)
	}

	if got.Command != "create" {
		t.Fatalf("Command = %q, want create", got.Command)
	}
	wantArgs := []string{"add_users_table", "sql"}
	if len(got.CommandArgs) != len(wantArgs) {
		t.Fatalf("CommandArgs = %v, want %v", got.CommandArgs, wantArgs)
	}
	for i := range wantArgs {
		if got.CommandArgs[i] != wantArgs[i] {
			t.Fatalf("CommandArgs = %v, want %v", got.CommandArgs, wantArgs)
		}
	}
}

func TestParseArgs_MissingCommandIsAnError(t *testing.T) {
	_, err := migrate.ParseArgs([]string{})
	if !errors.Is(err, migrate.ErrMissingCommand) {
		t.Fatalf("ParseArgs() error = %v, want ErrMissingCommand", err)
	}
}

func TestParseArgs_MissingCommandAfterFlagsIsAnError(t *testing.T) {
	_, err := migrate.ParseArgs([]string{"-config", "other.yaml"})
	if !errors.Is(err, migrate.ErrMissingCommand) {
		t.Fatalf("ParseArgs() error = %v, want ErrMissingCommand", err)
	}
}

func TestParseArgs_UnknownFlagIsAnError(t *testing.T) {
	_, err := migrate.ParseArgs([]string{"-bogus", "x", "up"})
	if err == nil {
		t.Fatal("ParseArgs() error = nil, want an error for an unknown flag")
	}
}
