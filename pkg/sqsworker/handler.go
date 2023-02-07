package sqsworker

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

type Handler interface {
	ExecuteTask(context.Context, types.Message) error
}

type HandlerMiddleware func(Handler) Handler
