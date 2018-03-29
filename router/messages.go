package router

import (
	"net/http"
	"time"

	"github.com/traPtitech/traQ/notification"
	"github.com/traPtitech/traQ/notification/events"

	"github.com/labstack/echo"
	"github.com/labstack/gommon/log"
	"github.com/traPtitech/traQ/model"
)

//MessageForResponse :クライアントに返す形のメッセージオブジェクト
type MessageForResponse struct {
	MessageID       string                `json:"messageId"`
	UserID          string                `json:"userId"`
	ParentChannelID string                `json:"parentChannelId"`
	Content         string                `json:"content"`
	Datetime        time.Time             `json:"datetime"`
	Pin             bool                  `json:"pin"`
	StampList       []*model.MessageStamp `json:"stampList"`
}

// GetMessageByID GET /messages/{messageID} のハンドラ
func GetMessageByID(c echo.Context) error {
	messageID := c.Param("messageID")
	m, err := validateMessageID(messageID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, formatMessage(m))
}

// GetMessagesByChannelID GET /channels/{channelID}/messages のハンドラ
func GetMessagesByChannelID(c echo.Context) error {
	queryParam := &struct {
		Limit  int `query:"limit"`
		Offset int `query:"offset"`
	}{}
	if err := c.Bind(queryParam); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid format")
	}

	// channelIDの検証
	channelID := c.Param("channelID")
	userID := c.Get("user").(*model.User).ID
	if _, err := validateChannelID(channelID, userID); err != nil {
		return err
	}

	messages, err := model.GetMessagesByChannelID(channelID, queryParam.Limit, queryParam.Offset)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Channel is not found")
	}

	res := make([]*MessageForResponse, 0)
	for _, message := range messages {
		res = append(res, formatMessage(message))
	}

	return c.JSON(http.StatusOK, res)
}

// PostMessage POST /channels/{channelID}/messages のハンドラ
func PostMessage(c echo.Context) error {
	post := &struct {
		Text string `json:"text"`
	}{}
	if err := c.Bind(post); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid format")
	}

	m := &model.Message{
		UserID:    c.Get("user").(*model.User).ID,
		Text:      post.Text,
		ChannelID: c.Param("channelID"),
	}
	if err := m.Create(); err != nil {
		c.Logger().Errorf("Message.Create() returned an error: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to insert your message")
	}

	go notification.Send(events.MessageCreated, events.MessageEvent{Message: *m})
	return c.JSON(http.StatusCreated, formatMessage(m))
}

// PutMessageByID PUT /messages/{messageID}のハンドラ
func PutMessageByID(c echo.Context) error {
	m, err := validateMessageID(c.Param("messageID"))
	if err != nil {
		return err
	}

	req := &struct {
		Text string `json:"text"`
	}{}
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid format")
	}

	m.Text = req.Text
	m.UpdaterID = c.Get("user").(*model.User).ID

	if err := m.Update(); err != nil {
		c.Logger().Errorf("message.Update() returned an error: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update the message")
	}

	go notification.Send(events.MessageUpdated, events.MessageEvent{Message: *m})
	return c.JSON(http.StatusOK, formatMessage(m))
}

// DeleteMessageByID : DELETE /message/{messageID} のハンドラ
func DeleteMessageByID(c echo.Context) error {
	userID := c.Get("user").(*model.User).ID
	messageID := c.Param("messageID")

	m, err := validateMessageID(messageID)
	if err != nil {
		return err
	}
	if m.UserID != userID {
		return echo.NewHTTPError(http.StatusForbidden, "you are not allowed to delete this message")
	}

	m.IsDeleted = true
	if err := m.Update(); err != nil {
		c.Logger().Errorf("message.Update() returned an error: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update the message")
	}

	if err := model.DeleteUnreadsByMessageID(messageID); err != nil {
		c.Logger().Errorf("model.DeleteUnreadsByMessageID returned an error: %v", err) //500エラーにはしない
	}

	go notification.Send(events.MessageDeleted, events.MessageEvent{Message: *m})
	return c.NoContent(http.StatusNoContent)
}

func formatMessage(raw *model.Message) *MessageForResponse {
	isPinned, err := raw.IsPinned()
	if err != nil {
		log.Error(err)
	}

	stampList, err := model.GetMessageStamps(raw.ID)
	if err != nil {
		log.Error(err)
	}

	res := &MessageForResponse{
		MessageID:       raw.ID,
		UserID:          raw.UserID,
		ParentChannelID: raw.ChannelID,
		Pin:             isPinned,
		Content:         raw.Text,
		Datetime:        raw.CreatedAt.Truncate(time.Second).UTC(),
		StampList:       stampList,
	}
	return res
}

// リクエストで飛んできたmessageIDを検証する。存在する場合はそのメッセージを返す
func validateMessageID(messageID string) (*model.Message, error) {
	m := &model.Message{ID: messageID}
	ok, err := m.Exists()
	if err != nil {
		log.Error(err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Cannot find message")
	}
	if !ok {
		return nil, echo.NewHTTPError(http.StatusNotFound, "Message is not found")
	}
	return m, nil
}
