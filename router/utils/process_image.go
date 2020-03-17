package utils

import (
	"bytes"
	"context"
	"github.com/disintegration/imaging"
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/utils/imagemagick"
	"time"
)

// ImageMagickPath imagemagick実行ファイルパス
var ImageMagickPath = ""

// SaveUploadImage MultipartFormでアップロードされた画像ファイルをリサイズして保存
func SaveUploadImage(c echo.Context, repo repository.Repository, name string, fType string, maxFileSize int64, maxImageSize int) (uuid.UUID, error) {
	// ファイルオープン
	src, file, err := c.Request().FormFile(name)
	if err != nil {
		return uuid.Nil, herror.BadRequest(err)
	}
	defer src.Close()

	// ファイルサイズ制限
	if file.Size > maxFileSize {
		return uuid.Nil, herror.BadRequest("too large image file (limit exceeded)")
	}

	// ファイルタイプ確認・必要があればリサイズ
	var (
		b    *bytes.Buffer
		mime string
	)
	switch file.Header.Get(echo.HeaderContentType) {
	case consts.MimeImagePNG, consts.MimeImageJPEG:
		// デコード
		img, err := imaging.Decode(src, imaging.AutoOrientation(true))
		if err != nil {
			return uuid.Nil, herror.BadRequest("bad image file")
		}

		// リサイズ
		if size := img.Bounds().Size(); size.X > maxImageSize || size.Y > maxImageSize {
			img = imaging.Fit(img, maxImageSize, maxImageSize, imaging.Linear)
		}

		// PNGに戻す
		b = &bytes.Buffer{}
		_ = imaging.Encode(b, img, imaging.PNG)
		mime = consts.MimeImagePNG
	case consts.MimeImageGIF:
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) // 10秒以内に終わらないファイルは無効
		defer cancel()

		// リサイズ
		b, err = imagemagick.ResizeAnimationGIF(ctx, ImageMagickPath, src, maxImageSize, maxImageSize, false)
		if err != nil {
			switch err {
			case imagemagick.ErrUnavailable:
				// gifは一時的にサポートされていない
				return uuid.Nil, herror.BadRequest("gif file is temporarily unsupported")
			case imagemagick.ErrUnsupportedType:
				// 不正なgifである
				return uuid.Nil, herror.BadRequest("bad image file")
			case context.DeadlineExceeded:
				// リサイズタイムアウト
				return uuid.Nil, herror.BadRequest("bad image file (resize timeout)")
			default:
				// 予期しないエラー
				return uuid.Nil, herror.InternalServerError(err)
			}
		}
		mime = consts.MimeImageGIF
	default:
		return uuid.Nil, herror.BadRequest("invalid image file")
	}

	// ファイル保存
	f, err := repo.SaveFile(repository.SaveFileArgs{
		FileName: file.Filename,
		FileSize: int64(b.Len()),
		MimeType: mime,
		FileType: fType,
		Src:      b,
	})
	if err != nil {
		return uuid.Nil, herror.InternalServerError(err)
	}

	return f.ID, nil
}
