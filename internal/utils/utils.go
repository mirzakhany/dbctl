package utils

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/signal"
	"sort"
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

// GetListHash generate a hash from list of strings
func GetListHash(list []string) string {
	sort.Strings(list)
	// create md5 hash of xx
	cc := sha256.Sum256([]byte(fmt.Sprintf("%x", list)))
	return fmt.Sprintf("%x", cc[:])
}

// OneOf returns true if s is one of list
func OneOf(s string, list ...string) bool {
	for _, l := range list {
		if s == l {
			return true
		}
	}
	return false
}
