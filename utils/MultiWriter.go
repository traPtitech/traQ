package utils

import "io"

type multiWriter struct {
	writers []io.Writer
}

// Write
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

// MultiWriter creates a writer that duplicates its writes to all the
// provided writers, similar to the Unix tee(1) command.
// io.MultiWriterとの違いはwritersがio.ErrClosedPipeを返す場合は処理を続行するという点
func MultiWriter(writers ...io.Writer) io.Writer {
	w := make([]io.Writer, len(writers))
	copy(w, writers)
	return &multiWriter{w}
}
