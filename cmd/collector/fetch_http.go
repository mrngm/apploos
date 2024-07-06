package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type HTTPFetcher struct {
	client *http.Client
}

// redirPreventerLogger prevents more than 2 redirects
func redirPreventerLogger(req *http.Request, via []*http.Request) error {
	viaLogs := make([]slog.Attr, len(via))
	for i, theVia := range via {
		viaLogs[i] = slog.Group("via"+strconv.Itoa(i), req2slog(theVia))
	}
	slog.Error("redirected", "vias", viaLogs, req2slog(req))
	if len(via) < 2 {
		return nil
	}
	return fmt.Errorf("preventing redirect (after %d earlier request(s)) to %v", len(via), req.URL)
}

func NewHTTPFetcher(timeout time.Duration) *HTTPFetcher {
	return &HTTPFetcher{
		client: &http.Client{
			CheckRedirect: redirPreventerLogger,
			Timeout:       timeout,
		},
	}
}

// FetchHTTPSource retrieves src (protocols: http://, https://) using GET method and returns the bytes and nil error.
// Otherwise, an appropriate error is returned. It's possible to customize the request using the options.
func (hf *HTTPFetcher) Fetch(ctx context.Context, src string, options ...HTTPFetchOption) (io.Reader, error) {
	// Try to make sense of the provided src
	if !(strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://")) {
		return nil, fmt.Errorf("invalid prefix, expected http:// or https://")
	}
	if *appname == "" { // flags
		*appname = "FIXME-to-be-nice"
	}
	options = append(options, WithUserAgent(*appname))

	// Generate a request ID (UUID), add it to the request and insert it into the context
	reqId := uuid.New() // may panic
	options = append(options, WithRequestIdAndAppname(reqId, *appname))

	ctx = NewContextWithRequestId(ctx, reqId)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, src, nil)
	if err != nil {
		slog.Error("FetchHTTPSource request creation failed", "err", err, slog.Group("request", "method", http.MethodGet, "src", src)) // req2slog wouldn't work
		return nil, fmt.Errorf("creating request failed: %v", err)
	}

	for _, opt := range options {
		err = opt(req)
		if err != nil {
			slog.Error("FetchHTTPSource option setting failed", "err", err, req2slog(req))
			return nil, fmt.Errorf("setting HTTPFetchOption failed: %v", err)
		}
	}

	slog.Debug("request created", "request-id", reqId, req2slog(req))

	resp, err := hf.client.Do(req)
	if err != nil {
		slog.Error("request failed", "err", err, req2slog(req))
		return nil, err
	}
	slog.Info("received response", resp2slog(resp))

	return nil, fmt.Errorf("unimplemented")
}

// vim: cc=120:
