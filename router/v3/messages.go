package v3

import (
	"fmt"
	vd "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/service/search"
	"github.com/traPtitech/traQ/utils/message"
	"net/http"
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

	// TODO SearchEngineで検索処理
	var q search.Query
	if err := bindAndValidate(c, &q); err != nil {
		return err
	}

	fmt.Println(q.Word)
	fmt.Println(q.After)

	// 仮置き
	r, err := h.SearchEngine.Do(&q)
	if err != nil {
		return herror.InternalServerError(err)
	}

	return c.JSON(http.StatusOK, r.Get())
}

// GetMessage GET /messages/:messageID
func (h *Handlers) GetMessage(c echo.Context) error {
	return c.JSON(http.StatusOK, formatMessage(getParamMessage(c)))
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

	// 投稿先チャンネル確認
	ch, err := h.Repo.GetChannel(m.ChannelID)
	if err != nil {
		return herror.InternalServerError(err)
	}
	if ch.IsArchived() {
		return herror.BadRequest(fmt.Sprintf("channel #%s has been archived", h.Repo.GetPublicChannelTree().GetChannelPath(ch.ID)))
	}

	var req PostMessageRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	// 他人のテキストは編集できない
	if userID != m.UserID {
		return herror.Forbidden("This is not your message")
	}

	if req.Embed {
		req.Content = message.NewReplacer(h.Repo).Replace(req.Content)
	}

	if err := h.Repo.UpdateMessage(m.ID, req.Content); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}

// DeleteMessage DELETE /messages/:messageID
func (h *Handlers) DeleteMessage(c echo.Context) error {
	userID := getRequestUserID(c)
	m := getParamMessage(c)

	if m.UserID != userID {
		mUser, err := h.Repo.GetUser(m.UserID, false)
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
				case repository.ErrNotFound:
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
				case repository.ErrNotFound:
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

	// 投稿先チャンネル確認
	ch, err := h.Repo.GetChannel(m.ChannelID)
	if err != nil {
		return herror.InternalServerError(err)
	}
	if ch.IsArchived() {
		return herror.BadRequest(fmt.Sprintf("channel #%s has been archived", h.Repo.GetPublicChannelTree().GetChannelPath(ch.ID)))
	}

	if err := h.Repo.DeleteMessage(m.ID); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}

// GetPin GET /messages/:messageID/pin
func (h *Handlers) GetPin(c echo.Context) error {
	m := getParamMessage(c)
	if m.Pin == nil {
		return herror.NotFound("this message is not pinned")
	}
	return c.JSON(http.StatusOK, formatMessagePin(m.Pin))
}

// CreatePin POST /messages/:messageID/pin
func (h *Handlers) CreatePin(c echo.Context) error {
	m := getParamMessage(c)
	if m.Pin != nil {
		return herror.BadRequest("this message has already been pinned")
	}

	p, err := h.Repo.PinMessage(m.ID, getRequestUserID(c))
	if err != nil {
		return herror.InternalServerError(err)
	}
	return c.JSON(http.StatusCreated, formatMessagePin(p))
}

// RemovePin DELETE /messages/:messageID/pin
func (h *Handlers) RemovePin(c echo.Context) error {
	m := getParamMessage(c)
	if m.Pin == nil {
		return herror.NotFound("this message is not pinned")
	}

	if err := h.Repo.UnpinMessage(m.ID, getRequestUserID(c)); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}

// GetMessageStamps GET /messages/:messageID/stamps
func (h *Handlers) GetMessageStamps(c echo.Context) error {
	return c.JSON(http.StatusOK, getParamMessage(c).Stamps)
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
	messageID := getParamAsUUID(c, consts.ParamMessageID)
	stampID := getParamAsUUID(c, consts.ParamStampID)

	// スタンプをメッセージに押す
	if _, err := h.Repo.AddStampToMessage(messageID, stampID, userID, req.Count); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}

// RemoveMessageStamp DELETE /messages/:messageID/stamps/:stampID
func (h *Handlers) RemoveMessageStamp(c echo.Context) error {
	userID := getRequestUserID(c)
	messageID := getParamAsUUID(c, consts.ParamMessageID)
	stampID := getParamAsUUID(c, consts.ParamStampID)

	// スタンプをメッセージから削除
	if err := h.Repo.RemoveStampFromMessage(messageID, stampID, userID); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}

// GetMessageClips GET /messages/:messageID/clips
func (h *Handlers) GetMessageClips(c echo.Context) error {
	userID := getRequestUserID(c)
	messageID := getParamAsUUID(c, consts.ParamMessageID)

	clips, err := h.Repo.GetMessageClips(userID, messageID)
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

	return serveMessages(c, h.Repo, req.convertC(channelID))
}

// PostMessage POST /channels/:channelID/messages
func (h *Handlers) PostMessage(c echo.Context) error {
	userID := getRequestUserID(c)
	ch := getParamChannel(c)

	if ch.IsArchived() {
		return herror.BadRequest(fmt.Sprintf("channel #%s has been archived", h.Repo.GetPublicChannelTree().GetChannelPath(ch.ID)))
	}

	var req PostMessageRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	if req.Embed {
		req.Content = message.NewReplacer(h.Repo).Replace(req.Content)
	}

	m, err := h.Repo.CreateMessage(userID, ch.ID, req.Content)
	if err != nil {
		return herror.InternalServerError(err)
	}

	return c.JSON(http.StatusCreated, formatMessage(m))
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
	ch, err := h.Repo.GetDirectMessageChannel(myID, targetID)
	if err != nil {
		return herror.InternalServerError(err)
	}

	return serveMessages(c, h.Repo, req.convertC(ch.ID))
}

// PostDirectMessage POST /users/:userId/messages
func (h *Handlers) PostDirectMessage(c echo.Context) error {
	myID := getRequestUserID(c)
	targetID := getParamAsUUID(c, consts.ParamUserID)

	var req PostMessageRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	// DMチャンネルを取得
	ch, err := h.Repo.GetDirectMessageChannel(myID, targetID)
	if err != nil {
		return herror.InternalServerError(err)
	}

	if req.Embed {
		req.Content = message.NewReplacer(h.Repo).Replace(req.Content)
	}

	m, err := h.Repo.CreateMessage(myID, ch.ID, req.Content)
	if err != nil {
		return herror.InternalServerError(err)
	}

	return c.JSON(http.StatusCreated, formatMessage(m))
}
