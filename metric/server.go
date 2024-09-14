package metric

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
)

func StartServer(ctx context.Context, addr, path string, l zerolog.Logger) {
	mux := http.NewServeMux()
	mux.Handle(path, promhttp.Handler())

	srv := http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		l.Debug().Str("addr", addr).Msg("metrics server starting")
		defer func() {
			l.Debug().Msg("metrics server stopped")
		}()

		err := srv.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			// ok
		} else if err != nil {
			l.Error().Err(err).Msg("metrics server listen and server failed")
		}
	}()

	go func() {
		<-ctx.Done()
		ctxSd, ctxSdC := context.WithTimeout(context.Background(), 10*time.Second)
		defer ctxSdC()

		l.Debug().Msg("metrics server shutting down")

		if err := srv.Shutdown(ctxSd); err != nil {
			l.Error().Err(err).Msg("metrics server shutdown failed")
		} else {
			l.Debug().Msg("metrics server shut down")
		}
	}()
}
