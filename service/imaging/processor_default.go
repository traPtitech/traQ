package imaging

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/draw"
	"image/gif"
	_ "image/jpeg" // image.Decode用
	_ "image/png"  // image.Decode用
	"io"
	"math"

	_ "golang.org/x/image/webp" // image.Decode用

	"github.com/disintegration/imaging"
	"github.com/go-audio/wav"
	"github.com/hajimehoshi/go-mp3"
	"github.com/motoki317/go-waveform"
	"golang.org/x/sync/semaphore"

	imaging2 "github.com/traPtitech/traQ/utils/imaging"
)

type defaultProcessor struct {
	c  Config
	sp *semaphore.Weighted
}

func NewProcessor(c Config) Processor {
	return &defaultProcessor{
		c:  c,
		sp: semaphore.NewWeighted(int64(c.Concurrency)),
	}
}

func (p *defaultProcessor) Thumbnail(src io.ReadSeeker) (image.Image, error) {
	return p.Fit(src, p.c.ThumbnailMaxSize.X, p.c.ThumbnailMaxSize.Y)
}

func (p *defaultProcessor) Fit(src io.ReadSeeker, width, height int) (image.Image, error) {
	_ = p.sp.Acquire(context.Background(), 1)
	defer p.sp.Release(1)

	imgCfg, _, err := image.DecodeConfig(src)
	if err != nil {
		if err == image.ErrFormat {
			return nil, ErrInvalidImageSrc
		}
		return nil, err
	}

	// 画素数チェック
	if imgCfg.Width*imgCfg.Height > p.c.MaxPixels {
		return nil, ErrPixelLimitExceeded
	}

	// 先頭に戻す
	if _, err := src.Seek(0, 0); err != nil {
		return nil, err
	}

	// 変換
	orig, err := imaging.Decode(src, imaging.AutoOrientation(true))
	if err != nil {
		return nil, ErrInvalidImageSrc
	}

	if imgCfg.Width > width || imgCfg.Height > height {
		// mks2013: フロントで使用している https://github.com/nodeca/pica のデフォルト
		return imaging.Fit(orig, width, height, mks2013Filter), nil
	}
	return orig, nil
}

func (p *defaultProcessor) FitAnimationGIF(src io.Reader, width, height int) (*bytes.Reader, error) {
	srcImage, err := gif.DecodeAll(src)
	if err != nil {
		return nil, ErrInvalidImageSrc
	}

	srcWidth, srcHeight := srcImage.Config.Width, srcImage.Config.Height
	// 画素数チェック
	if srcWidth*srcHeight > p.c.MaxPixels {
		return nil, ErrPixelLimitExceeded
	}
	// 画像が十分小さければスキップ
	if srcWidth <= width && srcHeight <= height {
		return imaging2.GifToBytesReader(srcImage)
	}

	// 元の比率を保つよう調整 & 拡大・縮小比率を計算
	floatSrcWidth, floatSrcHeight, floatWidth, floatHeight := float64(srcWidth), float64(srcHeight), float64(width), float64(height)
	ratio := floatWidth / floatSrcWidth
	if floatSrcWidth/floatSrcHeight > floatWidth/floatHeight {
		ratio = floatWidth / floatSrcWidth
		height = int(math.Round(floatSrcHeight * ratio))
	} else if floatSrcWidth/floatSrcHeight < floatWidth/floatHeight {
		ratio = floatHeight / floatSrcHeight
		width = int(math.Round(floatSrcWidth * ratio))
	}

	destImage := &gif.GIF{
		Image:     make([]*image.Paletted, len(srcImage.Image)),
		Delay:     srcImage.Delay,
		LoopCount: srcImage.LoopCount,
		Disposal:  srcImage.Disposal,
		Config: image.Config{
			ColorModel: srcImage.Config.ColorModel,
			Width:      width,
			Height:     height,
		},
		BackgroundIndex: srcImage.BackgroundIndex,
	}

	// フレームを重ねるためのキャンバス
	//	差分最適化されたGIFに対応するための処置
	// 	差分最適化されたGIFでは、1フレーム目以外、周りが透明ピクセルのフレームを
	// 	次々に重ねていくことでアニメーションを表現する
	// 	周りが透明ピクセルのフレームをそのまま縮小すると、周りの透明ピクセルと
	// 	混ざった色が透明色ではなくなってフレームの縁に黒っぽいノイズが入ってしまう
	// 	ため、キャンバスでフレームを重ねてから縮小する
	var tempCanvas *image.NRGBA

	for i, srcFrame := range srcImage.Image {
		// 元のフレームのサイズ
		//  差分最適化されたGIFでは、これが元GIFのサイズより小さいことがある
		srcBounds := srcFrame.Bounds()
		// フレームサイズを、上で計算した拡大・縮小比率に合わせる
		destBounds := image.Rect(
			int(math.Round(float64(srcBounds.Min.X)*ratio)),
			int(math.Round(float64(srcBounds.Min.Y)*ratio)),
			int(math.Round(float64(srcBounds.Max.X)*ratio)),
			int(math.Round(float64(srcBounds.Max.Y)*ratio)),
		)

		if i == 0 { // 1フレーム目は必ずGIFと同じサイズなので、これでキャンバスを初期化
			tempCanvas = image.NewNRGBA(srcBounds)
		}
		// それまでのフレームに読んだフレームを重ねる
		draw.Draw(tempCanvas, srcBounds, srcFrame, srcBounds.Min, draw.Over)

		// 重ねたフレームを縮小
		fittedImage := imaging.Resize(tempCanvas, width, height, mks2013Filter)

		// 縮小後のフレームサイズに合わせて、縮小されたイメージを切り抜き
		destFrame := image.NewPaletted(destBounds, srcFrame.Palette)
		draw.Draw(destFrame, destBounds, fittedImage.SubImage(destBounds), destBounds.Min, draw.Src)

		destImage.Image[i] = destFrame
	}

	return imaging2.GifToBytesReader(destImage)
}

func (p *defaultProcessor) WaveformMp3(src io.ReadSeeker, width, height int) (r io.Reader, err error) {
	defer func() {
		// workaround fix https://github.com/traPtitech/traQ/issues/1178
		if p := recover(); p != nil {
			if perr, ok := p.(error); ok {
				err = perr
			} else {
				err = fmt.Errorf("recovered: %v", p)
			}
		}
	}()
	d, err := mp3.NewDecoder(src)
	if err != nil {
		return nil, err
	}
	return waveform.OutputWaveformImageMp3(d, &waveform.Option{
		Resolution: width / 5,
		Width:      width,
		Height:     height,
	})
}

func (p *defaultProcessor) WaveformWav(src io.ReadSeeker, width, height int) (io.Reader, error) {
	d := wav.NewDecoder(src)
	return waveform.OutputWaveformImageWav(d, &waveform.Option{
		Resolution: width / 5,
		Width:      width,
		Height:     height,
	})
}
