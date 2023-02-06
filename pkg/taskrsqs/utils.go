package taskrsqs

import (
	"context"

	"github.com/WeTransfer/go-telemetry/log"
	"github.com/rs/zerolog"
)

func contextWithLogFields(ctx context.Context, fn func(zerolog.Context) zerolog.Context) context.Context {
	logCtx := fn(log.FromContext(ctx).With())
	return logCtx.Logger().WithContext(ctx)
}
