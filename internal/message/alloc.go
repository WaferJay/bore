package message

import "sync"

type FixedBufAllocator struct {
	bufSize  int
	bufPool  *sync.Pool
}

type BufAllocator struct {
	fixedAllocators *sync.Map
}

func NewFixedBufAllocator(bufSize int) *FixedBufAllocator {
	fa := &FixedBufAllocator{
		bufSize:  bufSize,
		bufPool: &sync.Pool{},
	}
	fa.bufPool.New = fa.newBuf

	return fa
}

func (fa *FixedBufAllocator) newBuf() interface{} {
	return newBufEntry(fa)
}

func (fa *FixedBufAllocator) Alloc() *BufEntry {
	return fa.bufPool.Get().(*BufEntry)
}

func (fa *FixedBufAllocator) Dealloc(entry *BufEntry) {
	if entry.Size != fa.bufSize {
		panic("invalid buffer")
	}

	entry.Clear()
	fa.bufPool.Put(entry)
}

func NewBufAllocator() *BufAllocator {
	return &BufAllocator{
		fixedAllocators: &sync.Map{},
	}
}

func (ba *BufAllocator) getFixedAllocator(bufSize int) *FixedBufAllocator {
	value, _ := ba.fixedAllocators.LoadOrStore(
		bufSize, NewFixedBufAllocator(bufSize))

	return value.(*FixedBufAllocator)
}

func (ba *BufAllocator) Alloc(bufSize int) *BufEntry {
	return ba.getFixedAllocator(bufSize).Alloc()
}

func (ba *BufAllocator) Dealloc(entry *BufEntry) {
	if alloc, ok := ba.fixedAllocators.Load(entry.Size); !ok || alloc != entry.alloc {
		panic("invalid buffer")
	}
	entry.alloc.Dealloc(entry)
}
