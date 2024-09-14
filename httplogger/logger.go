package httplogger

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

type writer struct {
	pwr io.Writer
	u   string
	un  string
	pw  string
	c   httpClient
}

func New(l zerolog.Logger, u, username, password string, c httpClient) (zerolog.Logger, error) {
	if u == "" {
		return l, fmt.Errorf("empty url")
	}

	if _, err := url.Parse(u); err != nil {
		return l, fmt.Errorf("invalid url: %w", err)
	}

	if c == nil {
		c = &http.Client{}
	}

	wr := &writer{
		pwr: l,
		u:   u,
		un:  username,
		pw:  password,
		c:   c,
	}

	return l.Output(wr), nil
}

func (l *writer) Write(b []byte) (int, error) {
	if n, err := l.pwr.Write(b); err != nil {
		return n, err
	}

	req, err := http.NewRequest(http.MethodPost, l.u, bytes.NewReader(b))
	if err != nil {
		return 0, fmt.Errorf("could not create request: %w", err)
	}

	if l.un != "" {
		req.SetBasicAuth(l.un, l.pw)
	}

	req.Header.Set("Content-Type", "application/json")

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
