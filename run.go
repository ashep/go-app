package apprun

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/ashep/go-cfgloader"
)

var (
	appName = "app"
	appVer  = "unknown"
)

type validatable interface {
	Validate() error
}

type App interface {
	Run(ctx context.Context, args []string) error
}

type factory[AT App, CT any] func(cfg CT, l zerolog.Logger) AT

func Run[AT App, CT any](f factory[AT, CT], cfg CT) {
	time.Local = time.UTC

	ll := zerolog.InfoLevel
	dbg := os.Getenv("APP_DEBUG")
	if dbg == "true" || dbg == "1" {
		ll = zerolog.DebugLevel
	}

	isTerminal := false
	if o, _ := os.Stdout.Stat(); (o.Mode() & os.ModeCharDevice) == os.ModeCharDevice {
		isTerminal = true
	}

	l := log.Logger.Level(ll).With().Str("app", appName).Str("app_v", appVer).Logger()
	if isTerminal {
		l = l.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	// Try to load config from default paths
	for _, base := range []string{"config", appName} {
		for _, ext := range []string{".yaml", ".json"} {
			cfgPath := base + ext
			err := cfgloader.LoadFromPath(cfgPath, &cfg, nil)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				logFatalError(fmt.Errorf("config load failed: %w", err), isTerminal, l)
			}

			l.Debug().Str("path", cfgPath).Msg("config loaded from file")
		}
	}

	// Load config from path defined  by an env variable
	if cfgPath := os.Getenv("APP_CONFIG_PATH"); cfgPath != "" {
		if err := cfgloader.LoadFromPath(cfgPath, &cfg, nil); err != nil {
			logFatalError(fmt.Errorf("config load failed: %w", err), isTerminal, l)
		}

		l.Debug().Str("path", cfgPath).Msg("config loaded from file")
	}

	if err := cfgloader.LoadFromEnv("APP", &cfg); err != nil {
		logFatalError(fmt.Errorf("config load failed: %w", err), isTerminal, l)
	}

	appEnvCfgName := strings.ReplaceAll(appName, "-", "_")
	appEnvCfgName = strings.ReplaceAll(appEnvCfgName, ".", "_")
	if err := cfgloader.LoadFromEnv(appEnvCfgName, &cfg); err != nil {
		logFatalError(fmt.Errorf("config load failed: %w", err), isTerminal, l)
	}

	var cfgV any = cfg
	if cfgVT, ok := cfgV.(validatable); ok {
		if err := cfgVT.Validate(); err != nil {
			logFatalError(fmt.Errorf("config validation failed: %w", err), isTerminal, l)
		}
	}

	ctx, ctxC := context.WithCancel(context.Background())
	defer ctxC()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		s := <-sig
		l.Info().Str("signal", s.String()).Msg("signal received")
		ctxC()
	}()

	if err := f(cfg, l).Run(ctx, os.Args); err != nil {
		logFatalError(err, isTerminal, l)
	}
}

func logFatalError(err error, isTerminal bool, l zerolog.Logger) {
	if isTerminal {
		fmt.Println(err.Error())
	} else {
		l.Error().Err(err).Msg("")
	}

	os.Exit(1)
}
