package router

import (
	"net/http"

	"github.com/traPtitech/traQ/notification"
	"github.com/traPtitech/traQ/notification/events"

	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/model"
)

// TagForResponse クライアントに返す形のタグ構造体
type TagForResponse struct {
	ID       string `json:"tagId"`
	Tag      string `json:"tag"`
	IsLocked bool   `json:"isLocked"`
}

// FIXME: 名前が良くない気がする

// TagListForResponse クライアントに返す形のタグリスト構造体
type TagListForResponse struct {
	ID    string             `json:"tagId"`
	Tag   string             `json:"tag"`
	Users []*UserForResponse `json:"users"`
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

	body := struct {
		Tag string `json:"tag"`
	}{}
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid format")
	}

	ut := &model.UsersTag{
		UserID: id,
	}
	if err := ut.Create(body.Tag); err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Failed to create tag")
	}

	res, err := getUserTags(id, c)
	if err != nil {
		return err
	}

	go notification.Send(events.UserTagsUpdated, events.UserEvent{ID: id})
	return c.JSON(http.StatusCreated, res)
}

// PutUserTag PUT /users/{userID}/tags/{tagID} のハンドラ
func PutUserTag(c echo.Context) error {
	userID := c.Param("userID")
	tagID := c.Param("tagID")

	body := struct {
		IsLocked bool `json:"isLocked"`
	}{}
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid format")
	}

	ut, err := validateTagID(tagID, userID, c)
	if err != nil {
		return err
	}

	ut.IsLocked = body.IsLocked

	if err := ut.Update(); err != nil {
		c.Logger().Error("failed to update usersTag: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update tag")
	}

	res, err := getUserTags(userID, c)
	if err != nil {
		return err
	}

	go notification.Send(events.UserTagsUpdated, events.UserEvent{ID: userID})
	return c.JSON(http.StatusOK, res)
}

// DeleteUserTag DELETE /users/{userID}/tags/{tagID} のハンドラ
func DeleteUserTag(c echo.Context) error {
	userID := c.Param("userID")
	tagID := c.Param("tagID")

	ut, err := validateTagID(tagID, userID, c)
	if err != nil {
		return err
	}

	// TODO: 既に削除されている場合にもエラーが出るか確認し、そのときは204を返すようにする
	if err := ut.Delete(); err != nil {
		c.Logger().Error("failed to delete usersTag: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to delete tag")
	}

	go notification.Send(events.UserTagsUpdated, events.UserEvent{ID: userID})
	return c.NoContent(http.StatusNoContent)
}

// GetAllTags GET /tags のハンドラ
func GetAllTags(c echo.Context) error {
	tags, err := model.GetAllTags()
	if err != nil {
		c.Echo().Logger.Errorf("failed to get all tags: %v", err)
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
			ID:    v.ID,
			Tag:   v.Name,
			Users: users,
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
		ID:    t.ID,
		Tag:   t.Name,
		Users: users,
	}

	return c.JSON(http.StatusOK, res)
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
		ID:       tag.ID,
		Tag:      tag.Name,
		IsLocked: ut.IsLocked,
	}, nil
}

func validateTagID(tagID, userID string, c echo.Context) (*model.UsersTag, error) {
	if _, err := model.GetTagByID(tagID); err != nil {
		switch err {
		case model.ErrNotFound:
			return nil, echo.NewHTTPError(http.StatusNotFound, "The specified tag does not exist")
		default:
			c.Logger().Errorf("failed to get tag: %v", err)
			return nil, echo.NewHTTPError(http.StatusInternalServerError, "An error occurred while checking existence of tag")
		}
	}

	userTag, err := model.GetTag(userID, tagID)
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return nil, echo.NewHTTPError(http.StatusNotFound, "The specified tag does not exist")
		default:
			c.Logger().Errorf("failed to get tag: %v", err)
			return nil, echo.NewHTTPError(http.StatusInternalServerError, "An error occurred while checking existence of tag")
		}
	}

	return userTag, nil
}
