package router

import (
	"encoding/json"
	"fmt"
	"github.com/go-sql-driver/mysql"
	"github.com/labstack/echo"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"golang.org/x/exp/utf8string"
	"gopkg.in/go-playground/webhooks.v3/github"
	"io/ioutil"
	"net/http"
	"strings"
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

// GetWebhooks GET /webhooks
func GetWebhooks(c echo.Context) error {
	user := c.Get("user").(*model.User)

	list, err := model.GetWebhooksByCreator(user.GetUID())
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
	user := c.Get("user").(*model.User)

	req := struct {
		Name        string `json:"name"        validate:"max=32,required"`
		Description string `json:"description" validate:"required"`
		ChannelID   string `json:"channelId"   validate:"uuid,required"`
	}{}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	if _, err := validateChannelID(req.ChannelID, user.ID); err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound, "this channel is not found")
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to find the specified channel")
		}
	}

	iconID, err := model.GenerateIcon(req.Name)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	w, err := model.CreateWebhook(req.Name, req.Description, uuid.Must(uuid.FromString(req.ChannelID)), user.GetUID(), uuid.Must(uuid.FromString(iconID)))
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	go event.Emit(event.UserJoined, event.UserEvent{ID: w.ID().String()})
	return c.JSON(http.StatusCreated, formatWebhook(w))
}

// GetWebhook GET /webhooks/:webhookID
func GetWebhook(c echo.Context) error {
	w, err := getWebhook(c, uuid.FromStringOrNil(c.Param("webhookID")), true)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, formatWebhook(w))
}

// PatchWebhook PATCH /webhooks/:webhookID
func PatchWebhook(c echo.Context) error {
	user := c.Get("user").(*model.User)

	w, err := getWebhook(c, uuid.FromStringOrNil(c.Param("webhookID")), true)
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

		if _, err := validateChannelID(req.ChannelID, user.ID); err != nil {
			switch err {
			case model.ErrNotFound:
				return echo.NewHTTPError(http.StatusBadRequest, "this channel is not found")
			default:
				c.Logger().Error(err)
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to find the specified channel")
			}
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

		go event.Emit(event.UserUpdated, event.UserEvent{ID: w.BotUserID().String()})
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
	w, err := getWebhook(c, uuid.FromStringOrNil(c.Param("webhookID")), true)
	if err != nil {
		return err
	}

	if err := model.DeleteWebhook(w.ID()); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}

// PostWebhook POST /webhooks/:webhookID
func PostWebhook(c echo.Context) error {
	w, err := getWebhook(c, uuid.FromStringOrNil(c.Param("webhookID")), false)
	if err != nil {
		return err
	}

	message := &model.Message{
		UserID:    w.BotUserID().String(),
		ChannelID: w.ChannelID().String(),
	}
	switch c.Request().Header.Get(echo.HeaderContentType) {
	case echo.MIMETextPlain, echo.MIMETextPlainCharsetUTF8:
		if b, err := ioutil.ReadAll(c.Request().Body); err == nil {
			message.Text = string(b)
		}
		if len(message.Text) == 0 {
			return echo.NewHTTPError(http.StatusBadRequest)
		}

	case echo.MIMEApplicationJSON, echo.MIMEApplicationForm, echo.MIMEApplicationJSONCharsetUTF8:
		req := struct {
			Text      string `json:"text"      form:"text"`
			ChannelID string `json:"channelId" form:"channelId"`
		}{}
		if err := c.Bind(&req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err)
		}
		if len(req.Text) == 0 {
			return echo.NewHTTPError(http.StatusBadRequest)
		}
		if len(req.ChannelID) == 36 {
			message.ChannelID = req.ChannelID
		}
		message.Text = req.Text

	default:
		return echo.NewHTTPError(http.StatusUnsupportedMediaType)
	}

	if err := message.Create(); err != nil {
		if errSQL, ok := err.(*mysql.MySQLError); ok {
			if errSQL.Number == 1452 { //外部キー制約
				return echo.NewHTTPError(http.StatusBadRequest, "invalid channelId")
			}
		}

		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	go event.Emit(event.MessageCreated, &event.MessageCreatedEvent{Message: *message})
	return c.NoContent(http.StatusNoContent)
}

// PutWebhookIcon PUT /webhooks/:webhookID/icon
func PutWebhookIcon(c echo.Context) error {
	w, err := getWebhook(c, uuid.FromStringOrNil(c.Param("webhookID")), true)
	if err != nil {
		return err
	}

	wu, err := model.GetUser(w.BotUserID().String())
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
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
	if err := wu.UpdateIconID(iconID.String()); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	go event.Emit(event.UserIconUpdated, event.UserEvent{ID: w.BotUserID().String()})
	return c.NoContent(http.StatusOK)
}

// PostWebhookByGithub POST /webhooks/:webhookID/github
func PostWebhookByGithub(c echo.Context) error {
	w, err := getWebhook(c, uuid.FromStringOrNil(c.Param("webhookID")), false)
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

	//MEMO 現在はサーバー側で簡単に整形してるけど、将来的にクライアント側に表示デザイン込みで任せたいよね
	message := &model.Message{
		UserID:    w.BotUserID().String(),
		ChannelID: w.ChannelID().String(),
	}

	switch githubEvent {
	case github.IssuesEvent: // Any time an Issue is assigned, unassigned, labeled, unlabeled, opened, edited, milestoned, demilestoned, closed, or reopened.
		var i github.IssuesPayload
		if err := json.NewDecoder(c.Request().Body).Decode(&i); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest)
		}

		switch i.Action {
		case "opened":
			message.Text = fmt.Sprintf("## Issue Opened\n[%s](%s) - [%s](%s)", i.Repository.FullName, i.Repository.HTMLURL, i.Issue.Title, i.Issue.HTMLURL)
		case "closed":
			message.Text = fmt.Sprintf("## Issue Closed\n[%s](%s) - [%s](%s)", i.Repository.FullName, i.Repository.HTMLURL, i.Issue.Title, i.Issue.HTMLURL)
		case "reopened":
			message.Text = fmt.Sprintf("## Issue Reopened\n[%s](%s) - [%s](%s)", i.Repository.FullName, i.Repository.HTMLURL, i.Issue.Title, i.Issue.HTMLURL)
		case "assigned", "unassigned", "labeled", "unlabeled", "edited", "milestoned", "demilestoned":
			// Unsupported
		}

	case github.PullRequestEvent: // Any time a pull request is assigned, unassigned, labeled, unlabeled, opened, edited, closed, reopened, or synchronized (updated due to a new push in the branch that the pull request is tracking). Also any time a pull request review is requested, or a review request is removed.
		var p github.PullRequestPayload
		if err := json.NewDecoder(c.Request().Body).Decode(&p); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest)
		}

		switch p.Action {
		case "opened":
			message.Text = fmt.Sprintf("## PullRequest Opened\n[%s](%s) - [%s](%s)", p.Repository.FullName, p.Repository.HTMLURL, p.PullRequest.Title, p.PullRequest.HTMLURL)
		case "closed":
			if p.PullRequest.Merged {
				message.Text = fmt.Sprintf("## PullRequest Merged\n[%s](%s) - [%s](%s)", p.Repository.FullName, p.Repository.HTMLURL, p.PullRequest.Title, p.PullRequest.HTMLURL)
			} else {
				message.Text = fmt.Sprintf("## PullRequest Closed\n[%s](%s) - [%s](%s)", p.Repository.FullName, p.Repository.HTMLURL, p.PullRequest.Title, p.PullRequest.HTMLURL)
			}
		case "reopened":
			message.Text = fmt.Sprintf("## PullRequest Reopened\n[%s](%s) - [%s](%s)", p.Repository.FullName, p.Repository.HTMLURL, p.PullRequest.Title, p.PullRequest.HTMLURL)
		case "assigned", "unassigned", "labeled", "unlabeled", "edited", "review_requested", "review_request_removed":
			// Unsupported
		}

	case github.PushEvent: // Any Git push to a Repository, including editing tags or branches. Commits via API actions that update references are also counted. This is the default event.
		var p github.PushPayload
		if err := json.NewDecoder(c.Request().Body).Decode(&p); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest)
		}
		if len(p.Commits) == 0 {
			break
		}

		message.Text = fmt.Sprintf("## %d Commit(s) Pushed by %s\n[%s](%s), refs: `%s`\n", len(p.Commits), p.Pusher.Name, p.Repository.FullName, p.Repository.HTMLURL, p.Ref)

		for _, v := range p.Commits {
			message.Text += fmt.Sprintf("+ [`%s`](%s) - `%s`\n", utf8string.NewString(v.ID).Slice(0, 7), v.URL, strings.Replace(v.Message, "\n", " ", -1))
		}

	default:
		// Currently Unsupported:
		// marketplace_purchase, fork, gollum, installation, installation_repositories, label, ping, member, membership,
		// organization, org_block, page_build, public, repository, status, team, team_add, watch, create, delete, deployment,
		// deployment_status, project_column, milestone, project_card, project, commit_comment, release, issue_comment,
		// pull_request_review, pull_request_review_comment
		// 上ので必要な場合は実装してプルリクを飛ばしてください
	}

	if len(message.Text) > 0 {
		if err := message.Create(); err != nil {
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
		go event.Emit(event.MessageCreated, &event.MessageCreatedEvent{Message: *message})
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
		if !ok || w.CreatorID() != user.GetUID() {
			return nil, echo.NewHTTPError(http.StatusForbidden)
		}
	}

	return w, nil
}

func formatWebhook(w model.Webhook) *webhookForResponse {
	return &webhookForResponse{
		WebhookID:   w.ID().String(),
		BotUserID:   w.BotUserID().String(),
		DisplayName: w.Name(),
		Description: w.Description(),
		ChannelID:   w.ChannelID().String(),
		CreatorID:   w.CreatorID().String(),
		CreatedAt:   w.CreatedAt(),
		UpdatedAt:   w.UpdatedAt(),
	}
}
