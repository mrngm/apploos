package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"
)

var (
	SupportedProtocols = []string{
		"http://", "https://", "file://",
	}
)

// IsSupportedSource returns the protocol and nil error if the given src is supported, or an appropriate message in
// error otherwise.
func IsSupportedSource(src string) (string, error) {
	if !strings.Contains(src, "://") {
		return "", fmt.Errorf("no protocol found, missing :// in %q", src)
	}

	for _, proto := range SupportedProtocols {
		if strings.HasPrefix(src, proto) {
			return proto, nil
		}
	}

	return "", fmt.Errorf("unsupported source protocol for value %q", src)
}

// FetchSource tries to fetch the supplied src. It returns an io.ReadCloser and nil error on success, or an appropriate
// error message otherwise.
func FetchSource(ctx context.Context, src string) (io.ReadCloser, error) {
	defer func() {
		if p := recover(); p != nil {
			slog.Error("FetchSource panicked", "src", src, "panic", p)
		}
	}()

	protocol, err := IsSupportedSource(src)
	if err != nil {
		return nil, err
	}

	switch protocol {
	case "http://", "https://":
		fetcher := NewHTTPFetcher(4 * time.Minute)
		return fetcher.Fetch(ctx, src)
	case "file://":
		return nil, fmt.Errorf("unimplemented")
	}

	return nil, fmt.Errorf("protocol %q seemed supported, but implementation is missing", protocol)
}

// vim: cc=120:
