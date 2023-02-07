package sqsworker

import (
	"context"
	"math/rand"
	"sync"

	"github.com/WeTransfer/x-go-taskr/pkg/worker"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/rs/zerolog/log"
)

func AtMostOnceMiddleware(repo AtMostOnceMiddlewareRepo) HandlerMiddleware {
	return func(next Handler) Handler {
		return worker.HandlerFunc(func(ctx context.Context, msg types.Message) (err error) {
			if err = repo.TryGet(ctx, *msg.MessageId); err != nil {
				return err
			}

			log.Ctx(ctx).Debug().Msg("AtMostOnce: Message Locked")

			err = next.ExecuteTask(ctx, msg)
			repo.Release(ctx, *msg.MessageId, err == nil)
			log.Ctx(ctx).Debug().Msg("AtMostOnce: Message Released")
			return err
		})
	}
}

type AtMostOnceMiddlewareRepo interface {
	TryGet(ctx context.Context, msgId string) error
	Release(ctx context.Context, msgId string, completed bool) error
}

// Provides a simple, thread-safe, in-memory implementation of AtMostOnceMiddleware
// suitable for local dev work. Also has the ability to simulate lock contention
// to test out consumer behaviours.
func InMemoryAtMostOnceMiddleware(simulateContentionPercent float32) HandlerMiddleware {
	return AtMostOnceMiddleware(&inMemoryStoreAtMostOnceMiddlewareRepo{
		mu:                      sync.Mutex{},
		msgs:                    make(map[string]bool),
		simulatedContentionOdds: simulateContentionPercent,
	})
}

type inMemoryStoreAtMostOnceMiddlewareRepo struct {
	mu                      sync.Mutex
	msgs                    map[string]bool
	simulatedContentionOdds float32
}

// Release implements atMostOnceMiddlewareRepo
func (r *inMemoryStoreAtMostOnceMiddlewareRepo) Release(ctx context.Context, msgId string, completed bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if completed {
		r.msgs[msgId] = true
	} else {
		delete(r.msgs, msgId)
	}

	return nil
}

// TryGet implements atMostOnceMiddlewareRepo
func (r *inMemoryStoreAtMostOnceMiddlewareRepo) TryGet(ctx context.Context, msgId string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.simulateContention(); err != nil {
		return err
	}

	isCompleted, exists := r.msgs[msgId]
	if exists {
		if isCompleted {
			return ErrMessageAlreadyProcessed
		} else {
			return ErrMessageLocked
		}
	}

	r.msgs[msgId] = false
	return nil
}

func (r *inMemoryStoreAtMostOnceMiddlewareRepo) simulateContention() error {
	if r.simulatedContentionOdds > 0 {
		exists, isCompleted := randomResult(r.simulatedContentionOdds)
		if exists {
			if isCompleted {
				return ErrMessageAlreadyProcessed
			} else {
				return ErrMessageLocked
			}
		}
	}
	return nil
}

func randomResult(odds float32) (bool, bool) {
	n := rand.Float32()
	if n <= odds {
		return true, rand.Float32() > 0.5
	} else {
		return false, false
	}
}
