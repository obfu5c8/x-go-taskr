package main

import (
	"context"
	"log"
	"time"

	telemetry "github.com/WeTransfer/go-telemetry"
	"github.com/WeTransfer/x-go-taskr/pkg/k8ssignals"
	"github.com/WeTransfer/x-go-taskr/pkg/redisam1"
	"github.com/WeTransfer/x-go-taskr/pkg/sqsworker"
	"github.com/WeTransfer/x-go-taskr/pkg/worker"
	"github.com/WeTransfer/x-go-taskr/pkg/zerologx"
	zerologaws "github.com/WeTransfer/x-go-taskr/pkg/zerologx/aws"
	"github.com/redis/go-redis/v9"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/caarlos0/env/v7"
)

type Config struct {
	Env             string `env:"ENV" envDefault:"local"`
	SQSQueueUrl     string `env:"SQS_QUEUE_URL"`
	NumWorkers      int    `env:"NUM_WORKERS"`
	WorkerBatchSize int    `env:"WORKER_BATCH_SIZE"`
}

func main() {

	c := Config{}
	if err := env.Parse(&c); err != nil {
		log.Fatalf("unable to parse config: %v", err)
	}

	logger, ctx := telemetry.MustInitFromEnv()
	defer telemetry.Close()
	// Grubby workaround to add caller to logs in dev. This should be added to WeTransfer/go-telemetry
	logger = logger.With().Caller().Logger()
	ctx = logger.WithContext(ctx)

	// Listen for SIGTERM or SIGINT to start a graceful shutdown
	// SIGKILL will cancel the context immediately too (does this matter?)
	ctx, shutdownChan := k8ssignals.WithShutdownSignals(ctx)

	// Set up the AWS client
	sqsClient := mustCreateSQSClient(ctx)
	queueUrl := c.SQSQueueUrl

	// The handler is what processes individual messages
	handler := sqsworker.HandlerWithMiddleware(
		MyHandler{},
		atMostOnceProcessingProtectionMiddleware(ctx),
	)

	// Worker definition describes the parameters for any workers we spawn
	workerDef := sqsworker.Define(sqsClient, queueUrl, handler,
		sqsworker.WithBatchSize(c.WorkerBatchSize),
		sqsworker.WithInitialVisibilityTimeout(time.Second*30),
		sqsworker.WithExtendedVisibilityTimeout(time.Second*15),
		sqsworker.WithVisibilityTimeoutGraceTime(time.Second*10))

	// Runner manages the lifecycle of workers
	runner, _ := worker.Launch(ctx, &workerDef,
		worker.WithInstances(c.NumWorkers))

	// Wait for signal to exit
	<-shutdownChan
	logger.Info().Msg("Shutting down... please wait...")

	// Drain all the workers gracefully
	drainStartTime := time.Now()
	runner.Shutdown()
	logger.Info().Dur("drain_time", time.Since(drainStartTime)).Msg("Shutdown Complete")

}

func atMostOnceProcessingProtectionMiddleware(ctx context.Context) sqsworker.HandlerMiddleware {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	redis.SetLogger(zerologx.GoRedisLogger{})
	return redisam1.Middleware(rdb)
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
		config.WithLogger(zerologaws.Logger{}),
		config.WithClientLogMode(aws.LogRetries|aws.LogRequestWithBody|aws.LogResponseWithBody))

	if err != nil {
		log.Fatalf("configuration error: %v", err)
	}

	return sqs.NewFromConfig(cfg)
}
