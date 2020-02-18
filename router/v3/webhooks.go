package v3

import (
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/router/extension/herror"
)

// GetWebhookIcon GET /webhooks/:webhookID/icon
func (h *Handlers) GetWebhookIcon(c echo.Context) error {
	w := getParamWebhook(c)

	// ユーザー取得
	user, err := h.Repo.GetUser(w.GetBotUserID())
	if err != nil {
		return herror.InternalServerError(err)
	}

	return serveUserIcon(c, h.Repo, user)
}

// ChangeWebhookIcon PUT /webhooks/:webhookID/icon
func (h *Handlers) ChangeWebhookIcon(c echo.Context) error {
	return changeUserIcon(c, h.Repo, getParamWebhook(c).GetBotUserID())
}
