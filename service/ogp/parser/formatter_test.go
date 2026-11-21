package ogpparser

import (
	"fmt"
	"testing"

	"github.com/dyatlov/go-opengraph/opengraph"
	"github.com/dyatlov/go-opengraph/opengraph/types/image"
	"github.com/dyatlov/go-opengraph/opengraph/types/video"
	"github.com/stretchr/testify/assert"
)

func TestMergeDefaultPageMetaAndOpenGraph(t *testing.T) {
	t.Parallel()
	t.Run("too many media", func(t *testing.T) {
		t.Parallel()
		og := opengraph.NewOpenGraph()
		for i := range maxMediaCount * 10 {
			// Width / Height を指定して実画像の取得を行わないようにする
			og.Images = append(og.Images, &image.Image{URL: fmt.Sprintf("https://example.com/%d.png", i), Width: 1, Height: 1})
			og.Videos = append(og.Videos, &video.Video{URL: fmt.Sprintf("https://example.com/%d.mp4", i), Width: 1, Height: 1})
		}
		result := MergeDefaultPageMetaAndOpenGraph(og, &DefaultPageMeta{})

		assert.Len(t, result.Images, maxMediaCount)
		assert.Len(t, result.Videos, maxMediaCount)
		assert.Equal(t, "https://example.com/0.png", result.Images[0].URL)
		assert.Equal(t, "https://example.com/0.mp4", result.Videos[0].URL)
	})
}
