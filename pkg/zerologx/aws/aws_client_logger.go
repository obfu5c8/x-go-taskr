package zerologxaws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/smithy-go/logging"
	"github.com/rs/zerolog/log"
)

func New() Logger {
	return Logger{}
}

func WithZerologLogging() config.LoadOptionsFunc {
	return config.WithLogger(Logger{})
}

type Logger struct {
	ctx context.Context
}

// Logf implements logging.Logger
func (l Logger) Logf(classification logging.Classification, format string, v ...interface{}) {
	if l.ctx == nil {
		// Can't do anything
		return
	}
	switch classification {
	case logging.Warn:
		log.Ctx(l.ctx).Warn().Msgf("AWS Client: "+format, v...)
	case logging.Debug:
		log.Ctx(l.ctx).Trace().Msgf("AWS Client: "+format, v...)
	}
}

func (l Logger) WithContext(ctx context.Context) logging.Logger {
	return Logger{ctx}
}
