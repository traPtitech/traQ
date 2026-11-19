//go:generate mockgen -source=$GOFILE -destination=mock_$GOPACKAGE/mock_$GOFILE
package imaging

import (
	"bytes"
	"context"
	"image"
	"io"
)

type Processor interface {
	ImageThumbnail(src io.ReadSeeker) (image.Image, error)
	VideoThumbnail(ctx context.Context, src io.Reader) (image.Image, error)
	Fit(src io.ReadSeeker, width, height int) (image.Image, error)
	FitAnimationGIF(src io.Reader, width, height int) (*bytes.Reader, error)
	WaveformMp3(src io.ReadSeeker, width, height int) (io.Reader, error)
	WaveformWav(src io.ReadSeeker, width, height int) (io.Reader, error)
}
