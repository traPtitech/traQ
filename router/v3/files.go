package v3

import (
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"net/http"
)

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
	return c.Stream(http.StatusOK, consts.MimeImagePNG, file)
}
