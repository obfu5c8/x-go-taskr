package main

import (
	"context"
	"errors"
	"math"
	"math/rand"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

type MyHandler struct {
}

func (h MyHandler) ExecuteTask(ctx context.Context, msg types.Message) error {
	// Sleep for random time to simulate work
	nSeconds := 1 + int64(math.Round(rand.Float64()*7))
	time.Sleep(time.Duration(int64(time.Second) * nSeconds))

	// Simulate failures in a few
	if rand.Intn(10) == 1 {
		return errors.New("simulated runtime failure")
	}

	return nil
}
