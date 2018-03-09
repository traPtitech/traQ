package router

import (
	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/notification"
	"github.com/traPtitech/traQ/notification/events"
	"io/ioutil"
	"net/http"
	"time"
)

type webhookForResponse struct {
	ID          string    `json:"id"`
	DisplayName string    `json:"displayName"`
	Description string    `json:"description"`
	IconFileID  string    `json:"iconFileId"`
	ChannelID   string    `json:"channelId"`
	Token       string    `json:"token"`
	Valid       bool      `json:"valid"`
	CreatorID   string    `json:"creatorId"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdaterID   string    `json:"updaterId"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// GetChannelWebhooks : GET /channels/:channelID/webhooks
func GetChannelWebhooks(c echo.Context) error {
	//TODO ユーザー権限によって403にする
	userID := c.Get("user").(*model.User).ID
	channelID := c.Param("channelID")

	if _, err := validateChannelID(channelID, userID); err != nil {
		return err
	}

	list, err := model.GetWebhooksByChannelID(channelID)
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

// PostChannelWebhooks : POST /channels/:channelID/webhooks
func PostChannelWebhooks(c echo.Context) error {
	//TODO ユーザー権限によって403にする
	userID := c.Get("user").(*model.User).ID
	channelID := c.Param("channelID")

	if _, err := validateChannelID(channelID, userID); err != nil {
		return err
	}

	req := struct {
		Name        string `json:"name" form:"name"`
		Description string `json:"description" form:"description"`
	}{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
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

	wb, err := model.CreateWebhook(req.Name, req.Description, channelID, userID, fileID)
	if err != nil {
		switch err {
		case model.ErrBotInvalidName:
			return echo.NewHTTPError(http.StatusBadRequest, err)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	return c.JSON(http.StatusCreated, formatWebhook(wb))
}

// GetWebhook : GET /webhooks/:webhookID
func GetWebhook(c echo.Context) error {
	//TODO ユーザー権限によって403にする
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

	return c.JSON(http.StatusOK, formatWebhook(wb))
}

// PatchWebhook : PATCH /webhooks/:webhookID
func PatchWebhook(c echo.Context) error {
	//TODO ユーザー権限によって403にする
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
	if !wb.IsValid {
		return echo.NewHTTPError(http.StatusForbidden)
	}

	req := struct {
		Name        string `json:"name" form:"name"`
		Description string `json:"description" form:"description"`
	}{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}
	if len(req.Name) > 32 {
		return echo.NewHTTPError(http.StatusBadRequest, model.ErrBotInvalidName)
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
	}
	if len(req.Name) > 0 {
		if err := wb.UpdateDisplayName(req.Name); err != nil {
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
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
	//TODO ユーザー権限によって403にする
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

	if err := wb.Invalidate(); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}

// PostWebhook : POST /webhooks/:webhookID/:token
func PostWebhook(c echo.Context) error {
	webhookID := c.Param("webhookID")
	token := c.Param("token")

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
	if wb.Token != token {
		return echo.NewHTTPError(http.StatusForbidden)
	}

	message := &model.Message{
		UserID:    wb.ID,
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
			Text string `json:"text" form:"text"`
		}{}
		if err := c.Bind(&req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err)
		}
		if len(req.Text) == 0 {
			return echo.NewHTTPError(http.StatusBadRequest)
		}
		message.Text = req.Text

	default:
		return echo.NewHTTPError(http.StatusUnsupportedMediaType)
	}

	if err := message.Create(); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	go notification.Send(events.MessageCreated, events.MessageEvent{Message: *message})
	return c.NoContent(http.StatusNoContent)
}

// PostWebhookByGithub : POST /webhooks/:webhookID/:token/github
func PostWebhookByGithub(c echo.Context) error {
	webhookID := c.Param("webhookID")
	token := c.Param("token")

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
	if wb.Token != token {
		return echo.NewHTTPError(http.StatusForbidden)
	}

	//TODO parse JSON and post message

	return c.NoContent(http.StatusNoContent)
}

func formatWebhook(w *model.WebhookBotUser) *webhookForResponse {
	return &webhookForResponse{
		ID:          w.ID,
		DisplayName: w.DisplayName,
		Description: w.Description,
		IconFileID:  w.Icon,
		ChannelID:   w.ChannelID,
		Token:       w.Token,
		Valid:       w.IsValid,
		CreatorID:   w.Bot.CreatorID,
		CreatedAt:   w.Bot.CreatedAt,
		UpdaterID:   w.Bot.UpdaterID,
		UpdatedAt:   w.Bot.UpdatedAt,
	}
}
