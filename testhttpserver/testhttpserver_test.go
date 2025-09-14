package testhttpserver_test

import (
	"io"
	"net/http"
	"testing"

	"github.com/ashep/go-app/testhttpserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPServer(main *testing.T) {
	main.Run("Ok", func(t *testing.T) {
		s := testhttpserver.New(t)
		s.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTeapot)
			_, _ = w.Write([]byte("Hello, World!"))
		})
		s.Run()

		resp, err := http.Get(s.URL("/test"))
		require.NoError(t, err)
		defer func() {
			require.NoError(t, resp.Body.Close())
		}()

		calls := s.Calls("/test")
		require.Len(t, calls, 1)
		assert.Equal(t, "Go-http-client/1.1", calls[0].Header.Get("User-Agent"))
		assert.Equal(t, []byte{}, calls[0].Body)

		assert.Equal(t, http.StatusTeapot, resp.StatusCode)
		b, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, []byte("Hello, World!"), b)
	})
}
