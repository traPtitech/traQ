package v3

import (
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/service/ws"
	"net/http"
)

func (h *Handlers) GetMyViewStates(c echo.Context) error {
	type viewState struct {
		Key       string    `json:"key"`
		ChannelID uuid.UUID `json:"channelId"`
		State     string    `json:"state"`
	}
	res := make([]viewState, 0)

	userID := getRequestUserID(c)
	h.WS.IterateSessions(func(session ws.Session) {
		if session.UserID() == userID {
			channelID, state := session.ViewState()
			res = append(res, viewState{
				Key:       session.Key(),
				ChannelID: channelID,
				State:     state.String(),
			})
		}
	})

	return c.JSON(http.StatusOK, res)
}
