package runner

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ashep/go-app/httplogwriter"
	"github.com/ashep/go-app/metrics"
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
	SrvMux     *http.ServeMux
	Logger     zerolog.Logger
}

type Runnable interface {
	Run(context.Context) error
}

type appFactory[RT Runnable, CT any] func(cfg CT, rt *Runtime) (RT, error)

type Runner[RT Runnable, CT any] struct {
	appName    string
	appVer     string
	cfg        CT
	appFactory appFactory[RT, CT]
	srvMux     *http.ServeMux
	srv        *http.Server
	logLevel   zerolog.Level
	logWriters []io.Writer
	bsLog      zerolog.Logger // bootstrap logger, used until Run called
}

func New[RT Runnable, CT any](f appFactory[RT, CT], cfg CT) *Runner[RT, CT] {
	time.Local = time.UTC
	logLevel := zerolog.InfoLevel

	if appName == "" {
		appName = os.Getenv("APP_NAME")
	}

	if appVer == "" {
		appVer = os.Getenv("APP_VERSION")
	}

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

	bsLog := zerolog.New(zerolog.MultiLevelWriter(logWriters...)).Level(logLevel).
		With().Str("app", appName).Str("app_v", appVer).Logger()

	return &Runner[RT, CT]{
		appName:    appName,
		appVer:     appVer,
		cfg:        cfg,
		appFactory: f,
		logLevel:   logLevel,
		logWriters: logWriters,
		bsLog:      bsLog,
	}
}

func (r *Runner[RT, CT]) WithLogWriter(w io.Writer) *Runner[RT, CT] {
	r.logWriters = append(r.logWriters, w)
	return r
}

func (r *Runner[RT, CT]) WithDefaultHTTPLogWriter(must bool) *Runner[RT, CT] {
	w, err := httplogwriter.NewFromEnv()
	if err != nil {
		if must {
			r.bsLog.Error().Err(err).Msg("error setting up http log writer")
			os.Exit(1)
		}
		r.bsLog.Warn().Err(err).Msg("http log writer has not been set up")
		return r
	}

	return r.WithLogWriter(w)
}

func (r *Runner[RT, CT]) WithHTTPServer(s *http.Server) *Runner[RT, CT] {
	if r.srv != nil {
		r.bsLog.Error().Msg("http server is already set")
		os.Exit(1)
	}

	r.srvMux = http.NewServeMux()
	s.Handler = r.srvMux
	r.srv = s

	return r
}

func (r *Runner[RT, CT]) WithDefaultHTPServer() *Runner[RT, CT] {
	addr := os.Getenv("APP_HTTP_SERVER_ADDR")
	if addr == "" {
		addr = ":9000"
	}

	return r.WithHTTPServer(&http.Server{
		Addr: addr,
	})
}

func (r *Runner[RT, CT]) WithMetricsHandler() *Runner[RT, CT] {
	if r.srv == nil {
		panic("http server is not set")
	}

	metrics.SetAppName(r.appName)
	metrics.SetAppVersion(r.appVer)

	r.srvMux.Handle("/metrics", promhttp.Handler())

	return r
}

func (r *Runner[RT, CT]) Run() {
	l := zerolog.New(zerolog.MultiLevelWriter(r.logWriters...)).Level(r.logLevel).
		With().Str("app", appName).Str("app_v", appVer).Logger()

	for _, base := range []string{"config", appName} {
		for _, ext := range []string{".yaml", ".json"} {
			cfgPath := base + ext
			err := cfgloader.LoadFromPath(cfgPath, &r.cfg, nil)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				l.Error().Err(err).Str("path", cfgPath).Msg("config file load failed")
				os.Exit(1)
			} else if err == nil {
				l.Debug().Str("path", cfgPath).Msg("config file loaded")
			}
		}
	}

	if cfgPath := os.Getenv("APP_CONFIG_PATH"); cfgPath != "" {
		if err := cfgloader.LoadFromPath(cfgPath, &r.cfg, nil); err != nil {
			l.Error().Err(err).Str("path", cfgPath).Msg("config file load failed")
			os.Exit(1)
		}

		l.Debug().Str("path", cfgPath).Msg("config file loaded")
	}

	if err := cfgloader.LoadFromEnv("APP", &r.cfg); err != nil {
		l.Error().Err(err).Msg("load config from env vars failed")
		os.Exit(1)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	rt := &Runtime{
		AppName:    r.appName,
		AppVersion: r.appVer,
		SrvMux:     r.srvMux,
		Logger:     l,
	}

	app, err := r.appFactory(r.cfg, rt)
	if err != nil {
		l.Error().Err(err).Msg("app init failed")
		os.Exit(1)
	}

	ctx, ctxC := context.WithCancel(context.Background())
	defer ctxC()

	go func() {
		s := <-sig
		l.Info().Str("signal", s.String()).Msg("signal received")
		ctxC()
	}()

	if r.srv != nil {
		go func() {
			l.Info().Str("addr", r.srv.Addr).Msg("http server is starting")
			if err := r.srv.ListenAndServe(); errors.Is(err, http.ErrServerClosed) {
				l.Info().Msg("http server closed")
			} else if err != nil {
				l.Error().Err(err).Msg("http server serve failed")
			}
		}()
	}

	if err := app.Run(ctx); err != nil {
		l.Error().Err(err).Msg("app run failed")
		os.Exit(1)
	}

	if r.srv != nil {
		l.Info().Msg("http server is shutting down")
		if err := r.srv.Shutdown(context.Background()); err != nil {
			l.Error().Err(err).Msg("http server shutdown failed")
		}
	}
}

func isTerminal() bool {
	if o, _ := os.Stdout.Stat(); (o.Mode() & os.ModeCharDevice) == os.ModeCharDevice {
		return true
	}
	return false
}
