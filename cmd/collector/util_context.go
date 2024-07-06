package main

import (
	"context"

	"github.com/google/uuid"
)

type requestIdCtxKey string

const ctxRequestId = "requestId"

func NewContextWithRequestId(ctx context.Context, requestId uuid.UUID) context.Context {
	return ctx.WithValue(ctxRequestId, requestId)
}

func RequestIdFromContext(ctx context.Context) (uuid.UUID, bool) {
	ret, ok := ctx.Value(ctxRequestId).(uuid.UUID)
	return ret, ok
}
