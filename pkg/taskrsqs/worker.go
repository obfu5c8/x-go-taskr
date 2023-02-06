package taskrsqs

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/WeTransfer/go-telemetry/log"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/rs/zerolog"
)

var ErrWorkerAlreadyRunning = errors.New("worker already running")

func NewWorker(ctx context.Context, config *WorkerConfig) Worker {
	worker := Worker{
		Config: config,
		ctx:    ctx,
	}

	return worker
}

type Worker struct {
	Config    *WorkerConfig
	ctx       context.Context
	cancelCtx context.CancelFunc
	running   bool
	drainChan chan bool
	doneChan  chan bool
}

func (w *Worker) IsRunning() bool {
	return w.running
}

func (w *Worker) Context() context.Context {
	return w.ctx
}

func (w *Worker) Start() {
	if w.running {
		panic(ErrWorkerAlreadyRunning)
	}
	w.ctx, w.cancelCtx = context.WithCancel(w.ctx)

	w.doneChan = make(chan bool)
	w.drainChan = make(chan bool)
	go w.run()
}

func (w *Worker) Stop() {
	w.drainChan <- true
	<-w.doneChan
}

func (w *Worker) run() {
	w.running = true
	nIterations := uint64(0)

iteration_loop:
	for true {
		select {
		case <-w.drainChan:
			log.FromContext(w.ctx).Info().Msg("Drain received, halting iterations")
			break iteration_loop
		default:
			nIterations++
			if nIterations >= ^uint64(0) {
				nIterations = 0
			}
			ctx := context.WithValue(w.ctx, workerIterationIdCtxKey, nIterations)

			if err := w.iterate(ctx); err != nil {
				log.From(w).Err(err).Msg("Iteration failed")
				time.Sleep(time.Second * 1) // Just to slow down devtime runaway loops of death
			}
		}
	}

	log.FromContext(w.ctx).Debug().Msg("Iteration loop broken, worker is done")
	w.doneChan <- true
}

func (w *Worker) iterate(ctx context.Context) error {

	ctx = contextWithLogFields(ctx, func(lc zerolog.Context) zerolog.Context {
		return lc.Uint64("wit", getWorkerIterationId(ctx))
	})
	logger := log.FromContext(ctx)

	// Get a batch of messages from the queue
	req := createReceiveMessageInput(w)
	logger.Trace().Interface("sqs_request", req).Msg("ReceiveMessageInput")

	logger.Debug().Msg("Looking for new messages")
	resp, err := w.Config.SQSClient.ReceiveMessage(w.Context(), req)
	if err != nil {
		return fmt.Errorf("failed to receive message: %w", err)
	}

	msgs := resp.Messages
	numMsgs := len(msgs)

	if numMsgs == 0 {
		// We didn't get any messages. Give up
		logger.Debug().Msg("No messages received")
		return errors.New("no new messages")
	}

	// Now we have our batch of messages, we need do two things in parallel
	//
	// Firstly, we need to start a routine to monitor the visibility time of the batch
	// and extend it if our messages are still being processed
	//
	// Then in parallel we fork out processing of each individual message
	logger.Debug().Msgf("Got %d messages", numMsgs)

	msgResults2 := make([]msgResult2, numMsgs)

	var taskWg sync.WaitGroup
	taskWg.Add(numMsgs)

	for i, msg := range msgs {
		go func(index int, msg types.Message) {

			handlerCtx := contextWithLogFields(ctx, func(lc zerolog.Context) zerolog.Context {
				return lc.Str("msg_id", *msg.MessageId)
			})

			result := w.Config.Handler.HandleTask(handlerCtx, msg)
			msgResults2[index] = msgResult2{
				msg:       msg,
				err:       result,
				processed: result == nil,
			}
			taskWg.Done()
		}(i, msg)
	}

	// Wait for all tasks to finish executing
	taskWg.Wait()

	// Stop the lease extender

	// Release/delete messages
	// No point hanging around while these messages are deleted,
	// whack it into a background routine and continue to the next iteration
	msgsToDelete := filterProcessed(msgResults2)
	logger.Debug().Interface("message_ids", msgsToDelete).Msgf("Got %d messages to delete", len(msgsToDelete))
	go w.deleteMessages(ctx, msgsToDelete)
	return nil

}

func (w *Worker) deleteMessages(ctx context.Context, messages []types.Message) {
	entries := make([]types.DeleteMessageBatchRequestEntry, len(messages))
	for i, msg := range messages {
		entries[i] = types.DeleteMessageBatchRequestEntry{
			Id:            msg.MessageId,
			ReceiptHandle: msg.ReceiptHandle,
		}
	}
	delReq := &sqs.DeleteMessageBatchInput{
		Entries:  entries,
		QueueUrl: &w.Config.QueueUrl,
	}
	log.FromContext(ctx).Trace().Interface("sqs_request", delReq).Msg("DeleteMessageInput")

	log.FromContext(ctx).Debug().Msg("Deleting message")
	delRes, err := w.Config.SQSClient.DeleteMessageBatch(w.ctx, delReq)
	log.FromContext(ctx).Trace().Interface("sqs_response", delRes).AnErr("sqs_error", err).Msg("DeleteMessage response")
	if err != nil {
		log.FromContext(ctx).Err(err).Msg("Message delete failed despite successful processing")
	}

}

func createReceiveMessageInput(w *Worker) *sqs.ReceiveMessageInput {
	return &sqs.ReceiveMessageInput{
		VisibilityTimeout:   int32(w.Config.InitialVisibilityTimeout / time.Second),
		QueueUrl:            &w.Config.QueueUrl,
		WaitTimeSeconds:     int32(w.Config.WaitTime / time.Second),
		MaxNumberOfMessages: int32(w.Config.BatchSize),
		MessageAttributeNames: []string{
			string(types.QueueAttributeNameAll),
		},
	}
}

func filterProcessed(src []msgResult2) []types.Message {
	matches := make([]types.Message, 0, len(src))
	for _, result := range src {
		if result.processed {
			matches = append(matches, result.msg)
		}
	}
	return matches
}

type msgResult struct {
	err       error
	processed bool
}

type msgResult2 struct {
	msg       types.Message
	err       error
	processed bool
}
