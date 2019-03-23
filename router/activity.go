package router

import (
	"github.com/gofrs/uuid"
	"github.com/labstack/echo"
	"go.uber.org/zap"
	"net/http"
)

// GET /activity/latest-messages
func (h *Handlers) GetActivityLatestMessages(c echo.Context) error {
	userID := getRequestUserID(c)

	req := struct {
		Limit         int  `query:"limit"  validate:"min=1,max=50"`
		SubscribeOnly bool `query:"subscribe"`
	}{
		Limit:         50,
		SubscribeOnly: true,
	}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	messages, err := h.Repo.GetChannelLatestMessagesByUserID(userID, req.Limit, req.SubscribeOnly)
	if err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	reports, err := h.Repo.GetMessageReportsByReporterID(userID)
	if err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError)
	}
	hidden := make(map[uuid.UUID]bool)
	for _, v := range reports {
		hidden[v.MessageID] = true
	}

	res := make([]*MessageForResponse, 0, len(messages))
	for _, message := range messages {
		ms := h.formatMessage(message)
		if hidden[message.ID] {
			ms.Reported = true
		}
		res = append(res, ms)
	}

	return c.JSON(http.StatusOK, res)
}
