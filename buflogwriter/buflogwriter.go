package buflogwriter

import (
	"bytes"
)

type BufLogWriter struct {
	b bytes.Buffer
}

func (w *BufLogWriter) Write(p []byte) (n int, err error) {
	return w.b.Write(p)
}

func (w *BufLogWriter) String() string {
	return w.b.String()
}

func New() *BufLogWriter {
	return &BufLogWriter{
		b: bytes.Buffer{},
	}
}
