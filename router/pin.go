package router

import (
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/event"
	"net/http"
	"time"

	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/model"
)

type pinForResponse struct {
	PinID     string              `json:"pinId"`
	ChannelID string              `json:"channelId"`
	UserID    string              `json:"userId"`
	DateTime  time.Time           `json:"dateTime"`
	Message   *MessageForResponse `json:"message"`
}

// GetChannelPin GET /channels/:channelID/pin"
func GetChannelPin(c echo.Context) error {
	userID := getRequestUserID(c)
	channelID := getRequestParamAsUUID(c, paramChannelID)

	// ユーザーからアクセス可能なチャンネルかどうか
	if ok, err := model.IsChannelAccessibleToUser(userID, channelID); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	} else if !ok {
		return echo.NewHTTPError(http.StatusNotFound)
	}

	res, err := getChannelPinResponse(channelID)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, res)
}

// PostPin POST /channels/:channelID/pin
func PostPin(c echo.Context) error {
	userID := getRequestUserID(c)
	channelID := getRequestParamAsUUID(c, paramChannelID)

	// ユーザーからアクセス可能なチャンネルかどうか
	if ok, err := model.IsChannelAccessibleToUser(userID, channelID); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	} else if !ok {
		return echo.NewHTTPError(http.StatusNotFound)
	}

	req := struct {
		MessageID string `json:"messageId" validate:"uuid,required"`
	}{}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	m, err := model.GetMessageByID(uuid.FromStringOrNil(req.MessageID))
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusBadRequest, "the message doesn't exist")
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}
	if m.GetCID() != channelID {
		return echo.NewHTTPError(http.StatusBadRequest, "the channel doesn't have the message")
	}

	pinID, err := model.CreatePin(m.GetID(), userID)
	if err != nil {
		if isMySQLDuplicatedRecordErr(err) {
			return echo.NewHTTPError(http.StatusBadRequest, "the message has already been pinned")
		}
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	go event.Emit(event.MessagePinned, &event.PinEvent{PinID: pinID, Message: *m})
	return c.JSON(http.StatusCreated, map[string]string{"id": pinID.String()})
}

// GetPin GET /pin/:pinID"
func GetPin(c echo.Context) error {
	userID := getRequestUserID(c)
	pinID := getRequestParamAsUUID(c, paramPinID)

	pin, err := model.GetPin(pinID)
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	// ユーザーからアクセス可能なチャンネルかどうか
	if ok, err := model.IsChannelAccessibleToUser(userID, pin.Message.GetCID()); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	} else if !ok {
		return echo.NewHTTPError(http.StatusNotFound)
	}

	return c.JSON(http.StatusOK, formatPin(pin))
}

// DeletePin DELETE /pin/:pinID
func DeletePin(c echo.Context) error {
	userID := getRequestUserID(c)
	pinID := getRequestParamAsUUID(c, paramPinID)

	pin, err := model.GetPin(pinID)
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	// ユーザーからアクセス可能なチャンネルかどうか
	if ok, err := model.IsChannelAccessibleToUser(userID, pin.Message.GetCID()); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	} else if !ok {
		return echo.NewHTTPError(http.StatusNotFound)
	}

	if err := model.DeletePin(pinID); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	go event.Emit(event.MessageUnpinned, &event.PinEvent{PinID: pinID, Message: pin.Message})
	return c.NoContent(http.StatusNoContent)
}

func getChannelPinResponse(channelID uuid.UUID) ([]*pinForResponse, error) {
	pins, err := model.GetPinsByChannelID(channelID)
	if err != nil {
		return nil, err
	}

	res := make([]*pinForResponse, len(pins))
	for i, pin := range pins {
		res[i] = formatPin(pin)
	}
	return res, nil
}

func formatPin(raw *model.Pin) *pinForResponse {
	return &pinForResponse{
		PinID:     raw.ID,
		ChannelID: raw.Message.ChannelID,
		UserID:    raw.UserID,
		DateTime:  raw.CreatedAt,
		Message:   formatMessage(&raw.Message),
	}
}
