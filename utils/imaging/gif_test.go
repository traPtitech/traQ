package imaging

import (
	"image/gif"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/traPtitech/traQ/testutils"
)

func TestGifToBytesReader(t *testing.T) {
	t.Parallel()

	tests := []string{"miku.gif", "new_year.gif", "tooth.gif"}

	for _, tt := range tests {
		tt := tt

		t.Run(tt, func(t *testing.T) {
			t.Parallel()

			f, err := gif.DecodeAll(testutils.MustOpenGif(tt))
			assert.Nil(t, err)

			_, err = GifToBytesReader(f)
			assert.Nil(t, err)
		})
	}
}
