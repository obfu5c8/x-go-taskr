package shutdown

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/WeTransfer/x-go-taskr/internal/util"
)

func ListenerWithDeadline(ctx context.Context, deadline time.Duration) (context.Context, chan struct{}) {
	return Listener(ctx, WithDeadline(deadline))
}

// Returns a channel that will receive a signal when either SIGINT or SIGTERM is
// recieved, notifying you to gracefully shut down the application.
// The returned context will also be cancelled when SIGKILL is received.
func Listener(ctx context.Context, opts ...ListenerOption) (context.Context, chan struct{}) {

	cfg := util.ApplyOpts(opts, listenerOptions{
		Deadline:        -1,
		ShutdownSignals: []os.Signal{syscall.SIGINT, syscall.SIGTERM},
		HaltSignals:     []os.Signal{syscall.SIGKILL},
	})

	ctx, cancelCtx := context.WithCancel(ctx)

	sigTermChan := make(chan os.Signal, 1)
	signal.Notify(sigTermChan, cfg.ShutdownSignals...)

	sigKillChan := make(chan os.Signal, 1)
	signal.Notify(sigKillChan, cfg.HaltSignals...)

	shutdownChan := make(chan struct{})

	go func() {
		<-sigKillChan
		cancelCtx()
	}()

	go func() {
		<-sigTermChan
		shutdownChan <- struct{}{}
		if cfg.Deadline > -1 {
			time.Sleep(cfg.Deadline)
			cancelCtx()
		}
	}()

	return ctx, shutdownChan
}

type listenerOptions struct {
	Deadline        time.Duration
	ShutdownSignals []os.Signal
	HaltSignals     []os.Signal
}

type ListenerOption = func(listenerOptions) listenerOptions

func WithDeadline(time time.Duration) ListenerOption {
	return func(lo listenerOptions) listenerOptions {
		lo.Deadline = time
		return lo
	}
}
