package testrunner_test

import (
	"testing"
	"time"

	"github.com/ashep/go-app/runner"
	"github.com/ashep/go-app/testrunner"
	"github.com/stretchr/testify/assert"
)

func TestRunner(main *testing.T) {
	main.Run("Run", func(t *testing.T) {
		cnt := 0
		testrunner.New(t, runMock, cfgMock{t: t, foo: "bar", cnt: &cnt}).Run()
		assert.Equal(t, 1, cnt)
	})

	main.Run("Start", func(t *testing.T) {
		cnt := 0

		testrunner.New(t, runMock, cfgMock{
			t:    t,
			foo:  "bar",
			cnt:  &cnt,
			wait: true,
		}).
			SetStartWaiter(func(cfg cfgMock) bool { *cfg.cnt++; return true }).
			Start()

		assert.Eventually(t, func() bool {
			return cnt == 2
		}, time.Millisecond*100, time.Millisecond*10)
	})
}

type cfgMock struct {
	t    *testing.T
	foo  string
	cnt  *int
	wait bool
}

func runMock(rt *runner.Runtime[cfgMock]) error {
	assert.Equal(rt.Cfg.t, rt.Cfg.foo, "bar")
	*rt.Cfg.cnt++

	if rt.Cfg.wait {
		<-rt.Ctx.Done()
	}

	return nil
}
