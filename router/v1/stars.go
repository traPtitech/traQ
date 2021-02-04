package v1

import (
	"net/http"

	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"

	"github.com/labstack/echo/v4"
)

// GetStars GET /users/me/stars
func (h *Handlers) GetStars(c echo.Context) error {
	userID := getRequestUserID(c)

	stars, err := h.Repo.GetStaredChannels(userID)
	if err != nil {
		return herror.InternalServerError(err)
	}

	return c.JSON(http.StatusOK, stars)
}

// PutStars PUT /users/me/stars/:channelID
func (h *Handlers) PutStars(c echo.Context) error {
	userID := getRequestUserID(c)
	channelID := getRequestParamAsUUID(c, consts.ParamChannelID)

	if err := h.Repo.AddStar(userID, channelID); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}

// DeleteStars DELETE /users/me/stars/:channelID
func (h *Handlers) DeleteStars(c echo.Context) error {
	userID := getRequestUserID(c)
	channelID := getRequestParamAsUUID(c, consts.ParamChannelID)

	if err := h.Repo.RemoveStar(userID, channelID); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}
