package metrics_test

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/ashep/go-app/metrics"
	"github.com/ashep/go-app/testhttpserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterServer(main *testing.T) {
	main.Run("Ok", func(t *testing.T) {
		s := testhttpserver.New(t)
		metrics.RegisterServer("an-app", "1.2.3", s)
		s.HandleFunc("/foo", func(w http.ResponseWriter, r *http.Request) {
			metrics.MeasureHTTPServerRequest(r, "/foo")(http.StatusOK)
			w.WriteHeader(http.StatusOK)
		})
		s.Run()

		_, err := http.Get(s.URL("/foo"))
		require.NoError(t, err)

		res, err := http.Get(s.URL(metrics.URLPath))
		require.NoError(t, err)

		b, err := io.ReadAll(res.Body)
		require.NoError(t, err)

		host := strings.TrimPrefix(s.BaseURL(), "http://")
		assert.Contains(t, string(b), `http_server_request_duration_seconds_count{app="an-app",app_v="1.2.3",code="200",host="`+host+`",method="GET",path="/foo"} 1`)
	})
}
