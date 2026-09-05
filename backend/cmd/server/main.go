package main

import (
	"context"
	"log"
	"time"

	"featureflag/internal/config"
	"featureflag/internal/db"
	httpapi "featureflag/internal/http"
	"featureflag/internal/migrateutil"
)

func main() {
	cfg := config.Load()

	if _, _, err := migrateutil.Up(cfg.MigrationsPath, cfg.DatabaseURL); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	database, err := db.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer database.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := database.Ping(ctx); err != nil {
		log.Fatalf("db ping: %v", err)
	}

	r := httpapi.NewRouter()
	log.Printf("M1 server listening on %s (healthz=/healthz)", cfg.HTTPAddr)
	if err := r.Run(cfg.HTTPAddr); err != nil {
		log.Fatal(err)
	}
}
