package dbmigrator

import (
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	pgxdrv "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

type MigrationResult struct {
	PrevVersion uint
	NewVersion  uint
}

func RunPostgres(url string, fs embed.FS, path string) (*MigrationResult, error) {
	srcDrv, err := iofs.New(fs, path)
	if err != nil {
		return nil, fmt.Errorf("load migrations: %w", err)
	}

	dbDrv, err := (&pgxdrv.Postgres{}).Open(url)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	mig, err := migrate.NewWithInstance("iofs", srcDrv, "postgres", dbDrv)
	if err != nil {
		return nil, fmt.Errorf("init migrator: %w", err)
	}

	prevVer, dirty, err := mig.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		return nil, fmt.Errorf("get db version before migration: %w", err)
	}

	if dirty {
		return nil, fmt.Errorf("current version is dirty: %d", prevVer)
	}

	err = mig.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return nil, fmt.Errorf("apply migrations: %w", err)
	}

	newVer, _, err := mig.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		return nil, fmt.Errorf("get db version after migration: %w", err)
	}

	return &MigrationResult{
		PrevVersion: prevVer,
		NewVersion:  newVer,
	}, nil
}
