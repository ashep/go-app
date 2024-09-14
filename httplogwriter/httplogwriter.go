package httplogwriter

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
)

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type Writer struct {
	u      string
	user   string
	passwd string
	c      httpClient
}

func New(u, user, passwd string, c httpClient) (*Writer, error) {
	if u == "" {
		return nil, fmt.Errorf("empty url")
	}

	if _, err := url.Parse(u); err != nil {
		return nil, fmt.Errorf("invalid url: %w", err)
	}

	if c == nil {
		c = &http.Client{}
	}

	return &Writer{
		u:      u,
		user:   user,
		passwd: passwd,
		c:      c,
	}, nil
}

func (l *Writer) Write(b []byte) (int, error) {
	req, err := http.NewRequest(http.MethodPost, l.u, bytes.NewReader(b))
	if err != nil {
		return 0, fmt.Errorf("could not create request: %w", err)
	}

	if l.user != "" {
		req.SetBasicAuth(l.user, l.passwd)
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
