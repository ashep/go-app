//go:build functest

package testpostgres_test

import (
	"context"
	"testing"

	"github.com/ashep/go-app/testpostgres"
	"github.com/stretchr/testify/require"
)

func TestPostgres(main *testing.T) {
	main.Run("Ok", func(t *testing.T) {
		tp := testpostgres.New(t)
		db := tp.DB()
		_, err := db.Exec(context.Background(), "SELECT 1;")
		require.NoError(t, err)
	})
}
