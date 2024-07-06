// package main in cmd/collector fetches information from external sources and saves the ingested data. It may
// format/lint the data such that it can efficiently tell through a checksum if the data was changed compared to an
// earlier fetch.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"
)

var (
	once            = flag.Bool("once", false, "If given, fetch source once, write to storage, and exit. Otherwise, keep running and fetch every interval.")
	refreshInterval = flag.Duration("interval", time.Duration(5*time.Minute), "Refresh source every duration with jitter. Ignored when -once is given")
	refreshJitter   = flag.Duration("jitter", time.Duration(23*time.Second), "Apply jitter up to (-)duration on refresh interval, e.g. 5m (interval) +/- 23s (jitter)")
	source          = flag.String("source", "", fmt.Sprintf("Fetch this source, prefixed with protocol://. Supported: %+q", SupportedProtocols))
	saveDir         = flag.String("storage", "", "Store results in this directory. If not supplied, a temporary directory will be created. If the supplied directory doesn't exist, it's created given enough permissions. Existing files in the supplied directory are never overwritten.")
	appname         = flag.String("appname", "", "Set the application name (used in e.g. user-agent and request-id)")
)

var (
	logLevel = new(slog.LevelVar)
)

func main() {
	flag.Parse()
	if flag.NArg() == 0 && flag.NFlag() == 0 {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
		return
	}
	logLevel.Set(slog.LevelDebug)

	if *appname == "" {
		*appname = "FIXME-to-be-nice"
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel}))
	slog.SetDefault(logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx.Done()

	srcContents, err := FetchSource(ctx, *source)
	if err != nil {
		logger.Error("FetchSource failed", "err", err)
		return
	}

	logger.Debug("FetchSource contents", "length", len(srcContents))

}

// vim: cc=120:
