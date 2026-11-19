package imaging

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	_ "image/png"
	"io"
	"os"
	"os/exec"
	"strings"

	ffmpeg "github.com/u2takey/ffmpeg-go"
)

// generateVideoThumbnail decodes the second video frame and returns it as an
// image. FFmpeg's select filter counts decoded frames from zero, so n=1 is the
// second frame.
func generateVideoThumbnail(ctx context.Context, src io.Reader, maxPixels int, maxSize image.Point) (image.Image, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if maxPixels <= 0 || maxSize.X <= 0 || maxSize.Y <= 0 {
		return nil, fmt.Errorf("invalid video thumbnail limits")
	}
	input, err := os.CreateTemp("", "traq-video-thumbnail-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary video file: %w", err)
	}
	inputPath := input.Name()
	defer os.Remove(inputPath)

	if _, err := io.Copy(input, &videoContextReader{ctx: ctx, src: src}); err != nil {
		_ = input.Close()
		return nil, fmt.Errorf("failed to copy video to temporary file: %w", err)
	}
	if err := input.Close(); err != nil {
		return nil, fmt.Errorf("failed to close temporary video file: %w", err)
	}
	// Inspect the encoded dimensions before starting thumbnail decoding.
	probe, err := exec.CommandContext(ctx, "ffprobe", "-v", "error", "-select_streams", "v:0",
		"-show_entries", "stream=width,height", "-of", "json", inputPath).Output()
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("failed to probe video dimensions: %w", err)
	}
	var metadata struct {
		Streams []struct{ Width, Height int }
	}
	if err := json.Unmarshal(probe, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse video dimensions: %w", err)
	}
	if len(metadata.Streams) == 0 || metadata.Streams[0].Width <= 0 || metadata.Streams[0].Height <= 0 {
		return nil, fmt.Errorf("missing video dimensions")
	}
	if metadata.Streams[0].Width > maxPixels/metadata.Streams[0].Height {
		return nil, ErrPixelLimitExceeded
	}

	var output bytes.Buffer
	var stderr bytes.Buffer
	stream := ffmpeg.Input(inputPath).Get("v:0").
		Filter("select", ffmpeg.Args{"eq(n,1)"}).
		Filter("scale", ffmpeg.Args{
			fmt.Sprintf("max(1,trunc(min(iw*sar,min(%d,%d*dar))))", maxSize.X, maxSize.Y),
			"max(1,trunc(ow/dar))",
		}).
		Filter("setsar", ffmpeg.Args{"1"}).
		Output("pipe:1", ffmpeg.KwArgs{
			"format":  "image2pipe",
			"vcodec":  "png",
			"vframes": 1,
		})
	// Set the request context before WithOutput adds its context values.
	stream.Context = ctx
	err = stream.WithOutput(&output).WithErrorOutput(&stderr).Run()
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("failed to decode second video frame: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	thumbnail, _, err := image.Decode(&output)
	if err != nil {
		return nil, fmt.Errorf("failed to decode generated thumbnail: %w", err)
	}
	return thumbnail, nil
}

// Check cancellation between reads; an arbitrary blocking Reader itself cannot
// be interrupted here. HTTP request bodies are closed by their request owner.
type videoContextReader struct {
	ctx context.Context
	src io.Reader
}

func (r *videoContextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.src.Read(p)
}
