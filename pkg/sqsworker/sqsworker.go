package sqsworker

import "context"

func Launch(ctx context.Context, numWorkers int, workerDef *WorkerDefinition) Runner {
	runner := Runner{}
	runner.Launch(ctx, numWorkers, workerDef)
	return runner
}
