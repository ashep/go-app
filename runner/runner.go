package runner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ashep/go-app/httplogwriter"
	"github.com/ashep/go-app/httpserver"
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
	Server     httpServer
	Logger     zerolog.Logger
}

type Runnable interface {
	Run(context.Context) error
}

type Validatable interface {
	Validate() error
}

type httpServer interface {
	Listener() net.Listener
	Handle(pattern string, handler http.Handler)
	HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request))
	Run() error
	Start(ctx context.Context) chan error
	Stop(ctx context.Context) error
}

type appFactory[RT Runnable, CT any] func(cfg *CT, rt *Runtime) (RT, error)

type Runner[RT Runnable, CT any] struct {
	appName    string
	appName2   string
	appVer     string
	appCfg     *CT
	logWriters []io.Writer
	srv        httpServer
	appFactory appFactory[RT, CT]
}

func New[RT Runnable, CT any](f appFactory[RT, CT]) *Runner[RT, CT] {
	time.Local = time.UTC

	if appName == "" {
		appName = os.Getenv("APP_NAME")
	}
	if appName == "" {
		appName = "app"
	}

	if appVer == "" {
		appVer = os.Getenv("APP_VERSION")
	}
	if appVer == "" {
		appVer = "0.0.1"
	}

	return &Runner[RT, CT]{
		appName:    appName,
		appName2:   strings.ToUpper(strings.ReplaceAll(appName, "-", "_")),
		appVer:     appVer,
		appCfg:     new(CT),
		appFactory: f,
		logWriters: []io.Writer{},
	}
}

func (r *Runner[RT, CT]) WithConfig(cfg *CT) *Runner[RT, CT] {
	r.appCfg = cfg
	return r
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
	var (
		w   *httplogwriter.Writer
		err error
	)

	for _, prefix := range []string{"APP", r.appName2} {
		if os.Getenv(prefix+"_LOGSERVER_URL") == "" {
			continue
		}
		if w, err = httplogwriter.NewFromEnv(prefix); err != nil {
			fmt.Printf("ERROR: setting up http log writer: %s\n", err)
			return r
		}
	}

	if w == nil {
		fmt.Printf("Neither APP_LOGSERVER_URL nor %s_LOGSERVER_URL env var defined\n", r.appName2)
		os.Exit(1)
	}

	return r.WithLogWriter(w)
}

func (r *Runner[RT, CT]) WithHTTPServer(s httpServer) *Runner[RT, CT] {
	if r.srv != nil {
		fmt.Println("http server is already set")
		os.Exit(1)
	}
	r.srv = s
	return r
}

func (r *Runner[RT, CT]) WithDefaultHTTPServer() *Runner[RT, CT] {
	addr := ""
	for _, prefix := range []string{"APP", r.appName2} {
		if addr = os.Getenv(prefix + "_HTTPSERVER_ADDR"); addr != "" {
			break
		}
	}
	if addr == "" {
		addr = ":9000"
	}
	return r.WithHTTPServer(httpserver.New(httpserver.WithAddr(addr)))
}

func (r *Runner[RT, CT]) WithDefaultMetricsHandler() *Runner[RT, CT] {
	if r.srv == nil {
		fmt.Println("http server is not set")
		os.Exit(1)
	}

	metrics.SetAppName(r.appName)
	metrics.SetAppVersion(r.appVer)
	r.srv.Handle("/metrics", promhttp.Handler())

	return r
}

func (r *Runner[RT, CT]) WithDefaultHealthHandler() *Runner[RT, CT] {
	if r.srv == nil {
		fmt.Println("http server is not set")
		os.Exit(1)
	}

	r.srv.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	return r
}

func (r *Runner[RT, CT]) Run() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	ctx, ctxC := context.WithCancel(context.Background())
	defer ctxC()

	go func() {
		<-sig
		ctxC()
	}()

	r.RunContext(ctx)
}

func (r *Runner[RT, CT]) RunContext(ctx context.Context) {
	logLevel := zerolog.InfoLevel
	if dbg := strings.ToLower(os.Getenv("APP_DEBUG")); dbg == "true" || dbg == "1" {
		logLevel = zerolog.DebugLevel
	}
	if dbg := strings.ToLower(os.Getenv(r.appName2 + "_DEBUG")); dbg == "true" || dbg == "1" {
		logLevel = zerolog.DebugLevel
	}

	l := zerolog.New(zerolog.MultiLevelWriter(r.logWriters...)).Level(logLevel).
		With().Str("app", r.appName).Str("app_v", r.appVer).Logger()

	// Load config from pre-defined files
	for _, base := range []string{"config", r.appName} {
		for _, ext := range []string{".yaml", ".yml", ".json"} {
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
	for _, prefix := range []string{"APP", r.appName2} {
		if cfgPath := os.Getenv(prefix + "_CONFIG_PATH"); cfgPath != "" {
			if err := cfgloader.LoadFromPath(cfgPath, r.appCfg, nil); err != nil {
				l.Error().Err(err).Str("path", cfgPath).Msg("config file load failed")
				os.Exit(1)
			}
			l.Debug().Str("path", cfgPath).Msg("config file loaded")
		}
	}

	// Load config from env
	for _, prefix := range []string{"APP", r.appName2} {
		if err := cfgloader.LoadFromEnv(prefix, r.appCfg); err != nil {
			l.Error().Err(err).Msgf("load config from %s_ env vars failed", prefix)
			os.Exit(1)
		}
	}

	if appCfgT, ok := any(r.appCfg).(Validatable); ok {
		if err := appCfgT.Validate(); err != nil {
			l.Error().Err(err).Msg("config validation failed")
			os.Exit(1)
		}
	}

	rt := &Runtime{
		AppName:    r.appName,
		AppVersion: r.appVer,
		Server:     r.srv,
		Logger:     l,
	}

	app, err := r.appFactory(r.appCfg, rt)
	if err != nil {
		l.Error().Err(err).Msg("app init failed")
		os.Exit(1)
	}

	if err := app.Run(ctx); err != nil {
		l.Error().Err(err).Msg("app run failed")
		os.Exit(1)
	}
}

func isTerminal() bool {
	if o, _ := os.Stdout.Stat(); (o.Mode() & os.ModeCharDevice) == os.ModeCharDevice {
		return true
	}
	return false
}
