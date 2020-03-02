package v3

import (
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	vd "github.com/go-ozzo/ozzo-validation"
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/rbac/permission"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/utils"
	"github.com/traPtitech/traQ/utils/message"
	"gopkg.in/guregu/null.v3"
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
	if isTrue(c.QueryParam("all")) && h.RBAC.IsGranted(user.Role, permission.AccessOthersWebhook) {
		list, err = h.Repo.GetAllWebhooks()
	} else {
		list, err = h.Repo.GetWebhooksByCreator(user.ID)
	}
	if err != nil {
		return herror.InternalServerError(err)
	}

	return c.JSON(http.StatusOK, formatWebhooks(list))
}

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
		vd.Field(&r.Secret, vd.RuneLength(0, 50)),
	)
}

// CreateWebhook POST /webhooks
func (h *Handlers) CreateWebhook(c echo.Context) error {
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
	w := getParamWebhook(c)
	return c.JSON(http.StatusOK, formatWebhook(w))
}

// PatchWebhookRequest PATCH /webhooks/:webhookID リクエストボディ
type PatchWebhookRequest struct {
	Name        null.String   `json:"name"`
	Description null.String   `json:"description"`
	ChannelID   uuid.NullUUID `json:"channelId"`
	Secret      null.String   `json:"secret"`
	OwnerID     uuid.NullUUID `json:"ownerId"`
}

func (r PatchWebhookRequest) Validate() error {
	return vd.ValidateStruct(&r,
		vd.Field(&r.Name, vd.RuneLength(1, 32)),
		vd.Field(&r.Description, vd.RuneLength(1, 1000)),
		vd.Field(&r.Secret, vd.RuneLength(0, 50)),
	)
}

// EditWebhook PATCH /webhooks/:webhookID
func (h *Handlers) EditWebhook(c echo.Context) error {
	w := getParamWebhook(c)

	var req PatchWebhookRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	args := repository.UpdateWebhookArgs{
		Name:        req.Name,
		Description: req.Description,
		ChannelID:   req.ChannelID,
		Secret:      req.Secret,
		CreatorID:   req.OwnerID,
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

// DeleteWebhook DELETE /webhooks/:webhookID
func (h *Handlers) DeleteWebhook(c echo.Context) error {
	w := getParamWebhook(c)

	if err := h.Repo.DeleteWebhook(w.GetID()); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}
