package sqsworker

import (
	"github.com/WeTransfer/x-go-taskr/pkg/worker"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

type Handler = worker.Handler[types.Message]
type HandlerMiddleware = worker.HandlerMiddleware[types.Message]

func HandlerWithMiddleware(handler Handler, middleware ...HandlerMiddleware) Handler {
	return worker.HandlerWithMiddleware(handler, middleware...)
}
