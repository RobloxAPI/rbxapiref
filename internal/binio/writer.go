package binio

import (
	"encoding/binary"
	"io"
)

// Writer is a wrapper that keeps track of the number of bytes written.
type Writer struct {
	w   io.Writer
	buf []byte
	n   int64
	Err error
}

func NewWriter(w io.Writer) *Writer {
	return &Writer{w: w, buf: make([]byte, 8)}
}

// Write implements the io.Writer interface.
func (w *Writer) Write(p []byte) (n int, err error) {
	n, err = w.w.Write(p)
	w.n += int64(n)
	w.Err = err
	return n, err
}

// BytesWritten returns the number of bytes written.
func (w *Writer) BytesWritten() int64 {
	return w.n
}

// End returns the number of written bytes and any error that occurred.
func (w *Writer) End() (n int64, err error) {
	return w.n, w.Err
}

// Bytes writes p as bytes.
func (w *Writer) Bytes(p []byte) (ok bool) {
	if w.Err != nil {
		return false
	}
	var n int
	n, w.Err = w.w.Write(p)
	w.n += int64(n)
	if n < len(p) {
		return false
	}
	return true
}

// Number writes data as a binary integer.
func (w *Writer) Number(data interface{}) (ok bool) {
	if w.Err != nil {
		return false
	}
	var b []byte
	switch data.(type) {
	case int8, uint8:
		b = w.buf[:1]
	case int16, uint16:
		b = w.buf[:2]
	case int32, uint32:
		b = w.buf[:4]
	case int64, uint64:
		b = w.buf[:8]
	default:
		goto invalid
	}
	switch data := data.(type) {
	case int8:
		b[0] = uint8(data)
	case uint8:
		b[0] = data
	case int16:
		binary.LittleEndian.PutUint16(b, uint16(data))
	case uint16:
		binary.LittleEndian.PutUint16(b, data)
	case int32:
		binary.LittleEndian.PutUint32(b, uint32(data))
	case uint32:
		binary.LittleEndian.PutUint32(b, data)
	case int64:
		binary.LittleEndian.PutUint64(b, uint64(data))
	case uint64:
		binary.LittleEndian.PutUint64(b, data)
	default:
		goto invalid
	}
	return w.Bytes(b)
invalid:
	panic("invalid type")
}

// String writes data as a short string. The first written byte is the length,
// followed by the byte content of the string.
func (w *Writer) String(data string) (ok bool) {
	if w.Err != nil {
		return false
	}
	if len(data) >= 1<<8 {
		panic("string too large")
	}
	if !w.Number(uint8(len(data))) {
		return false
	}
	return w.Bytes([]byte(data))
}
