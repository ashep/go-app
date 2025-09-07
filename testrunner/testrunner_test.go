package testrunner_test

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/ashep/go-app/runner"
	"github.com/ashep/go-app/testrunner"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type appConfig struct {
	Foo string
}

type testApp struct {
	rt *runner.Runtime[appConfig]
	l  zerolog.Logger
}

func (a *testApp) Run(ctx context.Context) error {
	a.l.Info().Msg("app started")

	if a.rt.Config.Foo != "" {
		a.l.Info().Str("foo", a.rt.Config.Foo).Msg("config value")
	}

	if a.rt.Server != nil {
		<-a.rt.Server.Start(ctx)
	}

	a.l.Info().Msg("app stopped")

	return nil
}

func appFactory(rt *runner.Runtime[appConfig]) (*testApp, error) {
	return &testApp{
		rt: rt,
		l:  rt.Logger,
	}, nil
}

func TestRunner(main *testing.T) {
	main.Run("Ok", func(t *testing.T) {
		tr := testrunner.New(t, appFactory, &appConfig{Foo: "bar"})
		err := tr.Run()

		assert.NoError(t, err)
		assert.Equal(t, `{"level":"info","message":"app started"}
{"level":"info","foo":"bar","message":"config value"}
{"level":"info","message":"app stopped"}
`, tr.Logs())
	})

	main.Run("OkWithHTTPServer", func(t *testing.T) {
		tr := testrunner.New(t, appFactory, &appConfig{}).WithServer()
		tr.Server().HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok"))
		})
		tr.Start()

		res, err := http.Get(tr.ServerURL("/"))
		require.NoError(t, err)
		b, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Equal(t, "ok", string(b))

		assert.Equal(t, `{"level":"info","message":"app started"}
`, tr.Logs())
	})
}
