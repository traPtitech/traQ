package router

import (
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/gofrs/uuid"
	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/rbac/permission"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/utils"
	"go.uber.org/zap"
	"gopkg.in/go-playground/webhooks.v5/github"
	"gopkg.in/guregu/null.v3"
	"io/ioutil"
	"net/http"
	"strings"
	"text/template"
	"time"
)

type webhookForResponse struct {
	WebhookID   string    `json:"webhookId"`
	BotUserID   string    `json:"botUserId"`
	DisplayName string    `json:"displayName"`
	Description string    `json:"description"`
	Secure      bool      `json:"secure"`
	ChannelID   string    `json:"channelId"`
	CreatorID   string    `json:"creatorId"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

var (
	webhookDefTmpls = template.New("")
)

// LoadWebhookTemplate Webhookのテンプレートファイルを読み込みます
func LoadWebhookTemplate(pattern string) {
	webhookDefTmpls = template.Must(template.New("").Funcs(template.FuncMap{
		"replace": strings.Replace,
	}).ParseGlob(pattern))
}

// GetWebhooks GET /webhooks
func (h *Handlers) GetWebhooks(c echo.Context) error {
	user := getRequestUser(c)

	var (
		list []model.Webhook
		err  error
	)
	if c.QueryParam("all") == "1" && h.RBAC.IsGranted(user.ID, user.Role, permission.AccessOthersWebhook) {
		list, err = h.Repo.GetAllWebhooks()
	} else {
		list, err = h.Repo.GetWebhooksByCreator(user.ID)
	}
	if err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	res := make([]*webhookForResponse, len(list))
	for i, v := range list {
		res[i] = formatWebhook(v)
	}

	return c.JSON(http.StatusOK, res)
}

// PostWebhooks POST /webhooks
func (h *Handlers) PostWebhooks(c echo.Context) error {
	userID := getRequestUserID(c)

	req := struct {
		Name        string    `json:"name"        validate:"max=32,required"`
		Description string    `json:"description" validate:"required"`
		ChannelID   uuid.UUID `json:"channelId"`
		Secret      string    `json:"secret"`
	}{}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	ch, err := h.Repo.GetChannel(req.ChannelID)
	if err != nil {
		switch err {
		case repository.ErrNotFound:
			return echo.NewHTTPError(http.StatusBadRequest)
		default:
			h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}
	if !ch.IsPublic {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	w, err := h.Repo.CreateWebhook(req.Name, req.Description, req.ChannelID, userID, req.Secret)
	if err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusCreated, formatWebhook(w))
}

// GetWebhook GET /webhooks/:webhookID
func (h *Handlers) GetWebhook(c echo.Context) error {
	w := getWebhookFromContext(c)
	return c.JSON(http.StatusOK, formatWebhook(w))
}

// PatchWebhook PATCH /webhooks/:webhookID
func (h *Handlers) PatchWebhook(c echo.Context) error {
	w := getWebhookFromContext(c)

	req := struct {
		Name        null.String   `json:"name"        validate:"max=32"`
		Description null.String   `json:"description"`
		ChannelID   uuid.NullUUID `json:"channelId"`
		Secret      null.String   `json:"secret"`
	}{}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	if req.Name.Valid && len(req.Name.String) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "name is empty")
	}

	if req.ChannelID.Valid {
		ch, err := h.Repo.GetChannel(req.ChannelID.UUID)
		if err != nil {
			switch err {
			case repository.ErrNotFound:
				return echo.NewHTTPError(http.StatusBadRequest)
			default:
				h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
				return echo.NewHTTPError(http.StatusInternalServerError)
			}
		}
		if !ch.IsPublic {
			return echo.NewHTTPError(http.StatusBadRequest)
		}
	}

	args := repository.UpdateWebhookArgs{
		Name:        req.Name,
		Description: req.Description,
		ChannelID:   req.ChannelID,
		Secret:      req.Secret,
	}

	if err := h.Repo.UpdateWebhook(w.GetID(), args); err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError)
	}
	return c.NoContent(http.StatusNoContent)
}

// DeleteWebhook DELETE /webhooks/:webhookID
func (h *Handlers) DeleteWebhook(c echo.Context) error {
	w := getWebhookFromContext(c)

	if err := h.Repo.DeleteWebhook(w.GetID()); err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError)
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
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return echo.NewHTTPError(http.StatusBadRequest)
	}
	if len(body) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	if len(w.GetSecret()) > 0 {
		sig, _ := hex.DecodeString(c.Request().Header.Get(headerSignature))
		if len(sig) == 0 {
			return echo.NewHTTPError(http.StatusBadRequest, "missing X-TRAQ-Signature header")
		}
		if subtle.ConstantTimeCompare(utils.CalcHMACSHA1(body, w.GetSecret()), sig) != 1 {
			return echo.NewHTTPError(http.StatusUnauthorized)
		}
	}

	if cid := c.Request().Header.Get(headerChannelID); len(cid) > 0 {
		id := uuid.FromStringOrNil(cid)
		if id == uuid.Nil {
			return echo.NewHTTPError(http.StatusBadRequest)
		}
		ch, err := h.Repo.GetChannel(id)
		if err != nil {
			switch err {
			case repository.ErrNotFound:
				return echo.NewHTTPError(http.StatusBadRequest)
			default:
				h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
				return echo.NewHTTPError(http.StatusInternalServerError)
			}
		}
		if !ch.IsPublic {
			return echo.NewHTTPError(http.StatusBadRequest)
		}
		channelID = id
	}

	if _, err := h.Repo.CreateMessage(w.GetBotUserID(), channelID, string(body)); err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}

// PutWebhookIcon PUT /webhooks/:webhookID/icon
func (h *Handlers) PutWebhookIcon(c echo.Context) error {
	w := getWebhookFromContext(c)

	// file確認
	uploadedFile, err := c.FormFile("file")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	iconID, err := h.processMultipartFormIconUpload(c, uploadedFile)
	if err != nil {
		return err
	}

	// アイコン変更
	if err := h.Repo.ChangeUserIcon(w.GetBotUserID(), iconID); err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
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
		return echo.NewHTTPError(http.StatusBadRequest, "missing X-GitHub-Event header")
	}

	body, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	if len(w.GetSecret()) > 0 {
		sig, _ := hex.DecodeString(strings.TrimPrefix(c.Request().Header.Get("X-Hub-Signature"), "sha1="))
		if len(sig) == 0 {
			return echo.NewHTTPError(http.StatusBadRequest, "missing X-Hub-Signature header")
		}
		if subtle.ConstantTimeCompare(utils.CalcHMACSHA1(body, w.GetSecret()), sig) != 1 {
			return echo.NewHTTPError(http.StatusUnauthorized)
		}
	}

	tmpl := webhookDefTmpls.Lookup(fmt.Sprintf("github_%s.tmpl", github.Event(ev)))
	if tmpl == nil {
		return c.NoContent(http.StatusNoContent)
	}

	var payload interface{}
	switch github.Event(ev) {
	case github.CommitCommentEvent:
		var d github.CommitCommentPayload
		err = json.Unmarshal(body, &d)
		payload = d
	case github.CreateEvent:
		var d github.CreatePayload
		err = json.Unmarshal(body, &d)
		payload = d
	case github.DeleteEvent:
		var d github.DeletePayload
		err = json.Unmarshal(body, &d)
		payload = d
	case github.DeploymentEvent:
		var d github.DeploymentPayload
		err = json.Unmarshal(body, &d)
		payload = d
	case github.DeploymentStatusEvent:
		var d github.DeploymentStatusPayload
		err = json.Unmarshal(body, &d)
		payload = d
	case github.ForkEvent:
		var d github.ForkPayload
		err = json.Unmarshal(body, &d)
		payload = d
	case github.GollumEvent:
		var d github.GollumPayload
		err = json.Unmarshal(body, &d)
		payload = d
	case github.InstallationEvent, github.IntegrationInstallationEvent:
		var d github.InstallationPayload
		err = json.Unmarshal(body, &d)
		payload = d
	case github.IssueCommentEvent:
		var d github.IssueCommentPayload
		err = json.Unmarshal(body, &d)
		payload = d
	case github.IssuesEvent:
		var d github.IssuesPayload
		err = json.Unmarshal(body, &d)
		payload = d
	case github.LabelEvent:
		var d github.LabelPayload
		err = json.Unmarshal(body, &d)
		payload = d
	case github.MemberEvent:
		var d github.MemberPayload
		err = json.Unmarshal(body, &d)
		payload = d
	case github.MembershipEvent:
		var d github.MembershipPayload
		err = json.Unmarshal(body, &d)
		payload = d
	case github.MilestoneEvent:
		var d github.MilestonePayload
		err = json.Unmarshal(body, &d)
		payload = d
	case github.OrganizationEvent:
		var d github.OrganizationPayload
		err = json.Unmarshal(body, &d)
		payload = d
	case github.OrgBlockEvent:
		var d github.OrgBlockPayload
		err = json.Unmarshal(body, &d)
		payload = d
	case github.PageBuildEvent:
		var d github.PageBuildPayload
		err = json.Unmarshal(body, &d)
		payload = d
	case github.PingEvent:
		var d github.PingPayload
		err = json.Unmarshal(body, &d)
		payload = d
	case github.ProjectCardEvent:
		var d github.ProjectCardPayload
		err = json.Unmarshal(body, &d)
		payload = d
	case github.ProjectColumnEvent:
		var d github.ProjectColumnPayload
		err = json.Unmarshal(body, &d)
		payload = d
	case github.ProjectEvent:
		var d github.ProjectPayload
		err = json.Unmarshal(body, &d)
		payload = d
	case github.PublicEvent:
		var d github.PublicPayload
		err = json.Unmarshal(body, &d)
		payload = d
	case github.PullRequestEvent:
		var d github.PullRequestPayload
		err = json.Unmarshal(body, &d)
		payload = d
	case github.PullRequestReviewEvent:
		var d github.PullRequestReviewPayload
		err = json.Unmarshal(body, &d)
		payload = d
	case github.PullRequestReviewCommentEvent:
		var d github.PullRequestReviewCommentPayload
		err = json.Unmarshal(body, &d)
		payload = d
	case github.PushEvent:
		var d github.PushPayload
		err = json.Unmarshal(body, &d)
		payload = d
	case github.ReleaseEvent:
		var d github.ReleasePayload
		err = json.Unmarshal(body, &d)
		payload = d
	case github.RepositoryEvent:
		var d github.RepositoryPayload
		err = json.Unmarshal(body, &d)
		payload = d
	case github.StatusEvent:
		var d github.StatusPayload
		err = json.Unmarshal(body, &d)
		payload = d
	case github.TeamEvent:
		var d github.TeamPayload
		err = json.Unmarshal(body, &d)
		payload = d
	case github.TeamAddEvent:
		var d github.TeamAddPayload
		err = json.Unmarshal(body, &d)
		payload = d
	case github.WatchEvent:
		var d github.WatchPayload
		err = json.Unmarshal(body, &d)
		payload = d
	}

	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	if payload == nil {
		return c.NoContent(http.StatusNoContent)
	}

	messageBuf := &strings.Builder{}
	if err := tmpl.Execute(messageBuf, payload); err != nil {
		messageBuf.WriteString("Webhook Template Execution Failed\n")
		messageBuf.WriteString(err.Error())
	}
	if messageBuf.Len() > 0 {
		_, err := h.Repo.CreateMessage(w.GetBotUserID(), w.GetChannelID(), messageBuf.String())
		if err != nil {
			h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	return c.NoContent(http.StatusNoContent)
}

// GetWebhookMessages GET /webhooks/:webhookID/messages
func (h *Handlers) GetWebhookMessages(c echo.Context) error {
	w := getWebhookFromContext(c)

	req := struct {
		Limit  int `query:"limit"  validate:"min=0"`
		Offset int `query:"offset" validate:"min=0"`
	}{}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}
	if req.Limit > 50 || req.Limit == 0 {
		req.Limit = 50
	}

	messages, err := h.Repo.GetMessagesByUserID(w.GetBotUserID(), req.Limit, req.Offset)
	if err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	res := make([]*MessageForResponse, 0, req.Limit)
	for _, message := range messages {
		res = append(res, h.formatMessage(message))
	}

	return c.JSON(http.StatusOK, res)
}

func formatWebhook(w model.Webhook) *webhookForResponse {
	return &webhookForResponse{
		WebhookID:   w.GetID().String(),
		BotUserID:   w.GetBotUserID().String(),
		DisplayName: w.GetName(),
		Description: w.GetDescription(),
		Secure:      len(w.GetSecret()) > 0,
		ChannelID:   w.GetChannelID().String(),
		CreatorID:   w.GetCreatorID().String(),
		CreatedAt:   w.GetCreatedAt(),
		UpdatedAt:   w.GetUpdatedAt(),
	}
}
