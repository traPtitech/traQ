package router

import (
	"encoding/json"
	"fmt"
	"github.com/labstack/echo"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"gopkg.in/go-playground/webhooks.v3/github"
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
func GetWebhooks(c echo.Context) error {
	userID := getRequestUserID(c)

	list, err := model.GetWebhooksByCreator(userID)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	res := make([]*webhookForResponse, len(list))
	for i, v := range list {
		res[i] = formatWebhook(v)
	}

	return c.JSON(http.StatusOK, res)
}

// PostWebhooks POST /webhooks
func PostWebhooks(c echo.Context) error {
	userID := getRequestUserID(c)

	req := struct {
		Name        string `json:"name"        validate:"max=32,required"`
		Description string `json:"description" validate:"required"`
		ChannelID   string `json:"channelId"   validate:"uuid,required"`
	}{}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}
	channelID := uuid.FromStringOrNil(req.ChannelID)

	if ok, err := model.IsChannelAccessibleToUser(userID, channelID); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	} else if !ok {
		return echo.NewHTTPError(http.StatusNotFound)
	}

	iconID, err := model.GenerateIcon(req.Name)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	w, err := model.CreateWebhook(req.Name, req.Description, channelID, userID, uuid.Must(uuid.FromString(iconID)))
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	go event.Emit(event.UserJoined, &event.UserEvent{ID: w.GetID()})
	return c.JSON(http.StatusCreated, formatWebhook(w))
}

// GetWebhook GET /webhooks/:webhookID
func GetWebhook(c echo.Context) error {
	webhookID := getRequestParamAsUUID(c, paramWebhookID)
	w, err := getWebhook(c, webhookID, true)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, formatWebhook(w))
}

// PatchWebhook PATCH /webhooks/:webhookID
func PatchWebhook(c echo.Context) error {
	userID := getRequestUserID(c)
	webhookID := getRequestParamAsUUID(c, paramWebhookID)

	w, err := getWebhook(c, webhookID, true)
	if err != nil {
		return err
	}

	req := struct {
		Name        string `json:"name"        validate:"max=32"`
		Description string `json:"description"`
		ChannelID   string `json:"channelId"`
	}{}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	if len(req.ChannelID) == 36 {
		cid := uuid.FromStringOrNil(req.ChannelID)
		if cid == uuid.Nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid channelId")
		}

		if ok, err := model.IsChannelAccessibleToUser(userID, cid); err != nil {
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		} else if !ok {
			return echo.NewHTTPError(http.StatusNotFound)
		}

		if err := model.UpdateWebhook(w, nil, nil, cid); err != nil {
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	if len(req.Name) > 0 {
		if err := model.UpdateWebhook(w, &req.Name, nil, uuid.Nil); err != nil {
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}

		go event.Emit(event.UserUpdated, &event.UserEvent{ID: w.GetBotUserID()})
	}

	if len(req.Description) > 0 {
		if err := model.UpdateWebhook(w, nil, &req.Description, uuid.Nil); err != nil {
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	return c.NoContent(http.StatusNoContent)
}

// DeleteWebhook DELETE /webhooks/:webhookID
func DeleteWebhook(c echo.Context) error {
	webhookID := getRequestParamAsUUID(c, paramWebhookID)
	w, err := getWebhook(c, webhookID, true)
	if err != nil {
		return err
	}

	if err := model.DeleteWebhook(w.GetID()); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}

// PostWebhook POST /webhooks/:webhookID
func PostWebhook(c echo.Context) error {
	webhookID := getRequestParamAsUUID(c, paramWebhookID)

	w, err := getWebhook(c, webhookID, false)
	if err != nil {
		return err
	}

	text := ""
	channelID := w.GetChannelID()
	switch c.Request().Header.Get(echo.HeaderContentType) {
	case echo.MIMETextPlain, echo.MIMETextPlainCharsetUTF8:
		if b, err := ioutil.ReadAll(c.Request().Body); err == nil {
			text = string(b)
		}
		if len(text) == 0 {
			return echo.NewHTTPError(http.StatusBadRequest)
		}

	case echo.MIMEApplicationJSON, echo.MIMEApplicationForm, echo.MIMEApplicationJSONCharsetUTF8:
		req := struct {
			Text      string `json:"text"      form:"text"      validate:"required"`
			ChannelID string `json:"channelId" form:"channelId"`
		}{}
		if err := bindAndValidate(c, &req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err)
		}
		if len(req.ChannelID) == 36 {
			channelID = uuid.FromStringOrNil(req.ChannelID)
			if ok, err := model.IsChannelAccessibleToUser(w.GetBotUserID(), channelID); err != nil {
				c.Logger().Error(err)
				return echo.NewHTTPError(http.StatusInternalServerError)
			} else if !ok {
				return echo.NewHTTPError(http.StatusBadRequest)
			}
		}
		text = req.Text

	default:
		return echo.NewHTTPError(http.StatusUnsupportedMediaType)
	}

	m, err := model.CreateMessage(w.GetBotUserID(), channelID, text)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	go event.Emit(event.MessageCreated, &event.MessageCreatedEvent{Message: *m})
	return c.NoContent(http.StatusNoContent)
}

// PutWebhookIcon PUT /webhooks/:webhookID/icon
func PutWebhookIcon(c echo.Context) error {
	webhookID := getRequestParamAsUUID(c, paramWebhookID)

	w, err := getWebhook(c, webhookID, true)
	if err != nil {
		return err
	}

	// file確認
	uploadedFile, err := c.FormFile("file")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	iconID, err := processMultipartFormIconUpload(c, uploadedFile)
	if err != nil {
		return err
	}

	// アイコン変更
	if err := model.ChangeUserIcon(w.GetBotUserID(), iconID); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	go event.Emit(event.UserIconUpdated, &event.UserEvent{ID: w.GetBotUserID()})
	return c.NoContent(http.StatusOK)
}

// PostWebhookByGithub POST /webhooks/:webhookID/github
func PostWebhookByGithub(c echo.Context) error {
	webhookID := getRequestParamAsUUID(c, paramWebhookID)

	w, err := getWebhook(c, webhookID, false)
	if err != nil {
		return err
	}

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

	githubEvent := github.Event(ev)
	tmpl := webhookDefTmpls.Lookup(fmt.Sprintf("github_%s.tmpl", githubEvent))
	if tmpl == nil {
		return c.NoContent(http.StatusNoContent)
	}

	var payload interface{}
	switch githubEvent {
	case github.CommitCommentEvent:
		var d github.CommitCommentPayload
		if err := json.NewDecoder(c.Request().Body).Decode(&d); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest)
		}
		payload = d
	case github.CreateEvent:
		var d github.CreatePayload
		if err := json.NewDecoder(c.Request().Body).Decode(&d); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest)
		}
		payload = d
	case github.DeleteEvent:
		var d github.DeletePayload
		if err := json.NewDecoder(c.Request().Body).Decode(&d); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest)
		}
		payload = d
	case github.DeploymentEvent:
		var d github.DeploymentPayload
		if err := json.NewDecoder(c.Request().Body).Decode(&d); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest)
		}
		payload = d
	case github.DeploymentStatusEvent:
		var d github.DeploymentStatusPayload
		if err := json.NewDecoder(c.Request().Body).Decode(&d); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest)
		}
		payload = d
	case github.ForkEvent:
		var d github.ForkPayload
		if err := json.NewDecoder(c.Request().Body).Decode(&d); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest)
		}
		payload = d
	case github.GollumEvent:
		var d github.GollumPayload
		if err := json.NewDecoder(c.Request().Body).Decode(&d); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest)
		}
		payload = d
	case github.InstallationEvent, github.IntegrationInstallationEvent:
		var d github.InstallationPayload
		if err := json.NewDecoder(c.Request().Body).Decode(&d); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest)
		}
		payload = d
	case github.IssueCommentEvent:
		var d github.IssueCommentPayload
		if err := json.NewDecoder(c.Request().Body).Decode(&d); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest)
		}
		payload = d
	case github.IssuesEvent:
		var d github.IssuesPayload
		if err := json.NewDecoder(c.Request().Body).Decode(&d); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest)
		}
		payload = d
	case github.LabelEvent:
		var d github.LabelPayload
		if err := json.NewDecoder(c.Request().Body).Decode(&d); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest)
		}
		payload = d
	case github.MemberEvent:
		var d github.MemberPayload
		if err := json.NewDecoder(c.Request().Body).Decode(&d); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest)
		}
		payload = d
	case github.MembershipEvent:
		var d github.MembershipPayload
		if err := json.NewDecoder(c.Request().Body).Decode(&d); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest)
		}
		payload = d
	case github.MilestoneEvent:
		var d github.MilestonePayload
		if err := json.NewDecoder(c.Request().Body).Decode(&d); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest)
		}
		payload = d
	case github.OrganizationEvent:
		var d github.OrganizationPayload
		if err := json.NewDecoder(c.Request().Body).Decode(&d); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest)
		}
		payload = d
	case github.OrgBlockEvent:
		var d github.OrgBlockPayload
		if err := json.NewDecoder(c.Request().Body).Decode(&d); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest)
		}
		payload = d
	case github.PageBuildEvent:
		var d github.PageBuildPayload
		if err := json.NewDecoder(c.Request().Body).Decode(&d); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest)
		}
		payload = d
	case github.PingEvent:
		var d github.PingPayload
		if err := json.NewDecoder(c.Request().Body).Decode(&d); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest)
		}
		payload = d
	case github.ProjectCardEvent:
		var d github.ProjectCardPayload
		if err := json.NewDecoder(c.Request().Body).Decode(&d); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest)
		}
		payload = d
	case github.ProjectColumnEvent:
		var d github.ProjectColumnPayload
		if err := json.NewDecoder(c.Request().Body).Decode(&d); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest)
		}
		payload = d
	case github.ProjectEvent:
		var d github.ProjectPayload
		if err := json.NewDecoder(c.Request().Body).Decode(&d); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest)
		}
		payload = d
	case github.PublicEvent:
		var d github.PublicPayload
		if err := json.NewDecoder(c.Request().Body).Decode(&d); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest)
		}
		payload = d
	case github.PullRequestEvent:
		var d github.PullRequestPayload
		if err := json.NewDecoder(c.Request().Body).Decode(&d); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest)
		}
		payload = d
	case github.PullRequestReviewEvent:
		var d github.PullRequestReviewPayload
		if err := json.NewDecoder(c.Request().Body).Decode(&d); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest)
		}
		payload = d
	case github.PullRequestReviewCommentEvent:
		var d github.PullRequestReviewCommentPayload
		if err := json.NewDecoder(c.Request().Body).Decode(&d); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest)
		}
		payload = d
	case github.PushEvent:
		var d github.PushPayload
		if err := json.NewDecoder(c.Request().Body).Decode(&d); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest)
		}
		payload = d
	case github.ReleaseEvent:
		var d github.ReleasePayload
		if err := json.NewDecoder(c.Request().Body).Decode(&d); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest)
		}
		payload = d
	case github.RepositoryEvent:
		var d github.RepositoryPayload
		if err := json.NewDecoder(c.Request().Body).Decode(&d); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest)
		}
		payload = d
	case github.StatusEvent:
		var d github.StatusPayload
		if err := json.NewDecoder(c.Request().Body).Decode(&d); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest)
		}
		payload = d
	case github.TeamEvent:
		var d github.TeamPayload
		if err := json.NewDecoder(c.Request().Body).Decode(&d); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest)
		}
		payload = d
	case github.TeamAddEvent:
		var d github.TeamAddPayload
		if err := json.NewDecoder(c.Request().Body).Decode(&d); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest)
		}
		payload = d
	case github.WatchEvent:
		var d github.WatchPayload
		if err := json.NewDecoder(c.Request().Body).Decode(&d); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest)
		}
		payload = d
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
		m, err := model.CreateMessage(w.GetBotUserID(), w.GetChannelID(), messageBuf.String())
		if err != nil {
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
		go event.Emit(event.MessageCreated, &event.MessageCreatedEvent{Message: *m})
	}

	return c.NoContent(http.StatusNoContent)
}

func getWebhook(c echo.Context, id uuid.UUID, strict bool) (model.Webhook, error) {
	if id == uuid.Nil {
		return nil, echo.NewHTTPError(http.StatusNotFound)
	}

	w, err := model.GetWebhook(id)
	if err != nil {
		c.Logger().Error(err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError)
	}
	if w == nil {
		return nil, echo.NewHTTPError(http.StatusNotFound)
	}
	if strict {
		user, ok := c.Get("user").(*model.User)
		if !ok || w.GetCreatorID() != user.GetUID() {
			return nil, echo.NewHTTPError(http.StatusForbidden)
		}
	}

	return w, nil
}

func formatWebhook(w model.Webhook) *webhookForResponse {
	return &webhookForResponse{
		WebhookID:   w.GetID().String(),
		BotUserID:   w.GetBotUserID().String(),
		DisplayName: w.GetName(),
		Description: w.GetDescription(),
		ChannelID:   w.GetChannelID().String(),
		CreatorID:   w.GetCreatorID().String(),
		CreatedAt:   w.GetCreatedAt(),
		UpdatedAt:   w.GetUpdatedAt(),
	}
}
