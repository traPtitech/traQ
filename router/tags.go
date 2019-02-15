package router

import (
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/repository"
	"net/http"
	"time"

	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/rbac/permission"
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

// TagListForResponse クライアントに返す形のタグリスト構造体
type TagListForResponse struct {
	ID       uuid.UUID          `json:"tagId"`
	Tag      string             `json:"tag"`
	Editable bool               `json:"editable"`
	Type     string             `json:"type"`
	Users    []*UserForResponse `json:"users"`
}

// GetUserTags GET /users/:userID/tags
func (h *Handlers) GetUserTags(c echo.Context) error {
	userID := getRequestParamAsUUID(c, paramUserID)

	// ユーザー確認
	if ok, err := h.Repo.UserExists(userID); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	} else if !ok {
		return echo.NewHTTPError(http.StatusNotFound)
	}

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
	req := struct {
		Tag string `json:"tag" validate:"required,max=30"`
	}{}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	// ユーザー確認
	if ok, err := h.Repo.UserExists(userID); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	} else if !ok {
		return echo.NewHTTPError(http.StatusNotFound)
	}

	// タグの確認
	t, err := h.Repo.GetOrCreateTagByName(req.Tag)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	// 操作制約付きタグ
	if t.Restricted {
		reqUser := getRequestUser(c)
		r := getRBAC(c)

		if !r.IsGranted(reqUser.ID, reqUser.Role, permission.OperateForRestrictedTag) {
			return echo.NewHTTPError(http.StatusForbidden)
		}
	}

	// ユーザーにタグを付与
	if err := h.Repo.AddUserTag(userID, t.ID); err != nil {
		switch err {
		case repository.ErrAlreadyExists:
			return c.NoContent(http.StatusNoContent)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
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
	body := struct {
		IsLocked bool `json:"isLocked"`
	}{}
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	// タグがつけられているかを見る
	ut, err := h.Repo.GetUserTag(userID, tagID)
	if err != nil {
		switch err {
		case repository.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	// 他人のロックは変更不可
	if me != userID {
		return echo.NewHTTPError(http.StatusForbidden)
	}

	// 操作制約付きタグは無効
	if ut.Tag.Restricted {
		return echo.NewHTTPError(http.StatusForbidden)
	}

	// 更新
	if err := h.Repo.ChangeUserTagLock(userID, ut.Tag.ID, body.IsLocked); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
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
		case repository.ErrNotFound: //既にない
			return c.NoContent(http.StatusNoContent)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	// 操作制約付きタグ
	if ut.Tag.Restricted {
		reqUser := getRequestUser(c)
		r := getRBAC(c)

		if !r.IsGranted(reqUser.ID, reqUser.Role, permission.OperateForRestrictedTag) {
			return echo.NewHTTPError(http.StatusForbidden)
		}
	}

	// 削除
	if err := h.Repo.DeleteUserTag(userID, ut.Tag.ID); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}

// GetAllTags GET /tags
func (h *Handlers) GetAllTags(c echo.Context) error {
	tags, err := h.Repo.GetAllTags()
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	res := make([]*TagListForResponse, len(tags))

	for i, v := range tags {
		var users []*UserForResponse
		users, err := h.getUsersByTagName(v.Name, c)
		if err != nil {
			return err
		}

		res[i] = &TagListForResponse{
			ID:       v.ID,
			Tag:      v.Name,
			Editable: !v.Restricted,
			Type:     v.Type,
			Users:    users,
		}
	}

	return c.JSON(http.StatusOK, res)
}

// GetUsersByTagID GET /tags/:tagID
func (h *Handlers) GetUsersByTagID(c echo.Context) error {
	tagID := getRequestParamAsUUID(c, paramTagID)

	t, err := h.Repo.GetTagByID(tagID)
	if err != nil {
		switch err {
		case repository.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound, "TagID doesn't exist")
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	users, err := h.getUsersByTagName(t.Name, c)
	if err != nil {
		return err
	}

	res := &TagListForResponse{
		ID:       t.ID,
		Tag:      t.Name,
		Editable: !t.Restricted,
		Type:     t.Type,
		Users:    users,
	}

	return c.JSON(http.StatusOK, res)
}

// PatchTag PATCH /tags/:tagID
func (h *Handlers) PatchTag(c echo.Context) error {
	tagID := getRequestParamAsUUID(c, paramTagID)

	// タグ存在確認
	_, err := h.Repo.GetTagByID(tagID)
	if err != nil {
		switch err {
		case repository.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	// リクエスト検証
	req := struct {
		Type     *string `json:"type"`
		Restrict *bool   `json:"restrict"`
	}{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	// 制約変更
	if req.Restrict != nil {
		reqUser := getRequestUser(c)
		r := getRBAC(c)

		if !r.IsGranted(reqUser.ID, reqUser.Role, permission.OperateForRestrictedTag) {
			return echo.NewHTTPError(http.StatusForbidden)
		}

		if err := h.Repo.ChangeTagRestrict(tagID, *req.Restrict); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	// タグタイプ変更
	if req.Type != nil {
		reqUser := getRequestUser(c)
		r := getRBAC(c)

		if !r.IsGranted(reqUser.ID, reqUser.Role, permission.OperateForRestrictedTag) {
			return echo.NewHTTPError(http.StatusForbidden)
		}

		if err := h.Repo.ChangeTagType(tagID, *req.Type); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *Handlers) getUserTags(userID uuid.UUID, c echo.Context) ([]*TagForResponse, error) {
	tagList, err := h.Repo.GetUserTagsByUserID(userID)
	if err != nil {
		c.Logger().Error(err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to get tagList")
	}

	res := make([]*TagForResponse, len(tagList))
	for i, v := range tagList {
		res[i] = formatTag(v)
	}
	return res, nil
}

func (h *Handlers) getUsersByTagName(name string, c echo.Context) ([]*UserForResponse, error) {
	users, err := h.Repo.GetUsersByTag(name)
	if err != nil {
		c.Logger().Error(err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to get userList")
	}
	res := make([]*UserForResponse, len(users))
	for i, v := range users {
		res[i] = h.formatUser(v)
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
