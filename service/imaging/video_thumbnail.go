package imaging

import (
	"bytes"
	"fmt"
	"image"
	_ "image/png"
	"io"
	"os"
	"strings"

	ffmpeg "github.com/u2takey/ffmpeg-go"
)

const (
	videoThumbnailMaxWidth  = 360
	videoThumbnailMaxHeight = 450
)

// generateVideoThumbnail decodes the second video frame and returns it as an
// image. FFmpeg's select filter counts decoded frames from zero, so n=1 is the
// second frame.
func generateVideoThumbnail(src io.Reader) (image.Image, error) {
	input, err := os.CreateTemp("", "traq-video-thumbnail-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary video file: %w", err)
	}
	inputPath := input.Name()
	defer os.Remove(inputPath)

	if _, err := io.Copy(input, src); err != nil {
		_ = input.Close()
		return nil, fmt.Errorf("failed to copy video to temporary file: %w", err)
	}
	if err := input.Close(); err != nil {
		return nil, fmt.Errorf("failed to close temporary video file: %w", err)
	}

	var output bytes.Buffer
	var stderr bytes.Buffer
	err = ffmpeg.Input(inputPath).
		Filter("select", ffmpeg.Args{"eq(n,1)"}).
		Filter("scale", ffmpeg.Args{
			fmt.Sprintf("min(%d,iw)", videoThumbnailMaxWidth),
			fmt.Sprintf("min(%d,ih)", videoThumbnailMaxHeight),
		}, ffmpeg.KwArgs{
			"force_original_aspect_ratio": "decrease",
			"force_divisible_by":          2,
		}).
		Output("pipe:1", ffmpeg.KwArgs{
			"format":  "image2pipe",
			"vcodec":  "png",
			"vframes": 1,
		}).
		WithOutput(&output).
		WithErrorOutput(&stderr).
		Run()
	if err != nil {
		return nil, fmt.Errorf("failed to decode second video frame: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	thumbnail, _, err := image.Decode(&output)
	if err != nil {
		return nil, fmt.Errorf("failed to decode generated thumbnail: %w", err)
	}
	return thumbnail, nil
}
