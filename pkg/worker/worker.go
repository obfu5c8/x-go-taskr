package worker

import "context"

func Launch(ctx context.Context, workerFactory WorkerFactory, opts ...LaunchOption) (Runtime, error) {
	runtime := NewRuntime()
	if err := runtime.Launch(ctx, workerFactory, opts...); err != nil {
		return runtime, err
	}
	return runtime, nil
}

// Workers are the things that do the work.
type Worker interface {
	Start(context.Context)
	Shutdown()
}

// Interface desribing a factory that creates Worker instances
type WorkerFactory interface {
	CreateWorker() Worker
}
