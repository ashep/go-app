package httpserver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"
)

type Option func(*Server)

func WithListener(lis net.Listener) Option {
	return func(s *Server) {
		s.lis = lis
	}
}

func WithAddr(addr string) Option {
	return func(s *Server) {
		lis, err := net.Listen("tcp", addr)
		if err != nil {
			panic(fmt.Sprintf("listen: %s", err))
		}
		WithListener(lis)(s)
	}
}

func WithRandomLocalAddr() Option {
	return func(s *Server) {
		WithAddr("127.0.0.1:0")(s)
	}
}

func WithHTTP1(v bool) Option {
	return func(s *Server) {
		s.srv.Protocols.SetHTTP1(v)
	}
}

func WithHTTP2(v bool) Option {
	return func(s *Server) {
		s.srv.Protocols.SetHTTP2(v)
	}
}

func WithUnencryptedHTTP2(v bool) Option {
	return func(s *Server) {
		s.srv.Protocols.SetUnencryptedHTTP2(v)
	}
}

type Server struct {
	lis     net.Listener
	srv     *http.Server
	mux     *http.ServeMux
	stopped chan struct{}
}

func New(opts ...Option) *Server {
	mux := http.NewServeMux()

	s := &Server{
		srv: &http.Server{
			Handler:   mux,
			Protocols: new(http.Protocols),
		},
		mux:     mux,
		stopped: make(chan struct{}),
	}

	for _, opt := range opts {
		opt(s)
	}

	if *s.srv.Protocols == *new(http.Protocols) {
		s.srv.Protocols.SetHTTP1(true)
	}

	if s.lis == nil {
		WithAddr("127.0.0.1:9000")(s)
	}

	return s
}

func (s *Server) Listener() net.Listener {
	return s.lis
}

func (s *Server) Handle(pattern string, handler http.Handler) {
	s.mux.Handle(pattern, handler)
}

func (s *Server) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	s.mux.HandleFunc(pattern, handler)
}

func (s *Server) Run(ctx context.Context) error {
	defer close(s.stopped)

	serveErr := make(chan error)

	go func() {
		defer close(serveErr)
		err := s.srv.Serve(s.lis)
		if errors.Is(err, http.ErrServerClosed) {
			serveErr <- nil
		} else if err != nil {
			serveErr <- err
		}
	}()

	go func() {
		<-ctx.Done()
		sCtx, sCtxC := context.WithTimeout(context.Background(), time.Second*5)
		defer sCtxC()
		_ = s.srv.Shutdown(sCtx)
	}()

	return <-serveErr
}

func (s *Server) Wait() error {
	<-s.stopped
	return nil
}
