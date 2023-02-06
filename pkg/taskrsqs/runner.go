package taskrsqs

import (
	"context"
	"errors"
	"sync"

	"github.com/WeTransfer/go-telemetry/log"
	"github.com/rs/zerolog"
)

func Run(ctx context.Context, numWorkers int, workerConfig *WorkerConfig) Runner {

	runner := Runner{
		ctx: ctx,
	}

	runner.Start(numWorkers, workerConfig)

	return runner
}

type Runner struct {
	ctx     context.Context
	workers []Worker
	started bool
}

func (r *Runner) Start(numWorkers int, workerConfig *WorkerConfig) {
	if r.started {
		panic(errors.New("runner already started. Don't call me twice!"))
	}
	r.started = true
	r.workers = make([]Worker, numWorkers)

	log.FromContext(r.ctx).Info().Msgf("Starting %d workers", numWorkers)

	for k := 0; k < numWorkers; k++ {
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
