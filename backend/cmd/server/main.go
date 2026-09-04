package main

import (
	"fmt"
	"log"
	"os"

	httpapi "featureflag/internal/http"
	"featureflag/internal/service"
	"featureflag/internal/store"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	dsn := envOr("DATABASE_URL", "postgres://flaguser:flagpass@localhost:5433/featureflag?sslmode=disable")
	migrationsPath := envOr("MIGRATIONS_PATH", "file://migrations")
	addr := envOr("HTTP_ADDR", ":8080")

	if err := runMigrations(migrationsPath, dsn); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	st, err := store.Open(dsn)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer st.Close()

	svc := service.New(st)
	r := httpapi.NewRouter(svc)
	log.Printf("listening on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
	}
}

func runMigrations(path, dsn string) error {
	m, err := migrate.New(path, dsn)
	if err != nil {
		return err
	}
	defer m.Close()
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}
	v, dirty, _ := m.Version()
	fmt.Printf("migrations at version=%d dirty=%v\n", v, dirty)
	return nil
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
