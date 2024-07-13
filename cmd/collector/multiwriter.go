package main

import (
	"bytes"
	"crypto/sha256"
)

// ChecksumWriter wraps an io.Writer around an internal buffer. Upon calling Sum256, the crypto/sha256 checksum is
// returned and the buffer is reset. This is unsafe for access by multiple goroutines.
type ChecksumWriter struct {
	buf *bytes.Buffer
}

func NewChecksumWriter() *ChecksumWriter {
	return &ChecksumWriter{
		buf: new(bytes.Buffer),
	}
}

func (csw *ChecksumWriter) Write(p []byte) (n int, err error) {
	return csw.buf.Write(p)
}

// Sum256 returns the crypto/sha256 checksum of the then collected bytes in the internal buffer. It resets the buffer
// after calculating the checksum, such that it is ready for another checksum.
func (csw *ChecksumWriter) Sum256() [sha256.Size]byte {
	ret := sha256.Sum256(csw.buf.Bytes())
	csw.buf.Reset()
	return ret
}

// Reset the internal buffer.
func (csw *ChecksumWriter) Reset() {
	csw.buf.Reset()
}

// vim: cc=120:
