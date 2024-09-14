package apprun

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/ashep/go-apprun/option"
	"github.com/ashep/go-apprun/runner"
)

var (
	appName = "app"
	appVer  = "unknown"
)

type App[CT any] interface {
	Run(context.Context, CT, zerolog.Logger) error
}

type factory[CT any] func() App[CT]

func Run[CT any](f factory[CT], cfg CT, l *zerolog.Logger, opts ...option.Option[CT]) int {
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

	ctx, ctxC := context.WithCancel(context.Background())
	defer ctxC()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		s := <-sig
		l.Info().Str("signal", s.String()).Msg("signal received")
		ctxC()
	}()

	r := &runner.Runner[CT]{
		Config: cfg,
		Logger: *l,
	}

	for _, opt := range opts {
		if err := opt(ctx, r); err != nil {
			r.Logger.Error().Err(err).Msg("failed to apply option")
			return 1
		}
	}

	if err := f().Run(ctx, r.Config, r.Logger); err != nil {
		r.Logger.Error().Err(err).Msg("")
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
