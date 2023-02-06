package taskrsqs

import (
	"errors"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

type WorkerConfig struct {
	// Number of messages per batch. Must be 1 <= n <= 10
	BatchSize int
	// URL of the SQS queue to fetch messages from
	QueueUrl string
	// VisibilityTimeout (lease time) to set on fetched messages initially.
	// This should be set as close as you can to expected processing duration
	InitialVisibilityTimeout time.Duration
	// Duration to extend VisibilityTimeout by if it looks like we'll run
	// over the InitialVisibilityTimeout
	ExtendVisibilityTimeout time.Duration
	// Time before the visibility timeout that we should extend at.
	// Example: If VisibilityTimeout is 30s and GraceTime is 10s, then
	//          after 20s we will extend the visibilityTimeout to hold
	//          on to the message lease
	ExtendVisibilityTimeoutGraceTime time.Duration
	// Long polling wait time - how long to wait for new messages to arrive
	// before aborting and trying again
	WaitTime time.Duration
	// Passed to SQS client ReceiveMessage call
	MessageAttributeNames []string
	// The handler definition used to process messages
	Handler TaskHandler
	// Configured SQS client
	SQSClient *sqs.Client
}

type NewWorkerConfigOption func(WorkerConfig) WorkerConfig

func NewWorkerConfig(queueUrl string, sqsClient *sqs.Client, handler TaskHandler, opts ...NewWorkerConfigOption) WorkerConfig {
	config := WorkerConfig{
		BatchSize:                10,
		InitialVisibilityTimeout: time.Second * 30,
		ExtendVisibilityTimeout:  time.Second * 30,
		WaitTime:                 time.Second * 20,
		MessageAttributeNames: []string{
			string(types.QueueAttributeNameAll),
		},
		QueueUrl:  queueUrl,
		SQSClient: sqsClient,
		Handler:   handler,
	}

	for _, opt := range opts {
		config = opt(config)
	}

	return config
}

func WithInitialVisibilityTimeout(timeout time.Duration) NewWorkerConfigOption {
	return func(wc WorkerConfig) WorkerConfig {
		wc.InitialVisibilityTimeout = timeout
		return wc
	}
}
func WithExtendVisibilityTimeout(timeout time.Duration) NewWorkerConfigOption {
	return func(wc WorkerConfig) WorkerConfig {
		wc.ExtendVisibilityTimeout = timeout
		return wc
	}
}
func WithWaitTime(timeout time.Duration) NewWorkerConfigOption {
	return func(wc WorkerConfig) WorkerConfig {
		wc.WaitTime = timeout
		return wc
	}
}
func WithMessageAttributeNames(names []string) NewWorkerConfigOption {
	return func(wc WorkerConfig) WorkerConfig {
		wc.MessageAttributeNames = names
		return wc
	}
}

func WithBatchSize(n int) NewWorkerConfigOption {
	if n < 1 || n > 10 {
		panic(errors.New("batch size must be 1 <= n <= 10"))
	}
	return func(wc WorkerConfig) WorkerConfig {
		wc.BatchSize = n
		return wc
	}
}
