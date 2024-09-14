package option

import (
	"context"

	"github.com/ashep/go-apprun/runner"
)

type Option[CT any] func(context.Context, *runner.Runner[CT]) error
