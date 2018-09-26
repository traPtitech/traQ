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
	switch mime {
	case "image/png":
		img, err := png.Decode(src)
		if err != nil {
			return nil, err
		}
		return Resize(ctx, img, ThumbnailMaxWidth, ThumbnailMaxHeight)

	case "image/gif":
		img, err := gif.Decode(src)
		if err != nil {
			return nil, err
		}
		return Resize(ctx, img, ThumbnailMaxWidth, ThumbnailMaxHeight)

	case "image/jpeg":
		img, err := jpeg.Decode(src)
		if err != nil {
			return nil, err
		}
		return Resize(ctx, img, ThumbnailMaxWidth, ThumbnailMaxHeight)

	case "image/bmp":
		img, err := bmp.Decode(src)
		if err != nil {
			return nil, err
		}
		return Resize(ctx, img, ThumbnailMaxWidth, ThumbnailMaxHeight)

	case "image/webp":
		img, err := webp.Decode(src)
		if err != nil {
			return nil, err
		}
		return Resize(ctx, img, ThumbnailMaxWidth, ThumbnailMaxHeight)

	case "image/tiff":
		img, err := tiff.Decode(src)
		if err != nil {
			return nil, err
		}
		return Resize(ctx, img, ThumbnailMaxWidth, ThumbnailMaxHeight)

	default: // Unsupported Type
		return nil, ErrFileThumbUnsupported
	}
}

// EncodeToPNG image.Imageをpngのバイトバッファにエンコードします
func EncodeToPNG(img image.Image) (b *bytes.Buffer, err error) {
	b = &bytes.Buffer{}
	err = png.Encode(b, img)
	return
}

// EncodeToJPG image.Imageをjpgのバイトバッファにエンコードします
func EncodeToJPG(img image.Image) (b *bytes.Buffer, err error) {
	b = &bytes.Buffer{}
	err = jpeg.Encode(b, img, &jpeg.Options{Quality: 100})
	return
}

// Resize imgをリサイズします。アスペクト比は保持されます。
func Resize(ctx context.Context, img image.Image, maxWidth, maxHeight int) (image.Image, error) {
	var dst draw.Image

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		thumbSize := CalcThumbnailSize(img.Bounds().Size(), image.Pt(maxWidth, maxHeight))
		dst = image.NewRGBA(image.Rectangle{Min: image.ZP, Max: thumbSize})
		draw.Draw(dst, dst.Bounds(), image.White, image.ZP, draw.Src)
		draw.ApproxBiLinear.Scale(dst, dst.Bounds(), img, img.Bounds(), draw.Src, nil)
	}

	return dst.(image.Image), nil
}
