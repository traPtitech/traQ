package v3

import (
	"fmt"
	"net/http"

	vd "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/labstack/echo/v4"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/service/message"
	"github.com/traPtitech/traQ/service/search"
)

// GetMyUnreadChannels GET /users/me/unread
func (h *Handlers) GetMyUnreadChannels(c echo.Context) error {
	userID := getRequestUserID(c)

	list, err := h.Repo.GetUserUnreadChannels(userID)
	if err != nil {
		return herror.InternalServerError(err)
	}

	return c.JSON(http.StatusOK, list)
}

// ReadChannel DELETE /users/me/unread/:channelID
func (h *Handlers) ReadChannel(c echo.Context) error {
	userID := getRequestUserID(c)
	channelID := getParamAsUUID(c, consts.ParamChannelID)

	if err := h.Repo.DeleteUnreadsByChannelID(channelID, userID); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}

// SearchMessages GET /messages
func (h *Handlers) SearchMessages(c echo.Context) error {
	if !h.SearchEngine.Available() {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "search service is currently unavailable")
	}

	var q search.Query
	if err := bindAndValidate(c, &q); err != nil {
		return err
	}

	if q.In.Valid {
		// ユーザーが該当チャンネルへのアクセス権限があるかを確認
		ok, err := h.ChannelManager.IsChannelAccessibleToUser(getRequestUserID(c), q.In.V)
		if err != nil {
			return herror.InternalServerError(err)
		}
		if !ok {
			return herror.Forbidden("invalid channelId")
		}
	}

	// 仮置き
	r, err := h.SearchEngine.Do(&q)
	if err != nil {
		return herror.InternalServerError(err)
	}

	type res struct {
		TotalHits int64             `json:"totalHits"`
		Hits      []message.Message `json:"hits"`
	}
	response := res{
		TotalHits: r.TotalHits(),
		Hits:      r.Hits(),
	}
	return c.JSON(http.StatusOK, response)
}

// GetMessage GET /messages/:messageID
func (h *Handlers) GetMessage(c echo.Context) error {
	userID := getRequestUserID(c)
	m := getParamMessage(c)

	// メッセージアクセス権確認
	if ok, err := h.MessageManager.IsAccessible(m, userID); err != nil {
		if err == message.ErrNotFound {
			return herror.NotFound()
		}
		return herror.InternalServerError(err)
	} else if !ok {
		return herror.NotFound()
	}

	return c.JSON(http.StatusOK, m)
}

// PostMessageRequest POST /channels/:channelID/messages等リクエストボディ
type PostMessageRequest struct {
	Content string `json:"content"`
	Embed   bool   `json:"embed" query:"embed"`
}

func (r PostMessageRequest) Validate() error {
	return vd.ValidateStruct(&r,
		vd.Field(&r.Content, vd.Required, vd.RuneLength(1, 10000)),
	)
}

// EditMessage PUT /messages/:messageID
func (h *Handlers) EditMessage(c echo.Context) error {
	userID := getRequestUserID(c)
	m := getParamMessage(c)

	// メッセージアクセス権確認
	if ok, err := h.MessageManager.IsAccessible(m, userID); err != nil {
		if err == message.ErrNotFound {
			return herror.NotFound()
		}
		return herror.InternalServerError(err)
	} else if !ok {
		return herror.NotFound()
	}

	var req PostMessageRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	// 他人のテキストは編集できない
	if userID != m.GetUserID() {
		return herror.Forbidden("This is not your message")
	}

	if req.Embed {
		req.Content = h.Replacer.Replace(req.Content)
	}

	if err := h.MessageManager.Edit(m.GetID(), req.Content); err != nil {
		switch err {
		case message.ErrChannelArchived:
			return herror.BadRequest("the channel of this message has been archived")
		default:
			return herror.InternalServerError(err)
		}
	}
	return c.NoContent(http.StatusNoContent)
}

// DeleteMessage DELETE /messages/:messageID
func (h *Handlers) DeleteMessage(c echo.Context) error {
	userID := getRequestUserID(c)
	m := getParamMessage(c)

	// メッセージアクセス権確認
	if ok, err := h.MessageManager.IsAccessible(m, userID); err != nil {
		if err == message.ErrNotFound {
			return herror.NotFound()
		}
		return herror.InternalServerError(err)
	} else if !ok {
		return herror.NotFound()
	}

	if muid := m.GetUserID(); muid != userID {
		mUser, err := h.Repo.GetUser(muid, false)
		if err != nil {
			return herror.InternalServerError(err)
		}

		switch mUser.GetUserType() {
		case model.UserTypeHuman:
			return herror.Forbidden("you are not allowed to delete this message")
		case model.UserTypeBot:
			// BOTのメッセージの削除権限の確認
			wh, err := h.Repo.GetBotByBotUserID(mUser.GetID())
			if err != nil {
				switch err {
				case repository.ErrNotFound: // deleted bot
					return herror.Forbidden("you are not allowed to delete this message")
				default:
					return herror.InternalServerError(err)
				}
			}

			if wh.CreatorID != userID {
				return herror.Forbidden("you are not allowed to delete this message")
			}
		case model.UserTypeWebhook:
			// Webhookのメッセージの削除権限の確認
			wh, err := h.Repo.GetWebhookByBotUserID(mUser.GetID())
			if err != nil {
				switch err {
				case repository.ErrNotFound: // deleted webhook
					return herror.Forbidden("you are not allowed to delete this message")
				default:
					return herror.InternalServerError(err)
				}
			}

			if wh.GetCreatorID() != userID {
				return herror.Forbidden("you are not allowed to delete this message")
			}
		}
	}

	if err := h.MessageManager.Delete(m.GetID()); err != nil {
		switch err {
		case message.ErrChannelArchived:
			return herror.BadRequest("the channel of this message has been archived")
		default:
			return herror.InternalServerError(err)
		}
	}
	return c.NoContent(http.StatusNoContent)
}

// GetPin GET /messages/:messageID/pin
func (h *Handlers) GetPin(c echo.Context) error {
	userID := getRequestUserID(c)
	m := getParamMessage(c)

	// メッセージアクセス権確認
	if ok, err := h.MessageManager.IsAccessible(m, userID); err != nil {
		if err == message.ErrNotFound {
			return herror.NotFound()
		}
		return herror.InternalServerError(err)
	} else if !ok {
		return herror.NotFound()
	}

	if m.GetPin() == nil {
		return herror.NotFound("this message is not pinned")
	}
	return c.JSON(http.StatusOK, formatMessagePin(m.GetPin()))
}

// CreatePin POST /messages/:messageID/pin
func (h *Handlers) CreatePin(c echo.Context) error {
	userID := getRequestUserID(c)
	m := getParamMessage(c)

	// メッセージアクセス権確認
	if ok, err := h.MessageManager.IsAccessible(m, userID); err != nil {
		if err == message.ErrNotFound {
			return herror.NotFound()
		}
		return herror.InternalServerError(err)
	} else if !ok {
		return herror.NotFound()
	}

	p, err := h.MessageManager.Pin(m.GetID(), userID)
	if err != nil {
		switch err {
		case message.ErrAlreadyExists:
			return herror.BadRequest("this message has already been pinned")
		case message.ErrChannelArchived:
			return herror.BadRequest("the channel of this message has been archived")
		case message.ErrPinLimitExceeded:
			return herror.BadRequest(fmt.Sprintf("cannot pin more than %d messages", message.PinLimit))
		default:
			return herror.InternalServerError(err)
		}
	}
	return c.JSON(http.StatusCreated, formatMessagePin(p))
}

// RemovePin DELETE /messages/:messageID/pin
func (h *Handlers) RemovePin(c echo.Context) error {
	userID := getRequestUserID(c)
	m := getParamMessage(c)

	// メッセージアクセス権確認
	if ok, err := h.MessageManager.IsAccessible(m, userID); err != nil {
		if err == message.ErrNotFound {
			return herror.NotFound()
		}
		return herror.InternalServerError(err)
	} else if !ok {
		return herror.NotFound()
	}

	if err := h.MessageManager.Unpin(m.GetID(), userID); err != nil {
		switch err {
		case message.ErrNotFound:
			return herror.NotFound("pin was not found")
		case message.ErrChannelArchived:
			return herror.BadRequest("the channel of this message has been archived")
		default:
			return herror.InternalServerError(err)
		}
	}
	return c.NoContent(http.StatusNoContent)
}

// GetMessageStamps GET /messages/:messageID/stamps
func (h *Handlers) GetMessageStamps(c echo.Context) error {
	userID := getRequestUserID(c)
	m := getParamMessage(c)

	// メッセージアクセス権確認
	if ok, err := h.MessageManager.IsAccessible(m, userID); err != nil {
		if err == message.ErrNotFound {
			return herror.NotFound()
		}
		return herror.InternalServerError(err)
	} else if !ok {
		return herror.NotFound()
	}

	return c.JSON(http.StatusOK, m.GetStamps())
}

// PostMessageStampRequest POST /messages/:messageID/stamps/:stampID リクエストボディ
type PostMessageStampRequest struct {
	Count int `json:"count"`
}

func (r *PostMessageStampRequest) Validate() error {
	if r.Count == 0 {
		r.Count = 1
	}
	return vd.ValidateStruct(r,
		vd.Field(&r.Count, vd.Required, vd.Min(1), vd.Max(100)),
	)
}

// AddMessageStamp POST /messages/:messageID/stamps/:stampID
func (h *Handlers) AddMessageStamp(c echo.Context) error {
	var req PostMessageStampRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	userID := getRequestUserID(c)
	m := getParamMessage(c)
	stampID := getParamAsUUID(c, consts.ParamStampID)

	// メッセージアクセス権確認
	if ok, err := h.MessageManager.IsAccessible(m, userID); err != nil {
		if err == message.ErrNotFound {
			return herror.NotFound()
		}
		return herror.InternalServerError(err)
	} else if !ok {
		return herror.NotFound()
	}

	// スタンプをメッセージに押す
	if _, err := h.MessageManager.AddStamps(m.GetID(), stampID, userID, req.Count); err != nil {
		switch err {
		case message.ErrChannelArchived:
			return herror.BadRequest("the channel of this message has been archived")
		default:
			return herror.InternalServerError(err)
		}
	}

	return c.NoContent(http.StatusNoContent)
}

// RemoveMessageStamp DELETE /messages/:messageID/stamps/:stampID
func (h *Handlers) RemoveMessageStamp(c echo.Context) error {
	userID := getRequestUserID(c)
	m := getParamMessage(c)
	stampID := getParamAsUUID(c, consts.ParamStampID)

	// メッセージアクセス権確認
	if ok, err := h.MessageManager.IsAccessible(m, userID); err != nil {
		if err == message.ErrNotFound {
			return herror.NotFound()
		}
		return herror.InternalServerError(err)
	} else if !ok {
		return herror.NotFound()
	}

	// スタンプをメッセージから削除
	if err := h.MessageManager.RemoveStamps(m.GetID(), stampID, userID); err != nil {
		switch err {
		case message.ErrChannelArchived:
			return herror.BadRequest("the channel of this message has been archived")
		default:
			return herror.InternalServerError(err)
		}
	}

	return c.NoContent(http.StatusNoContent)
}

// GetMessageClips GET /messages/:messageID/clips
func (h *Handlers) GetMessageClips(c echo.Context) error {
	userID := getRequestUserID(c)
	m := getParamMessage(c)

	// メッセージアクセス権確認
	if ok, err := h.MessageManager.IsAccessible(m, userID); err != nil {
		if err == message.ErrNotFound {
			return herror.NotFound()
		}
		return herror.InternalServerError(err)
	} else if !ok {
		return herror.NotFound()
	}

	clips, err := h.Repo.GetMessageClips(userID, m.GetID())
	if err != nil {
		return herror.InternalServerError(err)
	}

	return c.JSON(http.StatusOK, formatMessageClips(clips))
}

// GetMessages GET /channels/:channelID/messages
func (h *Handlers) GetMessages(c echo.Context) error {
	channelID := getParamAsUUID(c, consts.ParamChannelID)

	var req MessagesQuery
	if err := req.bind(c); err != nil {
		return err
	}

	return serveMessages(c, h.MessageManager, req.convertC(channelID))
}

// PostMessage POST /channels/:channelID/messages
func (h *Handlers) PostMessage(c echo.Context) error {
	userID := getRequestUserID(c)
	ch := getParamChannel(c)

	var req PostMessageRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	if req.Embed {
		req.Content = h.Replacer.Replace(req.Content)
	}

	m, err := h.MessageManager.Create(ch.ID, userID, req.Content)
	if err != nil {
		switch err {
		case message.ErrChannelArchived:
			return herror.BadRequest("this channel has been archived")
		default:
			return herror.InternalServerError(err)
		}
	}
	return c.JSON(http.StatusCreated, m)
}

// GetDirectMessages GET /users/:userId/messages
func (h *Handlers) GetDirectMessages(c echo.Context) error {
	myID := getRequestUserID(c)
	targetID := getParamAsUUID(c, consts.ParamUserID)

	var req MessagesQuery
	if err := req.bind(c); err != nil {
		return err
	}

	// DMチャンネルを取得
	ch, err := h.ChannelManager.GetDMChannel(myID, targetID)
	if err != nil {
		return herror.InternalServerError(err)
	}

	return serveMessages(c, h.MessageManager, req.convertC(ch.ID))
}

// PostDirectMessage POST /users/:userId/messages
func (h *Handlers) PostDirectMessage(c echo.Context) error {
	myID := getRequestUserID(c)
	targetID := getParamAsUUID(c, consts.ParamUserID)

	var req PostMessageRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	if req.Embed {
		req.Content = h.Replacer.Replace(req.Content)
	}

	m, err := h.MessageManager.CreateDM(myID, targetID, req.Content)
	if err != nil {
		return herror.InternalServerError(err)
	}
	return c.JSON(http.StatusCreated, m)
}
