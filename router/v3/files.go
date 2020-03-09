package v3

import (
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"net/http"
	"strconv"
)

// GetFileMeta GET /files/:fileID/meta
func (h *Handlers) GetFileMeta(c echo.Context) error {
	return c.JSON(http.StatusOK, formatFileInfo(getParamFile(c)))
}

// GetThumbnailImage GET /files/:fileID/thumbnail
func (h *Handlers) GetThumbnailImage(c echo.Context) error {
	meta := getParamFile(c)

	if !meta.HasThumbnail {
		return herror.NotFound()
	}

	c.Response().Header().Set(consts.HeaderFileMetaType, meta.Type)
	c.Response().Header().Set(consts.HeaderCacheFile, "true")

	// 直接アクセスURLが発行できる場合は、そっちにリダイレクト
	url, _ := h.Repo.GetFS().GenerateAccessURL(meta.GetThumbKey(), model.FileTypeThumbnail)
	if len(url) > 0 {
		return c.Redirect(http.StatusFound, url)
	}

	file, err := h.Repo.GetFS().OpenFileByKey(meta.GetThumbKey(), model.FileTypeThumbnail)
	if err != nil {
		return herror.InternalServerError(err)
	}
	defer file.Close()

	c.Response().Header().Set(consts.HeaderCacheControl, "private, max-age=31536000") // 1年間キャッシュ
	return c.Stream(http.StatusOK, meta.ThumbnailMime.String, file)
}

// GetFile GET /files/:fileID
func (h *Handlers) GetFile(c echo.Context) error {
	meta := getParamFile(c)

	c.Response().Header().Set(consts.HeaderFileMetaType, meta.Type)
	switch meta.Type {
	case model.FileTypeStamp, model.FileTypeIcon:
		c.Response().Header().Set(consts.HeaderCacheFile, "true")
	}

	// 直接アクセスURLが発行できる場合は、そっちにリダイレクト
	url, _ := h.Repo.GetFS().GenerateAccessURL(meta.GetKey(), meta.Type)
	if len(url) > 0 {
		return c.Redirect(http.StatusFound, url)
	}

	file, err := h.Repo.GetFS().OpenFileByKey(meta.GetKey(), meta.Type)
	if err != nil {
		return herror.InternalServerError(err)
	}
	defer file.Close()

	c.Response().Header().Set(echo.HeaderContentType, meta.Mime)
	c.Response().Header().Set(consts.HeaderETag, strconv.Quote(meta.Hash))
	c.Response().Header().Set(consts.HeaderCacheControl, "private, max-age=31536000") // 1年間キャッシュ
	if isTrue(c.QueryParam("dl")) {
		c.Response().Header().Set(echo.HeaderContentDisposition, fmt.Sprintf("attachment; filename=%s", meta.Name))
	}

	http.ServeContent(c.Response(), c.Request(), meta.Name, meta.CreatedAt, file)
	return nil
}
