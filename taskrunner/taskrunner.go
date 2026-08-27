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
	Wait() error
}

type Runner struct {
	mux   sync.Mutex
	tasks map[string]Runnable
	panic bool
	l     zerolog.Logger
}

func New(opt ...Option) *Runner {
	cfg := &config{
		panic: false,
		l:     zerolog.Nop(),
	}
	for _, o := range opt {
		o(cfg)
	}

	return &Runner{
		tasks: make(map[string]Runnable, 0),
		panic: cfg.panic,
		l:     cfg.l,
	}
}

func (g *Runner) Run(ctx context.Context, name string, r Runnable) error {
	g.mux.Lock()
	defer g.mux.Unlock()

	if _, ok := g.tasks[name]; ok {
		return errors.New("already running")
	}

	go func() {
		defer func() {
			if p := recover(); p != nil {
				if g.panic {
					panic(p)
				}
				g.l.Error().Err(fmt.Errorf("%v", p)).Msgf("%s panicked", name)
			}

			g.l.Info().Msgf("%s stopped", name)

			g.mux.Lock()
			defer g.mux.Unlock()
			delete(g.tasks, name)
		}()

		if err := r.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			g.l.Error().Err(err).Msgf("%s stopped with error", name)
		}
	}()

	g.tasks[name] = r
	g.l.Info().Msgf("%s started", name)

	return nil
}

func (g *Runner) Wait(ctx context.Context) error {
	<-ctx.Done()
	return g.shutdown(time.Second * 5)
}

func (g *Runner) shutdown(timeout time.Duration) error {
	g.mux.Lock()
	for name, task := range g.tasks {
		go func(n string) {
			defer func() {
				if p := recover(); p != nil {
					g.l.Error().Err(fmt.Errorf("%v", p)).Msgf("%s panicked on stop", n)
				}
			}()
			if err := task.Wait(); err != nil {
				g.l.Error().Err(err).Msgf("%s stopped with error", n)
			}
		}(name)
	}
	g.mux.Unlock()

	sCtx, sCancel := context.WithTimeout(context.Background(), timeout)
	defer sCancel()

	for {
		select {
		case <-sCtx.Done():
			return fmt.Errorf("timeout waiting for tasks to stop")
		default:
			g.mux.Lock()
			if len(g.tasks) == 0 {
				g.mux.Unlock()
				return nil
			}
			g.mux.Unlock()
		}
	}
}
