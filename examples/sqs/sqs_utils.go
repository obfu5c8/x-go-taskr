package main

import (
	"context"
	"sync"

	"github.com/WeTransfer/go-telemetry/log"
	"github.com/aws/smithy-go/middleware"
)

type threadSafeCounter struct {
	count uint64
	mutex sync.Mutex
}

func (c *threadSafeCounter) Increment() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.count += 1
}

func (c *threadSafeCounter) Get() uint64 {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	return c.count
}

var newRequestCounterMiddleware = func(counter *threadSafeCounter) middleware.DeserializeMiddleware {

	return middleware.DeserializeMiddlewareFunc("DefaultBucket", func(
		ctx context.Context, in middleware.DeserializeInput, next middleware.DeserializeHandler,
	) (
		out middleware.DeserializeOutput, metadata middleware.Metadata, err error,
	) {

		// Middleware must call the next middleware to be executed in order to continue execution of the stack.
		// If an error occurs, you can return to prevent further execution.
		counter.Increment()
		log.FromContext(ctx).Info().Uint64("total_requests", counter.Get()).Msg("AWS Request logged")
		return next.HandleDeserialize(ctx, in)
	})
}
