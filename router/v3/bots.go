package v3

import (
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/router/extension/herror"
)

// GetBotIcon GET /bots/:webhookID/icon
func (h *Handlers) GetBotIcon(c echo.Context) error {
	w := getParamBot(c)

	// ユーザー取得
	user, err := h.Repo.GetUser(w.BotUserID)
	if err != nil {
		return herror.InternalServerError(err)
	}

	return serveUserIcon(c, h.Repo, user)
}
