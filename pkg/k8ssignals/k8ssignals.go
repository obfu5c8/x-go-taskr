package k8ssignals

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

func WithShutdownSignals(ctx context.Context) (context.Context, chan os.Signal) {

	ctx, cancelCtx := context.WithCancel(ctx)

	sigTermChan := make(chan os.Signal, 1)
	signal.Notify(sigTermChan, syscall.SIGINT, syscall.SIGTERM)

	sigKillChan := make(chan os.Signal, 1)
	signal.Notify(sigKillChan, syscall.SIGKILL)

	go func() {
		<-sigKillChan
		cancelCtx()
	}()

	return ctx, sigTermChan
}
