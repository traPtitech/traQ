package imagemagick

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/traPtitech/traQ/config"
	"io"
	"os/exec"
)

// ResizeAnimationGIF Animation GIF画像をimagemagickでリサイズします
func ResizeAnimationGIF(ctx context.Context, src io.Reader, maxWidth, maxHeight int) (*bytes.Buffer, error) {
	if len(config.ImageMagickConverterExec) == 0 {
		return nil, ErrUnavailable
	}

	if maxHeight <= 0 || maxWidth <= 0 {
		return nil, errors.New("maxWidth or maxHeight is wrong")
	}

	cmd := exec.CommandContext(ctx, config.ImageMagickConverterExec, "-coalesce", "-resize", fmt.Sprintf("%dx%d", maxWidth, maxHeight), "-deconstruct", "-", "gif:-")

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	go func() {
		defer stdin.Close()
		io.Copy(stdin, src)
	}()

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	b := &bytes.Buffer{}
	io.Copy(b, stdout)

	if err := cmd.Wait(); err != nil {
		switch err.(type) {
		case *exec.ExitError:
			return nil, ErrUnsupportedType
		default:
			return nil, err
		}
	}

	return b, nil
}
