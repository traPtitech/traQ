package v3

import (
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/utils"
	"github.com/traPtitech/traQ/utils/message"
	"io/ioutil"
	"net/http"
	"strings"
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

// PostWebhook POST /webhooks/:webhookID
func (h *Handlers) PostWebhook(c echo.Context) error {
	w := getParamWebhook(c)
	channelID := w.GetChannelID()

	// text/plainのみ受け付ける
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

	// Webhookシークレット確認
	if len(w.GetSecret()) > 0 {
		sig, _ := hex.DecodeString(c.Request().Header.Get(consts.HeaderSignature))
		if len(sig) == 0 {
			return herror.BadRequest("missing X-TRAQ-Signature header")
		}
		if subtle.ConstantTimeCompare(utils.CalcHMACSHA1(body, w.GetSecret()), sig) != 1 {
			return herror.BadRequest("X-TRAQ-Signature is wrong")
		}
	}

	// 投稿先チャンネル変更
	if cid := c.Request().Header.Get(consts.HeaderChannelID); len(cid) > 0 {
		id := uuid.FromStringOrNil(cid)
		ch, err := h.Repo.GetChannel(id)
		if err != nil {
			switch err {
			case repository.ErrNotFound:
				return herror.BadRequest(fmt.Sprintf("invalid %s header", consts.HeaderChannelID))
			default:
				return herror.InternalServerError(err)
			}
		}
		if !ch.IsPublic {
			return herror.BadRequest("invalid channel")
		}
		channelID = id
	}

	// 埋め込み変換
	if isTrue(c.QueryParam("embed")) {
		body = []byte(message.NewReplacer(h.Repo).Replace(string(body)))
	}

	// メッセージ投稿
	if _, err := h.Repo.CreateMessage(w.GetBotUserID(), channelID, string(body)); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}
