package v3

import (
	"net/http"
	"net/url"

	"github.com/labstack/echo/v4"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
)

type CacheHitState int

// GetOgp GET /ogp?url={url}
func (h *Handlers) GetOgp(c echo.Context) error {
	u, parseErr := url.Parse(c.QueryParam(consts.ParamURL))
	if parseErr != nil || len(u.Scheme) == 0 || len(u.Host) == 0 {
		return herror.BadRequest("invalid url")
	}

	// キャッシュの削除に対応するために Cache-Control でのキャッシュはしない
	res, _, err := h.OGP.GetMeta(u)
	if err != nil {
		return herror.InternalServerError(err)
	}

	if res == nil {
		return c.JSON(http.StatusOK, model.Ogp{
			Type: "empty",
		})
	}
	return c.JSON(http.StatusOK, res)
}

// DeleteOgpCache DELETE /ogp/cache?url={url}
func (h *Handlers) DeleteOgpCache(c echo.Context) error {
	u, parseErr := url.Parse(c.QueryParam(consts.ParamURL))
	if parseErr != nil || len(u.Scheme) == 0 || len(u.Host) == 0 {
		return herror.BadRequest("invalid url")
	}

	err := h.OGP.DeleteCache(u)
	if err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}
