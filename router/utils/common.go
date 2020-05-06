package utils

import (
	"fmt"
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/router/sessions"
	"github.com/traPtitech/traQ/utils/imaging"
	"github.com/traPtitech/traQ/utils/optional"
	"net/http"
	"strconv"
)

// ChangeUserIcon userIDのユーザーのアイコン画像を変更する
func ChangeUserIcon(p imaging.Processor, c echo.Context, repo repository.Repository, userID uuid.UUID) error {
	iconID, err := SaveUploadIconImage(p, c, repo, "file")
	if err != nil {
		return err
	}

	// アイコン変更
	if err := repo.UpdateUser(userID, repository.UpdateUserArgs{IconFileID: optional.UUIDFrom(iconID)}); err != nil {
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

// ChangeUserPassword userIDのユーザーのパスワードを変更する
func ChangeUserPassword(c echo.Context, repo repository.Repository, userID uuid.UUID, newPassword string) error {
	if err := repo.UpdateUser(userID, repository.UpdateUserArgs{Password: optional.StringFrom(newPassword)}); err != nil {
		return herror.InternalServerError(err)
	}

	// ユーザーの全セッションを破棄(強制ログアウト)
	_ = sessions.DestroyByUserID(userID)
	return c.NoContent(http.StatusNoContent)
}

// ServeFileThumbnail metaのファイルのサムネイルをレスポンスとして返す
func ServeFileThumbnail(c echo.Context, meta model.FileMeta) error {
	if !meta.HasThumbnail() {
		return herror.NotFound()
	}

	file, err := meta.OpenThumbnail()
	if err != nil {
		return herror.InternalServerError(err)
	}
	defer file.Close()

	c.Response().Header().Set(consts.HeaderFileMetaType, meta.GetFileType().String())
	c.Response().Header().Set(consts.HeaderCacheFile, "true")
	c.Response().Header().Set(consts.HeaderCacheControl, "private, max-age=31536000") // 1年間キャッシュ
	return c.Stream(http.StatusOK, meta.GetThumbnailMIMEType(), file)
}

// ServeFile metaのファイル本体をレスポンスとして返す
func ServeFile(c echo.Context, meta model.FileMeta) error {
	// 直接アクセスURLが発行できる場合は、そっちにリダイレクト
	if url := meta.GetAlternativeURL(); len(url) > 0 {
		return c.Redirect(http.StatusFound, url)
	}

	file, err := meta.Open()
	if err != nil {
		return herror.InternalServerError(err)
	}
	defer file.Close()

	c.Response().Header().Set(echo.HeaderContentType, meta.GetMIMEType())
	c.Response().Header().Set(consts.HeaderETag, strconv.Quote(meta.GetMD5Hash()))
	c.Response().Header().Set(consts.HeaderCacheControl, "private, max-age=31536000") // 1年間キャッシュ
	if v, _ := strconv.ParseBool(c.QueryParam("dl")); v {
		c.Response().Header().Set(echo.HeaderContentDisposition, fmt.Sprintf("attachment; filename=%s", meta.GetFileName()))
	}
	c.Response().Header().Set(consts.HeaderFileMetaType, meta.GetFileType().String())
	switch meta.GetFileType() {
	case model.FileTypeStamp, model.FileTypeIcon:
		c.Response().Header().Set(consts.HeaderCacheFile, "true")
	}

	http.ServeContent(c.Response(), c.Request(), meta.GetFileName(), meta.GetCreatedAt(), file)
	return nil
}
