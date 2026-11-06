package taskrunner_test

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/ashep/go-app/taskrunner"
	"github.com/ashep/go-app/testlogger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunner_Run(main *testing.T) {
	main.Parallel()

	setUp := func(t *testing.T) (*taskrunner.Runner, *runnableMock, *testlogger.Logger) {
		t.Parallel()

		l := testlogger.New(t)
		tr := taskrunner.New(taskrunner.WithLogger(l.Logger()))
		rm := &runnableMock{l: l}

		return tr, rm, l
	}

	main.Run("RunAndWait", func(t *testing.T) {
		tr, runMock, l := setUp(t)

		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			err := tr.Run(ctx, "aTestTask", runMock)
			require.NoError(t, err)
			synctest.Wait()

			require.True(t, runMock.runCalled.Load())
			require.False(t, runMock.runExited.Load())
			require.False(t, runMock.shutdownCalled.Load())

			cancel()
			synctest.Wait()

			require.True(t, runMock.runCalled.Load())
			require.True(t, runMock.runExited.Load())
			require.False(t, runMock.shutdownCalled.Load())

			wErr := tr.Wait(ctx)
			require.NoError(t, wErr)
			require.True(t, runMock.runCalled.Load())
			require.True(t, runMock.runExited.Load())
			require.True(t, runMock.shutdownCalled.Load())

			assert.Equal(t, []string{
				`{"level":"info","message":"aTestTask started"}`,
				`{"level":"info","message":"aTestTask stopped"}`,
			}, l.Messages())
		})
	})

	main.Run("RunAndWaitMultiple", func(t *testing.T) {
		tr, _, _ := setUp(t)
		mux := sync.Mutex{}
		shutdownOrder := make([]int, 0)

		newRunMock := func(n int) *runnableMock {
			return &runnableMock{
				onShutdown: func() error {
					mux.Lock()
					defer mux.Unlock()
					shutdownOrder = append(shutdownOrder, n)
					return nil
				},
			}
		}

		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			err := tr.Run(ctx, "aTestTask1", newRunMock(1))
			require.NoError(t, err)

			err = tr.Run(ctx, "aTestTask2", newRunMock(2))
			require.NoError(t, err)

			err = tr.Run(ctx, "aTestTask3", newRunMock(3))
			require.NoError(t, err)

			cancel()
			wErr := tr.Wait(ctx)
			require.NoError(t, wErr)

			assert.Equal(t, []int{3, 2, 1}, shutdownOrder)
		})
	})

	main.Run("AutoRestart", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			var cnt = &atomic.Int32{}

			runMock := &runnableMock{
				onRun: func(ctx context.Context) error {
					cnt.Store(cnt.Load() + 1)
					return nil
				},
			}

			l := testlogger.New(t)
			tr := taskrunner.New(taskrunner.WithLogger(l.Logger()))
			ctx, cancel := context.WithCancel(t.Context())

			err := tr.Run(ctx, "aTestTask1", runMock, taskrunner.WithAutoRestart(time.Second*2))
			require.NoError(t, err)

			time.Sleep(time.Second * 10)
			cancel()

			err = tr.Wait(ctx)
			require.NoError(t, err)

			assert.Equal(t, 5, len(l.MessagesContainsAny(`{"level":"warn","message":"restarting aTestTask1"}`)))
		})
	})

	main.Run("AlreadyRun", func(t *testing.T) {
		tr, runMock, _ := setUp(t)

		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			err := tr.Run(ctx, "aTestTask", runMock)
			require.NoError(t, err)

			err = tr.Run(ctx, "aTestTask", runMock)
			require.EqualError(t, err, "aTestTask is already running")
		})
	})

	main.Run("PanicOnRun", func(t *testing.T) {
		tr, runMock, l := setUp(t)

		runMock.onRun = func(context.Context) error {
			panic("42")
			return nil
		}

		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			err := tr.Run(ctx, "aTestTask", runMock)
			require.NoError(t, err)

			cancel()
			wErr := tr.Wait(ctx)
			require.NoError(t, wErr)

			assert.Equal(t, []string{
				`{"level":"info","message":"aTestTask started"}`,
				`{"level":"error","error":"42","message":"aTestTask panicked"}`,
				`{"level":"info","message":"aTestTask stopped"}`,
			}, l.Messages())
		})
	})

	main.Run("ErrorOnShutdown", func(t *testing.T) {
		tr, runMock, l := setUp(t)

		runMock.onShutdown = func() error {
			return fmt.Errorf("shutdown: %w", assert.AnError)
		}

		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			err := tr.Run(ctx, "aTestTask", runMock)
			synctest.Wait()
			require.NoError(t, err)

			cancel()
			wErr := tr.Wait(ctx)
			require.EqualError(t, wErr, "shutdown: assert.AnError general error for testing")

			assert.Equal(t, []string{
				`{"level":"info","message":"aTestTask started"}`,
				`{"level":"info","message":"aTestTask stopped"}`,
			}, l.Messages())
		})
	})

	main.Run("MultipleErrorOnShutdown", func(t *testing.T) {
		tr, _, _ := setUp(t)

		newRunMock := func(n int) *runnableMock {
			return &runnableMock{
				onShutdown: func() error {
					if n%2 != 0 {
						return fmt.Errorf("shutdown: %w", assert.AnError)
					}
					return nil
				},
			}
		}

		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			err := tr.Run(ctx, "aTestTask1", newRunMock(1))
			require.NoError(t, err)

			err = tr.Run(ctx, "aTestTask2", newRunMock(2))
			require.NoError(t, err)

			err = tr.Run(ctx, "aTestTask3", newRunMock(3))
			require.NoError(t, err)

			cancel()
			synctest.Wait()

			wErr := tr.Wait(ctx)
			require.EqualError(t, wErr, `shutdown: assert.AnError general error for testing
shutdown: assert.AnError general error for testing`)
		})
	})

	main.Run("PanicOnShutdown", func(t *testing.T) {
		tr, runMock, l := setUp(t)

		runMock.onShutdown = func() error {
			panic("42")
			return nil
		}

		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			err := tr.Run(ctx, "aTestTask", runMock)
			synctest.Wait()
			require.NoError(t, err)

			cancel()
			synctest.Wait()

			wErr := tr.Wait(ctx)
			require.EqualError(t, wErr, "aTestTask panicked on shutdown: 42")

			assert.Equal(t, []string{
				`{"level":"info","message":"aTestTask started"}`,
				`{"level":"info","message":"aTestTask stopped"}`,
			}, l.Messages())
		})
	})

	main.Run("MultiplePanicOnShutdown", func(t *testing.T) {
		tr, _, _ := setUp(t)

		newRunMock := func(n int) *runnableMock {
			return &runnableMock{
				onShutdown: func() error {
					if n%2 != 0 {
						panic(fmt.Sprintf("some panic in %d", n))
					}
					return nil
				},
			}
		}

		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			err := tr.Run(ctx, "aTestTask1", newRunMock(1))
			require.NoError(t, err)

			err = tr.Run(ctx, "aTestTask2", newRunMock(2))
			require.NoError(t, err)

			err = tr.Run(ctx, "aTestTask3", newRunMock(3))
			require.NoError(t, err)

			cancel()
			synctest.Wait()

			wErr := tr.Wait(ctx)
			require.EqualError(t, wErr, `aTestTask3 panicked on shutdown: some panic in 3
aTestTask1 panicked on shutdown: some panic in 1`)
		})
	})
}

type runnableMock struct {
	onRun      func(ctx context.Context) error
	onShutdown func() error
	l          *testlogger.Logger

	runCalled      atomic.Bool
	runExited      atomic.Bool
	shutdownCalled atomic.Bool
}

func (r *runnableMock) Run(ctx context.Context) error {
	r.runCalled.Store(true)
	defer func() { r.runExited.Store(true) }()

	if r.onRun != nil {
		return r.onRun(ctx)
	}

	<-ctx.Done()

	return nil
}

func (r *runnableMock) Shutdown() error {
	defer func() { r.shutdownCalled.Store(true) }()

	if r.onShutdown != nil {
		return r.onShutdown()
	}

	return nil
}
