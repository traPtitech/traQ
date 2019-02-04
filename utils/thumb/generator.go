package thumb

import (
	"bytes"
	"context"
	"errors"
	"golang.org/x/image/bmp"
	"golang.org/x/image/draw"
	"golang.org/x/image/tiff"
	"golang.org/x/image/webp"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
)

const (
	// ThumbnailMaxWidth サムネイルの最大幅
	ThumbnailMaxWidth = 360
	// ThumbnailMaxHeight サムネイルの最大高さ
	ThumbnailMaxHeight = 480
)

var (
	// ErrFileThumbUnsupported この形式のファイルのサムネイル生成はサポートされていない
	ErrFileThumbUnsupported = errors.New("generating a thumbnail of the file is not supported")
)

// CalcThumbnailSize サムネイル画像のサイズを計算します
func CalcThumbnailSize(size image.Point, maxSize image.Point) image.Point {
	if size.X <= maxSize.X && size.Y <= maxSize.Y {
		// 元画像がサムネイル画像より小さい
		return size
	}

	ratio := float64(size.X) / float64(size.Y)
	boxRatio := float64(maxSize.X) / float64(maxSize.Y)

	if ratio > boxRatio {
		return image.Pt(maxSize.X, int(float64(maxSize.X)/ratio))
	}
	return image.Pt(int(float64(maxSize.Y)*ratio), maxSize.Y)
}

// Generate サムネイル画像を生成します
func Generate(ctx context.Context, src io.Reader, mime string) (image.Image, error) {
	var f func(io.Reader) (image.Image, error)
	switch mime {
	case "image/png":
		f = png.Decode
	case "image/gif":
		f = gif.Decode
	case "image/jpeg":
		f = jpeg.Decode
	case "image/bmp":
		f = bmp.Decode
	case "image/webp":
		f = webp.Decode
	case "image/tiff":
		f = tiff.Decode
	default: // Unsupported Type
		return nil, ErrFileThumbUnsupported
	}

	img, err := f(src)
	if err != nil {
		return nil, err
	}
	return Resize(img, ThumbnailMaxWidth, ThumbnailMaxHeight), nil
}

// EncodeToPNG image.Imageをpngのバイトバッファにエンコードします
func EncodeToPNG(img image.Image) (b *bytes.Buffer, err error) {
	b = &bytes.Buffer{}
	err = png.Encode(b, img)
	return
}

// Resize imgをリサイズします。アスペクト比は保持されます。
func Resize(img image.Image, maxWidth, maxHeight int) image.Image {
	var dst draw.Image = image.NewRGBA(image.Rectangle{Min: image.ZP, Max: CalcThumbnailSize(img.Bounds().Size(), image.Pt(maxWidth, maxHeight))})
	draw.Draw(dst, dst.Bounds(), image.White, image.ZP, draw.Src)
	draw.ApproxBiLinear.Scale(dst, dst.Bounds(), img, img.Bounds(), draw.Src, nil)
	return dst.(image.Image)
}
