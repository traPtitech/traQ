package router

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// GetStars GET /users/me/stars
func (h *Handlers) GetStars(c echo.Context) error {
	userID := getRequestUserID(c)

	stars, err := h.Repo.GetStaredChannels(userID)
	if err != nil {
		return internalServerError(err, h.requestContextLogger(c))
	}

	return c.JSON(http.StatusOK, stars)
}

// PutStars PUT /users/me/stars/:channelID
func (h *Handlers) PutStars(c echo.Context) error {
	userID := getRequestUserID(c)
	channelID := getRequestParamAsUUID(c, paramChannelID)

	if err := h.Repo.AddStar(userID, channelID); err != nil {
		return internalServerError(err, h.requestContextLogger(c))
	}

	return c.NoContent(http.StatusNoContent)
}

// DeleteStars DELETE /users/me/stars/:channelID
func (h *Handlers) DeleteStars(c echo.Context) error {
	userID := getRequestUserID(c)
	channelID := getRequestParamAsUUID(c, paramChannelID)

	if err := h.Repo.RemoveStar(userID, channelID); err != nil {
		return internalServerError(err, h.requestContextLogger(c))
	}

	return c.NoContent(http.StatusNoContent)
}
