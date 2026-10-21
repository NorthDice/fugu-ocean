// Command migrate applies goose migrations from the migrations/ directory
// against the database configured in config.yaml (or the path given via
// -config), using the same env-var overrides as the API binary.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" database/sql driver
	"github.com/pressly/goose/v3"

	"github.com/NorthDice/fugu-ocean/internal/config"
	"github.com/NorthDice/fugu-ocean/internal/migrate"
	"github.com/NorthDice/fugu-ocean/internal/platform/postgres"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}

func run(osArgs []string) error {
	args, err := migrate.ParseArgs(osArgs)
	if err != nil {
		return fmt.Errorf("parse arguments: %w", err)
	}

	cfg, err := config.Load(args.ConfigPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	db, err := sql.Open("pgx", postgres.DSN(cfg.Postgres))
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer func() {
		if cerr := db.Close(); cerr != nil {
			log.Printf("migrate: close database: %v", cerr)
		}
	}()

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set dialect: %w", err)
	}

	if err := goose.RunContext(context.Background(), args.Command, db, args.Dir, args.CommandArgs...); err != nil {
		return fmt.Errorf("run %s: %w", args.Command, err)
	}

	return nil
}
