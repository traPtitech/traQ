package router

import (
	"github.com/traPtitech/traQ/notification"
	"github.com/traPtitech/traQ/notification/events"
	"github.com/traPtitech/traQ/rbac"
	"github.com/traPtitech/traQ/rbac/permission"
	"gopkg.in/go-playground/validator.v9"
	"net/http"
	"time"

	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/model"
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

// GetUserTags GET /users/{userID}/tags のハンドラ
func GetUserTags(c echo.Context) error {
	id := c.Param("userID")
	res, err := getUserTags(id, c)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, res)
}

// PostUserTag POST /users/{userID}/tags のハンドラ
func PostUserTag(c echo.Context) error {
	id := c.Param("userID")

	// リクエスト検証
	req := struct {
		Tag string `json:"tag" validate:"required,max=30"`
	}{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}
	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	// ユーザー確認
	_, err := validateUserID(id)
	if err != nil {
		return err
	}

	// タグの確認
	t, err := model.GetTagByName(req.Tag)
	if err != nil {
		switch err {
		case model.ErrNotFound:
			// 存在しないので新規作成
			t = &model.Tag{
				Name: req.Tag,
			}
			if err := t.Create(); err != nil {
				c.Logger().Error(err)
				return echo.NewHTTPError(http.StatusInternalServerError)
			}
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
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
	ut := &model.UsersTag{
		UserID: id,
	}
	if err := ut.Create(req.Tag); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	go notification.Send(events.UserTagsUpdated, events.UserEvent{ID: id})
	return c.NoContent(http.StatusCreated)
}

// PatchUserTag PATCH /users/{userID}/tags/{tagID} のハンドラ
func PatchUserTag(c echo.Context) error {
	reqUser := c.Get("user").(*model.User)
	userID := c.Param("userID")
	tagID := c.Param("tagID")

	// リクエスト検証
	body := struct {
		IsLocked bool `json:"isLocked"`
	}{}
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	// タグがつけられているかを見る
	ut, err := model.GetTag(userID, tagID)
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
	if reqUser.ID != userID {
		return echo.NewHTTPError(http.StatusForbidden)
	}

	// タグ情報取得
	t, err := model.GetTagByID(tagID)
	if err != nil {
		switch err {
		case model.ErrNotFound: // ここには来ないはず
			c.Logger().Debug("UNEXPECTED CODE FLOW")
			return echo.NewHTTPError(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	// 操作制約付きタグは無効
	if t.Restricted {
		return echo.NewHTTPError(http.StatusForbidden)
	}

	// 更新
	ut.IsLocked = body.IsLocked
	if err := ut.Update(); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	go notification.Send(events.UserTagsUpdated, events.UserEvent{ID: userID})
	return c.NoContent(http.StatusNoContent)
}

// DeleteUserTag DELETE /users/{userID}/tags/{tagID} のハンドラ
func DeleteUserTag(c echo.Context) error {
	userID := c.Param("userID")
	tagID := c.Param("tagID")

	// タグがつけられているかを見る
	ut, err := model.GetTag(userID, tagID)
	if err != nil {
		switch err {
		case model.ErrNotFound: //既にない
			return c.NoContent(http.StatusNoContent)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	// タグ情報取得
	t, err := model.GetTagByID(tagID)
	if err != nil {
		switch err {
		case model.ErrNotFound: // ここには来ないはず
			c.Logger().Debug("UNEXPECTED CODE FLOW")
			return c.NoContent(http.StatusNoContent)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	// 操作制約付きタグ
	if t.Restricted {
		reqUser := c.Get("user").(*model.User)
		r := c.Get("rbac").(*rbac.RBAC)

		if !r.IsGranted(reqUser.GetUID(), reqUser.Role, permission.OperateForRestrictedTag) {
			return echo.NewHTTPError(http.StatusForbidden)
		}
	}

	// 削除
	if err := ut.Delete(); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	go notification.Send(events.UserTagsUpdated, events.UserEvent{ID: userID})
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
	id := c.Param("tagID")

	t, err := model.GetTagByID(id)
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound, "TagID doesn't exist")
		default:
			c.Logger().Errorf("failed to get tag: %v", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "An error occurred while checking existence of tag")
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
	id := c.Param("tagID")

	// リクエスト検証
	req := struct {
		Type     *string `json:"type"`
		Restrict *bool   `json:"restrict"`
		Name     *string `json:"name"`
	}{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	// タグ存在確認
	t, err := model.GetTagByID(id)
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	// タグ名変更
	if req.Name != nil {
		if _, err := model.GetTagByName(*req.Name); err != nil {
			switch err {
			case model.ErrNotFound:
				//OK
			default:
				c.Logger().Error(err)
				return echo.NewHTTPError(http.StatusInternalServerError)
			}
		} else {
			// 既に存在してかぶる
			return echo.NewHTTPError(http.StatusBadRequest, "name's tag has already existed.")
		}
		t.Name = *req.Name
	}

	// 制約変更
	if req.Restrict != nil {
		reqUser := c.Get("user").(*model.User)
		r := c.Get("rbac").(*rbac.RBAC)

		if !r.IsGranted(reqUser.GetUID(), reqUser.Role, permission.OperateForRestrictedTag) {
			return echo.NewHTTPError(http.StatusForbidden)
		}

		t.Restricted = *req.Restrict
	}

	// タグタイプ変更
	if req.Type != nil {
		reqUser := c.Get("user").(*model.User)
		r := c.Get("rbac").(*rbac.RBAC)

		if !r.IsGranted(reqUser.GetUID(), reqUser.Role, permission.OperateForRestrictedTag) {
			return echo.NewHTTPError(http.StatusForbidden)
		}

		t.Type = *req.Type
	}

	// 更新
	if err := t.Update(); err != nil {
		switch err.(type) {
		case *validator.ValidationErrors:
			return echo.NewHTTPError(http.StatusBadRequest, err)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	return c.NoContent(http.StatusNoContent)
}

func getUserTags(ID string, c echo.Context) ([]*TagForResponse, error) {
	tagList, err := model.GetUserTagsByUserID(ID)
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return nil, echo.NewHTTPError(http.StatusNotFound, "This user doesn't exist")
		default:
			c.Logger().Errorf("failed to get tagList: %v", err)
			return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to get tagList")
		}
	}

	var res []*TagForResponse
	for _, v := range tagList {
		t, err := formatTag(v)
		if err != nil {
			c.Logger().Errorf("failed to get tag by ID: %v", err)
			return nil, err
		}
		res = append(res, t)
	}
	return res, nil

}

func getUsersByTagName(name string, c echo.Context) ([]*UserForResponse, error) {
	var users []*UserForResponse

	idList, err := model.GetUserIDsByTags([]string{name})
	if err != nil {
		c.Logger().Errorf("failed to get users by tagName: %v", err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to get userList")
	}
	for _, v := range idList {
		u, err := model.GetUser(v.String())
		if err != nil {
			c.Logger().Errorf("failed to get user infomation: %v", err)
			return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to get users")
		}
		users = append(users, formatUser(u))
	}
	return users, nil
}

func formatTag(ut *model.UsersTag) (*TagForResponse, error) {
	tag, err := model.GetTagByID(ut.TagID)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to get tag infomation")
	}
	return &TagForResponse{
		ID:        tag.ID,
		Tag:       tag.Name,
		IsLocked:  ut.IsLocked || tag.Restricted,
		Editable:  !tag.Restricted,
		Type:      tag.Type,
		CreatedAt: ut.CreatedAt,
		UpdatedAt: ut.UpdatedAt,
	}, nil
}
