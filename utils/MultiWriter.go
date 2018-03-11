package utils

import "io"

type multiWriter struct {
	writers []io.Writer
}

func (t *multiWriter) Write(p []byte) (n int, err error) {
	for _, w := range t.writers {
		n, err = w.Write(p)
		if err != nil {
			if err == io.ErrClosedPipe {
				continue
			}
			return
		}
		if n != len(p) {
			err = io.ErrShortWrite
			return
		}
	}
	return len(p), nil
}

func MultiWriter(writers ...io.Writer) io.Writer {
	w := make([]io.Writer, len(writers))
	copy(w, writers)
	return &multiWriter{w}
}
