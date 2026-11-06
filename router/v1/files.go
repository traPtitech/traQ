package v1

import (
	"net/http"

	"github.com/labstack/echo/v5"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension"
	"github.com/traPtitech/traQ/router/utils"
)

// getAccessibleFile URLの:fileIDに対応するFileを取得し、リクエストユーザーがアクセス可能かを確認する
//
// ファイルが存在しない場合は404、アクセスできない場合は403を返す。
func (h *Handlers) getAccessibleFile(c *echo.Context) (model.File, error) {
	userID := c.Get(consts.KeyUser).(model.UserInfo).GetID()
	return extension.GetAccessibleFile(c, h.FileManager, userID)
}

// GetFileByID GET /files/:fileID
func (h *Handlers) GetFileByID(c *echo.Context) error {
	f, err := h.getAccessibleFile(c)
	if err != nil {
		return err
	}
	return utils.ServeFile(c, f)
}

// GetMetaDataByFileID GET /files/:fileID/meta
func (h *Handlers) GetMetaDataByFileID(c *echo.Context) error {
	f, err := h.getAccessibleFile(c)
	if err != nil {
		return err
	}
	c.Response().Header().Set(consts.HeaderCacheControl, "private, max-age=86400") // 1日キャッシュ
	return c.JSON(http.StatusOK, formatFile(f))
}

// GetThumbnailByID GET /files/:fileID/thumbnail
func (h *Handlers) GetThumbnailByID(c *echo.Context) error {
	f, err := h.getAccessibleFile(c)
	if err != nil {
		return err
	}
	return utils.ServeFileThumbnail(c, f, h.Repo, h.Logger)
}
