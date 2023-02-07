package sqsworker

import (
	"errors"
	"time"

	"github.com/WeTransfer/x-go-taskr/pkg/worker"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

type WorkerDefinition struct {
	// Message handler, invoked to process each individual message
	Handler Handler
	// BatchSize determines the maximum number of messages that are pulled
	// from the SQS queue on each iteration. AWS restricts this to 0..10.
	// The lower the batch size, the more efficient the compute usage will
	// be since less time is spent waiting for slow-processing messages, but
	// the more calls to SQS will be made, resulting in higher costs.
	BatchSize int
	// SQS client to use
	SQSClient *sqs.Client
	// URL of the SQS queue to pull messages from
	QueueUrl string
	// VisibilityTimeout (lease time) to set on fetched messages initially.
	// This should be set as close as you can to expected processing duration
	InitialVisibilityTimeout time.Duration
	// Amount of time to extend messages' visibilityTimeout by each time (if needed)
	ExtendedVisibilityTimeout time.Duration
	// How long before visibility timeout expires should we request an extension?
	VisibilityTimeoutGraceTime time.Duration
	// Long polling wait time - how long to wait for new messages to arrive
	// before aborting and trying again
	WaitTime time.Duration
	// Passed to SQS client ReceiveMessage call
	MessageAttributeNames []string
}

var _ worker.WorkerFactory = &WorkerDefinition{}

func (d WorkerDefinition) CreateWorker() worker.Worker {
	return &Worker{
		Def: &d,
	}
}

// Create a new WorkerDefinition with default values populated
func NewDefinition(sqsClient *sqs.Client, queueUrl string, handler Handler, options ...WorkerDefinitionOption) WorkerDefinition {
	def := WorkerDefinition{
		SQSClient:                  sqsClient,
		Handler:                    handler,
		QueueUrl:                   queueUrl,
		BatchSize:                  10,
		InitialVisibilityTimeout:   time.Minute * 5,
		ExtendedVisibilityTimeout:  time.Minute * 2,
		VisibilityTimeoutGraceTime: time.Second * 30,
		WaitTime:                   time.Second * 20,
		MessageAttributeNames: []string{
			string(types.QueueAttributeNameAll),
		},
	}

	for _, option := range options {
		def = option(def)
	}

	return def
}

type WorkerDefinitionOption func(WorkerDefinition) WorkerDefinition

// Apply middleware to the handler chain
// Middleware is applied in the order it is defined (outside to inside)
//
// Calling WithMiddleware multiple times will result in predictable but
// unintuitive ordering of middlewares. For example, if you call first
// WithMiddleware(A,B,C), and then separately WithMiddleware(D,E,F), the
// resulting chain will be D->E->F->A->B->C->handler
func WithMiddleware(middleware ...HandlerMiddleware) WorkerDefinitionOption {
	return func(wd WorkerDefinition) WorkerDefinition {
		for k := len(middleware) - 1; k >= 0; k-- {
			wd.Handler = middleware[k](wd.Handler)
		}
		return wd
	}
}

// Sets the worker batch size
func WithBatchSize(batchSize int) WorkerDefinitionOption {
	if batchSize < 0 || batchSize > 10 {
		panic(errors.New("BatchSize must be 0 <= n <= 10"))
	}
	return func(wd WorkerDefinition) WorkerDefinition {
		wd.BatchSize = batchSize
		return wd
	}
}

// Sets the visibility timeout to set when retrieving messages
func WithInitialVisibilityTimeout(timeout time.Duration) WorkerDefinitionOption {
	if timeout < 0 || timeout > time.Hour*12 {
		panic(errors.New("Maximum supported VisibilityTimeout is 12 hours"))
	}
	return func(wd WorkerDefinition) WorkerDefinition {
		wd.InitialVisibilityTimeout = timeout
		return wd
	}
}

// Sets the visibility timeout to set when extending message leases
func WithExtendedVisibilityTimeout(timeout time.Duration) WorkerDefinitionOption {
	if timeout < 0 || timeout > time.Hour*12 {
		panic(errors.New("Maximum supported VisibilityTimeout is 12 hours"))
	}
	return func(wd WorkerDefinition) WorkerDefinition {
		wd.ExtendedVisibilityTimeout = timeout
		return wd
	}
}

// Sets the VisibilityTimeoutGraceTime
func WithVisibilityTimeoutGraceTime(timeout time.Duration) WorkerDefinitionOption {
	return func(wd WorkerDefinition) WorkerDefinition {
		wd.VisibilityTimeoutGraceTime = timeout
		return wd
	}
}

// Sets the WaitTime
func WithWaitTime(waitTime time.Duration) WorkerDefinitionOption {
	if waitTime < 0 || waitTime > time.Second*20 {
		panic(errors.New("Maximum WaitTime is 20 seconds"))
	}
	return func(wd WorkerDefinition) WorkerDefinition {
		wd.WaitTime = waitTime
		return wd
	}
}

// Sets MessageAttributeNames
func WithMessageAttributeNames(names []string) WorkerDefinitionOption {
	return func(wd WorkerDefinition) WorkerDefinition {
		wd.MessageAttributeNames = names
		return wd
	}
}
