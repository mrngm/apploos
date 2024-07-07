package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

// SaveToDisk writes the data to a temporary file in *saveDir and syncs to disk for crash safety. After that, it tries
// to move the file into place with the supplied name and syncs the *saveDir directory such that the metadata is
// persisted as well.
//
// The return semantics are equivalent to io.Writer. It returns the number of bytes written (0 <= n <= len(data)) to the
// temporary file, and nil error if it succeeded. It may return 0 <= n <= len(data) and an error, indicating only part
// (or none) of the data was written, and an error message saying what failed exactly.
func SaveToDisk(ctx context.Context, name string, data []byte) (int, error) {
	fp := filepath.Join(*saveDir, name)
	slog.Debug("SaveToDisk", "fn", name, "dataLen", len(data), "dir", *saveDir, "fullPath", fp)

	// Already open a file descriptor to the directory such that we can sync metadata as well.
	dirFn, err := os.Open(*saveDir)
	if err != nil {
		slog.Error("SaveToDisk(dirFn.Open) failed", "err", err)
		return 0, err
	}
	defer func() {
		if err := dirFn.Close(); err != nil {
			slog.Error("(deferred) SaveToDisk(dirFn.Close) failed", "err", err, "dir", *saveDir)
		}
	}()

	patternTmp := "tmp-" + name + "-"
	fnTmp, err := os.CreateTemp(*saveDir, patternTmp)
	if err != nil {
		slog.Error("SaveToDisk(CreateTemp)", "dir", *saveDir, "pattern", patternTmp, "err", err)
		return 0, err
	}
	shouldCloseTmpFile := true
	defer func() {
		if shouldCloseTmpFile {
			if err := fnTmp.Close(); err != nil {
				slog.Error("(deferred) SaveToDisk(fnTmp.Close) failed", "err", err, "dir", *saveDir, "fnTmp", fnTmp.Name())
			}
		}
	}()

	n, err := fnTmp.Write(data)
	if err != nil {
		slog.Error("SaveToDisk(fnTmp.Write) failed", "err", err, "bytes_written", n, "dir", *saveDir, "fnTmp", fnTmp.Name())
		return n, err
	}
	// Sync tmpfile
	if err := fnTmp.Sync(); err != nil {
		slog.Error("SaveToDisk(fnTmp.Sync) failed", "err", err, "bytes_written", n, "dir", *saveDir, "fnTmp", fnTmp.Name())
		return n, err
	}

	// Sync directory of tmpfile
	if err := dirFn.Sync(); err != nil {
		slog.Error("SaveToDisk(fnTmp/dirFn.Sync) failed", "err", err, "bytes_written", n, "dir", *saveDir)
		return n, err
	}

	if err := fnTmp.Close(); err != nil {
		slog.Error("SaveToDisk(fnTmp.Close) failed", "err", err, "bytes_written", n, "dir", *saveDir, "fnTmp", fnTmp.Name())
		shouldCloseTmpFile = false
		return n, err
	}
	shouldCloseTmpFile = false

	// Check that the destination file doesn't exist. We do this after writing to the temporary file (and syncing) such
	// that the data itself is saved to disk, regardless of the possible existence of the destination. If we would first
	// check existence, data may be lost if the caller doesn't handle such cases.
	tryDest, err := os.OpenFile(fp, os.O_RDONLY, 0000) // perm is irrelevant
	if err == nil {
		closeErr := tryDest.Close()
		slog.Error("SaveToDisk failed, destination file already exists, leaving tmpfile", "fn", fp, "closeErr", closeErr, "dir", *saveDir, "tmpFn", fnTmp.Name(), "bytes_written", n)
		return n, fmt.Errorf("destination file %q already exists, preventing write, leaving tmpFile %q in *saveDir %q, closeErr: %v", fp, fnTmp.Name(), *saveDir, closeErr)
	}
	// Proposed destination doesn't exist after we just synced the containing directory. Rename is likely to succeed.

	if err := os.Rename(fnTmp.Name(), fp); err != nil {
		slog.Error("SaveToDisk(rename) failed", "err", err, "bytes_written", n, "dir", *saveDir, "oldpath", fnTmp.Name(), "newpath", fp)
		return n, fmt.Errorf("could not rename %q to %q: %v", fnTmp.Name(), fp)
	}

	// Sync metadata
	if err := dirFn.Sync(); err != nil {
		slog.Error("SaveToDisk(dirFn.Sync) failed", "err", err, "bytes_written", n, "dir", *saveDir)
		return n, err
	}

	return n, nil
}

// vim: cc=120:
