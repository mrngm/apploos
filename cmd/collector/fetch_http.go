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

	"github.com/mrngm/apploos/util"
)

type HTTPFetcher struct {
	client *http.Client
}

// redirPreventerLogger prevents more than 2 redirects
func redirPreventerLogger(req *http.Request, via []*http.Request) error {
	viaLogs := make([]slog.Attr, len(via))
	for i, theVia := range via {
		viaLogs[i] = slog.Group("via"+strconv.Itoa(i), util.Req2slog(theVia))
	}
	slog.Error("redirected", "vias", viaLogs, util.Req2slog(req))
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

// FetchHTTPSource retrieves src (protocols: http://, https://) using GET method and returns an io.ReadCloer and nil
// error.  Otherwise, an appropriate error is returned. It's possible to customize parts of the request using the
// options.
//
// The caller must close the returned io.ReadCloser on nil error.
//
// If a request ID couldn't be generated (UUID), this function may panic.
func (hf *HTTPFetcher) Fetch(ctx context.Context, src string, options ...HTTPFetchOption) (io.ReadCloser, error) {
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

	ctx = util.NewContextWithRequestId(ctx, reqId)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, src, nil)
	if err != nil {
		slog.Error("FetchHTTPSource request creation failed", "err", err, slog.Group("request", "method", http.MethodGet, "src", src)) // util.Req2slog wouldn't work
		return nil, fmt.Errorf("creating request failed: %v", err)
	}

	for _, opt := range options {
		err = opt(req)
		if err != nil {
			slog.Error("FetchHTTPSource option setting failed", "err", err, util.Req2slog(req))
			return nil, fmt.Errorf("setting HTTPFetchOption failed: %v", err)
		}
	}

	slog.Debug("request created", "request-id", reqId, util.Req2slog(req))

	resp, err := hf.client.Do(req)
	if err != nil {
		slog.Error("request failed", "err", err, util.Req2slog(req))
		return nil, err
	}
	slog.Info("received response", util.Resp2slog(resp))

	return resp.Body, nil
}

// vim: cc=120:
