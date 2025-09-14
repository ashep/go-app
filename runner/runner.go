package runner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/ashep/go-app/cfgloader"
	"github.com/ashep/go-app/httplogwriter"
	"github.com/rs/zerolog"
)

var (
	appName = "" //nolint:gochecknoglobals // set externally
	appVer  = "" //nolint:gochecknoglobals // set externally
)

type Validatable interface {
	Validate() error
}

type Runtime[CT any] struct {
	Ctx        context.Context
	AppName    string
	AppName2   string
	AppVersion string
	Cfg        *CT
	Log        zerolog.Logger
}

type Runner[RT func(*Runtime[CT]) error, CT any] struct {
	run        RT
	logWriters []io.Writer
	rt         *Runtime[CT]
}

func New[RT func(*Runtime[CT]) error, CT any](run RT) *Runner[RT, CT] {
	time.Local = time.UTC

	if appName == "" {
		appName = os.Getenv("APP_NAME")
	}
	if appName == "" {
		wd, err := os.Getwd()
		if err != nil {
			fmt.Println("Unable to determine current working directory")
			os.Exit(1)
		}
		appName = filepath.Base(wd)
	}

	if appVer == "" {
		appVer = os.Getenv("APP_VERSION")
	}
	if appVer == "" {
		appVer = "0.0.1"
	}

	rt := &Runtime[CT]{
		AppName:    appName,
		AppName2:   sanitizeAppName(appName),
		AppVersion: appVer,
		Cfg:        new(CT),
	}

	return &Runner[RT, CT]{
		run:        run,
		rt:         rt,
		logWriters: make([]io.Writer, 0),
	}
}

func (r *Runner[RT, CT]) SetAppName(name string) *Runner[RT, CT] {
	r.rt.AppName = name
	r.rt.AppName2 = sanitizeAppName(name)
	return r
}

func (r *Runner[RT, CT]) SetAppVersion(ver string) *Runner[RT, CT] {
	r.rt.AppVersion = ver
	return r
}

func (r *Runner[RT, CT]) SetConfig(cfg CT) *Runner[RT, CT] {
	r.rt.Cfg = &cfg
	return r
}

func (r *Runner[RT, CT]) LoadConfigFile(path string) *Runner[RT, CT] {
	err := cfgloader.LoadFromPath(path, r.rt.Cfg, nil)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		fmt.Printf("Error loading config from %s: %s\n", path, err)
		os.Exit(1)
	}

	// Load config from additional file
	for _, prefix := range []string{"APP", r.rt.AppName2} {
		if cfgPath := os.Getenv(prefix + "_CONFIG_PATH"); cfgPath != "" {
			if err := cfgloader.LoadFromPath(cfgPath, r.rt.Cfg, nil); err != nil {
				fmt.Printf("Error loading config from %s: %s\n", cfgPath, err)
				os.Exit(1)
			}
		}
	}

	return r
}

func (r *Runner[RT, CT]) LoadEnvConfig() *Runner[RT, CT] {
	// Load config from env
	for _, prefix := range []string{"APP", r.rt.AppName2} {
		if err := cfgloader.LoadFromEnv(prefix, r.rt.Cfg); err != nil {
			fmt.Printf("Error loading config from %s_ env vars failed", prefix)
			os.Exit(1)
		}
	}

	return r
}

func (r *Runner[RT, CT]) AddLogWriter(w io.Writer) *Runner[RT, CT] {
	r.logWriters = append(r.logWriters, w)
	return r
}

func (r *Runner[RT, CT]) AddConsoleLogWriter() *Runner[RT, CT] {
	var w io.Writer

	if isTerminal() {
		w = zerolog.ConsoleWriter{Out: os.Stderr}
	} else {
		w = os.Stderr
	}

	return r.AddLogWriter(w)
}

func (r *Runner[RT, CT]) AddHTTPLogWriter() *Runner[RT, CT] {
	var (
		w   *httplogwriter.Writer
		err error
	)

	for _, prefix := range []string{"APP", r.rt.AppName2} {
		if os.Getenv(prefix+"_LOGSERVER_URL") == "" {
			continue
		}
		if w, err = httplogwriter.NewFromEnv(prefix); err != nil {
			fmt.Printf("ERROR: setting up http log writer: %s\n", err)
			fmt.Println("WARN: HTTP logging is disabled")
			return r
		}
	}

	if w == nil {
		fmt.Printf("ERROR: neither APP_LOGSERVER_URL nor %s_LOGSERVER_URL env var defined\n", r.rt.AppName2)
		fmt.Println("WARN: HTTP logging is disabled")
		return r
	}

	return r.AddLogWriter(w)
}

func (r *Runner[RT, CT]) RunContext(ctx context.Context) {
	r.rt.Ctx = ctx

	logLevel := zerolog.InfoLevel
	if dbg := strings.ToLower(os.Getenv("APP_DEBUG")); dbg == "true" || dbg == "1" {
		logLevel = zerolog.DebugLevel
	}
	if dbg := strings.ToLower(os.Getenv(r.rt.AppName2 + "_DEBUG")); dbg == "true" || dbg == "1" {
		logLevel = zerolog.DebugLevel
	}
	r.rt.Log = zerolog.New(zerolog.MultiLevelWriter(r.logWriters...)).Level(logLevel).
		With().Str("app", r.rt.AppName).Str("app_v", r.rt.AppVersion).Logger()

	if appCfgT, ok := any(r.rt.Cfg).(Validatable); ok {
		if err := appCfgT.Validate(); err != nil {
			r.rt.Log.Error().Err(err).Msg("config validation failed")
			os.Exit(1)
		}
	}

	if err := r.run(r.rt); err != nil && !errors.Is(err, context.Canceled) {
		r.rt.Log.Error().Err(err).Msg("app run failed")
		os.Exit(1)
	}
}

func (r *Runner[RT, CT]) Run() {
	ctx, ctxC := context.WithCancel(context.Background())
	defer ctxC()

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		ctxC()
	}()

	r.RunContext(ctx)
}

func isTerminal() bool {
	if o, _ := os.Stdout.Stat(); (o.Mode() & os.ModeCharDevice) == os.ModeCharDevice {
		return true
	}
	return false
}

func sanitizeAppName(name string) string {
	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ReplaceAll(name, ".", "_")
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ToUpper(name)
	return name
}
