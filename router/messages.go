package router

import (
	"net/http"
	"strconv"
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
	user := c.Get("user").(*model.User)
	messageID := c.Param("messageID")
	m, err := validateMessageID(uuid.FromStringOrNil(messageID), user.GetUID())
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound)
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
	userID := c.Get("user").(*model.User).GetUID()
	res, err := getMessages(uuid.FromStringOrNil(c.Param("channelID")), userID, queryParam.Limit, queryParam.Offset)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, res)
}

// PostMessage POST /channels/{channelID}/messages のハンドラ
func PostMessage(c echo.Context) error {
	// 100KB制限
	if c.Request().ContentLength > 100*1024 {
		return echo.NewHTTPError(http.StatusRequestEntityTooLarge, "a request must be smaller than 100KB")
	}

	post := struct {
		Text string `json:"text" validate:"required"`
	}{}
	if err := bindAndValidate(c, &post); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	userID := c.Get("user").(*model.User).GetUID()
	channelID := uuid.FromStringOrNil(c.Param("channelID"))

	ch, err := validateChannelID(channelID, userID)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}
	if ch == nil {
		return echo.NewHTTPError(http.StatusNotFound)
	}

	m, err := createMessage(c, post.Text, userID, channelID)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusCreated, m)
}

// PutMessageByID PUT /messages/{messageID}のハンドラ
func PutMessageByID(c echo.Context) error {
	user := c.Get("user").(*model.User)
	m, err := validateMessageID(uuid.FromStringOrNil(c.Param("messageID")), user.GetUID())
	if err != nil {
		return err
	}

	// 他人のテキストは編集できない
	if user.ID != m.UserID {
		return echo.NewHTTPError(http.StatusForbidden, "This is not your message")
	}

	req := struct {
		Text string `json:"text" validate:"required"`
	}{}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	if err := model.UpdateMessage(m.GetID(), req.Text); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	go event.Emit(event.MessageUpdated, &event.MessageUpdatedEvent{Message: *m})
	return c.JSON(http.StatusOK, formatMessage(m))
}

// DeleteMessageByID : DELETE /message/{messageID} のハンドラ
func DeleteMessageByID(c echo.Context) error {
	user := c.Get("user").(*model.User)
	messageID := uuid.FromStringOrNil(c.Param("messageID"))

	m, err := validateMessageID(messageID, user.GetUID())
	if err != nil {
		return err
	}
	if m.UserID != user.ID {
		return echo.NewHTTPError(http.StatusForbidden, "you are not allowed to delete this message")
	}

	if err := model.DeleteMessage(m.GetID()); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	if err := model.DeleteUnreadsByMessageID(m.GetID()); err != nil {
		c.Logger().Error(err) //500エラーにはしない
	}

	go event.Emit(event.MessageDeleted, &event.MessageDeletedEvent{Message: *m})
	return c.NoContent(http.StatusNoContent)
}

// PostMessageReport POST /messages/{messageID}/report
func PostMessageReport(c echo.Context) error {
	user := c.Get("user").(*model.User)
	messageID := uuid.FromStringOrNil(c.Param("messageID"))

	req := struct {
		Reason string `json:"reason" validate:"max=100,required"`
	}{}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	m, err := validateMessageID(messageID, user.GetUID())
	if err != nil {
		return err
	}

	if err := model.CreateMessageReport(m.GetID(), user.GetUID(), req.Reason); err != nil {
		switch e := err.(type) {
		case *validator.ValidationErrors:
			return echo.NewHTTPError(http.StatusBadRequest, err)
		case *mysql.MySQLError:
			if isMySQLDuplicatedRecordErr(e) {
				return echo.NewHTTPError(http.StatusBadRequest, "already reported")
			}
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
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

// チャンネルのデータを取得する
func getMessages(channelID, userID uuid.UUID, limit, offset int) ([]*MessageForResponse, error) {
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

	reports, err := model.GetMessageReportsByReporterID(userID)
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
func validateMessageID(messageID, userID uuid.UUID) (*model.Message, error) {
	m, err := model.GetMessageByID(messageID)
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return nil, echo.NewHTTPError(http.StatusNotFound, "Message is not found")
		}
		log.Error(err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Cannot find message")
	}

	if _, err := validateChannelID(m.GetCID(), userID); err != nil {
		return nil, echo.NewHTTPError(http.StatusForbidden, "Message forbidden")
	}
	return m, nil
}
