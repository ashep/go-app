package testrunner

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/ashep/go-app/runner"
	"github.com/ashep/go-app/testhttpserver"
	"github.com/ashep/go-app/testlogger"
	"github.com/stretchr/testify/require"
)

type TestRunner[RT runner.Runnable, CT any] struct {
	t       *testing.T
	app     RT
	runtime *runner.Runtime[CT]
	lb      *testlogger.TestWriter
}

type appFactory[RT runner.Runnable, CT any] func(rt *runner.Runtime[CT]) (RT, error)

func New[RT runner.Runnable, CT any](
	t *testing.T,
	f appFactory[RT, CT],
	cfg *CT,
) *TestRunner[RT, CT] {
	if appCfgT, ok := any(cfg).(runner.Validatable); ok {
		require.NoError(t, appCfgT.Validate(), "config validation failed")
	}

	l, lb := testlogger.New()

	rt := &runner.Runtime[CT]{
		AppName:    "test",
		AppVersion: "test",
		Config:     cfg,
		Server:     nil,
		Logger:     l,
	}

	app, err := f(rt)
	require.NoError(t, err)

	return &TestRunner[RT, CT]{
		t:       t,
		app:     app,
		runtime: rt,
		lb:      lb,
	}
}

func (tr *TestRunner[RT, CT]) WithServer() *TestRunner[RT, CT] {
	require.Nil(tr.t, tr.runtime.Server, "http server is already enabled")
	tr.runtime.Server = testhttpserver.New()
	return tr
}

func (tr *TestRunner[RT, CT]) Server() *testhttpserver.Server {
	require.NotNil(tr.t, tr.runtime.Server, "http server is not enabled")
	return tr.runtime.Server.(*testhttpserver.Server)
}

func (tr *TestRunner[RT, CT]) ServerURL(path string) string {
	require.NotNil(tr.t, tr.runtime.Server, "http server is not enabled")
	return tr.runtime.Server.(*testhttpserver.Server).BaseURL() + path
}

func (tr *TestRunner[RT, CT]) Run() error {
	return tr.app.Run(context.Background())
}

func (tr *TestRunner[RT, CT]) RunContext(ctx context.Context) error {
	return tr.app.Run(ctx)
}

func (tr *TestRunner[RT, CT]) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		_ = tr.app.Run(ctx)
	}()

	if tr.runtime.Server != nil {
		require.Eventually(tr.t, func() bool {
			_, err := http.Get(tr.ServerURL("/"))
			return err == nil
		}, time.Second, 100*time.Millisecond, "server had not started within 1 second")
	}

	tr.t.Cleanup(func() {
		cancel()
		if tr.runtime.Server != nil {
			require.Eventually(tr.t, func() bool {
				_, err := http.Get(tr.ServerURL("/"))
				return err != nil
			}, time.Second, 100*time.Millisecond, "server had not stopped within 1 second")
		}
	})
}

func (tr *TestRunner[RT, CT]) Logs() string {
	return tr.lb.Content()
}
