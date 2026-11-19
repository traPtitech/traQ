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
	// VideoMaxPixels 動画入力の最大画素数。0の場合は4096*2160。
	VideoMaxPixels int
	// Concurrency 処理並列数
	Concurrency int
	// ThumbnailMaxSize 画像・動画サムネイルの最大出力サイズ
	ThumbnailMaxSize image.Point
}
