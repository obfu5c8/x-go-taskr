package dbg

import (
	"context"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func Trace(ctx context.Context) *zerolog.Event {
	return log.Ctx(ctx).Trace()
}

func Debug(ctx context.Context) *zerolog.Event {
	return log.Ctx(ctx).Debug()
}

func Info(ctx context.Context) *zerolog.Event {
	return log.Ctx(ctx).Info()
}

func Warn(ctx context.Context) *zerolog.Event {
	return log.Ctx(ctx).Warn()
}

func Error(ctx context.Context, err error) *zerolog.Event {
	return log.Ctx(ctx).Err(err)
}
