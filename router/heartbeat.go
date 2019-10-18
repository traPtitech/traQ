package router

import (
	"github.com/gofrs/uuid"
	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/model"
	"net/http"
)

// PostHeartbeat POST /heartbeat
func (h *Handlers) PostHeartbeat(c echo.Context) error {
	userID := getRequestUserID(c)

	req := struct {
		ChannelID uuid.UUID `json:"channelId"`
		Status    string    `json:"status"`
	}{}
	if err := bindAndValidate(c, &req); err != nil {
		return badRequest(err)
	}

	h.Realtime.HeartBeats.Beat(userID, req.ChannelID, req.Status)

	status := h.Realtime.ViewerManager.GetChannelViewers(req.ChannelID)
	result := &model.HeartbeatStatus{
		UserStatuses: make([]*model.UserStatus, 0, len(status)),
		ChannelID:    req.ChannelID,
	}
	for uid, s := range status {
		result.UserStatuses = append(result.UserStatuses, &model.UserStatus{UserID: uid, Status: s.String()})
	}
	return c.JSON(http.StatusOK, result)
}

// GetHeartbeat GET /heartbeat
func (h *Handlers) GetHeartbeat(c echo.Context) error {
	req := struct {
		ChannelID uuid.UUID `query:"channelId"`
	}{}
	if err := bindAndValidate(c, &req); err != nil {
		return badRequest(err)
	}

	status := h.Realtime.ViewerManager.GetChannelViewers(req.ChannelID)
	result := &model.HeartbeatStatus{
		UserStatuses: make([]*model.UserStatus, 0, len(status)),
		ChannelID:    req.ChannelID,
	}
	for uid, s := range status {
		result.UserStatuses = append(result.UserStatuses, &model.UserStatus{UserID: uid, Status: s.String()})
	}
	return c.JSON(http.StatusOK, result)
}
