package main

import (
	"context"
	"log"
	"math"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	telemetry "github.com/WeTransfer/go-telemetry"
	wtlog "github.com/WeTransfer/go-telemetry/log"

	"github.com/WeTransfer/x-go-taskr/pkg/taskr"
	"github.com/WeTransfer/x-go-taskr/pkg/taskrsqs"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/aws/smithy-go/middleware"
)

func main() {

	// var NUM_WORKERS = 1
	var SQS_QUEUE_URL = "http://localhost:4566//queue/eu-west-1/000000000000/test-queue"

	logger, ctx := telemetry.MustInitFromEnv()
	defer telemetry.Close()
	// Grubby workaround to add caller to logs in dev. This should be added to WeTransfer/go-telemetry
	ctx = logger.With().Caller().Logger().WithContext(ctx)

	// Listen for SIGINT to start a graceful shutdown
	cancelChan := make(chan os.Signal, 1)
	signal.Notify(cancelChan, syscall.SIGINT)

	// Set up the AWS client
	sqsClient := mustCreateSQSClient(ctx)
	queueUrl := SQS_QUEUE_URL

	// The handler is what processes individual messages
	handler := taskr.NewTaskHandler(createMessageHandlerFunc(),
		taskr.WithMiddleware(
			taskrsqs.InMemoryExactlyOnceProcessingMiddleware(),
		))

	// The worker config defines worker parameters
	workerConfig := taskrsqs.NewWorkerConfig(queueUrl, sqsClient, handler,
		taskrsqs.WithBatchSize(1))

	// Start running one or more workers
	taskRunner := taskrsqs.Run(ctx, &workerConfig,
		taskrsqs.WithNumberOfWorkers(1))

	<-cancelChan
	logger.Info().Msg("Draining workers... please wait...")

	taskRunner.Drain()
	logger.Info().Msg("All task workers drained")

}

func createMessageHandlerFunc() taskr.TaskHandlerFunc[sqstypes.Message] {
	return func(ctx context.Context, msg sqstypes.Message) error {

		// Sleep for random time to simulate work
		nSeconds := 1 + int64(math.Round(rand.Float64()*5))
		wtlog.FromContext(ctx).Info().Int64("delay", nSeconds).Msgf("Handling message %v", msg.MessageId)
		time.Sleep(time.Duration(int64(time.Second) * nSeconds))

		return nil
	}
}

func mustCreateSQSClient(ctx context.Context) *sqs.Client {
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			PartitionID:   "aws",
			URL:           "http://localhost:4566",
			SigningRegion: region,
		}, nil

	})

	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithEndpointResolverWithOptions(customResolver),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("dummy", "dummy", "dummy")),
		awsconfig.WithRegion("eu-west-1"))
	if err != nil {
		log.Fatalf("configuration error: %v", err)
	}

	reqCounter := threadSafeCounter{}

	return sqs.NewFromConfig(cfg, sqs.WithAPIOptions(func(s *middleware.Stack) error {
		return s.Deserialize.Add(newRequestCounterMiddleware(&reqCounter), middleware.After)
	}))
}
