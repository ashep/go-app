package testlogger

import (
	"strings"
)

type BufWriter struct {
	b strings.Builder
}

func (w *BufWriter) Write(s []byte) (int, error) {
	return w.b.Write(s)
}

func (w *BufWriter) Content() string {
	return w.b.String()
}
