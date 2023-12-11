package imaging

import (
	"errors"
	"image"
)

var (
	ErrPixelLimitExceeded = errors.New("the image exceeds max pixels limit")
	ErrInvalidImageSrc    = errors.New("invalid image src")
)

type Config struct {
	// MaxPixels 処理可能な最大画素数
	// この値を超える画素数の画像を処理しようとした場合、全てエラーになります
	MaxPixels int
	// Concurrency 処理並列数
	Concurrency int
	// ThumbnailMaxSize サムネイル画像サイズ
	ThumbnailMaxSize image.Point
}
