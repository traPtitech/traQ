package v3

import (
	"net/http"

	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"github.com/traPtitech/traQ/router/extension/herror"
)

type imageSearchWebhookRequest struct {
	ID     uuid.UUID `json:"id"`
	Status string    `json:"status"`
}

// ImageSearchWebhook POST /search/image-webhook
func (h *Handlers) ImageSearchWebhook(c echo.Context) error {
	var req imageSearchWebhookRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	if req.ID == uuid.Nil {
		return herror.BadRequest("invalid job id")
	}
	switch req.Status {
	case "completed", "failed":
	default:
		return herror.BadRequest("invalid status")
	}

	if err := h.SearchEngine.HandleImageWebhook(c.Request().Context(), req.ID); err != nil {
		// 通知元には常に204を返し、再送ループを避ける
		h.L(c).Debug("failed to handle image search webhook", zap.Error(err))
	}

	return c.NoContent(http.StatusNoContent)
}
