package testlogger

import (
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

func (l *Logger) Messages() []string {
	return strings.Split(strings.TrimRight(l.Content(), "\n"), "\n")
}

// MessagesContainsAny returns all log messages that contain any of the specified substrings.
func (l *Logger) MessagesContainsAny(s ...string) []string {
	res := make([]string, 0)
	for _, m := range l.Messages() {
		for _, sub := range s {
			if !strings.Contains(m, sub) {
				continue
			}
			res = append(res, m)
		}

	}
	return res
}

// MessagesContainsAll returns all log messages that contain all of the specified substrings.
func (l *Logger) MessagesContainsAll(s ...string) []string {
	res := make([]string, 0)
	for _, m := range l.Messages() {
		matchesAll := true
		for _, sub := range s {
			if !strings.Contains(m, sub) {
				matchesAll = false
				break
			}
		}
		if matchesAll {
			res = append(res, m)
		}
	}
	return res
}

func (l *Logger) FirstMessageContainsAny(s ...string) string {
	msgs := l.MessagesContainsAny(s...)
	if len(msgs) > 0 {
		return msgs[0]
	}
	return ""
}

func (l *Logger) FirstMessageContainsAll(s ...string) string {
	msgs := l.MessagesContainsAll(s...)
	if len(msgs) > 0 {
		return msgs[0]
	}
	return ""
}

func (l *Logger) AssertContains(s string) {
	assert.Contains(l.t, l.w.Content(), s)
}

func (l *Logger) AssertNotContains(s string) {
	assert.NotContains(l.t, l.w.Content(), s)
}

func (l *Logger) AssertNoErrors() {
	l.AssertNotContains(`"level":"error"`)
}

func (l *Logger) AssertContainsError() {
	l.AssertContains(`"level":"error"`)
}

func (l *Logger) AssertHasError(s string) {
	m := l.FirstMessageContainsAll(`"level":"error"`, s)
	if m == "" {
		assert.Fail(l.t, fmt.Sprintf("%#v should contain error message %#v", l.Content(), s))
	}
}

func (l *Logger) AssertNoWarns() {
	l.AssertNotContains(`"level":"warn"`)
}

func (l *Logger) AssertContainsWarn() {
	l.AssertContains(`"level":"warn"`)
}

func (l *Logger) AssertHasWarn(s string) {
	m := l.FirstMessageContainsAll(`"level":"warn"`, s)
	if m == "" {
		assert.Fail(l.t, fmt.Sprintf("%#v should contain warn message %#v", l.Content(), s))
	}
}

func (l *Logger) AssertNoWarnsAndErrors() {
	l.AssertNoWarns()
	l.AssertNoErrors()
}
