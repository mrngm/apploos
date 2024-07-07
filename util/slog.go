package util

import (
	"log/slog"
	"net/http"
)

func Req2slog(req *http.Request) slog.Attr {
	return slog.Group("request", "method", req.Method, "url", req.URL)
}

func Resp2slog(resp *http.Response) slog.Attr {
	return slog.Group("response", "status_code", resp.StatusCode, "content_length", resp.ContentLength, Req2slog(resp.Request), "headers", resp.Header)
}
