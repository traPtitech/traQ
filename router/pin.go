package router

import (
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/repository"
	"net/http"
	"time"

	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/model"
)

type pinForResponse struct {
	PinID     uuid.UUID           `json:"pinId"`
	ChannelID uuid.UUID           `json:"channelId"`
	UserID    uuid.UUID           `json:"userId"`
	DateTime  time.Time           `json:"dateTime"`
	Message   *MessageForResponse `json:"message"`
}

// GetChannelPin GET /channels/:channelID/pins
func (h *Handlers) GetChannelPin(c echo.Context) error {
	channelID := getRequestParamAsUUID(c, paramChannelID)

	res, err := h.getChannelPinResponse(channelID)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, res)
}

// PostPin POST /pins
func (h *Handlers) PostPin(c echo.Context) error {
	userID := getRequestUserID(c)

	req := struct {
		MessageID uuid.UUID `json:"messageId"`
	}{}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	m, err := h.Repo.GetMessageByID(req.MessageID)
	if err != nil {
		switch err {
		case repository.ErrNotFound:
			return echo.NewHTTPError(http.StatusBadRequest, "the message doesn't exist")
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	// ユーザーからアクセス可能なチャンネルかどうか
	if ok, err := h.Repo.IsChannelAccessibleToUser(userID, m.ChannelID); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	} else if !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "the message doesn't exist")
	}

	pinID, err := h.Repo.CreatePin(m.ID, userID)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusCreated, map[string]string{"id": pinID.String()})
}

// GetPin GET /pins/:pinID
func (h *Handlers) GetPin(c echo.Context) error {
	pin := getPinFromContext(c)
	return c.JSON(http.StatusOK, h.formatPin(pin))
}

// DeletePin DELETE /pins/:pinID
func (h *Handlers) DeletePin(c echo.Context) error {
	pinID := getRequestParamAsUUID(c, paramPinID)

	if err := h.Repo.DeletePin(pinID); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *Handlers) getChannelPinResponse(channelID uuid.UUID) ([]*pinForResponse, error) {
	pins, err := h.Repo.GetPinsByChannelID(channelID)
	if err != nil {
		return nil, err
	}

	res := make([]*pinForResponse, len(pins))
	for i, pin := range pins {
		res[i] = h.formatPin(pin)
	}
	return res, nil
}

func (h *Handlers) formatPin(raw *model.Pin) *pinForResponse {
	return &pinForResponse{
		PinID:     raw.ID,
		ChannelID: raw.Message.ChannelID,
		UserID:    raw.UserID,
		DateTime:  raw.CreatedAt,
		Message:   h.formatMessage(&raw.Message),
	}
}
