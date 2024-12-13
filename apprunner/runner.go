package apprunner

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ashep/go-cfgloader"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
)

var (
	appName = "" //nolint:gochecknoglobals // set externally
	appVer  = "" //nolint:gochecknoglobals // set externally
)

type Runtime struct {
	AppName    string
	AppVersion string
	Logger     zerolog.Logger
	SrvMux     *http.ServeMux
}

type runnable interface {
	Run(context.Context) error
}

type appFactory[CT any] func(cfg CT, rt *Runtime) (runnable, error)

type Runner[CT any] struct {
	cfg CT
	fct appFactory[CT]
	lw  []io.Writer
	srv *http.Server
	rt  *Runtime
}

func New[CT any](cfg CT, fct appFactory[CT]) *Runner[CT] {
	time.Local = time.UTC
	logLevel := zerolog.InfoLevel

	dbg := os.Getenv("APP_DEBUG")
	if dbg == "true" || dbg == "1" {
		logLevel = zerolog.DebugLevel
	}

	var logWriters []io.Writer
	if isTerminal() {
		logWriters = append(logWriters, zerolog.ConsoleWriter{Out: os.Stderr})
	} else {
		logWriters = append(logWriters, os.Stderr)
	}

	if appName == "" {
		appName = os.Getenv("APP_NAME")
	}

	if appVer == "" {
		appVer = os.Getenv("APP_VERSION")
	}

	l := zerolog.New(zerolog.MultiLevelWriter(logWriters...)).Level(logLevel).
		With().Str("app", appName).Str("app_v", appVer).Logger()

	return &Runner[CT]{
		cfg: cfg,
		fct: fct,
		lw:  logWriters,
		rt: Runtime{
			AppName:    appName,
			AppVersion: appVer,
			Logger:     l,
		},
	}
}

func (r *Runner[CT]) WithLogWriter(w io.Writer) *Runner[CT] {
	r.lw = append(r.lw, w)
	return r
}

func (r *Runner[CT]) WithHTTPServer(s *http.Server) *Runner[CT] {
	if r.srv != nil {
		panic("http server is already set")
	}

	r.rt.SrvMux = http.NewServeMux()
	r.srv = s

	return r
}

func (r *Runner[CT]) WithDefaultHTPServer() *Runner[CT] {
	addr := os.Getenv("APP_HTTP_SERVER_ADDR")
	if addr == "" {
		addr = ":9000"
	}

	return r.WithHTTPServer(&http.Server{
		Addr: addr,
	})
}

func (r *Runner[CT]) WithMetricsHandler() *Runner[CT] {
	if r.srv == nil {
		panic("http server is not set")
	}

	r.rt.SrvMux.Handle("/metrics", promhttp.Handler())

	return r
}

func (r *Runner[CT]) Run() int {
	for _, base := range []string{"config", appName} {
		for _, ext := range []string{".yaml", ".json"} {
			cfgPath := base + ext
			err := cfgloader.LoadFromPath(cfgPath, &r.cfg, nil)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				r.rt.Logger.Error().Err(err).Str("path", cfgPath).Msg("config file load failed")
				return 1
			} else if err == nil {
				r.rt.Logger.Debug().Str("path", cfgPath).Msg("config file loaded")
			}
		}
	}

	if cfgPath := os.Getenv("APP_CONFIG_PATH"); cfgPath != "" {
		if err := cfgloader.LoadFromPath(cfgPath, &r.cfg, nil); err != nil {
			r.rt.Logger.Error().Err(err).Str("path", cfgPath).Msg("config file load failed")
			return 1
		}

		r.rt.Logger.Debug().Str("path", cfgPath).Msg("config file loaded")
	}

	if err := cfgloader.LoadFromEnv("APP", &r.cfg); err != nil {
		r.rt.Logger.Error().Err(err).Msg("load config from env vars failed")
		return 1
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	app, err := r.fct(r.cfg, r.rt)
	if err != nil {
		r.rt.Logger.Error().Err(err).Msg("app init failed")
		return 1
	}

	ctx, ctxC := context.WithCancel(context.Background())
	defer ctxC()

	go func() {
		s := <-sig
		r.rt.Logger.Info().Str("signal", s.String()).Msg("signal received")
		ctxC()
	}()

	if r.srv != nil {
		go func() {
			r.rt.Logger.Info().Str("addr", r.srv.Addr).Msg("http server starting")

			if err := r.srv.ListenAndServe(); errors.Is(err, http.ErrServerClosed) {
				r.rt.Logger.Info().Msg("http server closed")
			} else if err != nil {
				r.rt.Logger.Error().Err(err).Msg("http server serve failed")
			}
		}()
	}

	if err := app.Run(ctx); err != nil {
		r.rt.Logger.Error().Err(err).Msg("app run failed")
		return 1
	}

	return 0
}

func isTerminal() bool {
	if o, _ := os.Stdout.Stat(); (o.Mode() & os.ModeCharDevice) == os.ModeCharDevice {
		return true
	}

	return false
}
