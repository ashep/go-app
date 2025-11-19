package testrunner

import (
	"net"
	"net/http"
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

func (r *Runner[RT, CT]) SetTCPReadyStartWaiter(addr string) *Runner[RT, CT] {
	nAddr, err := net.ResolveTCPAddr("tcp", addr)
	require.NoError(r.t, err)

	r.SetStartWaiter(func(CT) bool {
		_, err := net.DialTCP("tcp", nil, nAddr)
		return err == nil
	})

	return r
}

func (r *Runner[RT, CT]) SetHTTPReadyStartWaiter(url string) *Runner[RT, CT] {
	r.SetStartWaiter(func(CT) bool {
		res, err := http.DefaultClient.Get(url)
		if err != nil {
			return false
		}

		defer func() {
			_ = res.Body.Close()
		}()

		return res.StatusCode == http.StatusOK
	})

	return r
}

// Run runs the application and waits until it stops.
func (r *Runner[RT, CT]) Run() error {
	return runner.New(r.run).
		SetConfig(r.cfg).
		AddLogWriter(r.l).
		RunContext(r.t.Context())
}

// Start starts the application in a separate goroutine.
func (r *Runner[RT, CT]) Start() *Runner[RT, CT] {
	rnr := runner.New(r.run).
		SetConfig(r.cfg).
		AddLogWriter(r.l)

	go func() {
		if err := rnr.RunContext(r.t.Context()); err != nil {
			r.t.Logf("app run failed: %v", err)
		}
	}()

	if r.waitStart != nil {
		ok := assert.Eventually(r.t, func() bool {
			return r.waitStart(r.cfg)
		}, time.Second*3, time.Millisecond*100, "the app did not start in time")

		if !ok {
			r.t.Logf("Logs:\n%s", r.Logs())
		}
	}

	return r
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
