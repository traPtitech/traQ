package v3

import (
	vd "github.com/go-ozzo/ozzo-validation"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
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

// GetMessage GET /messages/:messageID
func (h *Handlers) GetMessage(c echo.Context) error {
	return c.JSON(http.StatusOK, formatMessage(getParamMessage(c)))
}

// GetMessageStamps GET /messages/:messageID/stamps
func (h *Handlers) GetMessageStamps(c echo.Context) error {
	messageID := getParamAsUUID(c, consts.ParamMessageID)

	stamps, err := h.Repo.GetMessageStamps(messageID)
	if err != nil {
		return herror.InternalServerError(err)
	}

	return c.JSON(http.StatusOK, stamps)
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
