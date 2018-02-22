package router

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/model"
)

//PinForResponse ピン留めのJSON
type PinForResponse struct {
	PinID     string              `json:"pinId"`
	ChannelID string              `json:"channelId"`
	UserID    string              `json:"userId"`
	DateTime  string              `json:"dateTime"`
	Message   *MessageForResponse `json:"message"`
}

//GetChannelPin Method Handler of "GET /channels/{channelID}/pin"
func GetChannelPin(c echo.Context) error {
	channelID := c.Param("channelID")

	responseBody, err := getChannelPinResponse(channelID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get pin: %v", err)
	}

	return c.JSON(http.StatusOK, responseBody)
}

//GetPin Method Handler of "GET /pin/{pinID}"
func GetPin(c echo.Context) error {
	pinID := c.Param("pinID")

	responseBody, err := getPinResponse(pinID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get pin: %v", err)
	}

	return c.JSON(http.StatusOK, responseBody)
}

//PostPin Method Handler of "POST /channels/{channelID}/pin"
func PostPin(c echo.Context) error {
	channelID := c.Param("channelID")
	me := c.Get("user").(*model.User)

	requestBody := struct {
		MessageID string `json:"messageId"`
	}{}
	if err := c.Bind(&requestBody); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Failed to bind request body.")
	}

	pin := &model.Pin{
		ChannelID: channelID,
		UserID:    me.ID,
		MessageID: requestBody.MessageID,
	}
	if err := pin.Create(); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create pin: %v", err))
	}

	responseBody, formatErr := formatPin(pin)
	if formatErr != nil {
		pin.Delete()
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to format pin: %v", formatErr)
	}

	c.Response().Header().Set(echo.HeaderLocation, "/pin/"+pin.ID)
	return c.JSON(http.StatusCreated, responseBody)
}

//DeletePin Method Handler of "DELETE /pin/{pinID}"
func DeletePin(c echo.Context) error {
	pinID := c.Param("pinID")

	pin := &model.Pin{
		ID: pinID,
	}

	if err := pin.Delete(); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to delete pin: %v", err))
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
	pin, err := model.GetPin(ID)
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
	rawMessage, err := model.GetMessage(raw.MessageID)
	if err != nil {
		return nil, fmt.Errorf("Failed to get message: %v", err)
	}

	message := formatMessage(rawMessage)

	return &PinForResponse{
		PinID:     raw.ID,
		ChannelID: raw.ChannelID,
		UserID:    raw.UserID,
		DateTime:  raw.CreatedAt,
		Message:   message,
	}, nil
}
