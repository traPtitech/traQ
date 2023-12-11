package testutils

import (
	"io/fs"

	"github.com/traPtitech/traQ/testdata/gif"
)

func MustOpenGif(name string) fs.File {
	f, err := gif.FS.Open(name)
	if err != nil {
		panic(err)
	}
	return f
}
