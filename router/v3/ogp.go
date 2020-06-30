package v3

import (
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/service/ogp"
	"golang.org/x/sync/singleflight"
	"net/http"
	"net/url"
	"time"
)

// GetOgp GET /ogp?url={url}
func (h *Handlers) GetOgp(c echo.Context) error {
	var group singleflight.Group

	u, parseErr := url.Parse(c.QueryParam(consts.ParamURL))
	if parseErr != nil || len(u.Scheme) == 0 || len(u.Host) == 0 {
		return herror.BadRequest("invalid url")
	}

	result, herr, _ := group.Do(u.String(), func() (interface{}, error) {
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
				return cache.Content, nil
			}
			// キャッシュがヒットしたがネガティブキャッシュだった
			return nil, herror.NotFound(err)
		}

		og, meta, err := ogp.ParseMetaForURL(u)
		if err == ogp.ErrClient || err == ogp.ErrParse || err == ogp.ErrNetwork {
			// 4xxエラー、パースエラー、名前解決などのネットワークエラーの場合はネガティブキャッシュを作成
			if shouldUpdateCache {
				updateErr := h.Repo.UpdateOgpCache(cacheURL, nil)
				if updateErr != nil {
					return nil, herror.InternalServerError(updateErr)
				}
			} else if shouldCreateCache {
				_, createErr := h.Repo.CreateOgpCache(cacheURL, nil)
				if createErr != nil {
					return nil, herror.InternalServerError(createErr)
				}
			}
			// キャッシュヒットしなかったので1週間キャッシュ
			c.Response().Header().Set(consts.HeaderCacheControl, "public, max-age=604800")
			return nil, herror.NotFound(err)
		} else if err != nil {
			// このパスは5xxエラーなのでクライアント側キャッシュつけない
			return nil,herror.NotFound(err)
		}

		content := ogp.MergeDefaultPageMetaAndOpenGraph(og, meta)

		if shouldUpdateCache {
			err = h.Repo.UpdateOgpCache(cacheURL, content)
			if err != nil {
				return nil, herror.InternalServerError(err)
			}
		} else if shouldCreateCache {
			_, err = h.Repo.CreateOgpCache(cacheURL, content)
			if err != nil {
				return nil, herror.InternalServerError(err)
			}
		}

		// キャッシュヒットしなかったので1週間キャッシュ
		c.Response().Header().Set(consts.HeaderCacheControl, "public, max-age=604800")
		return content, nil
	})
	if herr != nil {
		return herr
	}
	return c.JSON(http.StatusOK, result)
}
