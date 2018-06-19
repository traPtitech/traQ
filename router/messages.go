package router

import (
	"net/http"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/event"
	"gopkg.in/go-playground/validator.v9"

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
	CreatedAt       time.Time             `json:"createdAt"`
	UpdatedAt       time.Time             `json:"updatedAt"`
	Pin             bool                  `json:"pin"`
	Reported        bool                  `json:"reported"`
	StampList       []*model.MessageStamp `json:"stampList"`
}

// GetMessageByID GET /messages/{messageID} のハンドラ
func GetMessageByID(c echo.Context) error {
	userID := c.Get("user").(*model.User).ID
	messageID := c.Param("messageID")
	m, err := validateMessageID(messageID, userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "not found")
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
	userID := c.Get("user").(*model.User).ID
	res, err := getMessages(c.Param("channelID"), userID, queryParam.Limit, queryParam.Offset)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, res)
}

// PostMessage POST /channels/{channelID}/messages のハンドラ
func PostMessage(c echo.Context) error {
	// 10KB制限
	if c.Request().ContentLength > 10*1024 {
		return echo.NewHTTPError(http.StatusRequestEntityTooLarge, "a request must be smaller than 10KB")
	}

	post := &struct {
		Text string `json:"text"`
	}{}
	if err := c.Bind(post); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid format")
	}

	userID := c.Get("user").(*model.User).ID
	channelID := c.Param("channelID")

	_, err := validateChannelID(channelID, userID)
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound, "this channel is not found")
		default:
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to find the specified channel")
		}
	}

	m, err := createMessage(post.Text, userID, channelID)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusCreated, m)
}

// PutMessageByID PUT /messages/{messageID}のハンドラ
func PutMessageByID(c echo.Context) error {
	userID := c.Get("user").(*model.User).ID
	m, err := validateMessageID(c.Param("messageID"), userID)
	if err != nil {
		return err
	}

	// 他人のテキストは編集できない
	if userID != m.UserID {
		return echo.NewHTTPError(http.StatusForbidden, "This is not your message")
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

	go event.Emit(event.MessageUpdated, &event.MessageUpdatedEvent{Message: *m})
	return c.JSON(http.StatusOK, formatMessage(m))
}

// DeleteMessageByID : DELETE /message/{messageID} のハンドラ
func DeleteMessageByID(c echo.Context) error {
	userID := c.Get("user").(*model.User).ID
	messageID := c.Param("messageID")

	m, err := validateMessageID(messageID, userID)
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

	go event.Emit(event.MessageDeleted, &event.MessageDeletedEvent{Message: *m})
	return c.NoContent(http.StatusNoContent)
}

// PostMessageReport POST /messages/{messageID}/report
func PostMessageReport(c echo.Context) error {
	user := c.Get("user").(*model.User)
	messageID := c.Param("messageID")

	req := &struct {
		Reason string `json:"reason"`
	}{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	m, err := validateMessageID(messageID, user.ID)
	if err != nil {
		return err
	}

	mID := uuid.Must(uuid.FromString(messageID))

	if err := model.CreateMessageReport(mID, user.GetUID(), req.Reason); err != nil {
		switch e := err.(type) {
		case *validator.ValidationErrors:
			return echo.NewHTTPError(http.StatusBadRequest, err)
		case *mysql.MySQLError:
			if e.Number == errMySQLDuplicatedRecord {
				return echo.NewHTTPError(http.StatusBadRequest, "already reported")
			}
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	r, err := model.GetMessageReportsByMessageID(mID)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	// 5人でメッセージBAN
	if len(r) >= 5 {
		m.IsDeleted = true
		if err := m.Update(); err != nil {
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	return c.NoContent(http.StatusNoContent)
}

// dbにデータを入れる
func createMessage(text, userID, channelID string) (*MessageForResponse, error) {
	m := &model.Message{
		UserID:    userID,
		Text:      text,
		ChannelID: channelID,
	}
	if err := m.Create(); err != nil {
		log.Errorf("Message.Create() returned an error: %v", err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to insert your message")
	}

	go event.Emit(event.MessageCreated, &event.MessageCreatedEvent{Message: *m})
	return formatMessage(m), nil
}

// チャンネルのデータを取得する
func getMessages(channelID, userID string, limit, offset int) ([]*MessageForResponse, error) {
	if _, err := validateChannelID(channelID, userID); err != nil {
		switch err {
		case model.ErrNotFound:
			return nil, echo.NewHTTPError(http.StatusNotFound, "this channel is not found")
		default:
			return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to find the specified channel")
		}
	}

	messages, err := model.GetMessagesByChannelID(channelID, limit, offset)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusNotFound, "Channel is not found")
	}

	reports, err := model.GetMessageReportsByReporterID(uuid.FromStringOrNil(userID))
	if err != nil {
		log.Error(err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError)
	}
	hidden := make(map[string]bool)
	for _, v := range reports {
		hidden[v.MessageID] = true
	}

	res := make([]*MessageForResponse, 0, limit)

	for _, message := range messages {
		ms := formatMessage(message)
		if hidden[message.ID] {
			ms.Reported = true
		}
		res = append(res, ms)
	}
	return res, nil
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
		CreatedAt:       raw.CreatedAt.Truncate(time.Second).UTC(),
		UpdatedAt:       raw.UpdatedAt.Truncate(time.Second).UTC(),
		StampList:       stampList,
	}
	return res
}

// リクエストで飛んできたmessageIDを検証する。存在する場合はそのメッセージを返す
func validateMessageID(messageID, userID string) (*model.Message, error) {
	m := &model.Message{ID: messageID}
	ok, err := m.Exists()
	if err != nil {
		log.Error(err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Cannot find message")
	}
	if !ok {
		return nil, echo.NewHTTPError(http.StatusNotFound, "Message is not found")
	}

	if _, err := validateChannelID(m.ChannelID, userID); err != nil {
		return nil, echo.NewHTTPError(http.StatusForbidden, "Message forbidden")
	}
	return m, nil
}
