package buf

import (
	"io"
)

func ReadAtLeast(reader io.Reader, buf *BufEntry, least int) (int, error) {
	bs := buf.B[buf.WriterIndex():]
	n, err := io.ReadAtLeast(reader, bs, least)
	if err != nil {
		return n, err
	}

	err = buf.AddWriterIndex(n)
	if err != nil {
		panic("unreachable")
	}
	return n, err
}

func ReadFull(reader io.Reader, buf *BufEntry) (int, error) {
	bs := buf.B[buf.WriterIndex():]
	n, err := io.ReadFull(reader, bs)

	if err != nil {
		err = buf.AddWriterIndex(n)
		if err != nil {
			panic("unreachable")
		}
	}

	return n, err
}

func WriteBuf(writer io.Writer, buf *BufEntry) (int, error) {
	bs := buf.B[buf.ReaderIndex():buf.WriterIndex()]
	n, err := writer.Write(bs)

	if err == nil {
		err = buf.AddReaderIndex(n)
		if err != nil {
			panic("unreachable")
		}
	}
	if buf.SetReaderIndex(buf.WriterIndex()) != nil {
		panic("unreachable")
	}

	return n, err
}
