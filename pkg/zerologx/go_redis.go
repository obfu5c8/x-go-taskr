package zerologx

import (
	"context"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type GoRedisLogger struct {
	level zerolog.Level
}

func (l GoRedisLogger) Printf(ctx context.Context, format string, v ...interface{}) {
	log.Ctx(ctx).WithLevel(l.level).Msgf(format, v...)
}
