//go:build functest

package dbmigrator_test

import (
	"context"
	"embed"
	"testing"

	"github.com/ashep/go-app/dbmigrator"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

var (
	//go:embed testdata/*.sql
	fs embed.FS
)

const dbURL = "postgres://postgres:postgres@postgres:5432/postgres"

func TestDBMigrator(main *testing.T) {
	reset := func(t *testing.T) {
		ctx := context.Background()

		cn, err := pgx.Connect(ctx, dbURL)
		require.NoError(t, err, "failed to connect to postgres")

		_, err = cn.Exec(ctx, "DROP TABLE IF EXISTS test1;")
		require.NoError(t, err, "failed to drop table test1")

		_, err = cn.Exec(ctx, "DROP TABLE IF EXISTS test2;")
		require.NoError(t, err, "failed to drop table test2")

		_, err = cn.Exec(ctx, "DROP TABLE IF EXISTS schema_migrations;")
		require.NoError(t, err, "failed to drop test table")
	}

	main.Run("Success", func(t *testing.T) {
		reset(t)

		ver, err := dbmigrator.RunPostgres(dbURL, fs, "testdata")
		require.NoError(t, err)
		require.Equal(t, uint(2), ver)

		// Run again, should be no changes
		ver, err = dbmigrator.RunPostgres(dbURL, fs, "testdata")
		require.NoError(t, err)
		require.Equal(t, uint(2), ver)
	})
}
