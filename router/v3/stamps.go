package v3

import (
	vd "github.com/go-ozzo/ozzo-validation"
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/rbac/permission"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/utils/validator"
	"gopkg.in/guregu/null.v3"
	"net/http"
	"strconv"
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

// PatchStampRequest PATCH /users/me リクエストボディ
type PatchStampRequest struct {
	Name      null.String `json:"name"`
	CreatorID uuid.UUID   `json:"creatorId"`
}

func (r PatchStampRequest) Validate() error {
	return vd.ValidateStruct(&r,
		vd.Field(&r.Name, validator.StampNameRule...),
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
	if stamp.CreatorID != user.ID && !h.RBAC.IsGranted(user.Role, permission.EditStampCreatedByOthers) {
		return herror.Forbidden("you are not permitted to edit stamp created by others")
	}

	args := repository.UpdateStampArgs{}

	// 名前変更
	if req.Name.Valid {
		// 権限確認
		if !h.RBAC.IsGranted(user.Role, permission.EditStampName) {
			return herror.Forbidden("you are not permitted to change stamp name")
		}
		args.Name = req.Name
	}

	// 作成者変更
	if req.CreatorID != uuid.Nil {
		ok, err := h.Repo.UserExists(req.CreatorID)
		if err != nil {
			return herror.InternalServerError(err)
		}
		if !ok {
			return herror.BadRequest("invalid creatorId")
		}

		args.CreatorID = uuid.NullUUID{Valid: true, UUID: req.CreatorID}
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
	file, err := h.Repo.GetFS().OpenFileByKey(meta.GetKey(), meta.Type)
	if err != nil {
		return herror.InternalServerError(err)
	}
	defer file.Close()

	c.Response().Header().Set(echo.HeaderContentType, meta.Mime)
	c.Response().Header().Set(consts.HeaderETag, strconv.Quote(meta.Hash))
	http.ServeContent(c.Response(), c.Request(), meta.Name, meta.CreatedAt, file)
	return nil
}

// ChangeStampImage PUT /stamps/:stampID/image
func (h *Handlers) ChangeStampImage(c echo.Context) error {
	user := getRequestUser(c)
	stamp := getParamStamp(c)

	// ユーザー確認
	if stamp.CreatorID != user.ID && !h.RBAC.IsGranted(user.Role, permission.EditStampCreatedByOthers) {
		return herror.Forbidden("you are not permitted to edit stamp created by others")
	}

	// スタンプ画像保存
	fileID, err := saveUploadImage(c, h.Repo, "file", model.FileTypeStamp, 1<<20, 128)
	if err != nil {
		return err
	}

	args := repository.UpdateStampArgs{FileID: uuid.NullUUID{Valid: true, UUID: fileID}}
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
