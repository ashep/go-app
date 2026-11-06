package taskrunner

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

type Runnable interface {
	Run(context.Context) error
	Shutdown() error
}

type task struct {
	name     string
	runnable Runnable
	cfg      taskCfg
}

type Runner struct {
	mux       sync.Mutex
	taskNames map[string]struct{}
	tasks     []task
	l         zerolog.Logger
}

func New(opt ...RunnerOption) *Runner {
	cfg := &runnerCfg{
		l: zerolog.Nop(),
	}
	for _, o := range opt {
		o(cfg)
	}

	return &Runner{
		tasks:     make([]task, 0),
		taskNames: make(map[string]struct{}),
		l:         cfg.l,
	}
}

func (tr *Runner) Run(ctx context.Context, name string, r Runnable, opts ...TaskOption) error {
	tr.mux.Lock()
	defer tr.mux.Unlock()

	if _, ok := tr.taskNames[name]; ok {
		return fmt.Errorf("%s is already running", name)
	}

	cfg := taskCfg{
		shutdownTimeout: time.Second * 5,
	}
	for _, o := range opts {
		o(&cfg)
	}

	go func() {
		defer func() {
			if p := recover(); p != nil {
				tr.l.Error().Err(fmt.Errorf("%v", p)).Msgf("%s panicked", name)
			}

			tr.l.Info().Msgf("%s stopped", name)

			tr.mux.Lock()
			delete(tr.taskNames, name)
			newTasks := make([]task, 0, len(tr.tasks)-1)
			for _, t := range tr.tasks {
				if t.name != name {
					newTasks = append(newTasks, t)
				}
			}
			tr.mux.Unlock()

			if cfg.autoRestart > 0 {
				select {
				case <-ctx.Done():
					return
				case <-time.After(cfg.autoRestart):
					tr.l.Warn().Msgf("restarting %s", name)
					_ = tr.Run(ctx, name, r, opts...)
				}
			}
		}()

		if err := r.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			tr.l.Error().Err(err).Msgf("%s stopped with error", name)
		}
	}()

	tr.taskNames[name] = struct{}{}
	tr.tasks = append(tr.tasks, task{
		name:     name,
		runnable: r,
		cfg:      cfg,
	})
	tr.l.Info().Msgf("%s started", name)

	return nil
}

// Wait waits until context is done and then calls Shutdown on each runnable
func (tr *Runner) Wait(ctx context.Context) error {
	<-ctx.Done()

	tr.mux.Lock()
	defer tr.mux.Unlock()

	shutdownErr := make(chan error)
	resErr := make([]error, 0)

	for i := len(tr.tasks) - 1; i >= 0; i-- {
		go func(t *task) {
			var err error

			sCtx, sCancel := context.WithTimeout(context.Background(), t.cfg.shutdownTimeout)
			defer sCancel()

			defer func() {
				if p := recover(); p != nil {
					err = fmt.Errorf("%s panicked on shutdown: %v", t.name, p)
				}

				shutdownErr <- err
			}()

			select {
			case <-sCtx.Done():
				shutdownErr <- sCtx.Err()
			default:
				err = t.runnable.Shutdown()
			}
		}(&tr.tasks[i])

		if err := <-shutdownErr; err != nil {
			resErr = append(resErr, err)
		}
	}

	return errors.Join(resErr...)
}
