package apprun

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/ashep/go-cfgloader"
)

type app interface {
	Run(ctx context.Context) error
}

type factory func(cfg any, l zerolog.Logger) app

func Run(name string, f factory, cfg any) {
	if name == "" {
		panic("empty app name")
	}
	nameUpper := strings.ToUpper(name)

	time.Local = time.UTC

	l := log.Logger.With().Str("app", name).Logger()
	if o, _ := os.Stdout.Stat(); (o.Mode() & os.ModeCharDevice) == os.ModeCharDevice { // Terminal
		l = l.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	cfgPath := os.Getenv(nameUpper + "_CONFIG_PATH")
	if cfgPath != "" {
		if err := cfgloader.LoadFromPath(cfgPath, &cfg, nil); err != nil {
			l.Error().Err(err).Msg("load config from file failed")
			os.Exit(1)
		}
	}

	if err := cfgloader.LoadFromEnv(name, &cfg); err != nil {
		l.Error().Err(err).Msg("load config from env failed")
		os.Exit(1)
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

	if err := f(cfg, l).Run(ctx); err != nil {
		l.Error().Err(err).Msg("app run failed")
		os.Exit(1)
	}
}
