package testlogger

import (
	"strings"

	"github.com/rs/zerolog"
)

func New() (zerolog.Logger, *TestWriter) {
	tw := &TestWriter{
		b: strings.Builder{},
	}

	return zerolog.New(tw), tw
}

type TestWriter struct {
	b strings.Builder
}

func (w *TestWriter) Write(p []byte) (int, error) {
	return w.b.Write(p)
}

func (w *TestWriter) Content() string {
	return w.b.String()
}
