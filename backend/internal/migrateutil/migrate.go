package migrateutil

import (
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// Up 执行迁移至最新版本。
func Up(migrationsPath, databaseURL string) (version uint, dirty bool, err error) {
	m, err := migrate.New(migrationsPath, databaseURL)
	if err != nil {
		return 0, false, err
	}
	defer m.Close()
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return 0, false, err
	}
	v, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return 0, dirty, err
	}
	fmt.Printf("migrations at version=%d dirty=%v\n", v, dirty)
	return v, dirty, nil
}
