package sqsworker

import (
	"context"
	"errors"
	"sync"

	"github.com/WeTransfer/go-telemetry/log"
	"github.com/rs/zerolog"
)

// Runner is NOT thread safe
type Runner struct {
	state   RunningState
	workers []*Worker
}

func (r *Runner) Launch(ctx context.Context, numWorkers int, def *WorkerDefinition) error {

	if r.state == RunningStateDraining {
		return errors.New("Cannot start new workers while draining")
	}
	r.state = RunningStateRunning

	log.FromContext(ctx).Info().
		Interface("definition", def).
		Msgf("Launching %d workers", numWorkers)
	for k := 0; k < numWorkers; k++ {
		r.launchWorker(ctx, def)
	}

	return nil
}

func (r *Runner) launchWorker(ctx context.Context, def *WorkerDefinition) {
	worker := NewWorker(def)
	r.workers = append(r.workers, &worker)

	workerCtx := contextWithLogFields(ctx, func(lc zerolog.Context) zerolog.Context {
		return lc.Int("wid", len(r.workers))
	})
	log.FromContext(workerCtx).Debug().Msg("Starting worker")
	worker.Start(workerCtx)
}

// Gracefully halt all workers and then return once they're done
func (r *Runner) Stop() error {

	switch r.state {
	case RunningStateStopped:
		// Nothing to do, return
		return nil
	case RunningStateDraining:
		return errors.New("Already draining")
	}

	r.state = RunningStateDraining

	var wg sync.WaitGroup
	wg.Add(len(r.workers))
	for _, worker := range r.workers {
		go func(worker *Worker) {
			worker.Drain()
			wg.Done()
		}(worker)
	}

	// Wait for all workers to drain
	wg.Wait()

	r.workers = make([]*Worker, 1)
	r.state = RunningStateStopped
	return nil
}
