package v3

import (
	"net/http"

	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/router/extension/herror"
)

// GetSoundboardItems
func (h *Handlers) GetSoundboardItems(c echo.Context) error {
	items, err := h.Repo.GetAllSoundboardItems()
	if err != nil {
		return herror.InternalServerError(err)
	}
	return c.JSON(http.StatusOK, items)
}

// CreateSoundboardItem
func (h *Handlers) CreateSoundboardItem(c echo.Context) error {
	src, uploadedFile, err := c.Request().FormFile("file")
	if err != nil {
		return herror.BadRequest(err)
	}
	defer src.Close()
	if uploadedFile.Size == 0 {
		return herror.BadRequest("non-empty file is required")
	}

	mimeType := uploadedFile.Header.Get(echo.HeaderContentType)
	soundName := c.FormValue("name")
	creatorID := uuid.FromStringOrNil(c.FormValue("creatorId"))
	stampID := uuid.FromStringOrNil(c.FormValue("stampId"))

	if err := h.Soundboard.SaveSoundboardItem(uuid.Must(uuid.NewV7()), soundName, mimeType, model.FileTypeSoundboardItem, src, &stampID, creatorID); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}
