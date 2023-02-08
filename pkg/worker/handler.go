package worker

import "context"

type Handler[T any] interface {
	ExecuteTask(context.Context, T) error
}

type HandlerMiddleware[T any] func(Handler[T]) Handler[T]

// Create a Handler instance from a func
func HandlerFunc[T any](handler handlerFunc[T]) Handler[T] {
	return basicHandler[T]{handler}
}

// Apply middleware to a handler, wrapping it and returning a single Handler containing the whole chain
//
// Middlewares are applied in the order they are defined, from the outside in. For example, if you call
// HandlerWithMiddleware(handler, A, B, C), the resulting chain will be A->B->C->handler
func HandlerWithMiddleware[T any](handler Handler[T], middleware ...HandlerMiddleware[T]) Handler[T] {
	for k := len(middleware) - 1; k >= 0; k-- {
		handler = middleware[k](handler)
	}
	return handler
}

type handlerFunc[T any] func(context.Context, T) error

type basicHandler[T any] struct {
	handler handlerFunc[T]
}

func (h basicHandler[T]) ExecuteTask(ctx context.Context, input T) error {
	return h.handler(ctx, input)
}
