package v3

import (
	vd "github.com/go-ozzo/ozzo-validation"
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/extension/herror"
	"net/http"
	"time"
)

// GetOnlineUsers GET /activity/onlines
func (h *Handlers) GetOnlineUsers(c echo.Context) error {
	return c.JSON(http.StatusOK, h.Realtime.OnlineCounter.GetOnlineUserIDs())
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

	var (
		messages []*model.Message
		err      error
	)

	if req.PerChannel {
		messages, err = h.Repo.GetChannelLatestMessagesByUserID(userID, req.Limit, !req.All)
	} else {
		messages, _, err = h.Repo.GetMessages(repository.MessagesQuery{
			Limit:          req.Limit,
			ExcludeDMs:     true,
			DisablePreload: true,
		})
	}
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
