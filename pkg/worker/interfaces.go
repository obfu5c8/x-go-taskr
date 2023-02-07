package worker

import "context"

type Worker interface {
	Start(context.Context)
	Shutdown()
}

type WorkerFactory interface {
	CreateWorker() Worker
}
