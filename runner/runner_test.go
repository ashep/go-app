package runner_test

import (
	"net/http"
	"testing"

	"github.com/ashep/go-app/runner"
	"github.com/ashep/go-app/testhttpserver"
	"github.com/ashep/go-app/testlogger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type runCfg struct {
	Key1 string
	Key2 string `env:"KEY2"`
	Key3 string
}

func TestRunner(main *testing.T) {
	main.Run("Ok", func(t *testing.T) {
		l, lb := testlogger.New()
		lSrv := newLogServerMock(t)

		t.Setenv("APP_KEY2", "Key2Value")
		t.Setenv("FOO_BAR_KEY3", "Key3Value")
		t.Setenv("APP_LOGSERVER_URL", lSrv.URL("/log"))
		t.Setenv("APP_LOGSERVER_USERNAME", "aLogServerUsername")
		t.Setenv("APP_LOGSERVER_PASSWORD", "aLogServerPassword")

		runner.New(newRunMock(t)).
			SetAppName("foo-bar").
			SetAppVersion("1.2.3").
			SetConfig(runCfg{
				Key1: "Key1Value",
			}).
			LoadEnvConfig().
			AddLogWriter(l).
			AddHTTPLogWriter().
			Run()

		assert.Equal(t, `{"app":"foo-bar","app_v":"1.2.3","level":"info","message":"test log message"}
`, lb.Content())

		lSrvCalls := lSrv.Calls("/log")
		require.Equal(t, 1, len(lSrvCalls))
		assert.Equal(t, "Basic YUxvZ1NlcnZlclVzZXJuYW1lOmFMb2dTZXJ2ZXJQYXNzd29yZA==", lSrvCalls[0].Header.Get("Authorization"))
		assert.Equal(t, []byte(`{"level":"info","app":"foo-bar","app_v":"1.2.3","message":"test log message"}`+"\n"), lSrvCalls[0].Body)
	})
}

func newRunMock(t *testing.T) func(rt *runner.Runtime[runCfg]) error {
	return func(rt *runner.Runtime[runCfg]) error {
		rt.Log.Info().Msg("test log message")

		assert.Equal(t, "foo-bar", rt.AppName)
		assert.Equal(t, "FOO_BAR", rt.AppName2)
		assert.Equal(t, "1.2.3", rt.AppVersion)

		assert.Equal(t, "Key1Value", rt.Cfg.Key1)
		assert.Equal(t, "Key2Value", rt.Cfg.Key2)
		assert.Equal(t, "Key3Value", rt.Cfg.Key3)

		return nil
	}
}

func newLogServerMock(t *testing.T) *testhttpserver.Server {
	s := testhttpserver.New(t)
	s.HandleFunc("/log", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})
	s.Run()
	return s
}
