package testhttpserver_test

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/ashep/go-app/testhttpserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPServer(main *testing.T) {
	main.Run("Ok", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		s := testhttpserver.New()
		s.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTeapot)
			w.Write([]byte("Hello, World!"))
		})
		s.Start(ctx)
		defer s.Stop(ctx)

		resp, err := http.Get(s.BaseURL() + "/test")
		require.NoError(t, err)
		defer resp.Body.Close()

		calls := s.Calls("/test")
		require.Len(t, calls, 1)
		assert.Equal(t, "Go-http-client/1.1", calls[0].Header.Get("User-Agent"))
		assert.Equal(t, []byte{}, calls[0].Body)

		assert.Equal(t, http.StatusTeapot, resp.StatusCode)
		b, err := io.ReadAll(resp.Body)
		assert.Equal(t, []byte("Hello, World!"), b)
	})
}
