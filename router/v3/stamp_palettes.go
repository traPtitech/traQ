package v3

import (
	vd "github.com/go-ozzo/ozzo-validation"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/utils/validator"
	"net/http"
)

// GetStampPalettes GET /stamp-palettes
func (h *Handlers) GetStampPalettes(c echo.Context) error {
	userID := getRequestUserID(c)

	palettes, err := h.Repo.GetStampPalettes(userID)
	if err != nil {
		return herror.InternalServerError(err)
	}

	return c.JSON(http.StatusOK, palettes)
}

// CreateStampPaletteRequest POST /stamp-palettes リクエストボディ
type CreateStampPaletteRequest struct {
	Name      string `json:"name"`
	Description string   `json:"description"`
	Stamps	model.UUIDs	`json:"stamps"`
}

func (r CreateStampPaletteRequest) Validate() error {
	return vd.ValidateStruct(&r,
		vd.Field(&r.Name, validator.StampPaletteNameRule...),
		vd.Field(&r.Description, validator.StampPaletteDescriptionRule...),
	)
}

// CreateStampPalette POST /stamp-palettes
func (h *Handlers) CreateStampPalette(c echo.Context) error {
	userID := getRequestUserID(c)

	var req CreateStampPaletteRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	// スタンプパレット作成
	sp, err := h.Repo.CreateStampPalette(req.Name, req.Description, req.Stamps, userID)
	if err != nil {
		switch {
		case repository.IsArgError(err):
			return herror.BadRequest(err)
		case err == repository.ErrNilID:
			return herror.BadRequest(err)
		default:
			return herror.InternalServerError(err)
		}
	}
	return c.JSON(http.StatusCreated, sp)
}

// GetStampPalette GET /stamp-palette/:paletteID
func (h *Handlers) GetStampPalette(c echo.Context) error {
	return c.JSON(http.StatusOK, getParamStampPalette(c))
}

// DeleteStampPalette DELETE /stamps/:stampID
func (h *Handlers) DeleteStampPalette(c echo.Context) error {
	paletteID := getParamAsUUID(c, consts.ParamStampPaletteID)

	if err := h.Repo.DeleteStampPalette(paletteID); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}
