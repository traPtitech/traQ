package utils

import (
	"bytes"
	"io"
)

func Map[T, R any](s []T, mapper func(item T) R) []R {
	ret := make([]R, len(s))
	for i := range s {
		ret[i] = mapper(s[i])
	}
	return ret
}

func IoReaderToBytes(r io.Reader) ([]byte, error) {
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(r); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func MustIoReaderToBytes(r io.Reader) []byte {
	b, err := IoReaderToBytes(r)
	if err != nil {
		panic(err)
	}
	return b
}
