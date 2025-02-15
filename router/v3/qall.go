package v3

import (
	"net/http"

	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/router/extension/herror"
)

// CreateSoundBoardItemRequest POST /soundboard/items リクエストボディ
type CreateSoundBoardItemRequest struct {
	Name      string    `json:"name"`
	StampID   uuid.UUID `json:"stampId"`
	CreatorID uuid.UUID `json:"creatorId"`
}

// GetSoundboardItems
func (h *Handlers) GetSoundBoardItems(c echo.Context) error {
	items, err := h.Repo.GetAllSoundBoardItems()
	if err != nil {
		return herror.InternalServerError(err)
	}
	return c.JSON(http.StatusOK, items)
}

// CreateSoundBoardItem
func (h *Handlers) CreateSoundBoardItem(c echo.Context) error {
	var req CreateSoundBoardItemRequest
	if err := bindAndValidate(c, &req); err != nil {
		return herror.BadRequest(err)
	}

	if err := h.Repo.CreateSoundBoardItem(uuid.Must(uuid.NewV7()), req.Name, req.StampID, req.CreatorID); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}
