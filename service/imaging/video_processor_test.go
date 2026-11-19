package imaging

import (
	"bytes"
	"context"
	"image"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestVideoThumbnail_4KAndConfiguredOutput(t *testing.T) {
	encoded := generateTestVideo(t, 4096, 2160)
	p := NewProcessor(Config{
		Concurrency: 1, MaxPixels: 2560 * 1600,
		ThumbnailMaxSize: image.Pt(128, 128),
	})
	thumbnail, err := p.VideoThumbnail(context.Background(), bytes.NewReader(encoded))
	require.NoError(t, err)
	require.Equal(t, 128, thumbnail.Bounds().Dx())
	require.LessOrEqual(t, thumbnail.Bounds().Dy(), 128)
}

func TestVideoThumbnail_CancelWhileWaiting(t *testing.T) {
	p := NewProcessor(Config{Concurrency: 1, ThumbnailMaxSize: image.Pt(360, 480)}).(*defaultProcessor)
	require.NoError(t, p.sp.Acquire(context.Background(), 1))
	defer p.sp.Release(1)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	// A nil source ensures no input is read after cancellation while waiting.
	_, err := p.VideoThumbnail(ctx, nil)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}
