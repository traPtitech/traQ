package v1

import (
	vd "github.com/go-ozzo/ozzo-validation"
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"net/http"
)

// GetUserTags GET /users/:userID/tags
func (h *Handlers) GetUserTags(c echo.Context) error {
	userID := getRequestParamAsUUID(c, consts.ParamUserID)

	tags, err := h.Repo.GetUserTagsByUserID(userID)
	if err != nil {
		return herror.InternalServerError(err)
	}

	return c.JSON(http.StatusOK, formatTags(tags))
}

// PostUserTagRequest POST /users/:userID/tags リクエストボディ
type PostUserTagRequest struct {
	Tag string `json:"tag"`
}

func (r PostUserTagRequest) Validate() error {
	return vd.ValidateStruct(&r,
		vd.Field(&r.Tag, vd.Required, vd.Length(1, 30)),
	)
}

// PostUserTag POST /users/:userID/tags
func (h *Handlers) PostUserTag(c echo.Context) error {
	userID := getRequestParamAsUUID(c, consts.ParamUserID)

	// リクエスト検証
	var req PostUserTagRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	// タグの確認
	t, err := h.Repo.GetOrCreateTagByName(req.Tag)
	if err != nil {
		return herror.InternalServerError(err)
	}

	// ユーザーにタグを付与
	if err := h.Repo.AddUserTag(userID, t.ID); err != nil {
		switch err {
		case repository.ErrAlreadyExists:
			return c.NoContent(http.StatusNoContent)
		default:
			return herror.InternalServerError(err)
		}
	}

	return c.NoContent(http.StatusCreated)
}

// PatchUserTagRequest PATCH /users/:userID/tags/:tagID リクエストボディ
type PatchUserTagRequest struct {
	IsLocked bool `json:"isLocked"`
}

// PatchUserTag PATCH /users/:userID/tags/:tagID
func (h *Handlers) PatchUserTag(c echo.Context) error {
	me := getRequestUserID(c)
	userID := getRequestParamAsUUID(c, consts.ParamUserID)
	tagID := getRequestParamAsUUID(c, consts.ParamTagID)

	// リクエスト検証
	var req PatchUserTagRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	// タグがつけられているかを見る
	ut, err := h.Repo.GetUserTag(userID, tagID)
	if err != nil {
		switch err {
		case repository.ErrNotFound:
			return herror.NotFound()
		default:
			return herror.InternalServerError(err)
		}
	}

	// 他人のロックは変更不可
	if me != userID {
		return herror.Forbidden("this is not your tag")
	}

	// 更新
	if err := h.Repo.ChangeUserTagLock(userID, ut.Tag.ID, req.IsLocked); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}

// DeleteUserTag DELETE /users/:userID/tags/:tagID
func (h *Handlers) DeleteUserTag(c echo.Context) error {
	userID := getRequestParamAsUUID(c, consts.ParamUserID)
	tagID := getRequestParamAsUUID(c, consts.ParamTagID)

	// タグがつけられているかを見る
	ut, err := h.Repo.GetUserTag(userID, tagID)
	if err != nil {
		switch err {
		case repository.ErrNotFound: // 既にない
			return c.NoContent(http.StatusNoContent)
		default:
			return herror.InternalServerError(err)
		}
	}

	// 削除
	if err := h.Repo.DeleteUserTag(userID, ut.Tag.ID); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}

// GetUsersByTagID GET /tags/:tagID
func (h *Handlers) GetUsersByTagID(c echo.Context) error {
	type response struct {
		ID    uuid.UUID   `json:"tagId"`
		Tag   string      `json:"tag"`
		Users []uuid.UUID `json:"users"`
	}

	tagID := getRequestParamAsUUID(c, consts.ParamTagID)

	t, err := h.Repo.GetTagByID(tagID)
	if err != nil {
		switch err {
		case repository.ErrNotFound:
			return herror.NotFound()
		default:
			return herror.InternalServerError(err)
		}
	}

	users, err := h.Repo.GetUserIDsByTagID(t.ID)
	if err != nil {
		return herror.InternalServerError(err)
	}

	return c.JSON(http.StatusOK, &response{
		ID:    t.ID,
		Tag:   t.Name,
		Users: users,
	})
}
