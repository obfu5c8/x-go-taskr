package worker

import "context"

func Launch(ctx context.Context, workerFactory WorkerFactory, opts ...LaunchOption) (Runtime, error) {
	runtime := NewRuntime()
	if err := runtime.Launch(ctx, workerFactory, opts...); err != nil {
		return runtime, err
	}
	return runtime, nil
}
