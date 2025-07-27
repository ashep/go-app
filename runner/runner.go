package runner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ashep/go-app/httplogwriter"
	"github.com/ashep/go-app/metrics"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"

	"github.com/ashep/go-app/cfgloader"
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

type Validatable interface {
	Validate() error
}

type appFactory[RT Runnable, CT Validatable] func(cfg CT, rt *Runtime) (RT, error)

type Runner[RT Runnable, CT Validatable] struct {
	appName    string
	appVer     string
	appCfg     CT
	logWriters []io.Writer
	srvMux     *http.ServeMux
	srv        *http.Server
	appFactory appFactory[RT, CT]
}

func New[RT Runnable, CT Validatable](f appFactory[RT, CT]) *Runner[RT, CT] {
	time.Local = time.UTC

	if appName == "" {
		appName = os.Getenv("APP_NAME")
	}

	if appVer == "" {
		appVer = os.Getenv("APP_VERSION")
	}

	return &Runner[RT, CT]{
		appName:    appName,
		appVer:     appVer,
		appCfg:     *(new(CT)),
		appFactory: f,
		logWriters: []io.Writer{},
	}
}

func (r *Runner[RT, CT]) WithLogWriter(w io.Writer) *Runner[RT, CT] {
	r.logWriters = append(r.logWriters, w)
	return r
}

func (r *Runner[RT, CT]) WithConsoleLogWriter() *Runner[RT, CT] {
	var w io.Writer

	if isTerminal() {
		w = zerolog.ConsoleWriter{Out: os.Stderr}
	} else {
		w = os.Stderr
	}

	return r.WithLogWriter(w)
}

func (r *Runner[RT, CT]) WithDefaultHTTPLogWriter() *Runner[RT, CT] {
	w, err := httplogwriter.NewFromEnv()
	if err != nil {
		fmt.Printf("ERROR: setting up http log writer: %s\n", err)
		return r
	}

	return r.WithLogWriter(w)
}

func (r *Runner[RT, CT]) WithHTTPServer(s *http.Server) *Runner[RT, CT] {
	if r.srv != nil {
		fmt.Println("http server is already set")
		os.Exit(1)
	}

	r.srvMux = http.NewServeMux()
	s.Handler = r.srvMux
	r.srv = s

	return r
}

func (r *Runner[RT, CT]) WithDefaultHTTPServer() *Runner[RT, CT] {
	addr := os.Getenv("APP_HTTPSERVER_ADDR")
	if addr == "" {
		addr = ":9000"
	}

	return r.WithHTTPServer(&http.Server{
		Addr: addr,
	})
}

func (r *Runner[RT, CT]) WithDefaultMetricsHandler() *Runner[RT, CT] {
	if r.srv == nil {
		panic("http server is not set")
	}

	metrics.SetAppName(r.appName)
	metrics.SetAppVersion(r.appVer)

	r.srvMux.Handle("/metrics", promhttp.Handler())

	return r
}

func (r *Runner[RT, CT]) Run() {
	logLevel := zerolog.InfoLevel
	if dbg := strings.ToLower(os.Getenv("APP_DEBUG")); dbg == "true" || dbg == "1" {
		logLevel = zerolog.DebugLevel
	}

	l := zerolog.New(zerolog.MultiLevelWriter(r.logWriters...)).Level(logLevel).
		With().Str("app", r.appName).Str("app_v", r.appVer).Logger()

	// Load config from pre-defined files
	for _, base := range []string{"config", r.appName} {
		for _, ext := range []string{".yaml", ".json"} {
			cfgPath := base + ext
			err := cfgloader.LoadFromPath(cfgPath, r.appCfg, nil)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				l.Error().Err(err).Str("path", cfgPath).Msg("config file load failed")
				os.Exit(1)
			} else if err == nil {
				l.Debug().Str("path", cfgPath).Msg("config file loaded")
			}
		}
	}

	// Load config from additional file
	if cfgPath := os.Getenv("APP_CONFIG_PATH"); cfgPath != "" {
		if err := cfgloader.LoadFromPath(cfgPath, r.appCfg, nil); err != nil {
			l.Error().Err(err).Str("path", cfgPath).Msg("config file load failed")
			os.Exit(1)
		}
		l.Debug().Str("path", cfgPath).Msg("config file loaded")
	}

	// Load config from env
	if err := cfgloader.LoadFromEnv("APP", r.appCfg); err != nil {
		l.Error().Err(err).Msg("load config from env vars failed")
		os.Exit(1)
	}
	if err := cfgloader.LoadFromEnv(strings.ToUpper(appName), r.appCfg); err != nil {
		l.Error().Err(err).Msg("load config from env vars failed")
		os.Exit(1)
	}

	if err := r.appCfg.Validate(); err != nil {
		l.Error().Err(err).Msg("config validation failed")
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

	app, err := r.appFactory(r.appCfg, rt)
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
				l.Error().Err(err).Msg("http server listen and serve failed")
				ctxC()
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
