package v1

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
	"gopkg.in/go-playground/webhooks.v5/github"
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
	if c.QueryParam("all") == "1" && h.RBAC.IsGranted(user.Role, permission.AccessOthersWebhook) {
		list, err = h.Repo.GetAllWebhooks()
	} else {
		list, err = h.Repo.GetWebhooksByCreator(user.ID)
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
		vd.Field(&r.Name, vd.Required, vd.Length(1, 32)),
		vd.Field(&r.Description, vd.Required, vd.Length(1, 1000)),
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
	Name        null.String   `json:"name"`
	Description null.String   `json:"description"`
	ChannelID   uuid.NullUUID `json:"channelId"`
	Secret      null.String   `json:"secret"`
	CreatorID   uuid.NullUUID `json:"creatorId"`
}

func (r PatchWebhookRequest) Validate() error {
	return vd.ValidateStruct(&r,
		vd.Field(&r.Name, vd.Length(1, 32)),
		vd.Field(&r.Description, vd.Length(1, 1000)),
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
		if subtle.ConstantTimeCompare(utils.CalcHMACSHA1(body, w.GetSecret()), sig) != 1 {
			return herror.Unauthorized()
		}
	}

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

	if c.QueryParam("embed") == "1" {
		body = []byte(message.NewReplacer(h.Repo).Replace(string(body)))
	}

	if _, err := h.Repo.CreateMessage(w.GetBotUserID(), channelID, string(body)); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}

// GetWebhookIcon GET /webhooks/:webhookID/icon
func (h *Handlers) GetWebhookIcon(c echo.Context) error {
	w := getWebhookFromContext(c)

	// ユーザー取得
	user, err := h.Repo.GetUser(w.GetBotUserID())
	if err != nil {
		return herror.InternalServerError(err)
	}

	return h.getUserIcon(c, user)
}

// PutWebhookIcon PUT /webhooks/:webhookID/icon
func (h *Handlers) PutWebhookIcon(c echo.Context) error {
	return h.putUserIcon(c, getWebhookFromContext(c).GetBotUserID())
}

// PostWebhookByGithub POST /webhooks/:webhookID/github
func (h *Handlers) PostWebhookByGithub(c echo.Context) error {
	w := getWebhookFromContext(c)

	switch c.Request().Header.Get(echo.HeaderContentType) {
	case echo.MIMEApplicationJSON, echo.MIMEApplicationJSONCharsetUTF8:
		break
	default:
		return echo.NewHTTPError(http.StatusUnsupportedMediaType)
	}

	ev := c.Request().Header.Get("X-GitHub-Event")
	if len(ev) == 0 {
		return herror.BadRequest("missing X-GitHub-Event header")
	}

	body, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		return herror.InternalServerError(err)
	}

	if len(w.GetSecret()) > 0 {
		sig, _ := hex.DecodeString(strings.TrimPrefix(c.Request().Header.Get("X-Hub-Signature"), "sha1="))
		if len(sig) == 0 {
			return herror.BadRequest("missing X-TRAQ-Signature header")
		}
		if subtle.ConstantTimeCompare(utils.CalcHMACSHA1(body, w.GetSecret()), sig) != 1 {
			return herror.Unauthorized()
		}
	}

	tmpl := h.webhookDefTmpls.Lookup(fmt.Sprintf("github_%s.tmpl", github.Event(ev)))
	if tmpl == nil {
		return c.NoContent(http.StatusNoContent)
	}

	var payload interface{}
	switch github.Event(ev) {
	case github.CommitCommentEvent:
		payload = &github.CommitCommentPayload{}
	case github.CreateEvent:
		payload = &github.CreatePayload{}
	case github.DeleteEvent:
		payload = &github.DeletePayload{}
	case github.DeploymentEvent:
		payload = &github.DeploymentPayload{}
	case github.DeploymentStatusEvent:
		payload = &github.DeploymentStatusPayload{}
	case github.ForkEvent:
		payload = &github.ForkPayload{}
	case github.GollumEvent:
		payload = &github.GollumPayload{}
	case github.InstallationEvent, github.IntegrationInstallationEvent:
		payload = &github.InstallationPayload{}
	case github.IssueCommentEvent:
		payload = &github.IssueCommentPayload{}
	case github.IssuesEvent:
		payload = &github.IssuesPayload{}
	case github.LabelEvent:
		payload = &github.LabelPayload{}
	case github.MemberEvent:
		payload = &github.MemberPayload{}
	case github.MembershipEvent:
		payload = &github.MembershipPayload{}
	case github.MilestoneEvent:
		payload = &github.MilestonePayload{}
	case github.OrganizationEvent:
		payload = &github.OrganizationPayload{}
	case github.OrgBlockEvent:
		payload = &github.OrgBlockPayload{}
	case github.PageBuildEvent:
		payload = &github.PageBuildPayload{}
	case github.PingEvent:
		payload = &github.PingPayload{}
	case github.ProjectCardEvent:
		payload = &github.ProjectCardPayload{}
	case github.ProjectColumnEvent:
		payload = &github.ProjectColumnPayload{}
	case github.ProjectEvent:
		payload = &github.ProjectPayload{}
	case github.PublicEvent:
		payload = &github.PublicPayload{}
	case github.PullRequestEvent:
		payload = &github.PullRequestPayload{}
	case github.PullRequestReviewEvent:
		payload = &github.PullRequestReviewPayload{}
	case github.PullRequestReviewCommentEvent:
		payload = &github.PullRequestReviewCommentPayload{}
	case github.PushEvent:
		payload = &github.PushPayload{}
	case github.ReleaseEvent:
		payload = &github.ReleasePayload{}
	case github.RepositoryEvent:
		payload = &github.RepositoryPayload{}
	case github.StatusEvent:
		payload = &github.StatusPayload{}
	case github.TeamEvent:
		payload = &github.TeamPayload{}
	case github.TeamAddEvent:
		payload = &github.TeamAddPayload{}
	case github.WatchEvent:
		payload = &github.WatchPayload{}
	default:
		return c.NoContent(http.StatusNoContent)
	}

	if err := json.Unmarshal(body, payload); err != nil {
		return herror.BadRequest(err)
	}

	messageBuf := &strings.Builder{}
	if err := tmpl.Execute(messageBuf, payload); err != nil {
		messageBuf.WriteString("Webhook Template Execution Failed\n")
		messageBuf.WriteString(err.Error())
	}
	if messageBuf.Len() > 0 {
		_, err := h.Repo.CreateMessage(w.GetBotUserID(), w.GetChannelID(), messageBuf.String())
		if err != nil {
			return herror.InternalServerError(err)
		}
	}

	return c.NoContent(http.StatusNoContent)
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
