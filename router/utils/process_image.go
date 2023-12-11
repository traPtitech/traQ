package utils

import (
	"bytes"
	"image/png"
	"io"

	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/service/file"
	"github.com/traPtitech/traQ/service/imaging"
)

const (
	iconMaxFileSize   = 2 << 20 // 2MB
	iconMaxImageSize  = 256
	stampMaxFileSize  = 1 << 20 // 1MB
	stampMaxImageSize = 128
)

// SaveUploadIconImage MultipartFormでアップロードされたアイコン画像ファイルを保存
func SaveUploadIconImage(p imaging.Processor, c echo.Context, m file.Manager, name string) (uuid.UUID, error) {
	return saveUploadImage(p, c, m, name, model.FileTypeIcon, iconMaxFileSize, iconMaxImageSize)
}

// SaveUploadStampImage MultipartFormでアップロードされたスタンプ画像ファイルを保存
func SaveUploadStampImage(p imaging.Processor, c echo.Context, m file.Manager, name string) (uuid.UUID, error) {
	return saveUploadImage(p, c, m, name, model.FileTypeStamp, stampMaxFileSize, stampMaxImageSize)
}

func saveUploadImage(p imaging.Processor, c echo.Context, m file.Manager, name string, fType model.FileType, maxFileSize int64, maxImageSize int) (uuid.UUID, error) {
	const (
		tooLargeImage = "too large image"
		badImage      = "bad image"
	)

	// ファイルオープン
	src, fh, err := c.Request().FormFile(name)
	if err != nil {
		return uuid.Nil, herror.BadRequest(err)
	}
	defer src.Close()

	// ファイルサイズ制限
	if fh.Size > maxFileSize {
		return uuid.Nil, herror.BadRequest(tooLargeImage)
	}

	args := file.SaveArgs{
		FileName: fh.Filename,
		FileType: fType,
	}

	switch fh.Header.Get(echo.HeaderContentType) {
	case consts.MimeImagePNG, consts.MimeImageJPEG:
		img, err := p.Fit(src, maxImageSize, maxImageSize)
		if err != nil {
			switch err {
			case imaging.ErrInvalidImageSrc:
				return uuid.Nil, herror.BadRequest(badImage)
			case imaging.ErrPixelLimitExceeded:
				return uuid.Nil, herror.BadRequest(tooLargeImage)
			default:
				return uuid.Nil, herror.InternalServerError(err)
			}
		}

		// PNGに変換
		b := bytes.Buffer{}
		if err := png.Encode(&b, img); err != nil {
			return uuid.Nil, herror.InternalServerError(err)
		}

		args.Src = bytes.NewReader(b.Bytes())
		args.FileSize = int64(b.Len())
		args.MimeType = consts.MimeImagePNG
		args.Thumbnail = img // サムネイル画像より小さいという前提

	case consts.MimeImageGIF:
		// リサイズ
		b, err := p.FitAnimationGIF(src, maxImageSize, maxImageSize)
		if err != nil {
			switch err {
			case imaging.ErrInvalidImageSrc:
				// 不正なgifである
				return uuid.Nil, herror.BadRequest(badImage)
			default:
				// 予期しないエラー
				return uuid.Nil, herror.InternalServerError(err)
			}
		}

		args.Src = b
		args.FileSize = b.Size()
		args.MimeType = consts.MimeImageGIF

		args.Thumbnail, err = p.Thumbnail(b)
		if err != nil {
			return uuid.Nil, herror.InternalServerError(err)
		}
		_, _ = b.Seek(0, io.SeekStart)

	default:
		return uuid.Nil, herror.BadRequest(badImage)
	}

	// ファイル保存
	f, err := m.Save(args)
	if err != nil {
		return uuid.Nil, herror.InternalServerError(err)
	}

	return f.GetID(), nil
}
