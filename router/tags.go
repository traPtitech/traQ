package router

import (
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/event"
	"net/http"
	"time"

	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/rbac"
	"github.com/traPtitech/traQ/rbac/permission"
)

// TagForResponse クライアントに返す形のタグ構造体
type TagForResponse struct {
	ID        string    `json:"tagId"`
	Tag       string    `json:"tag"`
	IsLocked  bool      `json:"isLocked"`
	Editable  bool      `json:"editable"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// TagListForResponse クライアントに返す形のタグリスト構造体
type TagListForResponse struct {
	ID       string             `json:"tagId"`
	Tag      string             `json:"tag"`
	Editable bool               `json:"editable"`
	Type     string             `json:"type"`
	Users    []*UserForResponse `json:"users"`
}

// GetUserTags GET /users/:userID/tags
func GetUserTags(c echo.Context) error {
	userID := uuid.FromStringOrNil(c.Param("userID"))

	// ユーザー確認
	user, err := model.GetUser(userID)
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	res, err := getUserTags(user.GetUID(), c)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, res)
}

// PostUserTag POST /users/{userID}/tags のハンドラ
func PostUserTag(c echo.Context) error {
	userID := uuid.FromStringOrNil(c.Param("userID"))

	// リクエスト検証
	req := struct {
		Tag string `json:"tag" validate:"required,max=30"`
	}{}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	// ユーザー確認
	user, err := model.GetUser(userID)
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	// タグの確認
	t, err := model.GetOrCreateTagByName(req.Tag)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	// 操作制約付きタグ
	if t.Restricted {
		reqUser := c.Get("user").(*model.User)
		r := c.Get("rbac").(*rbac.RBAC)

		if !r.IsGranted(reqUser.GetUID(), reqUser.Role, permission.OperateForRestrictedTag) {
			return echo.NewHTTPError(http.StatusForbidden)
		}
	}

	// ユーザーにタグを付与
	if err := model.AddUserTag(user.GetUID(), t.GetID()); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	go event.Emit(event.UserTagsUpdated, event.UserEvent{ID: user.ID})
	return c.NoContent(http.StatusCreated)
}

// PatchUserTag PATCH /users/{userID}/tags/{tagID} のハンドラ
func PatchUserTag(c echo.Context) error {
	reqUser := c.Get("user").(*model.User)
	userID := uuid.FromStringOrNil(c.Param("userID"))
	tagID := uuid.FromStringOrNil(c.Param("tagID"))

	// リクエスト検証
	body := struct {
		IsLocked bool `json:"isLocked"`
	}{}
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	// タグがつけられているかを見る
	ut, err := model.GetUserTag(userID, tagID)
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	// 他人のロックは変更不可
	if reqUser.GetUID() != userID {
		return echo.NewHTTPError(http.StatusForbidden)
	}

	// 操作制約付きタグは無効
	if ut.Tag.Restricted {
		return echo.NewHTTPError(http.StatusForbidden)
	}

	// 更新
	if err := model.ChangeUserTagLock(userID, ut.Tag.GetID(), body.IsLocked); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	go event.Emit(event.UserTagsUpdated, event.UserEvent{ID: userID.String()})
	return c.NoContent(http.StatusNoContent)
}

// DeleteUserTag DELETE /users/{userID}/tags/{tagID} のハンドラ
func DeleteUserTag(c echo.Context) error {
	userID := uuid.FromStringOrNil(c.Param("userID"))
	tagID := uuid.FromStringOrNil(c.Param("tagID"))

	// タグがつけられているかを見る
	ut, err := model.GetUserTag(userID, tagID)
	if err != nil {
		switch err {
		case model.ErrNotFound: //既にない
			return c.NoContent(http.StatusNoContent)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	// 操作制約付きタグ
	if ut.Tag.Restricted {
		reqUser := c.Get("user").(*model.User)
		r := c.Get("rbac").(*rbac.RBAC)

		if !r.IsGranted(reqUser.GetUID(), reqUser.Role, permission.OperateForRestrictedTag) {
			return echo.NewHTTPError(http.StatusForbidden)
		}
	}

	// 削除
	if err := model.DeleteUserTag(userID, ut.Tag.GetID()); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	go event.Emit(event.UserTagsUpdated, event.UserEvent{ID: userID.String()})
	return c.NoContent(http.StatusNoContent)
}

// GetAllTags GET /tags のハンドラ
func GetAllTags(c echo.Context) error {
	tags, err := model.GetAllTags()
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	res := make([]*TagListForResponse, len(tags))

	for i, v := range tags {
		var users []*UserForResponse
		users, err := getUsersByTagName(v.Name, c)
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

// GetUsersByTagID GET /tags/{tagID} のハンドラ
func GetUsersByTagID(c echo.Context) error {
	id := uuid.FromStringOrNil(c.Param("tagID"))

	t, err := model.GetTagByID(id)
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound, "TagID doesn't exist")
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	users, err := getUsersByTagName(t.Name, c)
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

// PatchTag PATCH /tags/{tagID} のハンドラ
func PatchTag(c echo.Context) error {
	// タグ存在確認
	t, err := model.GetTagByID(uuid.FromStringOrNil(c.Param("tagID")))
	if err != nil {
		switch err {
		case model.ErrNotFound:
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
		reqUser := c.Get("user").(*model.User)
		r := c.Get("rbac").(*rbac.RBAC)

		if !r.IsGranted(reqUser.GetUID(), reqUser.Role, permission.OperateForRestrictedTag) {
			return echo.NewHTTPError(http.StatusForbidden)
		}

		if err := model.ChangeTagRestrict(t.GetID(), *req.Restrict); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	// タグタイプ変更
	if req.Type != nil {
		reqUser := c.Get("user").(*model.User)
		r := c.Get("rbac").(*rbac.RBAC)

		if !r.IsGranted(reqUser.GetUID(), reqUser.Role, permission.OperateForRestrictedTag) {
			return echo.NewHTTPError(http.StatusForbidden)
		}

		if err := model.ChangeTagType(t.GetID(), *req.Type); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	return c.NoContent(http.StatusNoContent)
}

func getUserTags(userID uuid.UUID, c echo.Context) ([]*TagForResponse, error) {
	tagList, err := model.GetUserTagsByUserID(userID)
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return nil, echo.NewHTTPError(http.StatusNotFound, "This user doesn't exist")
		default:
			c.Logger().Error(err)
			return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to get tagList")
		}
	}

	res := make([]*TagForResponse, len(tagList))
	for i, v := range tagList {
		res[i] = formatTag(v)
	}
	return res, nil
}

func getUsersByTagName(name string, c echo.Context) ([]*UserForResponse, error) {
	users, err := model.GetUsersByTag(name)
	if err != nil {
		c.Logger().Error(err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to get userList")
	}
	res := make([]*UserForResponse, len(users))
	for i, v := range users {
		res[i] = formatUser(v)
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
