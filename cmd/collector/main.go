// package main in cmd/collector fetches information from external sources and saves the ingested data. It may
// format/lint the data such that it can efficiently tell through a checksum if the data was changed compared to an
// earlier fetch.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/fs"
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
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel}))
	slog.SetDefault(logger)

	if *appname == "" {
		*appname = "FIXME-to-be-nice"
	}

	start := time.Now()
	if *saveDir == "" {
		// Create temporary directory in os.TempDir()
		tmpDir, err := os.MkdirTemp("", "collector-"+start.Format("20060102"))
		if err != nil && errors.Is(err, fs.ErrPermission) {
			logger.Error("cannot create tmpDir due to permissions", "err", err)
			os.Exit(1)
		} else if err != nil {
			logger.Error("error creating tmpDir", "err", err)
			os.Exit(1)
		}
		// TODO: we'll leave a tmp dir, perhaps add a flag that automatically removes it?
		logger.Info("created a temporary directory, it won't be removed", "dir", tmpDir)
	} else {
		// Try creating a file in the supplied directory and write something. If that fails, exit
		tmpFile, err := os.CreateTemp(*saveDir, "collector-"+start.Format("20060102"))
		if err != nil && errors.Is(err, fs.ErrPermission) {
			logger.Error("cannot create tmpFile due to permissions", "err", err, "dir", *saveDir)
			os.Exit(1)
		} else if err != nil {
			logger.Error("error creating tmpFile", "err", err, "dir", *saveDir)
			os.Exit(1)
		}

		defer func() {
			if err := os.Remove(tmpFile.Name()); err != nil {
				logger.Error("(deferred) removing tmpFile failed", "err", err, "fn", tmpFile, "dir", *saveDir)
			}
		}()

		shouldExit := false
		if _, err := tmpFile.Write([]byte("collector-write")); err != nil {
			logger.Error("could not write to tmpFile", "err", err, "fn", tmpFile)
			shouldExit = true
		}
		if err := tmpFile.Close(); err != nil {
			logger.Error("could not close tmpFile", "err", err, "fn", tmpFile)
		}
		if shouldExit {
			os.Exit(1)
		}
	}

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
