package file

import (
	"io"

	"github.com/sapphi-red/midec"
	// add gif, png, webp support
	_ "github.com/sapphi-red/midec/gif"
	_ "github.com/sapphi-red/midec/png"
	_ "github.com/sapphi-red/midec/webp"
)

func isAnimatedImage(r io.Reader) (bool, error) {
	return midec.IsAnimated(r)
}
