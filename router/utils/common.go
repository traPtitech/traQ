package utils

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/router/session"
	"github.com/traPtitech/traQ/service/file"
	imaging2 "github.com/traPtitech/traQ/service/imaging"
	"github.com/traPtitech/traQ/utils/optional"
	"github.com/traPtitech/traQ/utils/storage"
)

// ChangeUserIcon userIDのユーザーのアイコン画像を変更する
func ChangeUserIcon(p imaging2.Processor, c echo.Context, repo repository.Repository, m file.Manager, userID uuid.UUID) error {
	iconID, err := SaveUploadIconImage(p, c, m, "file")
	if err != nil {
		return err
	}

	// アイコン変更
	if err := repo.UpdateUser(userID, repository.UpdateUserArgs{IconFileID: optional.From(iconID)}); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}

// ServeUserIcon userのアイコン画像ファイルをレスポンスとして返す
func ServeUserIcon(c echo.Context, fm file.Manager, user model.UserInfo) error {
	// ファイルメタ取得
	meta, err := fm.Get(user.GetIconFileID())
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
func ChangeUserPassword(c echo.Context, repo repository.Repository, seStore session.Store, userID uuid.UUID, newPassword string) error {
	if err := repo.UpdateUser(userID, repository.UpdateUserArgs{Password: optional.From(newPassword)}); err != nil {
		return herror.InternalServerError(err)
	}

	// ユーザーの全セッションを破棄(強制ログアウト)
	_ = seStore.RevokeSessionsByUserID(userID)
	return c.NoContent(http.StatusNoContent)
}

// ServeFileThumbnail metaのファイルのサムネイルをレスポンスとして返す
func ServeFileThumbnail(c echo.Context, meta model.File, repo repository.Repository, logger *zap.Logger) error {
	typeStr := c.QueryParam("type")
	if len(typeStr) == 0 {
		typeStr = "image"
	}
	thumbnailType, err := model.ThumbnailTypeFromString(typeStr)
	if err != nil {
		return herror.BadRequest(err)
	}

	hasThumb, thumb := meta.GetThumbnail(thumbnailType)
	if !hasThumb {
		return herror.NotFound()
	}

	file, err := meta.OpenThumbnail(thumbnailType)
	if err != nil {
		// Check if the error is because the file doesn't exist in S3
		if errors.Is(err, storage.ErrFileNotFound) {
			// サムネイルが実際には存在しないのでDBの情報を更新する
			fileID := meta.GetID()
			err := repo.DeleteFileThumbnail(fileID, thumbnailType)
			if err != nil {
				logger.Warn("failed to delete thumbnail from database", zap.Error(err))
				return herror.InternalServerError(err)
			}
			logger.Info("removed non-existent thumbnail from database", zap.String("fileID", fileID.String()), zap.String("thumbnailType", thumbnailType.String()))
			return herror.NotFound()
		}
		return herror.InternalServerError(err)
	}
	defer file.Close()

	c.Response().Header().Set(consts.HeaderFileMetaType, meta.GetFileType().String())
	c.Response().Header().Set(consts.HeaderCacheFile, "true")
	c.Response().Header().Set(consts.HeaderCacheControl, "private, max-age=31536000") // 1年間キャッシュ
	return c.Stream(http.StatusOK, thumb.Mime, file)
}

// ServeFile metaのファイル本体をレスポンスとして返す
func ServeFile(c echo.Context, meta model.File) error {
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
		c.Response().Header().Set(echo.HeaderContentDisposition, fmt.Sprintf("attachment; filename*=UTF-8''%s", url.PathEscape(meta.GetFileName())))
	}
	c.Response().Header().Set(consts.HeaderFileMetaType, meta.GetFileType().String())
	switch meta.GetFileType() {
	case model.FileTypeStamp, model.FileTypeIcon:
		c.Response().Header().Set(consts.HeaderCacheFile, "true")
	}

	http.ServeContent(c.Response(), c.Request(), meta.GetFileName(), meta.GetCreatedAt(), file)
	return nil
}
