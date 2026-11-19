package imaging

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ffmpeg "github.com/u2takey/ffmpeg-go"
)

func TestGenerateVideoThumbnail(t *testing.T) {
	encoded := generateTestVideo(t, 16, 16)

	thumbnail, err := generateVideoThumbnail(context.Background(), bytes.NewReader(encoded), 4096*2160, image.Pt(360, 450))
	require.NoError(t, err)
	got := imageColorToRGBA(thumbnail.At(0, 0))
	assert.Greater(t, got.B, got.R)
	assert.Greater(t, got.B, got.G)
}

func TestGenerateVideoThumbnail_Resize(t *testing.T) {
	encoded := generateTestVideo(t, 1600, 1200)

	thumbnail, err := generateVideoThumbnail(context.Background(), bytes.NewReader(encoded), 4096*2160, image.Pt(360, 450))
	require.NoError(t, err)
	assert.Equal(t, 360, thumbnail.Bounds().Dx())
	assert.Equal(t, 270, thumbnail.Bounds().Dy())
}

func TestGenerateVideoThumbnail_DisplayAspectRatio(t *testing.T) {
	encoded := generateTestVideo(t, 720, 576, "16/15")
	thumbnail, err := generateVideoThumbnail(context.Background(), bytes.NewReader(encoded), 4096*2160, image.Pt(360, 450))
	require.NoError(t, err)
	assert.Equal(t, image.Pt(360, 270), thumbnail.Bounds().Size())
}

func TestGenerateVideoThumbnail_Limits(t *testing.T) {
	encoded := generateTestVideo(t, 160, 120)
	_, err := generateVideoThumbnail(context.Background(), bytes.NewReader(encoded), 160*120-1, image.Pt(360, 450))
	require.ErrorIs(t, err, ErrPixelLimitExceeded)
	thumbnail, err := generateVideoThumbnail(context.Background(), bytes.NewReader(encoded), 160*120, image.Pt(80, 40))
	require.NoError(t, err)
	assert.LessOrEqual(t, thumbnail.Bounds().Dx(), 80)
	assert.LessOrEqual(t, thumbnail.Bounds().Dy(), 40)
}

func TestGenerateVideoThumbnail_Canceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := generateVideoThumbnail(ctx, nil, 4096*2160, image.Pt(360, 450))
	require.ErrorIs(t, err, context.Canceled)
}

func TestGenerateVideoThumbnail_CancelRunningFFmpeg(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a POSIX shell executable")
	}
	encoded := generateTestVideo(t, 16, 16)
	dir := t.TempDir()
	command := filepath.Join(dir, "ffmpeg")
	require.NoError(t, os.WriteFile(command, []byte("#!/bin/sh\nprintf started > \"$0.started\"\nexec sleep 30\n"), 0700))
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := generateVideoThumbnail(ctx, bytes.NewReader(encoded), 4096*2160, image.Pt(360, 450))
	require.ErrorIs(t, err, context.DeadlineExceeded)
	_, err = os.Stat(command + ".started")
	require.NoError(t, err, "FFmpeg must have started before cancellation")
}

func generateTestVideo(t *testing.T, width, height int, sar ...string) []byte {
	t.Helper()

	// Two RGB frames: red, then blue. A regular MP4 keeps its index at
	// the end of the file, which verifies that thumbnail generation uses a
	// seekable FFmpeg input rather than stdin.
	pixels := width * height
	rawVideo := append(bytes.Repeat([]byte{0xff, 0, 0}, pixels), bytes.Repeat([]byte{0, 0, 0xff}, pixels)...)
	videoPath := filepath.Join(t.TempDir(), "video.mp4")
	stream := ffmpeg.Input("pipe:0", ffmpeg.KwArgs{
		"format":    "rawvideo",
		"framerate": 1,
		"pix_fmt":   "rgb24",
		"s":         fmt.Sprintf("%dx%d", width, height),
	})
	if len(sar) > 0 {
		stream = stream.Filter("setsar", ffmpeg.Args{sar[0]})
	}
	err := stream.Output(videoPath, ffmpeg.KwArgs{
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
