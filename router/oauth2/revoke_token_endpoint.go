package oauth2

import (
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/router/extension/herror"
	"net/http"
)

// RevokeTokenEndpointHandler トークン無効化エンドポイントのハンドラ
func (h *Config) RevokeTokenEndpointHandler(c echo.Context) error {
	var req struct {
		Token string `form:"token"`
	}
	if err := c.Bind(&req); err != nil {
		return herror.BadRequest(err)
	}

	if len(req.Token) == 0 {
		return c.NoContent(http.StatusOK)
	}

	if err := h.Repo.DeleteTokenByAccess(req.Token); err != nil {
		return herror.InternalServerError(err)
	}
	if err := h.Repo.DeleteTokenByRefresh(req.Token); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusOK)
}
