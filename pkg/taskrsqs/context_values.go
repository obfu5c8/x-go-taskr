package taskrsqs

import "context"

type workerIterationIdCtxKeyType int

const workerIterationIdCtxKey workerIterationIdCtxKeyType = 0

func getWorkerIterationId(ctx context.Context) uint64 {
	return ctx.Value(workerIterationIdCtxKey).(uint64)
}
