package output

import (
	"bufio"
	"io"
)

// countingWriter wraps an io.Writer and tracks how many bytes have been written.
// It is the core primitive that lets the serializer emit xref byte-offsets
// without keeping the entire PDF in memory.
type countingWriter struct {
	w   *bufio.Writer
	pos int64
	err error
}

// newCountingWriter creates a countingWriter wrapping w.
// A 256 KiB write-buffer is used to amortise syscall overhead.
func newCountingWriter(w io.Writer) *countingWriter {
	return &countingWriter{w: bufio.NewWriterSize(w, 256*1024)}
}

// WriteString writes s and updates the byte counter.
// Errors are sticky: once an error occurs all subsequent writes are no-ops.
func (cw *countingWriter) WriteString(s string) {
	if cw.err != nil {
		return
	}
	n, err := cw.w.WriteString(s)
	cw.pos += int64(n)
	if err != nil {
		cw.err = err
	}
}

// Write writes p and updates the byte counter.
func (cw *countingWriter) Write(p []byte) {
	if cw.err != nil {
		return
	}
	n, err := cw.w.Write(p)
	cw.pos += int64(n)
	if err != nil {
		cw.err = err
	}
}

// Pos returns the number of bytes written so far (post-flush).
func (cw *countingWriter) Pos() int64 { return cw.pos }

// Flush flushes the internal buffer to the underlying writer.
func (cw *countingWriter) Flush() error {
	if cw.err != nil {
		return cw.err
	}
	cw.err = cw.w.Flush()
	return cw.err
}

// Err returns the first error that occurred, or nil.
func (cw *countingWriter) Err() error { return cw.err }
