package worker

import "context"

type Handler[T any] interface {
	ExecuteTask(context.Context, T) error
}

type HandlerMiddleware[T any] func(Handler[T]) Handler[T]

type Worker interface {
	Start(context.Context)
	Shutdown()
}

type WorkerFactory interface {
	CreateWorker() Worker
}
