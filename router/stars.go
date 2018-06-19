package router

import (
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/event"
	"net/http"

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

// PutStars PUT /users/me/stars/{channelID} のハンドラ
func PutStars(c echo.Context) error {
	user := c.Get("user").(*model.User)
	channelID := c.Param("channelID")

	star := &model.Star{
		UserID:    user.ID,
		ChannelID: channelID,
	}

	if err := star.Create(); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create star")
	}

	go event.Emit(event.ChannelStared, &event.UserChannelEvent{UserID: user.GetUID(), ChannelID: uuid.Must(uuid.FromString(channelID))})
	return c.NoContent(http.StatusNoContent)
}

// DeleteStars DELETE /users/me/stars/{channelID} のハンドラ
func DeleteStars(c echo.Context) error {
	user := c.Get("user").(*model.User)

	channelID := c.Param("channelID")

	if _, err := validateChannelID(channelID, user.ID); err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound, "this channel is not found")
		default:
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to find the specified channel")
		}
	}

	star := &model.Star{
		UserID:    user.ID,
		ChannelID: channelID,
	}

	if err := star.Delete(); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to delete star")
	}

	go event.Emit(event.ChannelUnstared, &event.UserChannelEvent{UserID: user.GetUID(), ChannelID: uuid.Must(uuid.FromString(channelID))})
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
