package taskrunner

import (
	"github.com/rs/zerolog"
)

type config struct {
	panic bool
	l     zerolog.Logger
}

type Option func(*config)

func WithPanic(v bool) Option {
	return func(o *config) {
		o.panic = v
	}
}

func WithLogger(l zerolog.Logger) Option {
	return func(o *config) {
		o.l = l
	}
}
