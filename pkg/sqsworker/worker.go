package sqsworker

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/WeTransfer/x-go-taskr/pkg/worker"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/rs/zerolog/log"
)

type Worker struct {
	Def *WorkerDefinition

	ctx context.Context

	runningStateMutex sync.Mutex
	running           bool

	drainChan chan bool
	doneChan  chan bool
}

var _ worker.Worker = &Worker{}

func (w *Worker) Context() context.Context {
	return w.ctx
}

func (w *Worker) setRunning(running bool) {
	w.runningStateMutex.Lock()
	defer w.runningStateMutex.Unlock()
	w.running = running
}

func (w *Worker) Running() bool {
	w.runningStateMutex.Lock()
	defer w.runningStateMutex.Unlock()
	return w.running
}

// Drain any running worker tasks and exit
// This method will block until the worker is shut down
func (w *Worker) Shutdown() {
	if !w.Running() {
		// Already stopped, do nothing
		return
	}
	log.Ctx(w.ctx).Debug().Msg("Stop called")
	w.drainChan <- true
	log.Ctx(w.ctx).Debug().Msg("Drain complete")
	<-w.doneChan
}

// Start the worker in a separate goroutine, returning immediately
func (w *Worker) Start(ctx context.Context) {
	if w.Running() {
		panic(errors.New("worker already running"))
	}
	w.setRunning(true)

	w.ctx = ctx
	w.doneChan = make(chan bool)
	w.drainChan = make(chan bool)

	go w.run()
}

func (w *Worker) run() {
	w.running = true
	nIterations := uint64(0)

iteration_loop:
	for true {
		select {
		case <-w.drainChan:
			log.Ctx(w.ctx).Debug().Msg("Drain received, halting iterations")
			break iteration_loop
		default:
			nIterations++
			if nIterations >= ^uint64(0) {
				nIterations = 0
			}

			// Add iteration id to logging context
			ctx := log.Ctx(w.ctx).With().
				Uint64("wit", nIterations).
				Logger().WithContext(w.ctx)

			if err := w.iterate(ctx); err != nil {
				if !errors.Is(err, ErrNoNewMessages) {
					log.Ctx(ctx).Err(err).Msg("Iteration failed")
					// time.Sleep(time.Second * 1) // Just to slow down devtime runaway loops of death
				}
			}
		}
	}

	log.Ctx(w.ctx).Debug().Msg("Iteration loop broken, worker is done")
	w.doneChan <- true
}

func (w *Worker) iterate(ctx context.Context) error {

	logger := log.Ctx(ctx)

	logger.Debug().Msg("Starting iteration")

	// Get a batch of messages from the queue
	req := createReceiveMessageInput(w)

	logger.Debug().Msg("Looking for new messages")
	resp, err := w.Def.SQSClient.ReceiveMessage(w.Context(), req)
	if err != nil {
		return fmt.Errorf("failed to receive message: %w", err)
	}
	msgs := resp.Messages
	numMsgs := len(msgs)

	if numMsgs == 0 {
		// We didn't get any messages. Give up
		logger.Debug().Msg("No messages received")
		return ErrNoNewMessages
	}

	// Option to bail out early if a shutdown has been requested
	if !w.Running() {
		log.Ctx(ctx).Debug().Msg("Shutdown requested, aborting iteration")
		return ErrShutdownRequested
	}

	// Now we have our batch of messages, we need do two things in parallel
	//
	// Firstly, we need to start a routine to monitor the visibility time of the batch
	// and extend it if our messages are still being processed
	//
	// Then in parallel we fork out processing of each individual message
	logger.Debug().Msgf("Got %d messages", numMsgs)

	// Start a background routine to tell us when the lease is soon to expire
	timer := time.NewTimer(w.Def.InitialVisibilityTimeout - w.Def.VisibilityTimeoutGraceTime) // Initial visibility period

	// Collect results from handlers
	handlerResultC := make(chan dispatchResult, len(msgs))
	handlerResults := make([]dispatchResult, 0, len(msgs))

	// Spawn handler routines for each message in the background
	for _, msg := range msgs {
		go func(msg types.Message) {
			execCtx := log.Ctx(ctx).With().
				Str("id", *msg.MessageId).
				Logger().WithContext(ctx)

			result := w.Def.Handler.ExecuteTask(execCtx, msg)
			log.Ctx(execCtx).Info().
				Err(result).
				Bool("success", result == nil).
				Msg("Message processed")
			handlerResultC <- dispatchResult{
				msg,
				result,
			}
		}(msg)
	}

	// Wait for results or lease expiry events until all results are collected
	done := false
	for !done {
		select {
		case t := <-timer.C:
			fmt.Printf("t: %v\n", t)
			// Extend lease for all messages
			go func() {
				w.extendMessageLeases(ctx, msgs)
				timer.Reset(w.Def.ExtendedVisibilityTimeout - w.Def.VisibilityTimeoutGraceTime) // Extend visibility period
			}()

		case result := <-handlerResultC:
			// Capture result from handler
			handlerResults = append(handlerResults, result)
			if len(handlerResults) == len(msgs) {
				done = true
				timer.Stop()
			}
		}
	}

	log.Ctx(ctx).Debug().Msg("Dispatch complete, returning results")

	// Release/delete messages
	// No point hanging around while these messages are deleted,
	// whack it into a background routine and continue to the next iteration
	msgsToDelete := Map(filter(handlerResults, processedSuccessfully), getMessage)
	go w.deleteMessages(ctx, msgsToDelete)
	return nil

}

// Deletes a batch of messages from the queue once they have been processed
func (w *Worker) deleteMessages(ctx context.Context, messages []types.Message) {
	if len(messages) < 1 {
		return // nothing to do
	}

	entries := make([]types.DeleteMessageBatchRequestEntry, len(messages))
	ids := make([]string, len(messages))
	for i, msg := range messages {
		entries[i] = types.DeleteMessageBatchRequestEntry{
			Id:            msg.MessageId,
			ReceiptHandle: msg.ReceiptHandle,
		}
		ids[i] = *msg.MessageId
	}
	log.Ctx(ctx).Debug().Interface("message_ids", ids).Msgf("Got %d messages to delete", len(messages))

	delReq := &sqs.DeleteMessageBatchInput{
		Entries:  entries,
		QueueUrl: &w.Def.QueueUrl,
	}

	_, err := w.Def.SQSClient.DeleteMessageBatch(w.ctx, delReq)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("Message delete failed despite successful processing")
	}
}

// Extend visibilityPeriod lease for a batch of messages
func (w *Worker) extendMessageLeases(ctx context.Context, messages []types.Message) {
	log.Ctx(ctx).Debug().Dur("by", w.Def.ExtendedVisibilityTimeout).Msg("Extending message lease time")
	log.Ctx(ctx).Warn().Msg("Message lease extension not implemented")
}

func createReceiveMessageInput(w *Worker) *sqs.ReceiveMessageInput {
	return &sqs.ReceiveMessageInput{
		VisibilityTimeout:     int32(w.Def.InitialVisibilityTimeout / time.Second),
		QueueUrl:              &w.Def.QueueUrl,
		WaitTimeSeconds:       int32(w.Def.WaitTime / time.Second),
		MaxNumberOfMessages:   int32(w.Def.BatchSize),
		MessageAttributeNames: w.Def.MessageAttributeNames,
	}
}

func filterProcessed(src []taskResult) []types.Message {
	matches := make([]types.Message, 0, len(src))
	for _, result := range src {
		if result.processed {
			matches = append(matches, result.msg)
		}
	}
	return matches
}

type taskResult struct {
	msg       types.Message
	err       error
	processed bool
}

type dispatchResult struct {
	msg types.Message
	err error
}

func filter[T any](src []T, predicate func(T) bool) []T {
	var matches []T
	for _, item := range src {
		if predicate(item) {
			matches = append(matches, item)
		}
	}
	return matches
}

// Filter predicate matching handlers results with no errors
func processedSuccessfully(result dispatchResult) bool {
	return result.err == nil
}

func Map[TIn any, TOut any](src []TIn, mapFn func(TIn) TOut) []TOut {
	results := make([]TOut, len(src))

	for k := 0; k < len(src); k++ {
		results[k] = mapFn(src[k])
	}
	return results
}

func getMessage(result dispatchResult) types.Message {
	return result.msg
}
