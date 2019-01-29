package router

import (
	"github.com/labstack/echo"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/model"
	"net/http"
)

// GetPublicUserIcon GET /public/icon/{username}
func GetPublicUserIcon(c echo.Context) error {
	username := c.Param("username")
	if len(username) == 0 {
		return echo.NewHTTPError(http.StatusNotFound)
	}

	// ユーザー取得
	user, err := model.GetUserByName(username)
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	// ユーザーアイコンが設定されているかどうか
	if len(user.Icon) != 36 {
		return echo.NewHTTPError(http.StatusNotFound)
	}

	// ファイルメタ取得
	f, err := model.GetMetaFileDataByID(uuid.Must(uuid.FromString(user.Icon)))
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	// ファイルオープン
	if _, ok := c.QueryParams()["thumb"]; ok {
		if !f.HasThumbnail {
			return echo.NewHTTPError(http.StatusNotFound, "The specified file exists, but its thumbnail doesn't.")
		}
		r, err := f.OpenThumbnail()
		if err != nil {
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
		defer r.Close()
		c.Response().Header().Set(headerCacheControl, "public, max-age=3600") //1時間キャッシュ
		return c.Stream(http.StatusOK, mimeImagePNG, r)
	}

	r, err := f.Open()
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}
	defer r.Close()
	c.Response().Header().Set(headerCacheControl, "public, max-age=3600") //1時間キャッシュ
	return c.Stream(http.StatusOK, f.Mime, r)
}
