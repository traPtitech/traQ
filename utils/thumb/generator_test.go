package thumb

import (
	"bytes"
	"context"
	"github.com/jakobvarmose/go-qidenticon"
	"github.com/stretchr/testify/assert"
	"golang.org/x/image/bmp"
	"golang.org/x/image/tiff"
	"image"
	"image/gif"
	"image/jpeg"
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

func TestEncodeToJPG(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)

	jpeg, err := EncodeToJPG(img)
	if assert.NoError(err) {
		c, f, err := image.DecodeConfig(jpeg)
		if assert.NoError(err) {
			assert.Equal(img.Bounds().Size().X, c.Width)
			assert.Equal(img.Bounds().Size().Y, c.Height)
			assert.Equal("jpeg", f)
		}
	}
}

func TestGenerate(t *testing.T) {
	t.Parallel()

	png, _ := EncodeToPNG(img)
	b := png.Bytes()

	t.Run("Success1", func(t *testing.T) {
		t.Parallel()

		thumb, err := Generate(context.Background(), bytes.NewReader(b), "image/png")
		if assert.NoError(t, err) {
			assert.NotNil(t, thumb)
		}
	})

	t.Run("Success2", func(t *testing.T) {
		t.Parallel()

		_, err := Generate(context.Background(), nil, "application/json")
		assert.EqualError(t, err, ErrFileThumbUnsupported.Error())
	})

	t.Run("Success3", func(t *testing.T) {
		t.Parallel()

		b := &bytes.Buffer{}
		_ = jpeg.Encode(b, img, nil)

		thumb, err := Generate(context.Background(), b, "image/jpeg")
		if assert.NoError(t, err) {
			assert.NotNil(t, thumb)
		}
	})

	t.Run("Success4", func(t *testing.T) {
		t.Parallel()

		b := &bytes.Buffer{}
		_ = bmp.Encode(b, img)

		thumb, err := Generate(context.Background(), b, "image/bmp")
		if assert.NoError(t, err) {
			assert.NotNil(t, thumb)
		}
	})

	t.Run("Success5", func(t *testing.T) {
		t.Parallel()

		b := &bytes.Buffer{}
		_ = tiff.Encode(b, img, nil)

		thumb, err := Generate(context.Background(), b, "image/tiff")
		if assert.NoError(t, err) {
			assert.NotNil(t, thumb)
		}
	})

	t.Run("Success6", func(t *testing.T) {
		t.Parallel()

		b := &bytes.Buffer{}
		_ = gif.Encode(b, img, nil)

		thumb, err := Generate(context.Background(), b, "image/gif")
		if assert.NoError(t, err) {
			assert.NotNil(t, thumb)
		}
	})

	t.Run("Failure1", func(t *testing.T) {
		t.Parallel()

		_, err := Generate(context.Background(), bytes.NewReader(b), "image/bmp")
		assert.Error(t, err)
	})
}
