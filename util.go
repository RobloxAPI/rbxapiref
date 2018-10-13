package main

import (
	"io"
)

////////////////////////////////////////////////////////////////

// DecodeWrapper wraps a Reader so that the underlying error and byte count
// can be acquired after decoding.
type DecodeWrapper struct {
	r   io.Reader
	n   int64
	err error
}

func NewDecodeWrapper(r io.Reader) *DecodeWrapper {
	return &DecodeWrapper{r: r}
}

func (w *DecodeWrapper) Read(p []byte) (n int, err error) {
	n, err = w.r.Read(p)
	w.n += int64(n)
	w.err = err
	return n, err
}

func (w *DecodeWrapper) Result() (n int64, err error) {
	return w.n, w.err
}

func (w *DecodeWrapper) BytesRead() int64 {
	return w.n
}

func (w *DecodeWrapper) Error() error {
	return w.err
}

////////////////////////////////////////////////////////////////

// EncodeWrapper wraps a Writer so that the underlying error and byte count
// can be acquired after encoding.
type EncodeWrapper struct {
	w   io.Writer
	n   int64
	err error
}

func NewEncodeWrapper(w io.Writer) *EncodeWrapper {
	return &EncodeWrapper{w: w}
}

func (w *EncodeWrapper) Write(p []byte) (n int, err error) {
	n, err = w.w.Write(p)
	w.n += int64(n)
	w.err = err
	return n, err
}

func (w *EncodeWrapper) Result() (n int64, err error) {
	return w.n, w.err
}

func (w *EncodeWrapper) BytesRead() int64 {
	return w.n
}

func (w *EncodeWrapper) Error() error {
	return w.err
}

////////////////////////////////////////////////////////////////
