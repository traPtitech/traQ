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
	u, err := url.Parse(c.QueryParam(consts.ParamURL))
	if err != nil || len(u.Scheme) == 0 || len(u.Host) == 0 {
		return herror.BadRequest("invalid url")
	}

	cacheURL := u.String()
	cache, err := h.Repo.GetOgpCache(cacheURL)

	shouldUpdateCache := err == nil &&
		time.Now().After(cache.ExpiresAt)
	shouldCreateCache := err != nil

	if !shouldUpdateCache && !shouldCreateCache && err == nil {
		if cache.Valid {
			// キャッシュがヒットした
			return c.JSON(http.StatusOK, cache.Content)
		}
		// キャッシュがヒットしたがネガティブキャッシュだった
		return herror.BadRequest(err)
	}

	og, meta, err := ogp.ParseMetaForURL(u)
	println(err)
	if err == ogp.ErrClient {
		// 4xxエラーの場合はネガティブキャッシュを作成
		if shouldUpdateCache {
			updateErr := h.Repo.UpdateOgpCacheNegative(cacheURL)
			if updateErr != nil {
				return herror.InternalServerError(updateErr)
			}
		} else if shouldCreateCache {
			_, createErr := h.Repo.CreateOgpCacheNegative(cacheURL)
			if createErr != nil {
				return herror.InternalServerError(createErr)
			}
		}
		return herror.BadRequest(err)
	} else if err != nil {
		return herror.BadRequest(err)
	}

	content := ogp.MergeDefaultPageMetaAndOpenGraph(og, meta)

	if shouldUpdateCache {
		err = h.Repo.UpdateOgpCache(cacheURL, *content)
		if err != nil {
			return herror.InternalServerError(err)
		}
	} else if shouldCreateCache {
		_, err = h.Repo.CreateOgpCache(cacheURL, *content)
		if err != nil {
			return herror.InternalServerError(err)
		}
	}
	return c.JSON(http.StatusOK, content)
}
