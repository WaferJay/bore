package buf

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"io"
)

type BufEntry struct {
	B           []byte
	Size        int
	alloc       *FixedBufAllocator
	writerIndex int
	readerIndex int
}

var (
	ErrOutBound = errors.New("out of bound")
	ErrExcessSize = errors.New("excess size")
)

func newBufEntry(fa *FixedBufAllocator) *BufEntry {
	return &BufEntry{
		B:     make([]byte, fa.bufSize),
		Size:  fa.bufSize,
		alloc: fa,
	}
}

func (e *BufEntry) WriterIndex() int {
	return e.writerIndex
}

func (e *BufEntry) ReaderIndex() int {
	return e.readerIndex
}

func (e *BufEntry) Clear() *BufEntry {
	e.writerIndex = 0
	e.readerIndex = 0
	return e
}

func (e *BufEntry) SetWriterIndex(i int) error {
	if i < e.readerIndex || i > e.Size {
		return ErrOutBound
	}
	e.writerIndex = i
	return nil
}

func (e *BufEntry) SetReaderIndex(i int) error {
	if i > e.writerIndex || i < 0 {
		return ErrOutBound
	}
	e.readerIndex = i
	return nil
}

func (e *BufEntry) AddReaderIndex(i int) error {
	return e.SetReaderIndex(e.readerIndex + i)
}

func (e *BufEntry) AddWriterIndex(i int) error {
	return e.SetWriterIndex(e.writerIndex + i)
}

func (e *BufEntry) Length() int {
	return e.writerIndex - e.readerIndex
}

func (e *BufEntry) PutByte(b byte) error {
	if e.writerIndex+1 > e.Size {
		return ErrExcessSize
	}
	e.B[e.writerIndex] = b
	e.writerIndex++
	return nil
}

func (e *BufEntry) PutBytes(bs []byte) error {
	bsLen := len(bs)
	start := e.writerIndex
	end := start + bsLen
	if e.writerIndex+bsLen > e.Size {
		return ErrExcessSize
	}
	copy(e.B[start:end], bs)
	e.writerIndex = end
	return nil
}

func (e *BufEntry) WriteByte(b byte) error {
	return e.PutByte(b)
}

func (e *BufEntry) PutUint8(v uint8) error {
	if e.writerIndex+1 > e.Size {
		return ErrExcessSize
	}
	e.B[e.writerIndex] = byte(v)
	e.writerIndex++
	return nil
}

func (e *BufEntry) PutInt8(v int8) error {
	if e.writerIndex+1 > e.Size {
		return ErrExcessSize
	}
	e.B[e.writerIndex] = byte(v)
	e.writerIndex++
	return nil
}

func (e *BufEntry) PutUint16(v uint16, order binary.ByteOrder) error {
	const typeLen = 2

	if e.writerIndex+typeLen > e.Size {
		return ErrExcessSize
	}

	start := e.writerIndex
	end := e.writerIndex + typeLen

	order.PutUint16(e.B[start:end], v)
	e.writerIndex = end
	return nil
}

func (e *BufEntry) PutUint32(v uint32, order binary.ByteOrder) error {
	const typeLen = 4
	if e.writerIndex+typeLen > e.Size {
		return ErrExcessSize
	}

	start := e.writerIndex
	end := e.writerIndex + typeLen

	order.PutUint32(e.B[start:end], v)
	e.writerIndex = end
	return nil
}

func (e *BufEntry) PutUint64(v uint64, order binary.ByteOrder) error {
	const typeLen = 8
	if e.writerIndex+typeLen > e.Size {
		return ErrExcessSize
	}

	start := e.writerIndex
	end := e.writerIndex + typeLen

	order.PutUint64(e.B[start:end], v)
	e.writerIndex = end
	return nil
}

func (e *BufEntry) PutInt16(v int16, order binary.ByteOrder) error {
	const typeLen = 2
	if e.writerIndex+typeLen > e.Size {
		return ErrExcessSize
	}

	start := e.writerIndex
	end := e.writerIndex + typeLen

	order.PutUint16(e.B[start:end], uint16(v))
	e.writerIndex = typeLen
	return nil
}

func (e *BufEntry) PutInt32(v int32, order binary.ByteOrder) error {
	const typeLen = 4
	if e.writerIndex+typeLen > e.Size {
		return ErrExcessSize
	}

	start := e.writerIndex
	end := e.writerIndex + typeLen

	order.PutUint32(e.B[start:end], uint32(v))
	e.writerIndex = end
	return nil
}

func (e *BufEntry) PutInt64(v int64, order binary.ByteOrder) error {
	const typeLen = 8
	if e.writerIndex+typeLen > e.Size {
		return ErrExcessSize
	}

	start := e.writerIndex
	end := e.writerIndex + typeLen

	order.PutUint64(e.B[start:end], uint64(v))
	e.writerIndex = end
	return nil
}


func (e *BufEntry) GetByte() (byte, error) {
	if e.readerIndex >= e.writerIndex {
		return 0, ErrOutBound
	}
	return e.B[e.readerIndex], nil
}

func (e *BufEntry) GetInt8() (int8, error) {
	b, err := e.GetByte()
	return int8(b), err
}

func (e *BufEntry) GetUint8() (uint8, error) {
	v, err := e.GetByte()
	return uint8(v), err
}

func (e *BufEntry) GetUint16(order binary.ByteOrder) (uint16, error) {
	if e.readerIndex+2 > e.writerIndex {
		return 0, ErrOutBound
	}
	return order.Uint16(e.B[e.readerIndex:]), nil
}

func (e *BufEntry) GetUint32(order binary.ByteOrder) (uint32, error) {
	if e.readerIndex+4 > e.writerIndex {
		return 0, ErrOutBound
	}
	return order.Uint32(e.B[e.readerIndex:]), nil
}

func (e *BufEntry) GetUint64(order binary.ByteOrder) (uint64, error) {
	if e.readerIndex+8 > e.writerIndex {
		return 0, ErrOutBound
	}
	return order.Uint64(e.B[e.readerIndex:]), nil
}

func (e *BufEntry) GetInt16(order binary.ByteOrder) (int16, error) {
	u, err := e.GetUint16(order)
	if err != nil {
		return 0, err
	}
	return int16(u), nil
}

func (e *BufEntry) GetInt32(order binary.ByteOrder) (int32, error) {
	u, err := e.GetUint32(order)
	if err != nil {
		return 0, err
	}
	return int32(u), nil
}

func (e *BufEntry) GetInt64(order binary.ByteOrder) (int64, error) {
	u, err := e.GetUint32(order)
	if err != nil {
		return 0, err
	}
	return int64(u), nil
}

func (e *BufEntry) ReadByte() (b byte, err error) {
	b, err = e.GetByte()
	if err == ErrOutBound {
		err = io.EOF
		return
	}
	e.readerIndex++
	return
}

func (e *BufEntry) ReadBytes(bs []byte) (n int) {
	l := len(bs)
	copy(bs, e.B[e.readerIndex:e.writerIndex])
	if n = e.writerIndex - e.readerIndex; n > l {
		n = l
	}
	e.readerIndex += n
	return
}

func (e *BufEntry) ReadInt8() (v int8, err error) {
	v, err = e.GetInt8()
	e.readerIndex++
	return
}

func (e *BufEntry) ReadUint8() (v uint8, err error) {
	v, err = e.GetUint8()
	e.readerIndex++
	return
}

func (e *BufEntry) ReadUint16(order binary.ByteOrder) (v uint16, err error) {
	v, err = e.GetUint16(order)
	e.readerIndex += 2
	return
}

func (e *BufEntry) ReadUint32(order binary.ByteOrder) (v uint32, err error) {
	v, err = e.GetUint32(order)
	e.readerIndex += 4
	return
}

func (e *BufEntry) ReadUint64(order binary.ByteOrder) (v uint64, err error) {
	v, err = e.GetUint64(order)
	e.readerIndex += 8
	return
}

func (e *BufEntry) ReadInt16(order binary.ByteOrder) (int16, error) {
	v, err := e.GetUint16(order)
	e.readerIndex += 2
	return int16(v), err
}

func (e *BufEntry) ReadInt32(order binary.ByteOrder) (int32, error) {
	v, err := e.GetUint32(order)
	e.readerIndex += 4
	return int32(v), err
}

func (e *BufEntry) ReadInt64(order binary.ByteOrder) (int64, error) {
	v, err := e.GetUint64(order)
	e.readerIndex += 8
	return int64(v), err
}

func (e *BufEntry) Read(size int) (*BufEntry, error) {
	if e.writerIndex + size > e.Size {
		return nil, ErrOutBound
	}

	dst := make([]byte, size)
	copy(dst, e.B[e.writerIndex:e.writerIndex+size])
	e.writerIndex += size
	return &BufEntry{
		B: dst,
		Size: size,
	}, nil
}

func (e *BufEntry) Bytes() []byte {
	view := e.B[e.readerIndex:e.writerIndex]
	dst := make([]byte, len(view))
	copy(dst, view)
	return dst
}

func (e *BufEntry) HexString() string {
	view := e.B[e.readerIndex:e.writerIndex]
	return hex.EncodeToString(view)
}
