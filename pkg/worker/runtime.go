package worker

import (
	"context"
	"fmt"
	"sync"

	"github.com/WeTransfer/x-go-taskr/internal/dbg"
	"github.com/WeTransfer/x-go-taskr/internal/util"
	"github.com/rs/zerolog/log"
)

func NewRuntime() Runtime {
	return Runtime{
		state: runtimeState{
			&sync.RWMutex{},
			StateIdle,
		},
	}
}

// Runtime is NOT thread safe
type Runtime struct {
	state   runtimeState
	workers []Worker
}

type launchOptions struct {
	NumWorkers int
}

type LaunchOption = func(launchOptions) launchOptions

// Set the number of instances of a Worker that will be launched
func WithInstances(n int) LaunchOption {
	return func(lo launchOptions) launchOptions {
		lo.NumWorkers = n
		return lo
	}
}

// Launch one or more instances of a worker to run in the background
func (r *Runtime) Launch(ctx context.Context, workerFactory WorkerFactory, opts ...LaunchOption) error {

	cfg := util.ApplyOpts(opts, launchOptions{
		NumWorkers: 1,
	})

	if err := r.state.TrySet(StateActive); err != nil {
		return fmt.Errorf("cannot launch workers: %w", err)
	}

	log.Ctx(ctx).Info().
		Msgf("Launching %d workers", cfg.NumWorkers)

	for k := 0; k < cfg.NumWorkers; k++ {
		worker := workerFactory.CreateWorker()
		r.workers = append(r.workers, worker)

		workerCtx := log.Ctx(ctx).With().
			Int("wid", len(r.workers)).
			Logger().WithContext(ctx)

		dbg.Debug(workerCtx).Msg("Starting worker")
		go worker.Start(workerCtx)
	}

	return nil
}

// Gracefully shut down all workers, blocking until they have successfully terminated
func (r *Runtime) Shutdown() {
	if err := r.state.TrySet(StateDraining); err != nil {
		// Although the state change failed, it's ok because
		// it means a shutdown is already in progress, or
		// the runtime is already idle. Either way, we can
		// just return a success here I think
		return
	}

	util.ConcurrentForEach(r.workers, func(worker Worker) {
		worker.Shutdown()
	})

	r.workers = make([]Worker, 1)
	r.state.TrySet(StateIdle)
}
