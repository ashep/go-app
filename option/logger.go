package option

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ashep/go-apprun/runner"
	"github.com/ashep/go-apprun/zerologhttpwriter"
)

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func WithHTTPLogger[CT any](u string, c httpClient) Option[CT] {
	return func(_ context.Context, r *runner.Runner[CT]) error {
		httpWriter, err := zerologhttpwriter.New(r.Logger, u, c)
		if err != nil {
			return fmt.Errorf("http logger: %w", err)
		}

		r.Logger = r.Logger.Output(httpWriter)

		return nil
	}
}
