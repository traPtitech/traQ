package v1

import (
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	vd "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/rbac/permission"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/router/utils"
	"github.com/traPtitech/traQ/utils/hmac"
	"github.com/traPtitech/traQ/utils/message"
	"github.com/traPtitech/traQ/utils/optional"
	"io/ioutil"
	"net/http"
	"strings"
)

// GetWebhooks GET /webhooks
func (h *Handlers) GetWebhooks(c echo.Context) error {
	user := getRequestUser(c)

	var (
		list []model.Webhook
		err  error
	)
	if c.QueryParam("all") == "1" && h.RBAC.IsGranted(user.GetRole(), permission.AccessOthersWebhook) {
		list, err = h.Repo.GetAllWebhooks()
	} else {
		list, err = h.Repo.GetWebhooksByCreator(user.GetID())
	}
	if err != nil {
		return herror.InternalServerError(err)
	}

	return c.JSON(http.StatusOK, formatWebhooks(list))
}

// PostWebhooksRequest POST /webhooks リクエストボディ
type PostWebhooksRequest struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	ChannelID   uuid.UUID `json:"channelId"`
	Secret      string    `json:"secret"`
}

func (r PostWebhooksRequest) Validate() error {
	return vd.ValidateStruct(&r,
		vd.Field(&r.Name, vd.Required, vd.RuneLength(1, 32)),
		vd.Field(&r.Description, vd.Required, vd.RuneLength(1, 1000)),
	)
}

// PostWebhooks POST /webhooks
func (h *Handlers) PostWebhooks(c echo.Context) error {
	userID := getRequestUserID(c)

	var req PostWebhooksRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	w, err := h.Repo.CreateWebhook(req.Name, req.Description, req.ChannelID, userID, req.Secret)
	if err != nil {
		switch {
		case repository.IsArgError(err):
			return herror.BadRequest(err)
		default:
			return herror.InternalServerError(err)
		}
	}

	return c.JSON(http.StatusCreated, formatWebhook(w))
}

// GetWebhook GET /webhooks/:webhookID
func (h *Handlers) GetWebhook(c echo.Context) error {
	w := getWebhookFromContext(c)
	return c.JSON(http.StatusOK, formatWebhook(w))
}

// PatchWebhookRequest PATCH /webhooks/:webhookID リクエストボディ
type PatchWebhookRequest struct {
	Name        optional.String `json:"name"`
	Description optional.String `json:"description"`
	ChannelID   optional.UUID   `json:"channelId"`
	Secret      optional.String `json:"secret"`
	CreatorID   optional.UUID   `json:"creatorId"`
}

func (r PatchWebhookRequest) Validate() error {
	return vd.ValidateStruct(&r,
		vd.Field(&r.Name, vd.RuneLength(1, 32)),
		vd.Field(&r.Description, vd.RuneLength(1, 1000)),
	)
}

// PatchWebhook PATCH /webhooks/:webhookID
func (h *Handlers) PatchWebhook(c echo.Context) error {
	w := getWebhookFromContext(c)

	var req PatchWebhookRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	args := repository.UpdateWebhookArgs{
		Name:        req.Name,
		Description: req.Description,
		ChannelID:   req.ChannelID,
		Secret:      req.Secret,
		CreatorID:   req.CreatorID,
	}
	if err := h.Repo.UpdateWebhook(w.GetID(), args); err != nil {
		switch {
		case repository.IsArgError(err):
			return herror.BadRequest(err)
		default:
			return herror.InternalServerError(err)
		}
	}
	return c.NoContent(http.StatusNoContent)
}

// DeleteWebhook DELETE /webhooks/:webhookID
func (h *Handlers) DeleteWebhook(c echo.Context) error {
	w := getWebhookFromContext(c)

	if err := h.Repo.DeleteWebhook(w.GetID()); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}

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
	ch, err := h.Repo.GetChannel(channelID)
	if err != nil {
		switch err {
		case repository.ErrNotFound:
			return herror.BadRequest("invalid channel")
		default:
			return herror.InternalServerError(err)
		}
	}
	if !ch.IsPublic {
		return herror.BadRequest("invalid channel")
	}
	if ch.IsArchived() {
		return herror.BadRequest(fmt.Sprintf("channel has been archived"))
	}

	if c.QueryParam("embed") == "1" {
		body = []byte(message.NewReplacer(h.Repo).Replace(string(body)))
	}

	if _, err := h.Repo.CreateMessage(w.GetBotUserID(), ch.ID, string(body)); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}

// GetWebhookIcon GET /webhooks/:webhookID/icon
func (h *Handlers) GetWebhookIcon(c echo.Context) error {
	w := getWebhookFromContext(c)

	// ユーザー取得
	user, err := h.Repo.GetUser(w.GetBotUserID(), false)
	if err != nil {
		return herror.InternalServerError(err)
	}

	return utils.ServeUserIcon(c, h.Repo, user)
}

// PutWebhookIcon PUT /webhooks/:webhookID/icon
func (h *Handlers) PutWebhookIcon(c echo.Context) error {
	return utils.ChangeUserIcon(h.Imaging, c, h.Repo, getWebhookFromContext(c).GetBotUserID())
}

// GetWebhookMessages GET /webhooks/:webhookID/messages
func (h *Handlers) GetWebhookMessages(c echo.Context) error {
	w := getWebhookFromContext(c)

	var req messagesQuery
	if err := req.bind(c); err != nil {
		return err
	}

	return h.getMessages(c, req.convertU(w.GetBotUserID()), false)
}
