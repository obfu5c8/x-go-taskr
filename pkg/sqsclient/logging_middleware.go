package sqsclient

import (
	"context"

	"github.com/WeTransfer/go-telemetry/log"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/smithy-go/logging"
)

func WithLoggingMiddleware() config.LoadOptionsFunc {
	return config.WithLogger(loggingAdapter{})
}

type loggingAdapter struct {
	ctx context.Context
}

var _ logging.Logger = loggingAdapter{}

// Logf implements logging.Logger
func (l loggingAdapter) Logf(classification logging.Classification, format string, v ...interface{}) {
	if l.ctx == nil {
		// Can't do anything
		return
	}
	log.FromContext(l.ctx).Trace().Msgf("AWS "+format, v...)
}

func (l loggingAdapter) WithContext(ctx context.Context) logging.Logger {
	return loggingAdapter{ctx}
}
