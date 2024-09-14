package apprun_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/ashep/go-apprun"
	"github.com/ashep/go-apprun/option"
	"github.com/ashep/go-apprun/testlogger"
)

func TestRun_WithConfigFromFiles(tt *testing.T) {
	tt.Run("OK", func(t *testing.T) {
		l, _ := testlogger.New()
		cfg := cfgMock{}

		a := &appMock[cfgMock]{}
		defer a.AssertExpectations(t)

		fct := func() apprun.App[cfgMock] { return a }

		a.On("Run", mock.Anything, cfg, mock.Anything).
			Return(nil).
			Once()

		res := apprun.Run(fct, cfg, &l, option.WithConfigFromFiles[cfgMock]("anAppName"))
		assert.Equal(t, 0, res)
	})
}

func TestRun_WithConfigFromEnv(tt *testing.T) {
	tt.Run("OK", func(t *testing.T) {
		l, _ := testlogger.New()
		cfg := cfgMock{}

		a := &appMock[cfgMock]{}
		defer a.AssertExpectations(t)

		fct := func() apprun.App[cfgMock] { return a }

		t.Setenv("APP_THEKEY", "theValue")

		a.On("Run", mock.Anything, cfgMock{TheKey: "theValue"}, mock.Anything).
			Return(nil).
			Once()

		res := apprun.Run(fct, cfg, &l, option.WithConfigFromEnv[cfgMock]())
		assert.Equal(t, 0, res)
	})
}

func TestRun_WithHTTPLogger(tt *testing.T) {
	tt.Run("OK", func(t *testing.T) {
		l, _ := testlogger.New()
		cfg := cfgMock{}

		a := &appMock[cfgMock]{}
		defer a.AssertExpectations(t)

		fct := func() apprun.App[cfgMock] { return a }

		a.On("Run", mock.Anything, cfg, mock.Anything).
			Return(nil).
			Once()

		res := apprun.Run(fct, cfg, &l, option.WithHTTPLogger[cfgMock]("aTestURL", &httpClientMock{}))
		assert.Equal(t, 0, res)
	})
}

func TestRun_WithMetricsServer(tt *testing.T) {
	tt.Run("OK", func(t *testing.T) {
		l, lb := testlogger.New()
		cfg := cfgMock{}

		a := &appMock[cfgMock]{}
		defer a.AssertExpectations(t)

		fct := func() apprun.App[cfgMock] { return a }

		a.On("Run", mock.Anything, cfg, mock.Anything).
			Run(func(args mock.Arguments) {
			}).
			Return(nil).
			Once()

		res := apprun.Run(fct, cfg, &l, option.WithMetricsServer[cfgMock](":2112", "/metrics"))

		assert.Eventually(t, func() bool {
			return assert.Contains(t, lb.Content(), "metrics server stopped")
		}, time.Second, time.Millisecond*10)

		assert.Equal(t, 0, res)
		assert.Contains(t, lb.Content(), `{"level":"debug","addr":":2112","message":"metrics server starting"}`)
		assert.Contains(t, lb.Content(), `{"level":"debug","message":"metrics server shutting down"}`)
		assert.Contains(t, lb.Content(), `{"level":"debug","message":"metrics server shut down"}`)
		assert.Contains(t, lb.Content(), `{"level":"debug","message":"metrics server stopped"}`)
	})
}

type cfgMock struct {
	TheKey string `envconfig:"APP_THEKEY"`
}

type appMock[CT cfgMock] struct {
	mock.Mock
}

func (m *appMock[CT]) Run(ctx context.Context, cfg CT, l zerolog.Logger) error {
	return m.Called(ctx, cfg, l).Error(0)
}

type httpClientMock struct {
	mock.Mock
}

func (m *httpClientMock) Do(req *http.Request) (*http.Response, error) {
	args := m.Called(req)
	return args.Get(0).(*http.Response), args.Error(1)
}
