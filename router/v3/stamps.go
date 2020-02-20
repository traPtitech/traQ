package v3

import (
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"net/http"
)

// GetStamps GET /stamps
func (h *Handlers) GetStamps(c echo.Context) error {
	u := c.QueryParam("include-unicode")
	if len(u) == 0 {
		u = "1"
	}

	stamps, err := h.Repo.GetAllStamps(!isTrue(u))
	if err != nil {
		return herror.InternalServerError(err)
	}

	return c.JSON(http.StatusOK, stamps)
}

// CreateStamp POST /stamps
func (h *Handlers) CreateStamp(c echo.Context) error {
	userID := getRequestUserID(c)

	// スタンプ画像保存
	fileID, err := saveUploadImage(c, h.Repo, "file", model.FileTypeStamp, 1<<20, 128)
	if err != nil {
		return err
	}

	// スタンプ作成
	s, err := h.Repo.CreateStamp(c.FormValue("name"), fileID, userID)
	if err != nil {
		switch {
		case repository.IsArgError(err):
			return herror.BadRequest(err)
		case err == repository.ErrAlreadyExists:
			return herror.Conflict("this name has already been used")
		default:
			return herror.InternalServerError(err)
		}
	}
	return c.JSON(http.StatusCreated, s)
}

// GetStamp GET /stamps/:stampID
func (h *Handlers) GetStamp(c echo.Context) error {
	return c.JSON(http.StatusOK, getParamStamp(c))
}

// DeleteStamp DELETE /stamps/:stampID
func (h *Handlers) DeleteStamp(c echo.Context) error {
	stampID := getParamAsUUID(c, consts.ParamStampID)

	if err := h.Repo.DeleteStamp(stampID); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}
