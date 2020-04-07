package v3

import (
	vd "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/utils/validator"
	"net/http"
)

// GetMyStars GET /users/me/stars
func (h *Handlers) GetMyStars(c echo.Context) error {
	userID := getRequestUserID(c)

	stars, err := h.Repo.GetStaredChannels(userID)
	if err != nil {
		return herror.InternalServerError(err)
	}

	return c.JSON(http.StatusOK, stars)
}

// PostStarRequest POST /users/me/stars リクエストボディ
type PostStarRequest struct {
	ChannelID uuid.UUID `json:"channelId"`
}

func (r PostStarRequest) Validate() error {
	return vd.ValidateStruct(&r,
		vd.Field(&r.ChannelID, vd.Required, validator.NotNilUUID),
	)
}

// PostStar POST /users/me/stars
func (h *Handlers) PostStar(c echo.Context) error {
	var req PostStarRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	userID := getRequestUserID(c)
	if ok, err := h.Repo.IsChannelAccessibleToUser(userID, req.ChannelID); err != nil {
		return herror.InternalServerError(err)
	} else if !ok {
		return herror.BadRequest("bad channelID")
	}

	if err := h.Repo.AddStar(userID, req.ChannelID); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}

// RemoveMyStar DELETE /users/me/stars/:channelID
func (h *Handlers) RemoveMyStar(c echo.Context) error {
	userID := getRequestUserID(c)
	channelID := getParamAsUUID(c, consts.ParamChannelID)

	if err := h.Repo.RemoveStar(userID, channelID); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}
