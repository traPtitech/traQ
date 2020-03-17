package ioext

import "io"

// ReadSeekCloser io.Reader, io.Closer, io.Seekerの複合インターフェイス
type ReadSeekCloser interface {
	io.Reader
	io.Closer
	io.Seeker
}
