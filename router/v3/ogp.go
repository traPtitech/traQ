package v3

import (
	"fmt"
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
		// キャッシュがヒットしたので残りの有効時間までクライアント側にキャッシュ
		cacheDuration := int(time.Until(cache.ExpiresAt).Seconds())
		c.Response().Header().Set(consts.HeaderCacheControl, fmt.Sprintf("public, max-age=%d", cacheDuration))

		if cache.Valid {
			return c.JSON(http.StatusOK, cache.Content)
		}
		// キャッシュがヒットしたがネガティブキャッシュだった
		return herror.NotFound(err)
	}

	og, meta, err := ogp.ParseMetaForURL(u)
	if err == ogp.ErrClient {
		// 4xxエラーの場合はネガティブキャッシュを作成
		if shouldUpdateCache {
			updateErr := h.Repo.UpdateOgpCache(cacheURL, nil)
			if updateErr != nil {
				return herror.InternalServerError(updateErr)
			}
		} else if shouldCreateCache {
			_, createErr := h.Repo.CreateOgpCache(cacheURL, nil)
			if createErr != nil {
				return herror.InternalServerError(createErr)
			}
		}
		// キャッシュヒットしなかったので1週間キャッシュ
		c.Response().Header().Set(consts.HeaderCacheControl, "public, max-age=604800")
		return herror.NotFound(err)
	} else if err != nil {
		// このパスは5xxエラーなのでクライアント側キャッシュつけない
		return herror.NotFound(err)
	}

	content := ogp.MergeDefaultPageMetaAndOpenGraph(og, meta)

	if shouldUpdateCache {
		err = h.Repo.UpdateOgpCache(cacheURL, content)
		if err != nil {
			return herror.InternalServerError(err)
		}
	} else if shouldCreateCache {
		_, err = h.Repo.CreateOgpCache(cacheURL, content)
		if err != nil {
			return herror.InternalServerError(err)
		}
	}

	// キャッシュヒットしなかったので1週間キャッシュ
	c.Response().Header().Set(consts.HeaderCacheControl, "public, max-age=604800")
	return c.JSON(http.StatusOK, content)
}
