package router

import (
	"encoding/json"
	"fmt"
	"github.com/go-sql-driver/mysql"
	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/notification"
	"github.com/traPtitech/traQ/notification/events"
	"gopkg.in/go-playground/webhooks.v3/github"
	"io/ioutil"
	"net/http"
	"time"
)

type webhookForResponse struct {
	WebhookID   string    `json:"webhookID"`
	BotUserID   string    `json:"botUserId"`
	DisplayName string    `json:"displayName"`
	Description string    `json:"description"`
	IconFileID  string    `json:"iconFileId"`
	ChannelID   string    `json:"channelId"`
	Valid       bool      `json:"valid"`
	CreatorID   string    `json:"creatorId"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdaterID   string    `json:"updaterId"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// GetWebhooks : GET /webhooks
func GetWebhooks(c echo.Context) error {
	userID := c.Get("user").(*model.User).ID

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

// PostWebhooks : POST /webhooks
func PostWebhooks(c echo.Context) error {
	//TODO ユーザー権限によって403にする
	userID := c.Get("user").(*model.User).ID

	req := struct {
		Name        string `json:"name"        form:"name"`
		Description string `json:"description" form:"description"`
		ChannelID   string `json:"channelId"   form:"channelId"`
	}{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	if _, err := validateChannelID(req.ChannelID, userID); err != nil {
		return err
	}

	fileID := ""

	if c.Request().MultipartForm != nil {
		if uploadedFile, err := c.FormFile("file"); err == nil {
			contentType := uploadedFile.Header.Get(echo.HeaderContentType)
			switch contentType {
			case "image/png", "image/jpeg", "image/gif", "image/svg+xml":
				break
			default:
				return echo.NewHTTPError(http.StatusBadRequest, "invalid image file")
			}

			if uploadedFile.Size > 1024*1024 {
				return echo.NewHTTPError(http.StatusBadRequest, "too big image file")
			}

			file := &model.File{
				Name:      uploadedFile.Filename,
				Size:      uploadedFile.Size,
				CreatorID: userID,
			}

			src, err := uploadedFile.Open()
			if err != nil {
				c.Logger().Error(err)
				return echo.NewHTTPError(http.StatusInternalServerError)
			}
			defer src.Close()

			if err := file.Create(src); err != nil {
				c.Logger().Error(err)
				return echo.NewHTTPError(http.StatusInternalServerError)
			}
			fileID = file.ID
		} else if err != http.ErrMissingFile {
			return echo.NewHTTPError(http.StatusBadRequest, err)
		}
	}

	wb, err := model.CreateWebhook(req.Name, req.Description, req.ChannelID, userID, fileID)
	if err != nil {
		switch err {
		case model.ErrBotInvalidName:
			return echo.NewHTTPError(http.StatusBadRequest, err)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}
	go notification.Send(events.UserJoined, events.UserEvent{ID: wb.User.ID})

	return c.JSON(http.StatusCreated, formatWebhook(wb))
}

// GetWebhook : GET /webhooks/:webhookID
func GetWebhook(c echo.Context) error {
	webhookID := c.Param("webhookID")
	userID := c.Get("user").(*model.User).ID

	wb, err := model.GetWebhook(webhookID)
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}
	if wb.CreatorID != userID {
		return echo.NewHTTPError(http.StatusForbidden)
	}

	return c.JSON(http.StatusOK, formatWebhook(wb))
}

// PatchWebhook : PATCH /webhooks/:webhookID
func PatchWebhook(c echo.Context) error {
	webhookID := c.Param("webhookID")
	userID := c.Get("user").(*model.User).ID

	wb, err := model.GetWebhook(webhookID)
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}
	if wb.CreatorID != userID {
		return echo.NewHTTPError(http.StatusForbidden)
	}
	if !wb.IsValid {
		return echo.NewHTTPError(http.StatusForbidden)
	}

	req := struct {
		Name        string `json:"name"        form:"name"`
		Description string `json:"description" form:"description"`
		ChannelID   string `json:"channelId"   form:"channelId"`
	}{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}
	if len(req.Name) > 32 {
		return echo.NewHTTPError(http.StatusBadRequest, model.ErrBotInvalidName)
	}

	if len(req.ChannelID) > 0 {
		ch := &model.Channel{ID: req.ChannelID}
		ok, err := ch.Exists(userID)
		if err != nil {
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
		if !ok {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid channelId")
		}

		if err := wb.UpdateChannelID(ch.ID); err != nil {
			if errSQL, ok := err.(*mysql.MySQLError); ok {
				if errSQL.Number == 1452 { //外部キー制約
					return echo.NewHTTPError(http.StatusBadRequest, "invalid channelId")
				}
			}
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	fileID := ""

	if c.Request().MultipartForm != nil {
		if uploadedFile, err := c.FormFile("file"); err == nil {
			contentType := uploadedFile.Header.Get(echo.HeaderContentType)
			switch contentType {
			case "image/png", "image/jpeg", "image/gif", "image/svg+xml":
				break
			default:
				return echo.NewHTTPError(http.StatusBadRequest, "invalid image file")
			}

			if uploadedFile.Size > 1024*1024 {
				return echo.NewHTTPError(http.StatusBadRequest, "too big image file")
			}

			file := &model.File{
				Name:      uploadedFile.Filename,
				Size:      uploadedFile.Size,
				CreatorID: userID,
			}

			src, err := uploadedFile.Open()
			if err != nil {
				c.Logger().Error(err)
				return echo.NewHTTPError(http.StatusInternalServerError)
			}
			defer src.Close()

			if err := file.Create(src); err != nil {
				c.Logger().Error(err)
				return echo.NewHTTPError(http.StatusInternalServerError)
			}
			fileID = file.ID
		} else if err != http.ErrMissingFile {
			return echo.NewHTTPError(http.StatusBadRequest, err)
		}
	}

	if len(fileID) == 36 {
		if err := wb.UpdateIconID(fileID); err != nil {
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}

		go notification.Send(events.UserIconUpdated, events.UserEvent{ID: wb.User.ID})
	}
	if len(req.Name) > 0 {
		if err := wb.UpdateDisplayName(req.Name); err != nil {
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}

		go notification.Send(events.UserUpdated, events.UserEvent{ID: wb.User.ID})
	}
	if len(req.Description) > 0 {
		wb.Description = req.Description
	}

	if err := wb.Update(); err != nil {
		switch err {
		case model.ErrBotInvalidName:
			return echo.NewHTTPError(http.StatusBadRequest, err)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	return c.NoContent(http.StatusNoContent)
}

// DeleteWebhook : DELETE /webhooks/:webhookID
func DeleteWebhook(c echo.Context) error {
	webhookID := c.Param("webhookID")
	userID := c.Get("user").(*model.User).ID

	wb, err := model.GetWebhook(webhookID)
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}
	if wb.CreatorID != userID {
		return echo.NewHTTPError(http.StatusForbidden)
	}
	if !wb.IsValid {
		return echo.NewHTTPError(http.StatusForbidden)
	}

	if err := wb.Invalidate(); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}

// PostWebhook : POST /webhooks/:webhookID
func PostWebhook(c echo.Context) error {
	webhookID := c.Param("webhookID")

	wb, err := model.GetWebhook(webhookID)
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}
	if !wb.IsValid {
		return echo.NewHTTPError(http.StatusForbidden)
	}

	message := &model.Message{
		UserID:    wb.Webhook.UserID,
		ChannelID: wb.ChannelID,
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

	go notification.Send(events.MessageCreated, events.MessageEvent{Message: *message})
	return c.NoContent(http.StatusNoContent)
}

// PostWebhookByGithub : POST /webhooks/:webhookID/github
func PostWebhookByGithub(c echo.Context) error {
	webhookID := c.Param("webhookID")

	wb, err := model.GetWebhook(webhookID)
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}
	if !wb.IsValid {
		return echo.NewHTTPError(http.StatusForbidden)
	}

	switch c.Request().Header.Get(echo.HeaderContentType) {
	case echo.MIMEApplicationJSON, echo.MIMEApplicationJSONCharsetUTF8:
		break
	default:
		return echo.NewHTTPError(http.StatusUnsupportedMediaType)
	}

	event := c.Request().Header.Get("X-GitHub-Event")
	if len(event) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "missing X-GitHub-Event header")
	}

	githubEvent := github.Event(event)

	//MEMO 現在はサーバー側で簡単に整形してるけど、将来的にクライアント側に表示デザイン込みで任せたいよね
	message := &model.Message{
		UserID:    wb.Webhook.UserID,
		ChannelID: wb.ChannelID,
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

		message.Text = fmt.Sprintf("## %d Commit(s) Pushed by %s\n[%s](%s) , refs: `%s`, [compare](%s)\n", len(p.Commits), p.Pusher.Name, p.Repository.FullName, p.Repository.HTMLURL, p.Ref, p.Compare)

		for _, v := range p.Commits {
			message.Text += fmt.Sprintf("+ [`%7s`](%s) - %s \n", v.ID, v.URL, v.Message)
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
		go notification.Send(events.MessageCreated, events.MessageEvent{Message: *message})
	}

	return c.NoContent(http.StatusNoContent)
}

func formatWebhook(w *model.WebhookBotUser) *webhookForResponse {
	return &webhookForResponse{
		WebhookID:   w.Webhook.ID,
		BotUserID:   w.User.ID,
		DisplayName: w.DisplayName,
		Description: w.Description,
		IconFileID:  w.Icon,
		ChannelID:   w.ChannelID,
		Valid:       w.IsValid,
		CreatorID:   w.Bot.CreatorID,
		CreatedAt:   w.Bot.CreatedAt,
		UpdaterID:   w.Bot.UpdaterID,
		UpdatedAt:   w.Bot.UpdatedAt,
	}
}
