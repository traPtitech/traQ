package router

import (
	"github.com/labstack/echo"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/model"
	"net/http"
)

// GET /activity/latest-messages
func GetActivityLatestMessages(c echo.Context) error {
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

	messages, err := model.GetChannelLatestMessagesByUserID(userID, req.Limit, req.SubscribeOnly)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	reports, err := model.GetMessageReportsByReporterID(userID)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}
	hidden := make(map[uuid.UUID]bool)
	for _, v := range reports {
		hidden[v.MessageID] = true
	}

	res := make([]*MessageForResponse, 0, len(messages))
	for _, message := range messages {
		ms := formatMessage(message)
		if hidden[message.ID] {
			ms.Reported = true
		}
		res = append(res, ms)
	}

	return c.JSON(http.StatusOK, res)
}
