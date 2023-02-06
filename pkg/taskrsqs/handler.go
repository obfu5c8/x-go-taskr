package taskrsqs

import (
	"github.com/WeTransfer/x-go-taskr/pkg/taskr"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

type TaskHandler taskr.TaskHandler[types.Message]
type TaskHandlerFunc taskr.TaskHandlerFunc[types.Message]

type TaskHandlerMiddleware func(next TaskHandler) TaskHandler
