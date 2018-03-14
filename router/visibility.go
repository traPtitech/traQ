package router

import (
	"net/http"

	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/model"
)

// Visibility visibilityの受け渡し用構造体
type Visibility struct {
	Visible []string `json:"visible"`
	Hidden  []string `json:"hidden"`
}

// GetChannelsVisibility GET /users/me/channels/visibility
func GetChannelsVisibility(c echo.Context) error {
	userID := c.Get("user").(*model.User).ID

	res, err := getVisibilityResponse(c, userID)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, res)
}

// PutChannelsVisibility PUT /users/me/channels/visibility
func PutChannelsVisibility(c echo.Context) error {
	userID := c.Get("user").(*model.User).ID

	req := &Visibility{}
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request")
	}

	for _, v := range req.Hidden {
		i := model.UserInvisibleChannel{
			UserID:    userID,
			ChannelID: v,
		}

		ok, err := i.Exists()
		if err != nil {
			c.Logger().Errorf("failed to check users_invisible_channels: %v", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update channel visibility")
		}
		if !ok {
			if err := i.Create(); err != nil {
				c.Logger().Errorf("failed to create users_invisible_channels: %v", err)
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update channel visibility")
			}
		}
	}

	for _, v := range req.Visible {
		i := model.UserInvisibleChannel{
			UserID:    userID,
			ChannelID: v,
		}
		ok, err := i.Exists()
		if err != nil {
			c.Logger().Errorf("failed to check users_invisible_channels: %v", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update channel visibility")
		}
		if ok {
			if err := i.Delete(); err != nil {
				c.Logger().Errorf("failed to delete users_invisible_channels: %v", err)
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update channel visibility")
			}
		}
	}

	res, err := getVisibilityResponse(c, userID)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, res)
}

func getVisibilityResponse(c echo.Context, userID string) (*Visibility, error) {
	visible, err := model.GetVisibleChannelsByID(userID)
	if err != nil {
		c.Logger().Errorf("failed to get visible channels: %v", err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to get channel visibility")
	}

	hidden, err := model.GetInvisibleChannelsByID(userID)
	if err != nil {
		c.Logger().Errorf("failed to get hidden channels: %v", err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to get channel visibility")
	}

	res := &Visibility{
		Visible: visible,
		Hidden:  hidden,
	}
	return res, nil
}
