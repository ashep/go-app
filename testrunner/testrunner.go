package testrunner

import (
	"context"
	"testing"
	"time"

	"github.com/ashep/go-app/runner"
	"github.com/ashep/go-app/testlogger"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Runner[RT func(*runner.Runtime[CT]) error, CT any] struct {
	t         *testing.T
	cfg       CT
	run       RT
	waitStart func(CT) bool
	l         zerolog.Logger
	lb        *testlogger.BufWriter
}

func New[RT func(*runner.Runtime[CT]) error, CT any](t *testing.T, run RT, cfg CT) *Runner[RT, CT] {
	l, lb := testlogger.New()

	return &Runner[RT, CT]{
		t:   t,
		run: run,
		cfg: cfg,
		l:   l,
		lb:  lb,
	}
}

func (r *Runner[RT, CT]) SetStartWaiter(w func(CT) bool) *Runner[RT, CT] {
	r.waitStart = w
	return r
}

// Run runs the application and waits until it stops.
func (r *Runner[RT, CT]) Run() {
	runner.New(r.run).
		SetConfig(r.cfg).
		AddLogWriter(r.l).
		Run()
}

// Start starts the application in a separate goroutine.
func (r *Runner[RT, CT]) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	r.t.Cleanup(cancel)

	rnr := runner.New(r.run).
		SetConfig(r.cfg).
		AddLogWriter(r.l)

	go func() {
		rnr.RunContext(ctx)
	}()

	if r.waitStart != nil {
		require.Eventually(r.t, func() bool {
			return r.waitStart(r.cfg)
		}, time.Second*15, time.Millisecond*500, "the app did not start in time")
	}
}

func (r *Runner[RT, CT]) Logs() string {
	return r.lb.Content()
}

func (r *Runner[RT, CT]) AssertLogNoErrors() {
	assert.NotContains(r.t, r.Logs(), `"level":"error"`)
}

func (r *Runner[RT, CT]) AssertLogNoWarns() {
	assert.NotContains(r.t, r.Logs(), `"level":"warn"`)
}
