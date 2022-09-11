package pkg

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

var defaultExistSignals = []os.Signal{syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT}

// ContextWithOsSignal returns a context with by default is listening to
// SIGHUP, SIGINT, SIGTERM, SIGQUIT os signals to cancel
func ContextWithOsSignal(sig ...os.Signal) context.Context {
	if len(sig) == 0 {
		sig = defaultExistSignals
	}

	s := make(chan os.Signal, 1)
	signal.Notify(s, sig...)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-s
		cancel()
	}()
	return ctx
}
