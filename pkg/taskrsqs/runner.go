package taskrsqs

import (
	"context"
	"errors"
	"sync"

	"github.com/WeTransfer/go-telemetry/log"
	"github.com/rs/zerolog"
)

type NewRunnerOpt func(Runner) Runner

func Run(ctx context.Context, workerConfig *WorkerConfig, opts ...NewRunnerOpt) Runner {
	runner := NewRunner(ctx, opts...)
	runner.Start(workerConfig)
	return runner
}

func WithNumberOfWorkers(n int) func(Runner) Runner {
	return func(r Runner) Runner {
		r.numWorkers = n
		return r
	}
}

func NewRunner(ctx context.Context, opts ...NewRunnerOpt) Runner {
	runner := Runner{
		ctx:        ctx,
		numWorkers: 1,
	}

	for _, opt := range opts {
		runner = opt(runner)
	}

	return runner
}

type Runner struct {
	ctx        context.Context
	numWorkers int
	workers    []Worker
	started    bool
}

func (r *Runner) Start(workerConfig *WorkerConfig) {
	if r.started {
		panic(errors.New("runner already started. Don't call me twice!"))
	}
	r.started = true
	r.workers = make([]Worker, r.numWorkers)

	log.FromContext(r.ctx).Info().Msgf("Starting %d workers", r.numWorkers)

	for k := 0; k < r.numWorkers; k++ {
		ctx := contextWithLogFields(r.ctx, func(lc zerolog.Context) zerolog.Context {
			return lc.Int("worker", k)
		})
		r.workers[k] = NewWorker(ctx, workerConfig)
		r.workers[k].Start()
	}
}

func (r *Runner) Drain() {
	var wg sync.WaitGroup
	wg.Add(len(r.workers))

	for _, worker := range r.workers {
		go func() {
			worker.Stop()
			wg.Done()
		}()
	}

	wg.Wait()
}
