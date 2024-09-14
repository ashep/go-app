package zerologhttpwriter

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/rs/zerolog"
)

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type Logger struct {
	l zerolog.Logger
	u string
	c httpClient
}

func New(l zerolog.Logger, u string, c httpClient) (io.Writer, error) {
	if u == "" {
		return nil, fmt.Errorf("empty url")
	}

	if _, err := url.Parse(u); err != nil {
		return nil, fmt.Errorf("invalid url: %w", err)
	}

	if c == nil {
		return nil, fmt.Errorf("nil http client")
	}

	return &Logger{
		l: l,
		u: u,
		c: c,
	}, nil
}

func (l *Logger) Write(b []byte) (int, error) {
	if n, err := l.l.Write(b); err != nil {
		return n, err
	}

	req, err := http.NewRequest(http.MethodPost, l.u, bytes.NewReader(b))
	if err != nil {
		return 0, fmt.Errorf("could not create request: %w", err)
	}

	res, err := l.c.Do(req)
	if err != nil {
		return 0, fmt.Errorf("could not send request: %w", err)
	}

	if res.Body != nil {
		defer func() {
			_ = res.Body.Close()
		}()
	}

	if res.StatusCode != http.StatusCreated {
		return 0, fmt.Errorf("invalid response status code: %d", res.StatusCode)
	}

	return len(b), nil
}
