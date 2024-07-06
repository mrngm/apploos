package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

var HTTPHeaderRequestId = "x-YOURAPPNAMEHERE-request-id"

// HTTPFetchOption is a wrapper around net/http.Request. It provides a bit more easy access to various HTTP headers,
// setting basic authentication, etc. It may return an error if, for example, a validation on the correctness of the
// input failed.
type HTTPFetchOption func(*http.Request) error

func WithAcceptHeader(val string) HTTPFetchOption {
	return func(r *http.Request) error {
		if !strings.Contains(val, "/") {
			return fmt.Errorf("accept value doesn't contain /")
		}
		// TODO: if multiple types are given, validate each one
		r.Header.Set("accept", val)
		slog.Debug("WithAcceptHeader", "accept", val, slog.Group("request", "method", r.Method, "url", &r.URL))
		return nil
	}
}

func WithBasicAuth(user, pass string) HTTPFetchOption {
	return func(r *http.Request) error {
		r.SetBasicAuth(user, pass)
		slog.Debug("WithBasicAuth", "user", user, "pass", "<redacted>", slog.Group("request", "method", r.Method, "url", &r.URL))
		return nil
	}
}

func WithUserAgent(val string) HTTPFetchOption {
	return func(r *http.Request) error {
		r.Header.Set("user-agent", val)
		slog.Debug("WithUserAgent", "user-agent", val, slog.Group("request", "method", r.Method, "url", &r.URL))
		return nil
	}
}

func WithRequestIdAndAppname(id uuid.UUID, appname string) HTTPFetchOption {
	return func(r *http.Request) error {
		strVal := id.String()
		if strVal == "" {
			return fmt.Errorf("supplied UUID doesn't seem valid")
		}
		if appname == "" {
			appname = "FIXME-to-be-nice"
		}
		headerName := "x-" + appname + "-request-id"
		r.Header.Set(headerName, strVal)
		slog.Debug("WithRequestId", "request-id", strVal, "header-used", headerName, slog.Group("request", "method", r.Method, "url", &r.URL))
		return nil
	}
}

// vim: cc=120:
