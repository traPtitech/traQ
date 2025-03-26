package v1

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/utils"
)

// GetFileByID GET /files/:fileID
func (h *Handlers) GetFileByID(c echo.Context) error {
	return utils.ServeFile(c, getFileFromContext(c))
}

// GetMetaDataByFileID GET /files/:fileID/meta
func (h *Handlers) GetMetaDataByFileID(c echo.Context) error {
	meta := getFileFromContext(c)
	c.Response().Header().Set(consts.HeaderCacheControl, "private, max-age=86400") // 1日キャッシュ
	return c.JSON(http.StatusOK, formatFile(meta))
}

// GetThumbnailByID GET /files/:fileID/thumbnail
func (h *Handlers) GetThumbnailByID(c echo.Context) error {
	return utils.ServeFileThumbnail(c, getFileFromContext(c), h.Repo)
}
