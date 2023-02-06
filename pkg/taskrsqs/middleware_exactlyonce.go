package taskrsqs

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/WeTransfer/x-go-taskr/pkg/taskr"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

type exactlyOnceProcessingStore interface {
	LockMessage(id *string) error
	ReleaseMessage(id *string, processed bool) error
}

func ExactlyOnceProcessingMiddleware(store exactlyOnceProcessingStore) taskr.TaskHandlerMiddleware[types.Message] {
	return func(next taskr.TaskHandler[types.Message]) taskr.TaskHandler[types.Message] {

		return taskr.NewTaskHandler(func(ctx context.Context, msg types.Message) error {

			if err := store.LockMessage(msg.MessageId); err != nil {
				return fmt.Errorf("failed to lock message: %w", err)
			}

			var nextErr error

			defer func() {
				store.ReleaseMessage(msg.MessageId, nextErr == nil)
			}()

			nextErr = next.HandleTask(ctx, msg)
			return nextErr
		})
	}
}

func InMemoryExactlyOnceProcessingMiddleware() taskr.TaskHandlerMiddleware[types.Message] {
	inMemoryStore := inMemoryExactlyOnceProcessingStore{
		msgs: make(map[*string]bool),
	}
	return ExactlyOnceProcessingMiddleware(&inMemoryStore)
}

type inMemoryExactlyOnceProcessingStore struct {
	msgs  map[*string]bool
	mutex sync.Mutex
}

var _ exactlyOnceProcessingStore = &inMemoryExactlyOnceProcessingStore{}

// LockMessage implements exactlyOnceProcessingStore
func (s *inMemoryExactlyOnceProcessingStore) LockMessage(id *string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if _, ok := s.msgs[id]; ok {
		return errors.New("Locked")
	}

	s.msgs[id] = true
	return nil
}

// ReleaseMessage implements exactlyOnceProcessingStore
func (s *inMemoryExactlyOnceProcessingStore) ReleaseMessage(id *string, processed bool) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	delete(s.msgs, id)
	return nil
}
