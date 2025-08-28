package httpserver_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/ashep/go-app/httpserver"
	"github.com/stretchr/testify/require"
)

func TestHTTPServer(main *testing.T) {
	main.Run("RunAndStop", func(t *testing.T) {
		srv := httpserver.New(httpserver.WithRandomLocalAddr())
		done := make(chan error)

		go func() {
			done <- srv.Run()
		}()

		require.NoError(t, srv.Stop(context.Background()))
		require.ErrorIs(t, <-done, http.ErrServerClosed)
	})

	main.Run("StartAndStop", func(t *testing.T) {
		srv := httpserver.New(httpserver.WithRandomLocalAddr())

		ctx, ctxC := context.WithCancel(context.Background())
		done := srv.Start(ctx)

		go func() {
			ctxC()
		}()

		require.ErrorIs(t, <-done, http.ErrServerClosed)
	})
}
