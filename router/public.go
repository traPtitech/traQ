package router

import (
	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/repository"
	"net/http"
	"strconv"
)

// GetPublicUserIcon GET /public/icon/{username}
func (h *Handlers) GetPublicUserIcon(c echo.Context) error {
	username := c.Param("username")
	if len(username) == 0 {
		return echo.NewHTTPError(http.StatusNotFound)
	}

	// ユーザー取得
	user, err := h.Repo.GetUserByName(username)
	if err != nil {
		switch err {
		case repository.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	// ファイルオープン
	if _, ok := c.QueryParams()["thumb"]; ok {
		_, r, err := h.Repo.OpenThumbnailFile(user.Icon)
		if err != nil {
			switch err {
			case repository.ErrNotFound:
				return echo.NewHTTPError(http.StatusNotFound)
			default:
				c.Logger().Error(err)
				return echo.NewHTTPError(http.StatusInternalServerError)
			}
		}
		defer r.Close()
		c.Response().Header().Set(headerCacheControl, "public, max-age=3600") //1時間キャッシュ
		return c.Stream(http.StatusOK, mimeImagePNG, r)
	}

	f, r, err := h.Repo.OpenFile(user.Icon)
	if err != nil {
		switch err {
		case repository.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}
	defer r.Close()
	c.Response().Header().Set(echo.HeaderContentLength, strconv.FormatInt(f.Size, 10))
	c.Response().Header().Set(headerCacheControl, "public, max-age=3600") //1時間キャッシュ
	return c.Stream(http.StatusOK, f.Mime, r)
}
