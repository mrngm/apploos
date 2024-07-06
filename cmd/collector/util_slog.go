package main

import (
	"log/slog"
	"net/http"
)

func req2slog(req *http.Request) slog.Attr {
	return slog.Group("request", "method", req.Method, "url", req.URL)
}
