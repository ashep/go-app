package testrunner_test

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/ashep/go-app/runner"
	"github.com/ashep/go-app/testrunner"
	"github.com/stretchr/testify/assert"
)

func TestRunner(main *testing.T) {
	main.Run("Run", func(t *testing.T) {
		cnt := &atomic.Int64{}
		testrunner.New(t, runMock, cfgMock{t: t, foo: "bar", cnt: cnt}).Run()
		assert.Equal(t, int64(1), cnt.Load())
	})

	main.Run("Start", func(t *testing.T) {
		cnt := &atomic.Int64{}

		testrunner.New(t, runMock, cfgMock{
			t:    t,
			foo:  "bar",
			cnt:  cnt,
			wait: true,
		}).
			SetStartWaiter(func(cfg cfgMock) bool { cfg.cnt.Store(1); return true }).
			Start()

		assert.Eventually(t, func() bool {
			return cnt.Load() == 1
		}, time.Second, time.Millisecond*50)
	})
}

type cfgMock struct {
	t    *testing.T
	foo  string
	cnt  *atomic.Int64
	wait bool
}

func runMock(rt *runner.Runtime[cfgMock]) error {
	assert.Equal(rt.Cfg.t, rt.Cfg.foo, "bar")
	rt.Cfg.cnt.Add(1)

	if rt.Cfg.wait {
		<-rt.Ctx.Done()
	}

	return nil
}
