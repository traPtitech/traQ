package router

import (
	"github.com/gofrs/uuid"
	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/repository"
	"net/http"
)

// GetChannelPin GET /channels/:channelID/pins
func (h *Handlers) GetChannelPin(c echo.Context) error {
	channelID := getRequestParamAsUUID(c, paramChannelID)

	pins, err := h.Repo.GetPinsByChannelID(channelID)
	if err != nil {
		return internalServerError(err, h.requestContextLogger(c))
	}

	return c.JSON(http.StatusOK, formatPins(pins))
}

// PostPin POST /pins
func (h *Handlers) PostPin(c echo.Context) error {
	userID := getRequestUserID(c)

	var req struct {
		MessageID uuid.UUID `json:"messageId"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return badRequest(err)
	}

	m, err := h.Repo.GetMessageByID(req.MessageID)
	if err != nil {
		switch err {
		case repository.ErrNotFound:
			return badRequest("the message doesn't exist")
		default:
			return internalServerError(err, h.requestContextLogger(c))
		}
	}

	// ユーザーからアクセス可能なチャンネルかどうか
	if ok, err := h.Repo.IsChannelAccessibleToUser(userID, m.ChannelID); err != nil {
		return internalServerError(err, h.requestContextLogger(c))
	} else if !ok {
		return badRequest("the message doesn't exist")
	}

	pinID, err := h.Repo.CreatePin(m.ID, userID)
	if err != nil {
		return internalServerError(err, h.requestContextLogger(c))
	}

	return c.JSON(http.StatusCreated, map[string]string{"id": pinID.String()})
}

// GetPin GET /pins/:pinID
func (h *Handlers) GetPin(c echo.Context) error {
	pin := getPinFromContext(c)
	return c.JSON(http.StatusOK, formatPin(pin))
}

// DeletePin DELETE /pins/:pinID
func (h *Handlers) DeletePin(c echo.Context) error {
	pinID := getRequestParamAsUUID(c, paramPinID)

	if err := h.Repo.DeletePin(pinID); err != nil {
		return internalServerError(err, h.requestContextLogger(c))
	}

	return c.NoContent(http.StatusNoContent)
}
