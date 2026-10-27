package imaging

import (
	"bytes"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ffmpeg "github.com/u2takey/ffmpeg-go"
)

func TestGenerateVideoThumbnail(t *testing.T) {
	encoded := generateTestVideo(t, 16, 16)

	thumbnail, err := generateVideoThumbnail(bytes.NewReader(encoded))
	require.NoError(t, err)
	got := imageColorToRGBA(thumbnail.At(0, 0))
	assert.Greater(t, got.B, got.R)
	assert.Greater(t, got.B, got.G)
}

func TestGenerateVideoThumbnail_Resize(t *testing.T) {
	encoded := generateTestVideo(t, 1600, 1200)

	thumbnail, err := generateVideoThumbnail(bytes.NewReader(encoded))
	require.NoError(t, err)
	assert.Equal(t, 360, thumbnail.Bounds().Dx())
	assert.Equal(t, 270, thumbnail.Bounds().Dy())
}

func generateTestVideo(t *testing.T, width, height int) []byte {
	t.Helper()

	// Two RGB frames: red, then blue. A regular MP4 keeps its index at
	// the end of the file, which verifies that thumbnail generation uses a
	// seekable FFmpeg input rather than stdin.
	pixels := width * height
	rawVideo := append(bytes.Repeat([]byte{0xff, 0, 0}, pixels), bytes.Repeat([]byte{0, 0, 0xff}, pixels)...)
	videoPath := filepath.Join(t.TempDir(), "video.mp4")
	err := ffmpeg.Input("pipe:0", ffmpeg.KwArgs{
		"format":    "rawvideo",
		"framerate": 1,
		"pix_fmt":   "rgb24",
		"s":         fmt.Sprintf("%dx%d", width, height),
	}).Output(videoPath, ffmpeg.KwArgs{
		"vcodec": "mpeg4",
	}).OverWriteOutput().WithInput(bytes.NewReader(rawVideo)).Run()
	require.NoError(t, err)
	encoded, err := os.ReadFile(videoPath)
	require.NoError(t, err)
	return encoded
}

func imageColorToRGBA(c color.Color) color.RGBA {
	r, g, b, a := c.RGBA()
	return color.RGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}
}
