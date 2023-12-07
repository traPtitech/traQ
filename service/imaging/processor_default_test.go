package imaging

import (
	"bytes"
	"image"
	"image/png"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

const testdataFolder = "../../testdata/images/"

func mustOpen(path string) *os.File {
	fp, err := os.Open(testdataFolder + path)
	if err != nil {
		panic(err)
	}
	return fp
}

func setup() (Processor, *os.File) {
	processor := NewProcessor(Config{
		MaxPixels:        500 * 500,
		Concurrency:      1,
		ThumbnailMaxSize: image.Point{50, 50},
	})
	return processor, mustOpen("test.png")
}

func assertImg(t *testing.T, actualImg image.Image, expectedFilePath string) {
	actualImgBytesBuffer := &bytes.Buffer{}
	err := png.Encode(actualImgBytesBuffer, actualImg)
	if err != nil {
		panic(err)
	}
	actualImgBytes := actualImgBytesBuffer.Bytes()

	fpExpected := mustOpen(expectedFilePath)
	expectedImgBytes, err := io.ReadAll(fpExpected)
	if err != nil {
		panic(err)
	}

	assert.Equal(t, expectedImgBytes, actualImgBytes)
}

func TestProcessorDefault_Thumbnail(t *testing.T) {
	t.Parallel()

	processor, fpActual := setup()
	defer fpActual.Close()
	actualImg, err := processor.Thumbnail(fpActual)
	assert.Nil(t, err)
	assertImg(t, actualImg, "test_thumbnail.png")
}

func TestProcessorDefault_Fit(t *testing.T) {
	t.Parallel()

	processor, fp := setup()
	defer fp.Close()
	actualImg, err := processor.Fit(fp, 100, 100)
	assert.Nil(t, err)
	assertImg(t, actualImg, "test_fit.png")
}
