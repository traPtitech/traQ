package v1

import (
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"net/http"
)

// GetChannelPin GET /channels/:channelID/pins
func (h *Handlers) GetChannelPin(c echo.Context) error {
	channelID := getRequestParamAsUUID(c, consts.ParamChannelID)

	pins, err := h.Repo.GetPinsByChannelID(channelID)
	if err != nil {
		return herror.InternalServerError(err)
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
		return err
	}

	m, err := h.Repo.GetMessageByID(req.MessageID)
	if err != nil {
		switch err {
		case repository.ErrNotFound:
			return herror.BadRequest("the message doesn't exist")
		default:
			return herror.InternalServerError(err)
		}
	}

	// ユーザーからアクセス可能なチャンネルかどうか
	if ok, err := h.Repo.IsChannelAccessibleToUser(userID, m.ChannelID); err != nil {
		return herror.InternalServerError(err)
	} else if !ok {
		return herror.BadRequest("the message doesn't exist")
	}

	pin, err := h.Repo.CreatePin(m.ID, userID)
	if err != nil {
		return herror.InternalServerError(err)
	}

	return c.JSON(http.StatusCreated, echo.Map{"id": pin.ID})
}

// GetPin GET /pins/:pinID
func (h *Handlers) GetPin(c echo.Context) error {
	pin := getPinFromContext(c)
	return c.JSON(http.StatusOK, formatPin(pin))
}

// DeletePin DELETE /pins/:pinID
func (h *Handlers) DeletePin(c echo.Context) error {
	pinID := getRequestParamAsUUID(c, consts.ParamPinID)

	if err := h.Repo.DeletePin(pinID, getRequestUserID(c)); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}
