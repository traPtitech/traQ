package v3

import (
	"errors"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/service/ogp"
	"net/http"
	"net/url"
	"time"
)

type CacheHitState int

const (
	positiveCacheHit = iota
	negativeCacheHit
	positiveCacheCreated
	negativeCacheCreated
)

type cacheResult struct {
	CacheHit  CacheHitState
	Content   *model.Ogp
	ExpiresAt time.Time
}

// GetOgp GET /ogp?url={url}
func (h *Handlers) GetOgp(c echo.Context) error {
	u, parseErr := url.Parse(c.QueryParam(consts.ParamURL))
	if parseErr != nil || len(u.Scheme) == 0 || len(u.Host) == 0 {
		return herror.BadRequest("invalid url")
	}

	result, herr, _ := h.SFGroup.Do(u.String(), func() (interface{}, error) {
		cacheURL := u.String()
		cache, err := h.Repo.GetOgpCache(cacheURL)

		shouldUpdateCache := err == nil &&
			time.Now().After(cache.ExpiresAt)
		shouldCreateCache := err != nil

		if !shouldUpdateCache && !shouldCreateCache && err == nil {
			if cache.Valid {
				return cacheResult{
					CacheHit:  positiveCacheHit,
					Content:   &cache.Content,
					ExpiresAt: cache.ExpiresAt,
				}, nil
			}
			// キャッシュがヒットしたがネガティブキャッシュだった
			return cacheResult{
				CacheHit:  negativeCacheHit,
				Content:   nil,
				ExpiresAt: cache.ExpiresAt,
			}, nil
		}

		og, meta, err := ogp.ParseMetaForURL(u)
		if err == ogp.ErrClient || err == ogp.ErrParse || err == ogp.ErrNetwork || err == ogp.ErrContentTypeNotSupported {
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
			return cacheResult{
				CacheHit:  negativeCacheCreated,
				Content:   nil,
				ExpiresAt: ogp.GetCacheExpireDate(),
			}, nil
		} else if err != nil {
			// このパスは5xxエラーなのでクライアント側キャッシュつけない
			return nil, herror.NotFound(err)
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
		return cacheResult{
			CacheHit:  positiveCacheCreated,
			Content:   content,
			ExpiresAt: ogp.GetCacheExpireDate(),
		}, nil
	})
	if herr != nil {
		return herr
	}

	cr, ok := result.(cacheResult)
	if !ok {
		return herror.InternalServerError(errors.New("assertion failed"))
	}

	cacheDuration := int(time.Until(cr.ExpiresAt).Seconds())
	c.Response().Header().Set(consts.HeaderCacheControl, fmt.Sprintf("public, max-age=%d", cacheDuration))

	if cr.CacheHit == negativeCacheCreated || cr.CacheHit == negativeCacheHit {
		return herror.NotFound()
	}
	return c.JSON(http.StatusOK, cr.Content)
}
