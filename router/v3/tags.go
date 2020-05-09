package v3

import (
	"fmt"
	vd "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"net/http"
)

// GetUserTags GET /users/:userID/tags
func (h *Handlers) GetUserTags(c echo.Context) error {
	return serveUserTags(c, h.Repo, getParamAsUUID(c, consts.ParamUserID))
}

// GetMyUserTags GET /users/me/tags
func (h *Handlers) GetMyUserTags(c echo.Context) error {
	return serveUserTags(c, h.Repo, getRequestUserID(c))
}

// serveUserTags 指定したユーザーのタグ一覧をレスポンスとして返す
func serveUserTags(c echo.Context, repo repository.Repository, userID uuid.UUID) error {
	tags, err := repo.GetUserTagsByUserID(userID)
	if err != nil {
		return herror.InternalServerError(err)
	}
	return c.JSON(http.StatusOK, formatUserTags(tags))
}

// PostUserTagRequest POST /users/:userID/tags リクエストボディ
type PostUserTagRequest struct {
	Tag string `json:"tag"`
}

func (r PostUserTagRequest) Validate() error {
	return vd.ValidateStruct(&r,
		vd.Field(&r.Tag, vd.Required, vd.RuneLength(1, 30)),
	)
}

// AddUserTag POST /users/:userID/tags
func (h *Handlers) AddUserTag(c echo.Context) error {
	if getParamUser(c).GetUserType() == model.UserTypeWebhook {
		return herror.Forbidden("tags cannot be added to webhook user")
	}
	return addUserTags(c, h.Repo, getParamAsUUID(c, consts.ParamUserID))
}

// AddMyUserTag POST /users/me/tags
func (h *Handlers) AddMyUserTag(c echo.Context) error {
	return addUserTags(c, h.Repo, getRequestUserID(c))
}

// addUserTags 指定したユーザーにタグを追加するハンドラ
func addUserTags(c echo.Context, repo repository.Repository, userID uuid.UUID) error {
	var req PostUserTagRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	// タグの確認
	t, err := repo.GetOrCreateTag(req.Tag)
	if err != nil {
		return herror.InternalServerError(err)
	}

	// ユーザーにタグを付与
	if err := repo.AddUserTag(userID, t.ID); err != nil {
		switch err {
		case repository.ErrAlreadyExists:
			return c.NoContent(http.StatusConflict)
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

// EditUserTag PATCH /users/:userID/tags/:tagID
func (h *Handlers) EditUserTag(c echo.Context) error {
	me := getRequestUserID(c)
	userID := getParamAsUUID(c, consts.ParamUserID)

	// 他人のロックは変更不可
	if me != userID {
		return herror.Forbidden(fmt.Sprintf("you are not user (%s)", userID))
	}

	var req PatchUserTagRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	tagID := getParamAsUUID(c, consts.ParamTagID)

	// 更新
	if err := h.Repo.ChangeUserTagLock(userID, tagID, req.IsLocked); err != nil {
		switch err {
		case repository.ErrNotFound:
			return herror.NotFound()
		default:
			return herror.InternalServerError(err)
		}
	}

	return c.NoContent(http.StatusNoContent)
}

// EditMyUserTag PATCH /users/me/tags/:tagID
func (h *Handlers) EditMyUserTag(c echo.Context) error {
	var req PatchUserTagRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	userID := getRequestUserID(c)
	tagID := getParamAsUUID(c, consts.ParamTagID)

	// 更新
	if err := h.Repo.ChangeUserTagLock(userID, tagID, req.IsLocked); err != nil {
		switch err {
		case repository.ErrNotFound:
			return herror.NotFound()
		default:
			return herror.InternalServerError(err)
		}
	}

	return c.NoContent(http.StatusNoContent)
}

// RemoveUserTag DELETE /users/:userID/tags/:tagID
func (h *Handlers) RemoveUserTag(c echo.Context) error {
	return removeUserTag(c, h.Repo, getParamAsUUID(c, consts.ParamUserID))
}

// RemoveMyUserTag DELETE /users/me/tags/:tagID
func (h *Handlers) RemoveMyUserTag(c echo.Context) error {
	return removeUserTag(c, h.Repo, getRequestUserID(c))
}

// removeUserTag 指定したユーザーからタグを削除するハンドラ
func removeUserTag(c echo.Context, repo repository.Repository, userID uuid.UUID) error {
	tagID := getParamAsUUID(c, consts.ParamTagID)

	// タグがつけられているかを見る
	ut, err := repo.GetUserTag(userID, tagID)
	if err != nil {
		switch err {
		case repository.ErrNotFound: // 既にない
			return c.NoContent(http.StatusNoContent)
		default:
			return herror.InternalServerError(err)
		}
	}

	if ut.GetIsLocked() {
		return herror.Forbidden("this tag is locked")
	}

	// 削除
	if err := repo.DeleteUserTag(userID, ut.GetTagID()); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}

// GetTag GET /tags/:tagID
func (h *Handlers) GetTag(c echo.Context) error {
	tagID := getParamAsUUID(c, consts.ParamTagID)

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

	return c.JSON(http.StatusOK, echo.Map{
		"id":    t.ID,
		"tag":   t.Name,
		"users": users,
	})
}
