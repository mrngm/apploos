package main

import (
	"context"
	"os"
	"os/signal"
)

func HandleSignals(ctx context.Context, shutdownCh chan struct{}) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh)

sigLoop:
	for sig := range sigCh {
		switch sig {
		case os.Interrupt:
			close(shutdownCh)
			break sigLoop
		default:
			// Ignore received signal
		}
	}
	signal.Stop(sigCh)
	close(sigCh)
}

// vim: cc=120:
