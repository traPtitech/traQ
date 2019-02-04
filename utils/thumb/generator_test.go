package thumb

import (
	"bytes"
	"context"
	"github.com/jakobvarmose/go-qidenticon"
	"github.com/stretchr/testify/assert"
	"image"
	"testing"
)

var img = qidenticon.Render(qidenticon.Code("test"), 500, qidenticon.DefaultSettings())

func TestCalcThumbnailSize(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)
	maxSize := image.Pt(ThumbnailMaxWidth, ThumbnailMaxHeight)

	assert.EqualValues(image.Pt(100, 100), CalcThumbnailSize(image.Pt(100, 100), maxSize))
	assert.EqualValues(image.Pt(ThumbnailMaxWidth, 100), CalcThumbnailSize(image.Pt(ThumbnailMaxWidth, 100), maxSize))
	assert.EqualValues(image.Pt(ThumbnailMaxWidth, 50), CalcThumbnailSize(image.Pt(ThumbnailMaxWidth*2, 100), maxSize))
	assert.EqualValues(image.Pt(50, ThumbnailMaxHeight), CalcThumbnailSize(image.Pt(100, ThumbnailMaxHeight*2), maxSize))
	assert.EqualValues(image.Pt(ThumbnailMaxWidth, ThumbnailMaxHeight), CalcThumbnailSize(image.Pt(ThumbnailMaxWidth*2, ThumbnailMaxHeight*2), maxSize))
}

func TestResize(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)

	resized := Resize(img, 200, 300)
	assert.EqualValues(CalcThumbnailSize(img.Bounds().Size(), image.Pt(200, 300)), resized.Bounds().Size())
}

func TestEncodeToPNG(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)

	png, err := EncodeToPNG(img)
	if assert.NoError(err) {
		c, f, err := image.DecodeConfig(png)
		if assert.NoError(err) {
			assert.Equal(img.Bounds().Size().X, c.Width)
			assert.Equal(img.Bounds().Size().Y, c.Height)
			assert.Equal("png", f)
		}
	}
}

func TestGenerate(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)

	png, _ := EncodeToPNG(img)
	b := png.Bytes()

	t.Run("Success1", func(t *testing.T) {
		t.Parallel()

		thumb, err := Generate(context.Background(), bytes.NewReader(b), "image/png")
		if assert.NoError(err) {
			assert.NotNil(thumb)
		}
	})

	t.Run("Success2", func(t *testing.T) {
		t.Parallel()

		_, err := Generate(context.Background(), nil, "application/json")
		assert.EqualError(err, ErrFileThumbUnsupported.Error())
	})

	t.Run("Failure1", func(t *testing.T) {
		t.Parallel()

		_, err := Generate(context.Background(), bytes.NewReader(b), "image/bmp")
		assert.Error(err)
	})
}
