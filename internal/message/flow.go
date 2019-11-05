package message

import (
	"encoding/binary"
	"log"
	"reflect"
	"runtime"
)

type BufWriteFlow struct {
	buf    *BufEntry
	trace  []*TraceRecord

	err    error
}

type TraceRecord struct {
	method string
	args []interface{}
}

func NewBufWriteFlow(buf *BufEntry) *BufWriteFlow {
	return &BufWriteFlow{
		buf:    buf,
		err:    nil,
	}
}

func (e *BufEntry) Flow() *BufWriteFlow {
	return NewBufWriteFlow(e)
}

func (f *BufWriteFlow) callBuf(method interface{}, args ...interface{}) {
	if f.err != nil {
		return
	}

	valueArgs := make([]reflect.Value, len(args))
	for i, a := range args {
		valueArgs[i] = reflect.ValueOf(a)
	}

	valFunc := reflect.ValueOf(method)
	results := valFunc.Call(valueArgs)

	if result := results[0]; !result.IsNil() {
		err := result.Interface().(error)
		methodName := runtime.FuncForPC(valFunc.Pointer()).Name()

		f.err = err
		f.trace = append(f.trace, &TraceRecord{
			method: methodName,
			args:   args,
		})
	}
}

func (f *BufWriteFlow) Byte(b byte) *BufWriteFlow {
	f.callBuf(f.buf.PutByte, b)
	return f
}

func (f *BufWriteFlow) Bytes(bs []byte) *BufWriteFlow {
	f.callBuf(f.buf.PutBytes, bs)
	return f
}

func (f *BufWriteFlow) Uint8(v uint8) *BufWriteFlow {
	f.callBuf(f.buf.PutUint8, v)
	return f
}

func (f *BufWriteFlow) Int8(v int8) *BufWriteFlow {
	f.callBuf(f.buf.PutInt8, v)
	return f
}

func (f *BufWriteFlow) Uint16(v uint16, order binary.ByteOrder) *BufWriteFlow {
	f.callBuf(f.buf.PutUint16, v, order)
	return f
}

func (f *BufWriteFlow) Uint32(v uint32, order binary.ByteOrder) *BufWriteFlow {
	f.callBuf(f.buf.PutUint32, v, order)
	return f
}

func (f *BufWriteFlow) Uint64(v uint64, order binary.ByteOrder) *BufWriteFlow {
	f.callBuf(f.buf.PutUint64, v, order)
	return f
}

func (f *BufWriteFlow) Int16(v int16, order binary.ByteOrder) *BufWriteFlow {
	f.callBuf(f.buf.PutInt16, v, order)
	return f
}

func (f *BufWriteFlow) Int32(v int32, order binary.ByteOrder) *BufWriteFlow {
	f.callBuf(f.buf.PutInt32, v, order)
	return f
}

func (f *BufWriteFlow) Int64(v int64, order binary.ByteOrder) *BufWriteFlow {
	f.callBuf(f.buf.PutInt64, v, order)
	return f
}

func (f *BufWriteFlow) Error() error {
	return f.err
}

func (f *BufWriteFlow) LogTrace() {
	for _, record := range f.trace {
		log.Printf("Called %s, Args: %s", record.method, record.args)
	}
}

func (f *BufWriteFlow) LogError() {
	f.LogTrace()
	log.Printf("Raised error: %s [Size: %d, WriterIndex: %d, ReaderIndex: %d]",
		f.err, f.buf.Size, f.buf.WriterIndex(), f.buf.ReaderIndex())
}

func (f *BufWriteFlow) FatalIfError() {
	if f.err != nil {
		f.LogTrace()
		log.Fatalf("Raised error: %s [Size: %d, WriterIndex: %d, ReaderIndex: %d]",
			f.err, f.buf.Size, f.buf.WriterIndex(), f.buf.ReaderIndex())
	}
}
