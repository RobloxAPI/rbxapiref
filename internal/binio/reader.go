package binio

import (
	"encoding/binary"
	"io"
)

// Reader is a wrapper that keeps track of the number of bytes written.
type Reader struct {
	r   io.Reader
	buf []byte
	n   int64
	Err error
}

func NewReader(r io.Reader) *Reader {
	return &Reader{r: r, buf: make([]byte, 256)}
}

// Read implements the io.Reader interface.
func (r *Reader) Read(p []byte) (n int, err error) {
	n, err = r.r.Read(p)
	r.n += int64(n)
	r.Err = err
	return n, err
}

// BytesRead returns the number of bytes read.
func (r *Reader) BytesRead() int64 {
	return r.n
}

// End returns the number of read bytes and any error that occurred.
func (r *Reader) End() (n int64, err error) {
	return r.n, r.Err
}

// Bytes reads len(p) bytes into p.
func (r *Reader) Bytes(p []byte) (ok bool) {
	if r.Err != nil {
		return false
	}
	var n int
	n, r.Err = io.ReadFull(r.r, p)
	r.n += int64(n)
	if r.Err != nil {
		return false
	}
	return true
}

// Number reads a binary integer. data must be a pointer to a number type.
func (r *Reader) Number(data interface{}) (ok bool) {
	if r.Err != nil {
		return false
	}
	var b []byte
	switch data.(type) {
	case *int8, *uint8:
		b = r.buf[:1]
	case *int16, *uint16:
		b = r.buf[:2]
	case *int32, *uint32:
		b = r.buf[:4]
	case *int64, *uint64:
		b = r.buf[:8]
	default:
		goto invalid
	}
	if !r.Bytes(b) {
		return false
	}
	switch data := data.(type) {
	case *int8:
		*data = int8(b[0])
	case *uint8:
		*data = b[0]
	case *int16:
		*data = int16(binary.LittleEndian.Uint16(b))
	case *uint16:
		*data = binary.LittleEndian.Uint16(b)
	case *int32:
		*data = int32(binary.LittleEndian.Uint32(b))
	case *uint32:
		*data = binary.LittleEndian.Uint32(b)
	case *int64:
		*data = int64(binary.LittleEndian.Uint64(b))
	case *uint64:
		*data = binary.LittleEndian.Uint64(b)
	default:
		goto invalid
	}
	return true
invalid:
	panic("invalid type")
}

// String reads a short string into data. The first byte is read, indicating the
// length of the string, and then a number of bytes is read equal to the length.
func (r *Reader) String(data *string) (ok bool) {
	if r.Err != nil {
		return false
	}
	var length uint8
	if !r.Number(&length) {
		return false
	}
	s := r.buf[:int(length)]
	if !r.Bytes(s) {
		return false
	}
	*data = string(s)
	return true
}
