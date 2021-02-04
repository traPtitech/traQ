package v3

import (
	"context"
	"net/http"

	vd "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/router/utils"
	"github.com/traPtitech/traQ/utils/validator"
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

func (r PostStarRequest) ValidateWithContext(ctx context.Context) error {
	return vd.ValidateStructWithContext(ctx, &r,
		vd.Field(&r.ChannelID, vd.Required, validator.NotNilUUID, utils.IsPublicChannelID),
	)
}

// PostStar POST /users/me/stars
func (h *Handlers) PostStar(c echo.Context) error {
	var req PostStarRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	if err := h.Repo.AddStar(getRequestUserID(c), req.ChannelID); err != nil {
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
