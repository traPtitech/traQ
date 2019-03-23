package router

import (
	"github.com/gofrs/uuid"
	"github.com/labstack/gommon/log"
	"github.com/traPtitech/traQ/repository"
	"go.uber.org/zap"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/model"
)

// MessageForResponse クライアントに返す形のメッセージオブジェクト
type MessageForResponse struct {
	MessageID       uuid.UUID             `json:"messageId"`
	UserID          uuid.UUID             `json:"userId"`
	ParentChannelID uuid.UUID             `json:"parentChannelId"`
	Content         string                `json:"content"`
	CreatedAt       time.Time             `json:"createdAt"`
	UpdatedAt       time.Time             `json:"updatedAt"`
	Pin             bool                  `json:"pin"`
	Reported        bool                  `json:"reported"`
	StampList       []*model.MessageStamp `json:"stampList"`
}

// GetMessageByID GET /messages/:messageID
func (h *Handlers) GetMessageByID(c echo.Context) error {
	m := getMessageFromContext(c)
	return c.JSON(http.StatusOK, h.formatMessage(m))
}

// PutMessageByID PUT /messages/:messageID
func (h *Handlers) PutMessageByID(c echo.Context) error {
	userID := getRequestUserID(c)
	messageID := getRequestParamAsUUID(c, paramMessageID)
	m := getMessageFromContext(c)

	// 他人のテキストは編集できない
	if userID != m.UserID {
		return echo.NewHTTPError(http.StatusForbidden, "This is not your message")
	}

	req := struct {
		Text string `json:"text" validate:"required"`
	}{}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	if err := h.Repo.UpdateMessage(messageID, req.Text); err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}

// DeleteMessageByID DELETE /message/:messageID
func (h *Handlers) DeleteMessageByID(c echo.Context) error {
	userID := getRequestUserID(c)
	messageID := getRequestParamAsUUID(c, paramMessageID)
	m := getMessageFromContext(c)

	if m.UserID != userID {
		return echo.NewHTTPError(http.StatusForbidden, "you are not allowed to delete this message")
	}

	if err := h.Repo.DeleteMessage(messageID); err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}

// GetMessagesByChannelID GET /channels/:channelID/messages
func (h *Handlers) GetMessagesByChannelID(c echo.Context) error {
	req := struct {
		Limit  int `query:"limit"  validate:"min=0"`
		Offset int `query:"offset" validate:"min=0"`
	}{}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	userID := getRequestUserID(c)
	channelID := getRequestParamAsUUID(c, paramChannelID)

	messages, err := h.Repo.GetMessagesByChannelID(channelID, req.Limit, req.Offset)
	if err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	reports, err := h.Repo.GetMessageReportsByReporterID(userID)
	if err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError)
	}
	hidden := make(map[uuid.UUID]bool)
	for _, v := range reports {
		hidden[v.MessageID] = true
	}

	res := make([]*MessageForResponse, 0, req.Limit)
	for _, message := range messages {
		ms := h.formatMessage(message)
		if hidden[message.ID] {
			ms.Reported = true
		}
		res = append(res, ms)
	}

	return c.JSON(http.StatusOK, res)
}

// PostMessage POST /channels/:channelID/messages
func (h *Handlers) PostMessage(c echo.Context) error {
	post := struct {
		Text string `json:"text" validate:"required"`
	}{}
	if err := bindAndValidate(c, &post); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	userID := getRequestUserID(c)
	channelID := getRequestParamAsUUID(c, paramChannelID)

	m, err := h.createMessage(c, post.Text, userID, channelID)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusCreated, m)
}

// GetDirectMessages GET /users/:userId/messages
func (h *Handlers) GetDirectMessages(c echo.Context) error {
	req := struct {
		Limit  int `query:"limit"  validate:"min=0"`
		Offset int `query:"offset" validate:"min=0"`
	}{}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	myID := getRequestUserID(c)
	targetID := getRequestParamAsUUID(c, paramUserID)

	// DMチャンネルを取得
	ch, err := h.Repo.GetDirectMessageChannel(myID, targetID)
	if err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	// メッセージ取得
	messages, err := h.Repo.GetMessagesByChannelID(ch.ID, req.Limit, req.Offset)
	if err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	// 整形
	res := make([]*MessageForResponse, 0, req.Limit)
	for _, message := range messages {
		res = append(res, h.formatMessage(message))
	}

	return c.JSON(http.StatusOK, res)
}

// PostDirectMessage POST /users/:userId/messages
func (h *Handlers) PostDirectMessage(c echo.Context) error {
	req := struct {
		Text string `json:"text" validate:"required"`
	}{}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	myID := getRequestUserID(c)
	targetID := getRequestParamAsUUID(c, paramUserID)

	// DMチャンネルを取得
	ch, err := h.Repo.GetDirectMessageChannel(myID, targetID)
	if err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	// 投稿
	m, err := h.createMessage(c, req.Text, myID, ch.ID)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusCreated, m)
}

// PostMessageReport POST /messages/:messageID/report
func (h *Handlers) PostMessageReport(c echo.Context) error {
	userID := getRequestUserID(c)
	messageID := getRequestParamAsUUID(c, paramMessageID)

	req := struct {
		Reason string `json:"reason" validate:"max=100,required"`
	}{}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	if err := h.Repo.CreateMessageReport(messageID, userID, req.Reason); err != nil {
		if isMySQLDuplicatedRecordErr(err) {
			return echo.NewHTTPError(http.StatusBadRequest, "already reported")
		}
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}

// GetMessageReports GET /reports
func (h *Handlers) GetMessageReports(c echo.Context) error {
	p, _ := strconv.Atoi(c.QueryParam("p"))

	reports, err := h.Repo.GetMessageReports(p*50, 50)
	if err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, reports)
}

// GetUnread GET /users/me/unread
func (h *Handlers) GetUnread(c echo.Context) error {
	userID := getRequestUserID(c)

	unreads, err := h.Repo.GetUnreadMessagesByUserID(userID)
	if err != nil {
		c.Logger().Error()
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	responseBody := make([]*MessageForResponse, len(unreads))
	for i, v := range unreads {
		responseBody[i] = h.formatMessage(v)
	}

	return c.JSON(http.StatusOK, responseBody)
}

// DeleteUnread DELETE /users/me/unread/:channelID
func (h *Handlers) DeleteUnread(c echo.Context) error {
	userID := getRequestUserID(c)
	channelID := getRequestParamAsUUID(c, paramChannelID)

	if err := h.Repo.DeleteUnreadsByChannelID(channelID, userID); err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}

// dbにデータを入れる
func (h *Handlers) createMessage(c echo.Context, text string, userID, channelID uuid.UUID) (*MessageForResponse, error) {
	m, err := h.Repo.CreateMessage(userID, channelID, text)
	if err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return nil, echo.NewHTTPError(http.StatusInternalServerError)
	}
	return h.formatMessage(m), nil
}

func (h *Handlers) formatMessage(raw *model.Message) *MessageForResponse {
	isPinned, err := h.Repo.IsPinned(raw.ID)
	if err != nil {
		log.Error(err)
	}

	stampList, err := h.Repo.GetMessageStamps(raw.ID)
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
func (h *Handlers) validateMessageID(c echo.Context, messageID, userID uuid.UUID) (*model.Message, error) {
	m, err := h.Repo.GetMessageByID(messageID)
	if err != nil {
		switch err {
		case repository.ErrNotFound:
			return nil, echo.NewHTTPError(http.StatusNotFound, "Message is not found")
		default:
			h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
			return nil, echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	if ok, err := h.Repo.IsChannelAccessibleToUser(userID, m.ChannelID); err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return nil, echo.NewHTTPError(http.StatusInternalServerError)
	} else if !ok {
		return nil, echo.NewHTTPError(http.StatusNotFound)
	}

	return m, nil
}
