package utils

import (
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"net/http"
	"strconv"
)

// ChangeUserIcon userIDのユーザーのアイコン画像を変更するハンドラ
func ChangeUserIcon(c echo.Context, repo repository.Repository, userID uuid.UUID) error {
	iconID, err := SaveUploadImage(c, repo, "file", model.FileTypeIcon, 2<<20, 256)
	if err != nil {
		return err
	}

	// アイコン変更
	if err := repo.ChangeUserIcon(userID, iconID); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}

// ServeUserIcon userのアイコン画像ファイルをレスポンスとして返す
func ServeUserIcon(c echo.Context, repo repository.Repository, user model.UserInfo) error {
	// ファイルメタ取得
	meta, err := repo.GetFileMeta(user.GetIconFileID())
	if err != nil {
		return herror.InternalServerError(err)
	}

	// ファイルオープン
	file, err := meta.Open()
	if err != nil {
		return herror.InternalServerError(err)
	}
	defer file.Close()

	// レスポンスヘッダ設定
	c.Response().Header().Set(echo.HeaderContentType, meta.GetMIMEType())
	c.Response().Header().Set(consts.HeaderETag, strconv.Quote(meta.GetMD5Hash()))

	// ファイル送信
	http.ServeContent(c.Response(), c.Request(), meta.GetFileName(), meta.GetCreatedAt(), file)
	return nil
}
