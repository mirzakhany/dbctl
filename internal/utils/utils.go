package utils

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

var DefaultExistSignals = []os.Signal{syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT}

// ContextWithOsSignal returns a context with by default is listening to
// SIGHUP, SIGINT, SIGTERM, SIGQUIT os signals to cancel
func ContextWithOsSignal(sig ...os.Signal) context.Context {
	if len(sig) == 0 {
		sig = DefaultExistSignals
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

func Contain(src []string, target, alias string) bool {
	for _, s := range src {
		if s == target || s == alias {
			return true
		}
	}
	return false
}
