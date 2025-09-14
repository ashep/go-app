package testhttpserver

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/ashep/go-app/httpserver"
	"github.com/stretchr/testify/require"
)

type Call struct {
	Header http.Header
	Body   []byte
}

type Server struct {
	t     *testing.T
	s     *httpserver.Server
	calls map[string][]Call
}

func New(t *testing.T) *Server {
	return &Server{
		t:     t,
		calls: make(map[string][]Call),
		s:     httpserver.New(httpserver.WithRandomLocalAddr()),
	}
}

func (s *Server) Listener() net.Listener {
	return s.s.Listener()
}

func (s *Server) Handle(pattern string, handler http.Handler) {
	s.s.Handle(pattern, s.wrap(handler))
}

func (s *Server) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	s.s.Handle(pattern, s.wrap(http.HandlerFunc(handler)))
}

func (s *Server) Run() {
	ctx, cancel := context.WithCancel(context.Background())
	s.t.Cleanup(cancel)

	go func() {
		err := s.s.Run(ctx)
		require.NoError(s.t, err)
	}()

	require.Eventually(s.t, func() bool {
		_, err := http.Get(s.BaseURL())
		return err == nil
	}, time.Second*3, time.Microsecond*100, "http test server did not start")
}

func (s *Server) BaseURL() string {
	return "http://" + s.Listener().Addr().String()
}

func (s *Server) URL(path string) string {
	return s.BaseURL() + path
}

func (s *Server) Calls(path string) []Call {
	return s.calls[path]
}

func (s *Server) wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqBody, err := io.ReadAll(r.Body)
		if err != nil {
			panic(fmt.Errorf("failed to read request body: %w", err))
		}
		if err := r.Body.Close(); err != nil {
			panic(fmt.Errorf("failed to close request body: %w", err))
		}

		if s.calls[r.URL.Path] == nil {
			s.calls[r.URL.Path] = make([]Call, 0)
		}

		s.calls[r.URL.Path] = append(s.calls[r.URL.Path], Call{
			Header: r.Header.Clone(),
			Body:   reqBody,
		})

		r.Body = io.NopCloser(bytes.NewBuffer(reqBody))
		next.ServeHTTP(w, r)
	})
}
