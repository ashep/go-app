package testlogger

import (
	"strings"
	"sync"
)

type BufWriter struct {
	mux sync.RWMutex
	b   strings.Builder
}

func (w *BufWriter) Write(s []byte) (int, error) {
	w.mux.Lock()
	defer w.mux.Unlock()

	return w.b.Write(s)
}

func (w *BufWriter) Content() string {
	w.mux.RLock()
	defer w.mux.RUnlock()
	return w.b.String()
}
