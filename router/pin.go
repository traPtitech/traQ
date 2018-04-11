package router

import (
	"fmt"
	"net/http"
	"time"

	"github.com/traPtitech/traQ/notification"
	"github.com/traPtitech/traQ/notification/events"

	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/model"
)

//PinForResponse ピン留めのJSON
type PinForResponse struct {
	PinID     string              `json:"pinId"`
	ChannelID string              `json:"channelId"`
	UserID    string              `json:"userId"`
	DateTime  time.Time           `json:"dateTime"`
	Message   *MessageForResponse `json:"message"`
}

//GetChannelPin Method Handler of "GET /channels/{channelID}/pin"
func GetChannelPin(c echo.Context) error {
	userID := c.Get("user").(*model.User).ID
	channelID := c.Param("channelID")
	if _, err := validateChannelID(channelID, userID); err != nil {
		return err
	}

	res, err := getChannelPinResponse(channelID)
	if err != nil {
		c.Logger().Errorf("an error occurred while getting pins: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get pin")
	}

	return c.JSON(http.StatusOK, res)
}

//GetPin Method Handler of "GET /pin/{pinID}"
func GetPin(c echo.Context) error {
	pinID := c.Param("pinID")

	res, err := getPinResponse(pinID)
	if err != nil {
		c.Logger().Errorf("An error occurred while getting pin: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get pin")
	}

	return c.JSON(http.StatusOK, res)
}

//PostPin Method Handler of "POST /channels/{channelID}/pin"
func PostPin(c echo.Context) error {
	channelID := c.Param("channelID")
	myID := c.Get("user").(*model.User).ID

	req := struct {
		MessageID string `json:"messageId"`
	}{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Failed to bind request body.")
	}

	m, err := model.GetMessageByID(req.MessageID)
	if err != nil {
		c.Logger().Error("An error occurred while getting message: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get message you requested")
	}
	if m.ChannelID != channelID {
		return echo.NewHTTPError(http.StatusBadRequest, "This messageis not a member of this channel")
	}

	pin := &model.Pin{
		ChannelID: channelID,
		UserID:    myID,
		MessageID: req.MessageID,
	}
	if err := pin.Create(); err != nil {
		c.Logger().Errorf("Failed to create pin: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create pin")
	}

	pin.CreatedAt = pin.CreatedAt.Truncate(time.Second) //自前で秒未満切り捨てしないと駄目
	res, formatErr := formatPin(pin)
	if formatErr != nil {
		pin.Delete()
		c.Logger().Errorf("An error occurred while formatting pin: %v", formatErr)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to format pin")
	}

	c.Response().Header().Set(echo.HeaderLocation, "/pin/"+pin.ID)
	go notification.Send(events.MessagePinned, events.PinEvent{PinID: pin.ID, Message: *m})
	return c.JSON(http.StatusCreated, res)
}

//DeletePin Method Handler of "DELETE /pin/{pinID}"
func DeletePin(c echo.Context) error {
	pinID := c.Param("pinID")
	pin, err := validatePinID(pinID)
	if err != nil {
		return err
	}

	if err := pin.Delete(); err != nil {
		c.Logger().Errorf("an error occurred while deleting pin: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to delete pin")
	}

	if m, err := model.GetMessageByID(pin.MessageID); err != nil {
		go notification.Send(events.MessageUnpinned, events.PinEvent{PinID: pin.ID, Message: *m})
	}
	return c.NoContent(http.StatusNoContent)
}

func getChannelPinResponse(channelID string) ([]*PinForResponse, error) {
	pins, err := model.GetPinsByChannelID(channelID)
	if err != nil {
		return nil, err
	}

	responseBody := make([]*PinForResponse, 0)
	for _, pin := range pins {
		res, err := formatPin(pin)
		if err != nil {
			return nil, err
		}
		responseBody = append(responseBody, res)
	}
	return responseBody, nil
}

func getPinResponse(ID string) (*PinForResponse, error) {
	pin, err := validatePinID(ID)
	if err != nil {
		return nil, err
	}

	responseBody, formatErr := formatPin(pin)
	if err != nil {
		return nil, formatErr
	}

	return responseBody, nil
}

func formatPin(raw *model.Pin) (*PinForResponse, error) {
	rawMessage, err := model.GetMessageByID(raw.MessageID)
	if err != nil {
		return nil, fmt.Errorf("Failed to get message: %v", err)
	}

	message := formatMessage(rawMessage)

	return &PinForResponse{
		PinID:     raw.ID,
		ChannelID: raw.ChannelID,
		UserID:    raw.UserID,
		DateTime:  raw.CreatedAt.Truncate(time.Second).UTC(),
		Message:   message,
	}, nil
}

func validatePinID(pinID string) (*model.Pin, error) {
	p := &model.Pin{ID: pinID}
	ok, err := p.Exists()
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "An error occurred in the server while get pin")
	}
	if !ok {
		return nil, echo.NewHTTPError(http.StatusNotFound, "The specified pin does not exist")
	}
	return p, nil
}
