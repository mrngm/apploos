package main

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
)

func SaveToDisk(ctx context.Context, name string, data []byte) (int, error) {
	fp := filepath.Join(*saveDir, name)
	slog.Debug("SaveToDisk", "fn", name, "dataLen", len(data), "dir", *saveDir, "fullPath", fp)

	return 0, fmt.Errorf("unimplemented")
}
