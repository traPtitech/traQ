package router

import (
	"github.com/gofrs/uuid"
	"github.com/labstack/echo"
	"net/http"
	"time"
)

// GET /activity/latest-messages
func (h *Handlers) GetActivityLatestMessages(c echo.Context) error {
	type responseMessage struct {
		MessageID       uuid.UUID `json:"messageId"`
		UserID          uuid.UUID `json:"userId"`
		ParentChannelID uuid.UUID `json:"parentChannelId"`
		Content         string    `json:"content"`
		CreatedAt       time.Time `json:"createdAt"`
		UpdatedAt       time.Time `json:"updatedAt"`
	}

	userID := getRequestUserID(c)

	req := struct {
		Limit         int  `query:"limit"`
		SubscribeOnly bool `query:"subscribe"`
	}{
		Limit:         50,
		SubscribeOnly: true,
	}
	if err := bindAndValidate(c, &req); err != nil {
		return badRequest(err)
	}

	if req.Limit <= 0 || req.Limit > 50 {
		req.Limit = 50
	}

	messages, err := h.Repo.GetChannelLatestMessagesByUserID(userID, req.Limit, req.SubscribeOnly)
	if err != nil {
		return internalServerError(err, h.requestContextLogger(c))
	}

	res := make([]responseMessage, len(messages))
	for i, raw := range messages {
		res[i] = responseMessage{
			MessageID:       raw.ID,
			UserID:          raw.UserID,
			ParentChannelID: raw.ChannelID,
			Content:         raw.Text,
			CreatedAt:       raw.CreatedAt,
			UpdatedAt:       raw.UpdatedAt,
		}
	}

	return c.JSON(http.StatusOK, res)
}
