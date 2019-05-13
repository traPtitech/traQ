package router

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/repository"
	"net/http"
	"time"

	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/model"
)

// TagForResponse クライアントに返す形のタグ構造体
type TagForResponse struct {
	ID        uuid.UUID `json:"tagId"`
	Tag       string    `json:"tag"`
	IsLocked  bool      `json:"isLocked"`
	Editable  bool      `json:"editable"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// GetUserTags GET /users/:userID/tags
func (h *Handlers) GetUserTags(c echo.Context) error {
	userID := getRequestParamAsUUID(c, paramUserID)

	res, err := h.getUserTags(userID, c)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, res)
}

// PostUserTag POST /users/:userID/tags
func (h *Handlers) PostUserTag(c echo.Context) error {
	userID := getRequestParamAsUUID(c, paramUserID)

	// リクエスト検証
	var req struct {
		Tag string `json:"tag"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return badRequest(err)
	}

	// タグの確認
	t, err := h.Repo.GetOrCreateTagByName(req.Tag)
	if err != nil {
		switch {
		case repository.IsArgError(err):
			return badRequest(err)
		case err == repository.ErrNotFound:
			return badRequest("empty tag")
		default:
			return internalServerError(err, h.requestContextLogger(c))
		}
	}

	// ユーザーにタグを付与
	if err := h.Repo.AddUserTag(userID, t.ID); err != nil {
		switch err {
		case repository.ErrAlreadyExists:
			return c.NoContent(http.StatusNoContent)
		default:
			return internalServerError(err, h.requestContextLogger(c))
		}
	}

	return c.NoContent(http.StatusCreated)
}

// PatchUserTag PATCH /users/:userID/tags/:tagID
func (h *Handlers) PatchUserTag(c echo.Context) error {
	me := getRequestUserID(c)
	userID := getRequestParamAsUUID(c, paramUserID)
	tagID := getRequestParamAsUUID(c, paramTagID)

	// リクエスト検証
	var req struct {
		IsLocked bool `json:"isLocked"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return badRequest(err)
	}

	// タグがつけられているかを見る
	ut, err := h.Repo.GetUserTag(userID, tagID)
	if err != nil {
		switch err {
		case repository.ErrNotFound:
			return notFound()
		default:
			return internalServerError(err, h.requestContextLogger(c))
		}
	}

	// 他人のロックは変更不可
	if me != userID {
		return forbidden("this is not your tag")
	}

	// 更新
	if err := h.Repo.ChangeUserTagLock(userID, ut.Tag.ID, req.IsLocked); err != nil {
		return internalServerError(err, h.requestContextLogger(c))
	}

	return c.NoContent(http.StatusNoContent)
}

// DeleteUserTag DELETE /users/:userID/tags/:tagID
func (h *Handlers) DeleteUserTag(c echo.Context) error {
	userID := getRequestParamAsUUID(c, paramUserID)
	tagID := getRequestParamAsUUID(c, paramTagID)

	// タグがつけられているかを見る
	ut, err := h.Repo.GetUserTag(userID, tagID)
	if err != nil {
		switch err {
		case repository.ErrNotFound: // 既にない
			return c.NoContent(http.StatusNoContent)
		default:
			return internalServerError(err, h.requestContextLogger(c))
		}
	}

	// 削除
	if err := h.Repo.DeleteUserTag(userID, ut.Tag.ID); err != nil {
		return internalServerError(err, h.requestContextLogger(c))
	}

	return c.NoContent(http.StatusNoContent)
}

// GetUsersByTagID GET /tags/:tagID
func (h *Handlers) GetUsersByTagID(c echo.Context) error {
	type response struct {
		ID       uuid.UUID   `json:"tagId"`
		Tag      string      `json:"tag"`
		Editable bool        `json:"editable"`
		Type     string      `json:"type"`
		Users    []uuid.UUID `json:"users"`
	}

	tagID := getRequestParamAsUUID(c, paramTagID)

	t, err := h.Repo.GetTagByID(tagID)
	if err != nil {
		switch err {
		case repository.ErrNotFound:
			return notFound()
		default:
			return internalServerError(err, h.requestContextLogger(c))
		}
	}

	users, err := h.Repo.GetUserIDsByTagID(t.ID)
	if err != nil {
		return internalServerError(err, h.requestContextLogger(c))
	}

	return c.JSON(http.StatusOK, &response{
		ID:       t.ID,
		Tag:      t.Name,
		Editable: !t.Restricted,
		Type:     t.Type,
		Users:    users,
	})
}

func (h *Handlers) getUserTags(userID uuid.UUID, c echo.Context) ([]*TagForResponse, error) {
	tagList, err := h.Repo.GetUserTagsByUserID(userID)
	if err != nil {
		return nil, internalServerError(err, h.requestContextLogger(c))
	}

	res := make([]*TagForResponse, len(tagList))
	for i, v := range tagList {
		res[i] = formatTag(v)
	}
	return res, nil
}

func formatTag(ut *model.UsersTag) *TagForResponse {
	tag := ut.Tag
	return &TagForResponse{
		ID:        tag.ID,
		Tag:       tag.Name,
		IsLocked:  ut.IsLocked || tag.Restricted,
		Editable:  !tag.Restricted,
		Type:      tag.Type,
		CreatedAt: ut.CreatedAt,
		UpdatedAt: ut.UpdatedAt,
	}
}
