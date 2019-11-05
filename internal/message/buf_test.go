package message

import (
	"testing"
	"log"
)

func TestBufEntry_ReaderIndex(t *testing.T) {
	alloc := NewFixedBufAllocator(8)

	buf := alloc.Alloc()
	defer alloc.Dealloc(buf)

	if ri := buf.ReaderIndex(); ri != 0 {
		log.Fatalf("ReaderIndex(): %d != %d", ri, 0)
	}
}

func TestBufEntry_WriterIndex(t *testing.T) {
	alloc := NewFixedBufAllocator(8)

	buf := alloc.Alloc()
	defer alloc.Dealloc(buf)

	if wi := buf.WriterIndex(); wi != 0 {
		log.Fatalf("WriterIndex(): %d != %d", wi, 0)
	}
}

func TestBufEntry_AddReaderIndex(t *testing.T) {
}

func TestBufEntry_AddWriterIndex(t *testing.T) {
}

func TestBufEntry_SetReaderIndex(t *testing.T) {
	alloc := NewFixedBufAllocator(8)

	buf := alloc.Alloc()
	defer alloc.Dealloc(buf)

	if err := buf.SetReaderIndex(1); err == nil {
		log.Fatal("SetReaderIndex(1) failed: no error is returned")
	}

	buf.SetWriterIndex(2)
	if err := buf.SetReaderIndex(2); err != nil {
		log.Fatal("SetReaderIndex(2) failed: ", err)
	}
	if wi := buf.WriterIndex(); wi != 2 {
		log.Fatalf("SetReaderIndex(2) failed: ReaderIndex(%d) != 2", wi)
	}

	if err := buf.SetReaderIndex(0); err != nil {
		log.Fatal("SetReaderIndex(0) failed: ", err)
	}
	if err := buf.SetWriterIndex(-1); err == nil {
		log.Fatal("SetReaderIndex(-1) failed: no error is returned")
	}

}

func TestBufEntry_SetWriterIndex(t *testing.T) {
	logFailed := func(i int, err error) {
		log.Fatalf("SetWriterIndex(%d) failed: %s", i, err)
	}
	logNoError := func(i int, ri int) {
		log.Fatalf("SetWriterIndex(%d) failed: no error is returned (ReaderIndex %d)", i, ri)
	}

	alloc := NewFixedBufAllocator(8)

	buf := alloc.Alloc()
	defer alloc.Dealloc(buf)

	if err := buf.SetWriterIndex(2); err != nil {
		logFailed(2, err)
	}
	if wi := buf.WriterIndex(); wi != 2 {
		log.Fatalf("SetWriterIndex(2) failed: WriterIndex(%d) != 2", wi)
	}

	if err := buf.SetWriterIndex(8); err != nil {
		logFailed(8, err)
	}

	if err := buf.SetWriterIndex(9); err == nil {
		logNoError(9, buf.ReaderIndex())
	}

	if err := buf.SetWriterIndex(0); err != nil {
		logFailed(0, err)
	}
	if err := buf.SetWriterIndex(-1); err == nil {
		logNoError(-1, buf.ReaderIndex())
	}

	buf.SetWriterIndex(8)
	buf.SetReaderIndex(3)
	if err := buf.SetWriterIndex(3); err != nil {
		logFailed(3, err)
	}
	if err := buf.SetWriterIndex(2); err == nil {
		logNoError(2, buf.ReaderIndex())
	}
}

func TestBufEntry_Clear(t *testing.T) {
	alloc := NewFixedBufAllocator(32)
	buf := alloc.Alloc()

	buf.SetWriterIndex(16)
	buf.SetReaderIndex(10)

	buf.Clear()
	if ri, wi := buf.ReaderIndex(), buf.WriterIndex(); ri != 0 || wi != 0 {
		log.Fatalf("Clear() failed: ReaderIndex = %d, WriterIndex = %d", ri, wi)
	}
}
