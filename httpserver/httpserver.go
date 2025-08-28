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

type Server struct {
	lis net.Listener
	srv *http.Server
	mux *http.ServeMux
}

func New(opts ...Option) *Server {
	s := &Server{
		srv: &http.Server{},
		mux: http.NewServeMux(),
	}

	for _, opt := range opts {
		opt(s)
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

func (s *Server) Run() error {
	if s.lis == nil {
		return errors.New("no address specified")
	}

	return s.srv.Serve(s.lis)
}

func (s *Server) Start(ctx context.Context) chan error {
	done := make(chan error)

	go func() {
		defer close(done)
		done <- s.Run()
	}()

	go func() {
		<-ctx.Done()
		sCtx, sCtxC := context.WithTimeout(context.Background(), time.Second*5)
		defer sCtxC()
		_ = s.Stop(sCtx)
	}()

	return done
}

func (s *Server) Stop(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}
