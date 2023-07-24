package v3

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/service/file"
	"github.com/traPtitech/traQ/utils/jwt"
)

// GetVersion GET /version
func (h *Handlers) GetVersion(c echo.Context) error {
	extLogins := make([]string, 0, len(h.EnabledExternalAccountProviders))
	for p := range h.EnabledExternalAccountProviders {
		extLogins = append(extLogins, p)
	}
	return c.JSON(http.StatusOK, echo.Map{
		"version":  h.Version,
		"revision": h.Revision,
		"flags": echo.Map{
			"externalLogin": extLogins,
			"signUpAllowed": h.Config.AllowSignUp,
		},
	})
}

// GetJWKS GET /jwks
func (h *Handlers) GetJWKS(c echo.Context) error {
	jwks, err := jwt.JWKSet(c.Request().Context())
	if err != nil {
		return herror.InternalServerError(err)
	}
	return c.JSON(http.StatusOK, jwks)
}

// GetPublicUserIcon GET /public/icon/{username}
func (h *Handlers) GetPublicUserIcon(c echo.Context) error {
	username := c.Param("username")

	// ユーザー取得
	user, err := h.Repo.GetUserByName(username, false)
	if err != nil {
		switch err {
		case repository.ErrNotFound:
			return herror.NotFound()
		default:
			return herror.InternalServerError(err)
		}
	}

	// ファイルメタ取得
	meta, err := h.FileManager.Get(user.GetIconFileID())
	if err != nil {
		switch err {
		case file.ErrNotFound:
			return herror.NotFound()
		default:
			return herror.InternalServerError(err)
		}
	}

	// ファイルオープン
	file, err := meta.Open()
	if err != nil {
		return herror.InternalServerError(err)
	}
	defer file.Close()

	c.Response().Header().Set(echo.HeaderContentType, meta.GetMIMEType())
	c.Response().Header().Set(consts.HeaderETag, strconv.Quote(meta.GetMD5Hash()))
	c.Response().Header().Set(consts.HeaderCacheControl, "public, max-age=3600") // 1時間キャッシュ
	http.ServeContent(c.Response(), c.Request(), meta.GetFileName(), meta.GetCreatedAt(), file)
	return nil
}
