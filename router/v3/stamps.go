package v3

import (
	"context"
	vd "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/rbac/permission"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/router/utils"
	"github.com/traPtitech/traQ/utils/optional"
	"github.com/traPtitech/traQ/utils/validator"
	"net/http"
	"strconv"
)

// GetStamps GET /stamps
func (h *Handlers) GetStamps(c echo.Context) error {
	u := c.QueryParam("include-unicode")
	if len(u) == 0 {
		u = "1"
	}

	b, updatedAt, err := h.Repo.GetStampsJSON(!isTrue(u))
	if err != nil {
		return herror.InternalServerError(err)
	}

	extension.SetLastModified(c, updatedAt)
	if done, err := extension.CheckPreconditions(c, updatedAt); done {
		return err
	}
	return c.JSONBlob(http.StatusOK, b)
}

// CreateStamp POST /stamps
func (h *Handlers) CreateStamp(c echo.Context) error {
	userID := getRequestUserID(c)

	// スタンプ画像保存
	fileID, err := utils.SaveUploadStampImage(h.Imaging, c, h.Repo, "file")
	if err != nil {
		return err
	}

	// スタンプ作成
	s, err := h.Repo.CreateStamp(repository.CreateStampArgs{Name: c.FormValue("name"), FileID: fileID, CreatorID: userID})
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

// PatchStampRequest PATCH /stamps/:stampID リクエストボディ
type PatchStampRequest struct {
	Name      optional.String `json:"name"`
	CreatorID optional.UUID   `json:"creatorId"`
}

func (r PatchStampRequest) ValidateWithContext(ctx context.Context) error {
	return vd.ValidateStructWithContext(ctx, &r,
		vd.Field(&r.Name, validator.StampNameRule...),
		vd.Field(&r.CreatorID, validator.NotNilUUID, utils.IsActiveHumanUserID),
	)
}

// EditStamp PATCH /stamps/:stampID
func (h *Handlers) EditStamp(c echo.Context) error {
	var req PatchStampRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	user := getRequestUser(c)
	stamp := getParamStamp(c)

	// ユーザー確認
	if stamp.CreatorID != user.GetID() && !h.RBAC.IsGranted(user.GetRole(), permission.EditStampCreatedByOthers) {
		return herror.Forbidden("you are not permitted to edit stamp created by others")
	}

	args := repository.UpdateStampArgs{
		Name:      req.Name,
		CreatorID: req.CreatorID,
	}

	// 更新
	if err := h.Repo.UpdateStamp(stamp.ID, args); err != nil {
		switch {
		case repository.IsArgError(err):
			return herror.BadRequest(err)
		case err == repository.ErrAlreadyExists:
			return herror.Conflict("this name has already been used")
		default:
			return herror.InternalServerError(err)
		}
	}
	return c.NoContent(http.StatusNoContent)
}

// DeleteStamp DELETE /stamps/:stampID
func (h *Handlers) DeleteStamp(c echo.Context) error {
	stampID := getParamAsUUID(c, consts.ParamStampID)

	if err := h.Repo.DeleteStamp(stampID); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}

// GetStampImage GET /stamps/:stampID/image
func (h *Handlers) GetStampImage(c echo.Context) error {
	stamp := getParamStamp(c)

	// ファイルメタ取得
	meta, err := h.Repo.GetFileMeta(stamp.FileID)
	if err != nil {
		return herror.InternalServerError(err)
	}

	// ファイルオープン
	file, err := meta.Open()
	if err != nil {
		return herror.InternalServerError(err)
	}
	defer file.Close()

	c.Response().Header().Set(echo.HeaderContentType, meta.GetMIMEType())
	c.Response().Header().Set(consts.HeaderETag, strconv.Quote(meta.GetMD5Hash()))
	http.ServeContent(c.Response(), c.Request(), meta.GetFileName(), meta.GetCreatedAt(), file)
	return nil
}

// ChangeStampImage PUT /stamps/:stampID/image
func (h *Handlers) ChangeStampImage(c echo.Context) error {
	user := getRequestUser(c)
	stamp := getParamStamp(c)

	// ユーザー確認
	if stamp.CreatorID != user.GetID() && !h.RBAC.IsGranted(user.GetRole(), permission.EditStampCreatedByOthers) {
		return herror.Forbidden("you are not permitted to edit stamp created by others")
	}

	// スタンプ画像保存
	fileID, err := utils.SaveUploadStampImage(h.Imaging, c, h.Repo, "file")
	if err != nil {
		return err
	}

	args := repository.UpdateStampArgs{FileID: optional.UUIDFrom(fileID)}
	// 更新
	if err := h.Repo.UpdateStamp(stamp.ID, args); err != nil {
		switch {
		case repository.IsArgError(err):
			return herror.BadRequest(err)
		default:
			return herror.InternalServerError(err)
		}
	}
	return c.NoContent(http.StatusNoContent)
}
