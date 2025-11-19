package testlogger

import (
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

type Logger struct {
	t *testing.T
	l zerolog.Logger
	w *BufWriter
}

type msg struct {
	Message string `json:"message"`
}

func New(t *testing.T) *Logger {
	w := &BufWriter{b: strings.Builder{}}

	return &Logger{
		t: t,
		l: zerolog.New(w),
		w: w,
	}
}

func (l *Logger) Logger() zerolog.Logger {
	return l.l
}

func (l *Logger) Content() string {
	return l.w.Content()
}

func (l *Logger) AssertContains(s string) {
	assert.Contains(l.t, l.w.Content(), s, "logs do not contain expected string")
}

func (l *Logger) AssertNoErrors() {
	assert.NotContains(l.t, l.w.Content(), `"level":"error"`, "logs contain error level")
}

func (l *Logger) AssertNoWarns() {
	assert.NotContains(l.t, l.w.Content(), `"level":"warn"`, "logs contain warn level")
}

func (l *Logger) AssertNoWarnsAndErrors() {
	l.AssertNoWarns()
	l.AssertNoErrors()
}
