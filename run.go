package apprun

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ashep/go-cfgloader"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	appName = "app"
	appVer  = "unknown"
)

type App[CT any] interface {
	Run(context.Context) error
}

type factory[CT any] func(cfg CT, l zerolog.Logger) App[CT]

func Run[CT any](f factory[CT], cfg CT, l *zerolog.Logger) int {
	time.Local = time.UTC

	if l == nil {
		ll := zerolog.InfoLevel
		dbg := os.Getenv("APP_DEBUG")
		if dbg == "true" || dbg == "1" {
			ll = zerolog.DebugLevel
		}

		nl := log.Logger.Level(ll).With().Str("app", appName).Str("app_v", appVer).Logger()
		if isTerminal() {
			nl = nl.Output(zerolog.ConsoleWriter{Out: os.Stderr})
		}

		l = &nl
	}

	// Try to load from "standard" paths
	for _, base := range []string{"config", appName} {
		for _, ext := range []string{".yaml", ".json"} {
			cfgPath := base + ext
			err := cfgloader.LoadFromPath(cfgPath, &cfg, nil)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				l.Error().Err(err).Str("path", cfgPath).Msg("config file load failed")
				return 1
			}

			l.Debug().Str("path", cfgPath).Msg("config file loaded")
		}
	}

	// From a path defined by an env variable
	if cfgPath := os.Getenv("APP_CONFIG_PATH"); cfgPath != "" {
		if err := cfgloader.LoadFromPath(cfgPath, &cfg, nil); err != nil {
			l.Error().Err(err).Str("path", cfgPath).Msg("config envs load failed")
			return 1
		}

		l.Debug().Str("path", cfgPath).Msg("config env loaded")
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

	if err := f(cfg, *l).Run(ctx); err != nil {
		l.Error().Err(err).Msg("app run failed")
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
