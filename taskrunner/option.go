package taskrunner

import (
	"time"

	"github.com/rs/zerolog"
)

type runnerCfg struct {
	l zerolog.Logger
}

type RunnerOption func(*runnerCfg)

func WithLogger(l zerolog.Logger) RunnerOption {
	return func(o *runnerCfg) {
		o.l = l
	}
}

type taskCfg struct {
	autoRestart     time.Duration
	shutdownTimeout time.Duration
}

type TaskOption func(*taskCfg)

func WithShutdownTimeout(d time.Duration) TaskOption {
	return func(o *taskCfg) {
		o.shutdownTimeout = d
	}
}

func WithAutoRestart(a time.Duration) TaskOption {
	return func(o *taskCfg) {
		o.autoRestart = a
	}
}
