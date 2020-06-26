package v3

import (
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/service/ogp"
	"net/http"
	"net/url"
	"time"
)

// GetOgp GET /ogp?url={url}
func (h *Handlers) GetOgp(c echo.Context) error {
	u, err := url.Parse(c.QueryParam(consts.ParamUrl))
	if err != nil || len(u.Scheme) == 0 || len(u.Host) == 0 {
		return herror.BadRequest("invalid url")
	}

	cacheUrl := u.String()
	cache, err := h.Repo.GetOgpCache(cacheUrl)

	shouldUpdateCache := err == nil &&
		time.Now().After(cache.ExpiresAt)
	shouldCreateCache := err != nil

	if !shouldUpdateCache && !shouldCreateCache && err == nil {
		return c.JSON(http.StatusOK, cache.Content)
	}

	og, meta, err := ogp.ParseMetaForUrl(u)
	if err != nil {
		return herror.BadRequest(err)
	}

	content := ogp.MergeDefaultPageMetaAndOpenGraph(og, meta)

	if shouldUpdateCache {
		err = h.Repo.UpdateOgpCache(cacheUrl, *content)
		if err != nil {
			return herror.InternalServerError(err)
		}
	} else if shouldCreateCache {
		_, err = h.Repo.CreateOgpCache(cacheUrl, *content)
		if err != nil {
			return herror.InternalServerError(err)
		}
	}
	return c.JSON(http.StatusOK, content)
}
