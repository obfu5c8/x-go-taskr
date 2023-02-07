package redisam1

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/WeTransfer/x-go-taskr/pkg/sqsworker"
	"github.com/redis/go-redis/v9"
)

func Middleware(redisClient *redis.Client) sqsworker.HandlerMiddleware {
	repo := NewRepo(redisClient)
	return sqsworker.AtMostOnceMiddleware(&repo)
}

func NewRepo(redisClient *redis.Client) RedisAtMostOnceMiddlewareRepo {
	return RedisAtMostOnceMiddlewareRepo{
		redis: redisClient,
	}
}

var _ sqsworker.AtMostOnceMiddlewareRepo = &RedisAtMostOnceMiddlewareRepo{}

type RedisAtMostOnceMiddlewareRepo struct {
	redis *redis.Client
}

var ErrFailedToLock = errors.New("redisam1: failed to lock message")
var ErrFailedToRelease = errors.New("redisam1: failed to release message")

// Release implements sqsworker.AtMostOnceMiddlewareRepo
func (r *RedisAtMostOnceMiddlewareRepo) Release(ctx context.Context, msgId string, completed bool) error {
	if completed {
		_, err := r.redis.Set(ctx, msgId, "complete", time.Hour).Result()
		if err != nil {
			return fmt.Errorf("%w: %v", ErrFailedToRelease, err)
		}
	} else {
		_, err := r.redis.Del(ctx, msgId).Result()
		if err != nil {
			return fmt.Errorf("%w: %v", ErrFailedToRelease, err)
		}
	}
	return nil
}

// TryGet implements sqsworker.AtMostOnceMiddlewareRepo
func (r *RedisAtMostOnceMiddlewareRepo) TryGet(ctx context.Context, msgId string) error {

	// Only set the value if it doesn't already exist
	// SET <key> pending NX GET EX 3600
	cmd := r.redis.Do(ctx, "SET", msgId, "pending", "NX", "GET", "EX", 3600)
	oldVal, err := cmd.Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			// All good, no error here
			return nil
		}
		return fmt.Errorf("%w: %v", ErrFailedToLock, err)
	}
	if oldVal != nil {
		if oldVal == "complete" {
			return sqsworker.ErrMessageAlreadyProcessed
		} else {
			return sqsworker.ErrMessageLocked
		}
	}

	return nil
}
