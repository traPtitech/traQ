package router

import (
	"fmt"
	"net/http"

	"github.com/traPtitech/traQ/notification"
	"github.com/traPtitech/traQ/notification/events"

	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/model"
)

// GetStars /users/me/starsのGETメソッドハンドラ
func GetStars(c echo.Context) error {
	user := c.Get("user").(*model.User)

	responseBody, err := getStarsResponse(user.ID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get stared channels response")
	}

	return c.JSON(http.StatusOK, responseBody)
}

// PostStars /users/me/starsのPOSTメソッドハンドラ
func PostStars(c echo.Context) error {
	user := c.Get("user").(*model.User)

	requestBody := struct {
		ChannelID string `json:"channelId"`
	}{}

	if err := c.Bind(&requestBody); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Failed to bind request body.")
	}

	star := &model.Star{
		UserID:    user.ID,
		ChannelID: requestBody.ChannelID,
	}

	if err := star.Create(); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create star: %s", err.Error()))
	}

	responseBody, err := getStarsResponse(user.ID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get stared channels response")
	}

	go notification.Send(events.ChannelStared, events.UserChannelEvent{UserID: user.ID, ChannelID: requestBody.ChannelID})
	return c.JSON(http.StatusCreated, responseBody)
}

// DeleteStars /users/me/starsのDELETEメソッドハンドラ
func DeleteStars(c echo.Context) error {
	user := c.Get("user").(*model.User)

	requestBody := struct {
		ChannelID string `json:"channelId"`
	}{}

	if err := c.Bind(&requestBody); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Failed to bind request body.")
	}

	if _, err := validateChannelID(requestBody.ChannelID, user.ID); err != nil {
		return err
	}

	star := &model.Star{
		UserID:    user.ID,
		ChannelID: requestBody.ChannelID,
	}

	if err := star.Delete(); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to delete star: %s", err.Error()))
	}

	responseBody, err := getStarsResponse(user.ID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get stared channels response")
	}

	go notification.Send(events.ChannelUnstared, events.UserChannelEvent{UserID: user.ID, ChannelID: requestBody.ChannelID})
	return c.JSON(http.StatusOK, responseBody)
}

func getStarsResponse(userID string) ([]*ChannelForResponse, error) {
	staredChannels, err := model.GetStaredChannels(userID)
	if err != nil {
		return nil, err
	}
	responseBody := make([]*ChannelForResponse, 0)
	for _, channel := range staredChannels {
		channelForResponse := formatChannel(channel)
		channelForResponse.Children, err = channel.Children(userID)
		if err != nil {
			return nil, err
		}
		responseBody = append(responseBody, channelForResponse)
	}
	return responseBody, nil
}
