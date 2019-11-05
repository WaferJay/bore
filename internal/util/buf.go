package util

import (
	"io"
	"../message"
)

func ReadAtLeast(reader io.Reader, buf *message.BufEntry, least int) (int, error) {
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

func ReadFull(reader io.Reader, buf *message.BufEntry) (int, error) {
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

func WriteBuf(writer io.Writer, buf *message.BufEntry) (int, error) {
	bs := buf.B[buf.ReaderIndex():buf.WriterIndex()]
	n, err := writer.Write(bs)

	if err == nil {
		err = buf.AddReaderIndex(n)
		if err != nil {
			panic("unreachable")
		}
	}
	buf.SetReaderIndex(buf.WriterIndex())

	return n, err
}
