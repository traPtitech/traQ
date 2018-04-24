package router

import (
	"net/http"

	"github.com/traPtitech/traQ/notification"
	"github.com/traPtitech/traQ/notification/events"

	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/model"
)

// GetStars GET /users/me/stars のハンドラ
func GetStars(c echo.Context) error {
	myID := c.Get("user").(*model.User).ID

	res, err := getStarsResponse(myID)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get stared channels response")
	}

	return c.JSON(http.StatusOK, res)
}

// PostStars /users/me/starsのPOSTメソッドハンドラ
func PostStars(c echo.Context) error {
	myID := c.Get("user").(*model.User).ID

	req := struct {
		ChannelID string `json:"channelId"`
	}{}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Failed to bind request body.")
	}

	star := &model.Star{
		UserID:    myID,
		ChannelID: req.ChannelID,
	}

	if err := star.Create(); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create star")
	}

	go notification.Send(events.ChannelStared, events.UserChannelEvent{UserID: myID, ChannelID: req.ChannelID})
	return c.NoContent(http.StatusNoContent)
}

// DeleteStars DELETE /users/me/stars/{channelID} のハンドラ
func DeleteStars(c echo.Context) error {
	myID := c.Get("user").(*model.User).ID

	channelID := c.Param("channelID")

	if _, err := validateChannelID(channelID, myID); err != nil {
		return err
	}

	star := &model.Star{
		UserID:    myID,
		ChannelID: channelID,
	}

	if err := star.Delete(); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to delete star")
	}

	go notification.Send(events.ChannelUnstared, events.UserChannelEvent{UserID: myID, ChannelID: channelID})
	return c.NoContent(http.StatusNoContent)
}

func getStarsResponse(userID string) ([]*ChannelForResponse, error) {
	staredChannels, err := model.GetStaredChannels(userID)
	if err != nil {
		return nil, err
	}
	res := make([]*ChannelForResponse, 0)
	for _, ch := range staredChannels {
		childIDs, err := ch.Children(userID)
		if err != nil {
			return nil, err
		}
		members, err := model.GetPrivateChannelMembers(ch.ID)
		if err != nil {
			return nil, err
		}
		res = append(res, formatChannel(ch, childIDs, members))
	}
	return res, nil
}
