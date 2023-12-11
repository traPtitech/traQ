package imaging

import (
	"bytes"
	"image"
	"image/png"
	"io"
	"os"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/traPtitech/traQ/testutils"
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

func TestProcessorDefault_FitAnimationGIF(t *testing.T) {
	t.Parallel()

	test := []struct {
		name   string
		file   string
		reader io.Reader
		want   []byte
		err    error
	}{
		{
			name:   "invalid (empty)",
			reader: bytes.NewBufferString(""),
			want:   nil,
			err:    ErrInvalidImageSrc,
		},
		{
			name:   "invalid (invalid gif)",
			reader: io.LimitReader(testutils.MustOpenGif("tooth.gif"), 10),
			want:   nil,
			err:    ErrInvalidImageSrc,
		},
		{
			name: "success (tooth 正方形、Disposal設定アリ)",
			file: "tooth.gif",
			want: lo.Must(io.ReadAll(testutils.MustOpenGif("tooth_resized.gif"))),
			err:  nil,
		},
		{
			name: "success (new_year 横長)",
			file: "new_year.gif",
			want: lo.Must(io.ReadAll(testutils.MustOpenGif("new_year_resized.gif"))),
			err:  nil,
		},
		{
			name: "success (miku 縦長、差分最適化)",
			file: "miku.gif",
			want: lo.Must(io.ReadAll(testutils.MustOpenGif("miku_resized.gif"))),
			err:  nil,
		},
	}

	for _, tt := range test {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			processor, _ := setup()
			if tt.file != "" { // ファイルはこのタイミングで開かないと正常なデータにならない
				tt.reader = testutils.MustOpenGif(tt.file)
			}

			actual, err := processor.FitAnimationGIF(tt.reader, 256, 256)
			if tt.err != nil {
				assert.Equal(t, tt.err, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tt.want, lo.Must(io.ReadAll(actual)))
			}
		})
	}
}
