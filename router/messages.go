package router

import (
	"net/http"
	"strconv"
	"time"

	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/event"

	"github.com/labstack/echo"
	"github.com/labstack/gommon/log"
	"github.com/traPtitech/traQ/model"
)

// MessageForResponse クライアントに返す形のメッセージオブジェクト
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

// GetMessageByID GET /messages/:messageID
func GetMessageByID(c echo.Context) error {
	userID := getRequestUserID(c)
	messageID := getRequestParamAsUUID(c, paramMessageID)

	m, err := validateMessageID(c, messageID, userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound)
	}
	return c.JSON(http.StatusOK, formatMessage(m))
}

// PutMessageByID PUT /messages/:messageID
func PutMessageByID(c echo.Context) error {
	userID := getRequestUserID(c)
	messageID := getRequestParamAsUUID(c, paramMessageID)

	m, err := validateMessageID(c, messageID, userID)
	if err != nil {
		return err
	}
	// 他人のテキストは編集できない
	if userID != m.GetUID() {
		return echo.NewHTTPError(http.StatusForbidden, "This is not your message")
	}

	req := struct {
		Text string `json:"text" validate:"required"`
	}{}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	if err := model.UpdateMessage(messageID, req.Text); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	go event.Emit(event.MessageUpdated, &event.MessageUpdatedEvent{Message: *m})
	return c.JSON(http.StatusOK, formatMessage(m))
}

// DeleteMessageByID DELETE /message/:messageID
func DeleteMessageByID(c echo.Context) error {
	userID := getRequestUserID(c)
	messageID := getRequestParamAsUUID(c, paramMessageID)

	m, err := validateMessageID(c, messageID, userID)
	if err != nil {
		return err
	}
	if m.GetUID() != userID {
		return echo.NewHTTPError(http.StatusForbidden, "you are not allowed to delete this message")
	}

	if err := model.DeleteMessage(messageID); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	if err := model.DeleteUnreadsByMessageID(messageID); err != nil {
		c.Logger().Error(err) //500エラーにはしない
	}

	go event.Emit(event.MessageDeleted, &event.MessageDeletedEvent{Message: *m})
	return c.NoContent(http.StatusNoContent)
}

// GetMessagesByChannelID GET /channels/:channelID/messages
func GetMessagesByChannelID(c echo.Context) error {
	req := struct {
		Limit  int `query:"limit"  validate:"min=0"`
		Offset int `query:"offset" validate:"min=0"`
	}{}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	userID := getRequestUserID(c)
	channelID := getRequestParamAsUUID(c, paramChannelID)

	if ok, err := model.IsChannelAccessibleToUser(userID, channelID); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	} else if !ok {
		return echo.NewHTTPError(http.StatusNotFound)
	}

	messages, err := model.GetMessagesByChannelID(channelID, req.Limit, req.Offset)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	reports, err := model.GetMessageReportsByReporterID(userID)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}
	hidden := make(map[string]bool)
	for _, v := range reports {
		hidden[v.MessageID] = true
	}

	res := make([]*MessageForResponse, 0, req.Limit)
	for _, message := range messages {
		ms := formatMessage(message)
		if hidden[message.ID] {
			ms.Reported = true
		}
		res = append(res, ms)
	}

	return c.JSON(http.StatusOK, res)
}

// PostMessage POST /channels/:channelID/messages
func PostMessage(c echo.Context) error {
	post := struct {
		Text string `json:"text" validate:"required"`
	}{}
	if err := bindAndValidate(c, &post); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	userID := getRequestUserID(c)
	channelID := getRequestParamAsUUID(c, paramChannelID)

	if ok, err := model.IsChannelAccessibleToUser(userID, channelID); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	} else if !ok {
		return echo.NewHTTPError(http.StatusNotFound)
	}

	m, err := createMessage(c, post.Text, userID, channelID)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusCreated, m)
}

// GetDirectMessages GET /users/:userId/messages
func GetDirectMessages(c echo.Context) error {
	req := struct {
		Limit  int `query:"limit"  validate:"min=0"`
		Offset int `query:"offset" validate:"min=0"`
	}{}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	myID := getRequestUserID(c)
	targetID := getRequestParamAsUUID(c, paramUserID)

	// ユーザー確認
	if ok, err := model.UserExists(targetID); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	} else if !ok {
		return echo.NewHTTPError(http.StatusNotFound)
	}

	// DMチャンネルを取得
	ch, err := model.GetOrCreateDirectMessageChannel(myID, targetID)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	// メッセージ取得
	messages, err := model.GetMessagesByChannelID(ch.GetCID(), req.Limit, req.Offset)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	// 整形
	res := make([]*MessageForResponse, 0, req.Limit)
	for _, message := range messages {
		res = append(res, formatMessage(message))
	}

	return c.JSON(http.StatusOK, res)
}

// PostDirectMessage POST /users/:userId/messages
func PostDirectMessage(c echo.Context) error {
	req := struct {
		Text string `json:"text" validate:"required"`
	}{}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	myID := getRequestUserID(c)
	targetID := getRequestParamAsUUID(c, paramUserID)

	// ユーザー確認
	if ok, err := model.UserExists(targetID); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	} else if !ok {
		return echo.NewHTTPError(http.StatusNotFound)
	}

	// DMチャンネルを取得
	ch, err := model.GetOrCreateDirectMessageChannel(myID, targetID)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	// 投稿
	m, err := createMessage(c, req.Text, myID, ch.GetCID())
	if err != nil {
		return err
	}

	return c.JSON(http.StatusCreated, m)
}

// PostMessageReport POST /messages/:messageID/report
func PostMessageReport(c echo.Context) error {
	userID := getRequestUserID(c)
	messageID := getRequestParamAsUUID(c, paramMessageID)

	req := struct {
		Reason string `json:"reason" validate:"max=100,required"`
	}{}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	_, err := validateMessageID(c, messageID, userID)
	if err != nil {
		return err
	}

	if err := model.CreateMessageReport(messageID, userID, req.Reason); err != nil {
		if isMySQLDuplicatedRecordErr(err) {
			return echo.NewHTTPError(http.StatusBadRequest, "already reported")
		}
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}

// GetMessageReports GET /reports
func GetMessageReports(c echo.Context) error {
	p, _ := strconv.Atoi(c.QueryParam("p"))

	reports, err := model.GetMessageReports(p*50, 50)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, reports)
}

// dbにデータを入れる
func createMessage(c echo.Context, text string, userID, channelID uuid.UUID) (*MessageForResponse, error) {
	m, err := model.CreateMessage(userID, channelID, text)
	if err != nil {
		c.Logger().Error(err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError)
	}

	go event.Emit(event.MessageCreated, &event.MessageCreatedEvent{Message: *m})
	return formatMessage(m), nil
}

func formatMessage(raw *model.Message) *MessageForResponse {
	isPinned, err := model.IsPinned(raw.GetID())
	if err != nil {
		log.Error(err)
	}

	stampList, err := model.GetMessageStamps(raw.GetID())
	if err != nil {
		log.Error(err)
	}

	res := &MessageForResponse{
		MessageID:       raw.ID,
		UserID:          raw.UserID,
		ParentChannelID: raw.ChannelID,
		Pin:             isPinned,
		Content:         raw.Text,
		CreatedAt:       raw.CreatedAt,
		UpdatedAt:       raw.UpdatedAt,
		StampList:       stampList,
	}
	return res
}

// リクエストで飛んできたmessageIDを検証する。存在する場合はそのメッセージを返す
func validateMessageID(c echo.Context, messageID, userID uuid.UUID) (*model.Message, error) {
	m, err := model.GetMessageByID(messageID)
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return nil, echo.NewHTTPError(http.StatusNotFound, "Message is not found")
		default:
			c.Logger().Error(err)
			return nil, echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	if ok, err := model.IsChannelAccessibleToUser(userID, m.GetCID()); err != nil {
		c.Logger().Error(err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError)
	} else if !ok {
		return nil, echo.NewHTTPError(http.StatusNotFound)
	}

	return m, nil
}
