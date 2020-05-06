package utils

import (
	"bytes"
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/utils/imaging"
	"image/png"
)

const (
	iconMaxFileSize   = 2 << 20 // 2MB
	iconMaxImageSize  = 256
	stampMaxFileSize  = 1 << 20 // 1MB
	stampMaxImageSize = 128
)

// SaveUploadIconImage MultipartFormでアップロードされたアイコン画像ファイルを保存
func SaveUploadIconImage(p imaging.Processor, c echo.Context, repo repository.Repository, name string) (uuid.UUID, error) {
	return saveUploadImage(p, c, repo, name, model.FileTypeIcon, iconMaxFileSize, iconMaxImageSize)
}

// SaveUploadStampImage MultipartFormでアップロードされたスタンプ画像ファイルを保存
func SaveUploadStampImage(p imaging.Processor, c echo.Context, repo repository.Repository, name string) (uuid.UUID, error) {
	return saveUploadImage(p, c, repo, name, model.FileTypeStamp, stampMaxFileSize, stampMaxImageSize)
}

func saveUploadImage(p imaging.Processor, c echo.Context, repo repository.Repository, name string, fType string, maxFileSize int64, maxImageSize int) (uuid.UUID, error) {
	const (
		tooLargeImage = "too large image"
		badImage      = "bad image"
	)

	// ファイルオープン
	src, file, err := c.Request().FormFile(name)
	if err != nil {
		return uuid.Nil, herror.BadRequest(err)
	}
	defer src.Close()

	// ファイルサイズ制限
	if file.Size > maxFileSize {
		return uuid.Nil, herror.BadRequest(tooLargeImage)
	}

	args := repository.SaveFileArgs{
		FileName: file.Filename,
		FileType: fType,
	}

	switch file.Header.Get(echo.HeaderContentType) {
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
		var b = bytes.Buffer{}
		if err := png.Encode(&b, img); err != nil {
			return uuid.Nil, herror.InternalServerError(err)
		}

		args.Src = &b
		args.FileSize = int64(b.Len())
		args.MimeType = consts.MimeImagePNG
		args.Thumbnail = img // サムネイル画像より小さいという前提

	case consts.MimeImageGIF:
		// リサイズ
		b, err := p.FitAnimationGIF(src, maxImageSize, maxImageSize)
		if err != nil {
			switch err {
			case imaging.ErrImageMagickUnavailable:
				// gifは一時的にサポートされていない
				return uuid.Nil, herror.BadRequest("gif file is temporarily unsupported")
			case imaging.ErrInvalidImageSrc, imaging.ErrTimeout:
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

	default:
		return uuid.Nil, herror.BadRequest(badImage)
	}

	// ファイル保存
	f, err := repo.SaveFile(args)
	if err != nil {
		return uuid.Nil, herror.InternalServerError(err)
	}

	return f.GetID(), nil
}
