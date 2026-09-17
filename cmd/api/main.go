package main

import (
	"log"

	"github.com/NorthDice/fugu-ocean/internal/config"
)

func main() {
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("config loaded: server port=%s, postgres db=%s", cfg.Server.Port, cfg.Postgres.DBName)
}
