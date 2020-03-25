package v3

import (
	"net/http"

	vd "github.com/go-ozzo/ozzo-validation"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/utils/validator"
	"gopkg.in/guregu/null.v3"
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
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Stamps      model.UUIDs `json:"stamps"`
}

func (r CreateStampPaletteRequest) Validate() error {
	return vd.ValidateStruct(&r,
		vd.Field(&r.Name, validator.StampPaletteNameRuleRequired...),
		vd.Field(&r.Description, validator.StampPaletteDescriptionRule...),
		vd.Field(&r.Stamps, validator.StampPaletteStampsRule...),
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
		default:
			return herror.InternalServerError(err)
		}
	}
	return c.JSON(http.StatusCreated, sp)
}

// PatchStampPaletteRequest PATCH /stamp-palettes/:paletteID リクエストボディ
type PatchStampPaletteRequest struct {
	Name        null.String `json:"name"`
	Description null.String `json:"description"`
	Stamps      model.UUIDs `json:"stamps"`
}

func (r PatchStampPaletteRequest) Validate() error {
	return vd.ValidateStruct(&r,
		vd.Field(&r.Name, validator.StampPaletteNameRule...),
		vd.Field(&r.Description, validator.StampPaletteDescriptionRule...),
		vd.Field(&r.Stamps, validator.StampPaletteStampsRule...),
	)
}

// EditStampPalette PATCH /stamp-palettes/:paletteID
func (h *Handlers) EditStampPalette(c echo.Context) error {
	user := getRequestUser(c)
	stampPalette := getParamStampPalette(c)

	// 権限チェック
	if user.GetID() != stampPalette.CreatorID {
		return herror.Forbidden("you are not permitted to edit stamp-palette created by others")
	}
	var req PatchStampPaletteRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	args := repository.UpdateStampPaletteArgs{
		Name:        req.Name,
		Description: req.Description,
		Stamps:      req.Stamps,
	}

	// スタンプパレット更新
	if err := h.Repo.UpdateStampPalette(stampPalette.ID, args); err != nil {
		switch {
		case repository.IsArgError(err):
			return herror.BadRequest(err)
		default:
			return herror.InternalServerError(err)
		}
	}
	return c.NoContent(http.StatusNoContent)
}

// GetStampPalette GET /stamp-palette/:paletteID
func (h *Handlers) GetStampPalette(c echo.Context) error {
	return c.JSON(http.StatusOK, getParamStampPalette(c))
}

// DeleteStampPalette DELETE /stamps/:stampID
func (h *Handlers) DeleteStampPalette(c echo.Context) error {
	user := getRequestUser(c)
	stampPalette := getParamStampPalette(c)

	// 権限チェック
	if user.GetID() != stampPalette.CreatorID {
		return herror.Forbidden("you are not permitted to delete stamp-palette created by others")
	}

	if err := h.Repo.DeleteStampPalette(stampPalette.ID); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}
