package v3

import (
	vd "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/service/message"
	"net/http"
	"time"
)

// GetOnlineUsers GET /activity/onlines
func (h *Handlers) GetOnlineUsers(c echo.Context) error {
	return c.JSON(http.StatusOK, h.OC.GetOnlineUserIDs())
}

// GetActivityTimelineRequest GET /activity/timeline リクエストボディ
type GetActivityTimelineRequest struct {
	Limit      int  `query:"limit"`
	All        bool `query:"all"`
	PerChannel bool `query:"per_channel"`
}

func (r *GetActivityTimelineRequest) Validate() error {
	if r.Limit == 0 {
		r.Limit = 50
	}
	return vd.ValidateStruct(r,
		vd.Field(&r.Limit, vd.Required, vd.Min(1), vd.Max(50)),
	)
}

// GetActivityTimeline GET /activity/timeline
func (h *Handlers) GetActivityTimeline(c echo.Context) error {
	userID := getRequestUserID(c)

	var req GetActivityTimelineRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	if !req.PerChannel {
		query := message.TimelineQuery{
			Limit:          req.Limit,
			ExcludeDMs:     true,
			DisablePreload: true,
		}
		if !req.All {
			query.ChannelsSubscribedByUser = userID
		}

		timeline, err := h.MessageManager.GetTimeline(query)
		if err != nil {
			return herror.InternalServerError(err)
		}
		return c.JSON(http.StatusOK, timeline.Records())
	}

	messages, err := h.Repo.GetChannelLatestMessagesByUserID(userID, req.Limit, !req.All)
	if err != nil {
		return herror.InternalServerError(err)
	}

	type responseMessage struct {
		ID        uuid.UUID `json:"id"`
		UserID    uuid.UUID `json:"userId"`
		ChannelID uuid.UUID `json:"channelId"`
		Content   string    `json:"content"`
		CreatedAt time.Time `json:"createdAt"`
		UpdatedAt time.Time `json:"updatedAt"`
	}
	res := make([]responseMessage, len(messages))
	for i, raw := range messages {
		res[i] = responseMessage{
			ID:        raw.ID,
			UserID:    raw.UserID,
			ChannelID: raw.ChannelID,
			Content:   raw.Text,
			CreatedAt: raw.CreatedAt,
			UpdatedAt: raw.UpdatedAt,
		}
	}

	return c.JSON(http.StatusOK, res)
}
