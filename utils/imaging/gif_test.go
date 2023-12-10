package imaging

import (
	"image/gif"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/traPtitech/traQ/testutils"
)

func TestGifToBytesReader(t *testing.T) {
	t.Parallel()

	tests := []string{"cube.gif", "miku.gif", "parapara.gif", "miku2.gif", "rabbit.gif"}

	for _, tt := range tests {
		t.Run(tt, func(t *testing.T) {
			f, err := gif.DecodeAll(testutils.MustOpenGif(tt))
			assert.Nil(t, err)

			_, err = GifToBytesReader(f)
			assert.Nil(t, err)
		})
	}
}
