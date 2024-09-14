package runner

import (
	"github.com/rs/zerolog"
)

type Runner[CT any] struct {
	Config CT
	Logger zerolog.Logger
}
