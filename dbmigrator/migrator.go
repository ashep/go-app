package dbmigrator

import (
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	pgxdrv "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

func RunPostgres(url string, fs embed.FS, path string) (uint, error) {
	srcDrv, err := iofs.New(fs, path)
	if err != nil {
		return 0, fmt.Errorf("load migrations: %w", err)
	}

	dbDrv, err := (&pgxdrv.Postgres{}).Open(url)
	if err != nil {
		return 0, fmt.Errorf("open db: %w", err)
	}

	mig, err := migrate.NewWithInstance("iofs", srcDrv, "postgres", dbDrv)
	if err != nil {
		return 0, fmt.Errorf("init migrator: %w", err)
	}

	ver, dirty, err := mig.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		return 0, fmt.Errorf("get db version before migration: %w", err)
	}

	if dirty {
		return ver, fmt.Errorf("current version is dirty: %d", ver)
	}

	err = mig.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return 0, fmt.Errorf("apply migrations: %w", err)
	}

	ver, _, err = mig.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		return 0, fmt.Errorf("get db version after migration: %w", err)
	}

	return ver, nil
}
