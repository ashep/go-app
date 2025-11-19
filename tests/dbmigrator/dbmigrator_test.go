//go:build functest

package dbmigrator_test

import (
	"context"
	"embed"
	"testing"

	"github.com/ashep/go-app/dbmigrator"
	"github.com/ashep/go-app/testlogger"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
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

		l, lb := testlogger.New()

		res, err := dbmigrator.RunPostgres(dbURL, fs, "testdata", l)
		require.NoError(t, err)
		require.Equal(t, &dbmigrator.MigrationResult{
			PrevVersion: 0,
			NewVersion:  2,
		}, res)
		assert.NotContains(t, lb.Content(), "error", "there should be no error logs")

		// Run again, should be no changes
		res, err = dbmigrator.RunPostgres(dbURL, fs, "testdata", l)
		require.NoError(t, err)
		require.Equal(t, &dbmigrator.MigrationResult{
			PrevVersion: 2,
			NewVersion:  2,
		}, res)
		assert.NotContains(t, lb.Content(), "error", "there should be no error logs")
	})
}
