package buf

import (
	"encoding/binary"
	"log"
	"testing"
)

func TestNewFixedBufAllocator(t *testing.T) {
	const bufSize = 3
	alloc := NewFixedBufAllocator(bufSize)
	if alloc.bufSize != bufSize {
		log.Fatalf("bufSize: %d != %d", alloc.bufSize, bufSize)
	}
}

func TestFixedBufAllocator_Alloc(t *testing.T) {
	const bufSize = 8
	alloc := NewFixedBufAllocator(bufSize)
	buf := alloc.Alloc()
	if buf.Size != bufSize {
		log.Fatalf("valBuf.Size: %d != %d", buf.Size, bufSize)
	}
	if bufLen := len(buf.B); bufLen != buf.Size {
		log.Fatalf("len(valBuf.B): %d != %d", bufLen, buf.Size)
	}
	if buf.alloc != alloc {
		log.Fatalf("Alloc failed. valBuf.alloc: [%+v/%p] != [%+v/%p]",
			buf.alloc, buf.alloc, alloc, alloc)
	}
}

func TestFixedBufAllocator_Dealloc(t *testing.T) {
	const bufSize = 8
	alloc := NewFixedBufAllocator(bufSize)
	oldBuf := alloc.Alloc()
	alloc.Dealloc(oldBuf)

	newBuf := alloc.Alloc()
	if newBuf != oldBuf {
		log.Fatalf("Dealloc incorrectly")
	}

	newBuf.PutUint32(0xaabbccdd, binary.LittleEndian)
	alloc.Dealloc(newBuf)
	if wi := newBuf.writerIndex; wi != 0 {
		log.Fatalf("Dealloc incorrectly: writerIndex(%d) not reset", wi)
	}
	if ri := newBuf.readerIndex; ri != 0 {
		log.Fatalf("Dealloc ncorrectly: readerIndex(%d) not reset", ri)
	}
}

func TestNewBufAllocator(t *testing.T) {
	alloc := NewBufAllocator()
	if alloc.fixedAllocators == nil {
		log.Fatal("initialize BufAllocator failed")
	}
}

func TestBufAllocator_Alloc(t *testing.T) {
	testBufSize := []int {1, 2, 3, 512, 1024}

	alloc := NewBufAllocator()

	for _, v := range testBufSize {
		buf := alloc.Alloc(v)

		if buf.Size != v {
			log.Fatalf("Alloc(%d) failed. valBuf.Size: %d != %d", v, buf.Size, v)
		}
	}

	buf := alloc.Alloc(1)
	oneLenAlloc := alloc.getFixedAllocator(1)
	if buf.alloc != oneLenAlloc {
		log.Fatalf("Alloc failed. valBuf.alloc: [%+v/%p] != [%+v/%p]",
			buf.alloc, buf.alloc, oneLenAlloc, oneLenAlloc)
	}
}

func TestBufAllocator_Dealloc(t *testing.T) {
	testBufSize := []int {1, 2, 3, 512, 1024}

	alloc := NewBufAllocator()

	for _, v := range testBufSize {
		buf := alloc.Alloc(v)
		alloc.Dealloc(buf)
		newBuf := alloc.Alloc(v)

		if buf != newBuf {
			log.Fatalf("Dealloc incorrectly: [%+v/%p] != [%+v/%p]",
				newBuf, newBuf, buf, buf)
		}
	}
}
