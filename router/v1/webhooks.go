package v1

import (
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"

	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/service/channel"
	"github.com/traPtitech/traQ/service/message"
	"github.com/traPtitech/traQ/utils/hmac"
)

// PostWebhook POST /webhooks/:webhookID
func (h *Handlers) PostWebhook(c echo.Context) error {
	w := getWebhookFromContext(c)
	channelID := w.GetChannelID()

	switch strings.ToLower(c.Request().Header.Get(echo.HeaderContentType)) {
	case echo.MIMETextPlain, strings.ToLower(echo.MIMETextPlainCharsetUTF8):
		break
	default:
		return echo.NewHTTPError(http.StatusUnsupportedMediaType)
	}

	body, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		return herror.InternalServerError(err)
	}
	if len(body) == 0 {
		return herror.BadRequest("empty body")
	}

	if len(w.GetSecret()) > 0 {
		sig, _ := hex.DecodeString(c.Request().Header.Get(consts.HeaderSignature))
		if len(sig) == 0 {
			return herror.BadRequest("missing X-TRAQ-Signature header")
		}
		if subtle.ConstantTimeCompare(hmac.SHA1(body, w.GetSecret()), sig) != 1 {
			return herror.Unauthorized()
		}
	}

	// 投稿先チャンネル変更
	if cid := c.Request().Header.Get(consts.HeaderChannelID); len(cid) > 0 {
		id, err := uuid.FromString(cid)
		if err != nil {
			return herror.BadRequest(fmt.Sprintf("invalid %s header", consts.HeaderChannelID))
		}
		channelID = id
	}

	// 投稿先チャンネル確認
	ch, err := h.ChannelManager.GetChannel(channelID)
	if err != nil {
		switch err {
		case channel.ErrChannelNotFound:
			return herror.BadRequest("invalid channel")
		default:
			return herror.InternalServerError(err)
		}
	}
	if !ch.IsPublic {
		return herror.BadRequest("invalid channel")
	}

	if c.QueryParam("embed") == "1" {
		body = []byte(h.Replacer.Replace(string(body)))
	}

	if _, err := h.MessageManager.Create(ch.ID, w.GetBotUserID(), string(body)); err != nil {
		switch err {
		case message.ErrChannelArchived:
			return herror.BadRequest("the channel has been archived")
		default:
			return herror.InternalServerError(err)
		}
	}

	return c.NoContent(http.StatusNoContent)
}
