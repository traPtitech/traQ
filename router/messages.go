package router

import (
	"fmt"
	"github.com/gofrs/uuid"
	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/repository"
	"gopkg.in/guregu/null.v3"
	"net/http"
	"strconv"
	"time"
)

// GetMessageByID GET /messages/:messageID
func (h *Handlers) GetMessageByID(c echo.Context) error {
	m := getMessageFromContext(c)
	return c.JSON(http.StatusOK, formatMessage(m))
}

// PutMessageByID PUT /messages/:messageID
func (h *Handlers) PutMessageByID(c echo.Context) error {
	userID := getRequestUserID(c)
	messageID := getRequestParamAsUUID(c, paramMessageID)
	m := getMessageFromContext(c)

	var req struct {
		Text string `json:"text"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return badRequest(err)
	}

	// 他人のテキストは編集できない
	if userID != m.UserID {
		return forbidden("This is not your message")
	}

	if err := h.Repo.UpdateMessage(messageID, req.Text); err != nil {
		switch {
		case repository.IsArgError(err):
			return badRequest(err)
		default:
			return internalServerError(err, h.requestContextLogger(c))
		}
	}

	return c.NoContent(http.StatusNoContent)
}

// DeleteMessageByID DELETE /message/:messageID
func (h *Handlers) DeleteMessageByID(c echo.Context) error {
	userID := getRequestUserID(c)
	messageID := getRequestParamAsUUID(c, paramMessageID)
	m := getMessageFromContext(c)

	if m.UserID != userID {
		mUser, err := h.Repo.GetUser(m.UserID)
		if err != nil {
			return internalServerError(err, h.requestContextLogger(c))
		}

		if !mUser.Bot {
			return forbidden("you are not allowed to delete this message")
		}

		// Webhookのメッセージの削除権限の確認
		wh, err := h.Repo.GetWebhookByBotUserID(mUser.ID)
		if err != nil {
			switch err {
			case repository.ErrNotFound:
				return forbidden("you are not allowed to delete this message")
			default:
				return internalServerError(err, h.requestContextLogger(c))
			}
		}

		if wh.GetCreatorID() != userID {
			return forbidden("you are not allowed to delete this message")
		}
	}

	if err := h.Repo.DeleteMessage(messageID); err != nil {
		return internalServerError(err, h.requestContextLogger(c))
	}

	return c.NoContent(http.StatusNoContent)
}

// GetMessagesByChannelID GET /channels/:channelID/messages
func (h *Handlers) GetMessagesByChannelID(c echo.Context) error {
	channelID := getRequestParamAsUUID(c, paramChannelID)

	var req messagesQuery
	if err := req.bind(c); err != nil {
		return badRequest(err)
	}

	return h.getMessages(c, req.convertC(channelID), true)
}

// PostMessage POST /channels/:channelID/messages
func (h *Handlers) PostMessage(c echo.Context) error {
	userID := getRequestUserID(c)
	channelID := getRequestParamAsUUID(c, paramChannelID)

	var req struct {
		Text string `json:"text"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return badRequest(err)
	}

	m, err := h.Repo.CreateMessage(userID, channelID, req.Text)
	if err != nil {
		switch {
		case repository.IsArgError(err):
			return badRequest(err)
		default:
			return internalServerError(err, h.requestContextLogger(c))
		}
	}

	return c.JSON(http.StatusCreated, formatMessage(m))
}

// GetDirectMessages GET /users/:userId/messages
func (h *Handlers) GetDirectMessages(c echo.Context) error {
	myID := getRequestUserID(c)
	targetID := getRequestParamAsUUID(c, paramUserID)

	var req messagesQuery
	if err := req.bind(c); err != nil {
		return badRequest(err)
	}

	// DMチャンネルを取得
	ch, err := h.Repo.GetDirectMessageChannel(myID, targetID)
	if err != nil {
		return internalServerError(err, h.requestContextLogger(c))
	}

	return h.getMessages(c, req.convertC(ch.ID), false)
}

// PostDirectMessage POST /users/:userId/messages
func (h *Handlers) PostDirectMessage(c echo.Context) error {
	myID := getRequestUserID(c)
	targetID := getRequestParamAsUUID(c, paramUserID)

	var req struct {
		Text string `json:"text"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return badRequest(err)
	}

	// DMチャンネルを取得
	ch, err := h.Repo.GetDirectMessageChannel(myID, targetID)
	if err != nil {
		return internalServerError(err, h.requestContextLogger(c))
	}

	m, err := h.Repo.CreateMessage(myID, ch.ID, req.Text)
	if err != nil {
		switch {
		case repository.IsArgError(err):
			return badRequest(err)
		default:
			return internalServerError(err, h.requestContextLogger(c))
		}
	}

	return c.JSON(http.StatusCreated, formatMessage(m))
}

// PostMessageReport POST /messages/:messageID/report
func (h *Handlers) PostMessageReport(c echo.Context) error {
	userID := getRequestUserID(c)
	messageID := getRequestParamAsUUID(c, paramMessageID)

	var req struct {
		Reason string `json:"reason"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return badRequest(err)
	}

	if len(req.Reason) == 0 {
		return badRequest("reason is required")
	}

	if err := h.Repo.CreateMessageReport(messageID, userID, req.Reason); err != nil {
		switch err {
		case repository.ErrAlreadyExists:
			return badRequest("already reported")
		default:
			return internalServerError(err, h.requestContextLogger(c))
		}
	}
	return c.NoContent(http.StatusNoContent)
}

// GetMessageReports GET /reports
func (h *Handlers) GetMessageReports(c echo.Context) error {
	p, _ := strconv.Atoi(c.QueryParam("p"))

	reports, err := h.Repo.GetMessageReports(p*50, 50)
	if err != nil {
		return internalServerError(err, h.requestContextLogger(c))
	}

	return c.JSON(http.StatusOK, reports)
}

// DeleteUnread DELETE /users/me/unread/channels/:channelID
func (h *Handlers) DeleteUnread(c echo.Context) error {
	userID := getRequestUserID(c)
	channelID := getRequestParamAsUUID(c, paramChannelID)

	if err := h.Repo.DeleteUnreadsByChannelID(channelID, userID); err != nil {
		return internalServerError(err, h.requestContextLogger(c))
	}

	return c.NoContent(http.StatusNoContent)
}

// GetUnreadChannels GET /users/me/unread/channels
func (h *Handlers) GetUnreadChannels(c echo.Context) error {
	userID := getRequestUserID(c)

	list, err := h.Repo.GetUserUnreadChannels(userID)
	if err != nil {
		return internalServerError(err, h.requestContextLogger(c))
	}

	return c.JSON(http.StatusOK, list)
}

type messagesQuery struct {
	Limit     int        `query:"limit"`
	Offset    int        `query:"offset"`
	Since     *timestamp `query:"since"`
	Until     *timestamp `query:"until"`
	Inclusive bool       `query:"inclusive"`
}

func (q *messagesQuery) bind(c echo.Context) error {
	return bindAndValidate(c, q)
}

func (q *messagesQuery) convert() repository.MessagesQuery {
	return repository.MessagesQuery{
		Since:     null.TimeFromPtr((*time.Time)(q.Since)),
		Until:     null.TimeFromPtr((*time.Time)(q.Until)),
		Inclusive: q.Inclusive,
		Limit:     q.Limit,
		Offset:    q.Offset,
	}
}

func (q *messagesQuery) convertC(cid uuid.UUID) repository.MessagesQuery {
	r := q.convert()
	r.Channel = cid
	return r
}

func (q *messagesQuery) convertU(uid uuid.UUID) repository.MessagesQuery {
	r := q.convert()
	r.User = uid
	return r
}

func (h *Handlers) getMessages(c echo.Context, query repository.MessagesQuery, filterByReport bool) error {
	var (
		res  []*messageResponse
		more bool
	)

	if query.Limit > 200 || query.Limit == 0 {
		query.Limit = 200 // １度に取れるのは200メッセージまで
	}

	// TODO singleflightを使うべき所を精査する
	if query.Until.Valid || query.Since.Valid || query.User != uuid.Nil {
		messages, _more, err := h.Repo.GetMessages(query)
		if err != nil {
			return internalServerError(err, h.requestContextLogger(c))
		}
		res = formatMessages(messages)
		more = _more
	} else {
		type sRes struct {
			Messages []*messageResponse
			More     bool
		}

		resI, err, _ := h.messagesResponseCacheGroup.Do(fmt.Sprintf("%s/%d/%d", query.Channel, query.Limit, query.Offset), func() (interface{}, error) {
			messages, more, err := h.Repo.GetMessages(query)
			return sRes{Messages: formatMessages(messages), More: more}, err
		})
		if err != nil {
			return internalServerError(err, h.requestContextLogger(c))
		}
		res = resI.(sRes).Messages
		more = resI.(sRes).More
	}

	if filterByReport {
		userID := getRequestUserID(c)

		reports, err := h.Repo.GetMessageReportsByReporterID(userID)
		if err != nil {
			return internalServerError(err, h.requestContextLogger(c))
		}
		hidden := make(map[uuid.UUID]bool)
		for _, v := range reports {
			hidden[v.MessageID] = true
		}
		for _, v := range res {
			v.Reported = hidden[v.MessageID]
		}
	}

	c.Response().Header().Set(headerMore, strconv.FormatBool(more))
	return c.JSON(http.StatusOK, res)
}
