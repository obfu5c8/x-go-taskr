package taskr

import "context"

type TaskHandlerFunc[T any] func(context.Context, T) error

type TaskHandlerMiddleware[T any] func(TaskHandler[T]) TaskHandler[T]

type TaskHandler[T any] interface {
	HandleTask(context.Context, T) error
}

// Utility function for creating a TaskHandler struct from a function
func NewTaskHandler[T any](handlerFn TaskHandlerFunc[T], opts ...NewTaskHandlerOption[T]) TaskHandler[T] {
	var handler TaskHandler[T] = taskHandler[T]{
		handler: handlerFn,
	}

	for _, opt := range opts {
		handler = opt(handler)
	}

	return handler
}

type taskHandler[T any] struct {
	handler TaskHandlerFunc[T]
}

func (h taskHandler[T]) HandleTask(ctx context.Context, input T) error {
	return h.handler(ctx, input)
}

// Applies a middleware chain to a root handler from the inside out to produce
// a single MessageHandler that will execute all the supplied handlers.
//
// Middlewares are wrapped in the order they are defined, meaning the actual
// execution order will be the reverse of the order they are defined as arguments
func ChainMiddleware[T any](rootHandler TaskHandler[T], middleware ...TaskHandlerMiddleware[T]) TaskHandler[T] {
	chain := rootHandler

	for k := len(middleware) - 1; k >= 0; k-- {
		next := middleware[k]
		chain = next(chain)
	}

	return chain
}

type NewTaskHandlerOption[T any] func(TaskHandler[T]) TaskHandler[T]

func WithMiddleware[T any](middleware ...TaskHandlerMiddleware[T]) NewTaskHandlerOption[T] {
	return func(handler TaskHandler[T]) TaskHandler[T] {
		// Reverse the order of middleware to make it read sensibly
		return ChainMiddleware(handler, reverse(middleware)...)
	}
}

func reverse[T any](in []T) []T {
	out := make([]T, 0, len(in))
	for k := len(in) - 1; k >= 0; k-- {
		out = append(out, in[k])
	}
	return out
}
