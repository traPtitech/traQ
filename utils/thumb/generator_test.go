package thumb

import (
	"github.com/stretchr/testify/assert"
	"image"
	"testing"
)

func TestCalcThumbnailSize(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)

	assert.EqualValues(image.Pt(100, 100), CalcThumbnailSize(image.Pt(100, 100)))
	assert.EqualValues(image.Pt(ThumbnailMaxWidth, 100), CalcThumbnailSize(image.Pt(ThumbnailMaxWidth, 100)))
	assert.EqualValues(image.Pt(ThumbnailMaxWidth, 50), CalcThumbnailSize(image.Pt(ThumbnailMaxWidth*2, 100)))
	assert.EqualValues(image.Pt(50, ThumbnailMaxHeight), CalcThumbnailSize(image.Pt(100, ThumbnailMaxHeight*2)))
	assert.EqualValues(image.Pt(ThumbnailMaxWidth, ThumbnailMaxHeight), CalcThumbnailSize(image.Pt(ThumbnailMaxWidth*2, ThumbnailMaxHeight*2)))
}
