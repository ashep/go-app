package health

import (
	"net/http"
)

const (
	URLPath = "/health"
)

type httpServer interface {
	HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request))
}

func RegisterServer(srv httpServer) {
	srv.HandleFunc(URLPath, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}
