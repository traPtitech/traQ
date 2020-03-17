package utils

import (
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/router/sessions"
	"gopkg.in/guregu/null.v3"
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
	if err := repo.UpdateUser(userID, repository.UpdateUserArgs{IconFileID: uuid.NullUUID{UUID: iconID, Valid: true}}); err != nil {
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

// ChangeUserPassword userIDのユーザーのパスワードを変更するハンドラ
func ChangeUserPassword(c echo.Context, repo repository.Repository, userID uuid.UUID, newPassword string) error {
	if err := repo.UpdateUser(userID, repository.UpdateUserArgs{Password: null.StringFrom(newPassword)}); err != nil {
		return herror.InternalServerError(err)
	}

	// ユーザーの全セッションを破棄(強制ログアウト)
	_ = sessions.DestroyByUserID(userID)
	return c.NoContent(http.StatusNoContent)
}
