package testhttpserver

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"

	"github.com/ashep/go-app/httpserver"
)

type Call struct {
	Header http.Header
	Body   []byte
}

type Server struct {
	calls map[string][]Call
	s     *httpserver.Server
}

func New() *Server {
	s := httpserver.New(httpserver.WithRandomLocalAddr())

	return &Server{
		calls: make(map[string][]Call),
		s:     s,
	}
}

func (s *Server) Listener() net.Listener {
	return s.s.Listener()
}

func (s *Server) Handle(pattern string, handler http.Handler) {
	s.s.Handle(pattern, s.middleware(handler))
}

func (s *Server) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	s.s.Handle(pattern, s.middleware(http.HandlerFunc(handler)))
}

func (s *Server) Run() error {
	return s.s.Run()
}

func (s *Server) Start(ctx context.Context) chan error {
	return s.s.Start(ctx)
}

func (s *Server) Stop(ctx context.Context) error {
	return s.s.Stop(ctx)
}

func (s *Server) BaseURL() string {
	return "http://" + s.Listener().Addr().String()
}

func (s *Server) Calls(path string) []Call {
	return s.calls[path]
}

func (s *Server) middleware(next http.Handler) http.Handler {
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
