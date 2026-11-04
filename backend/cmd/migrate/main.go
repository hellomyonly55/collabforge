package main

import (
	"context"
	"os"
	"taskflow/internal/config"
	"taskflow/internal/infrastructure/database"
	"taskflow/internal/infrastructure/logger"
	"taskflow/migrations"
	"time"
)

func main() { os.Exit(run()) }
func run() int {
	log := logger.New()
	cfg, err := config.Load()
	if err != nil {
		log.Error("invalid configuration", "error", err.Error())
		return 1
	}
	db, err := database.Open(cfg.Database)
	if err != nil {
		log.Error("cannot initialize database client")
		return 1
	}
	defer db.SQL.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if err = migrations.Run(ctx, db.SQL); err != nil {
		log.Error("migration failed; check MySQL availability, permissions and migration SQL")
		return 1
	}
	log.Info("migrations applied", "version", migrations.LatestVersion)
	return 0
}
