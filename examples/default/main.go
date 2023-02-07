package main

import (
	"context"
	"log"
	"time"

	telemetry "github.com/WeTransfer/go-telemetry"
	"github.com/WeTransfer/x-go-taskr/pkg/k8ssignals"
	"github.com/WeTransfer/x-go-taskr/pkg/sqsclient"
	"github.com/WeTransfer/x-go-taskr/pkg/sqsworker"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

func main() {

	// var NUM_WORKERS = 1
	var SQS_QUEUE_URL = "http://localhost:4566//queue/eu-west-1/000000000000/test-queue"

	logger, ctx := telemetry.MustInitFromEnv()
	defer telemetry.Close()
	// Grubby workaround to add caller to logs in dev. This should be added to WeTransfer/go-telemetry
	ctx = logger.With().Caller().Logger().WithContext(ctx)

	// Listen for SIGTERM or SIGINT to start a graceful shutdown
	ctx, shutdownChan := k8ssignals.WithShutdownSignals(ctx)

	// Set up the AWS client
	sqsClient := mustCreateSQSClient(ctx)
	queueUrl := SQS_QUEUE_URL

	// The handler is what processes individual messages
	handler := MyHandler{}

	// Worker definition describes the parameters for any workers we spawn
	workerDef := sqsworker.NewDefinition(sqsClient, queueUrl, handler,
		sqsworker.WithBatchSize(10),
		sqsworker.WithInitialVisibilityTimeout(time.Second*30),
		sqsworker.WithExtendedVisibilityTimeout(time.Second*15),
		sqsworker.WithVisibilityTimeoutGraceTime(time.Second*10))

	// Runner manages the lifecycle of workers
	runner := sqsworker.Launch(ctx, 10, &workerDef)

	// Wait for signal to exit
	<-shutdownChan
	logger.Info().Msg("Draining workers... please wait...")

	// Drain all the workers gracefully
	drainStartTime := time.Now()
	runner.Stop()
	logger.Info().Dur("draintime", time.Since(drainStartTime)).Msg("All task workers drained")

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
		awsconfig.WithRegion("eu-west-1"),
		// Try a logging adapter
		sqsclient.WithLoggingMiddleware(),
		config.WithClientLogMode(aws.LogRetries|aws.LogRequestWithBody|aws.LogResponseWithBody))

	if err != nil {
		log.Fatalf("configuration error: %v", err)
	}

	return sqs.NewFromConfig(cfg)
}
