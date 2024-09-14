package option

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/ashep/go-apprun/runner"
)

func WithMetricsServer[CT any](addr, path string) Option[CT] {
	return func(ctx context.Context, r *runner.Runner[CT]) error {
		mux := http.NewServeMux()
		mux.Handle(path, promhttp.Handler())

		srv := http.Server{
			Addr:    addr,
			Handler: mux,
		}

		go func() {
			r.Logger.Debug().Str("addr", addr).Msg("metrics server starting")
			defer func() {
				r.Logger.Debug().Msg("metrics server stopped")
			}()

			err := srv.ListenAndServe()
			if errors.Is(err, http.ErrServerClosed) {
				// ok
			} else if err != nil {
				r.Logger.Error().Err(err).Msg("metrics server listen and server failed")
			}
		}()

		go func() {
			<-ctx.Done()
			ctxSd, ctxSdC := context.WithTimeout(context.Background(), 10*time.Second)
			defer ctxSdC()

			r.Logger.Debug().Msg("metrics server shutting down")

			if err := srv.Shutdown(ctxSd); err != nil {
				r.Logger.Error().Err(err).Msg("metrics server shutdown failed")
			} else {
				r.Logger.Debug().Msg("metrics server shut down")
			}
		}()

		return nil
	}
}
