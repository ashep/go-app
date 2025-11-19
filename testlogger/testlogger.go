package testlogger

import (
	"encoding/json"
	"fmt"
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

type BufWriter struct {
	b strings.Builder
}

func (w *BufWriter) Write(s []byte) (int, error) {
	outer := msg{}
	if err := json.Unmarshal(s, &outer); err != nil {
		return 0, fmt.Errorf("unmarshal 1: %w", err)
	}

	inner := make(map[string]interface{})
	if err := json.Unmarshal([]byte(outer.Message), &inner); err != nil {
		return 0, fmt.Errorf("unmarshal 2: %w", err)
	}

	out, err := json.Marshal(inner)
	if err != nil {
		return 0, fmt.Errorf("marshal: %w", err)
	}

	out = append(out, '\n')

	return w.b.Write(out)
}

func (w *BufWriter) Content() string {
	return w.b.String()
}
