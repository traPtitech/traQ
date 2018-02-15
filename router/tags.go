package router

import (
	"github.com/traPtitech/traQ/notification"
	"github.com/traPtitech/traQ/notification/events"
	"net/http"

	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/model"
)

// TagForResponse クライアントに返す形のタグ構造体
type TagForResponse struct {
	ID       string `json:"tagId"`
	Tag      string `json:"tag"`
	IsLocked bool   `json:"isLocked"`
}

// GetUserTags /users/{userID}/tags のGETメソッド
func GetUserTags(c echo.Context) error {
	ID := c.Param("userID")
	res, err := getUserTags(ID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, res)
}

// PostUserTag /users/{userID}/tags のPOSTメソッド
func PostUserTag(c echo.Context) error {
	userID := c.Param("userID")

	reqBody := struct {
		Tag string `json:"tag"`
	}{}
	if err := c.Bind(&reqBody); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid format")
	}

	tag := &model.UsersTag{
		UserID: userID,
	}
	if err := tag.Create(reqBody.Tag); err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Failed to create tag")
	}

	res, err := getUserTags(userID)
	if err != nil {
		return err
	}

	go notification.Send(events.UserTagsUpdated, events.UserEvent{ID: userID})
	return c.JSON(http.StatusCreated, res)
}

// PutUserTag /users/{userID}/tags/{tagID} のPUTメソッド
func PutUserTag(c echo.Context) error {
	userID := c.Param("userID")
	tagID := c.Param("tagID")

	reqBody := struct {
		IsLocked bool `json:"isLocked"`
	}{}
	if err := c.Bind(&reqBody); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid format")
	}

	tag, err := model.GetTag(userID, tagID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Failed to get tag")
	}
	tag.IsLocked = reqBody.IsLocked

	if err := tag.Update(); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update tag")
	}

	res, err := getUserTags(userID)
	if err != nil {
		return err
	}

	go notification.Send(events.UserTagsUpdated, events.UserEvent{ID: userID})
	return c.JSON(http.StatusOK, res)
}

// DeleteUserTag /users/{userID}/tags/{tagID} のDELETEメソッド
func DeleteUserTag(c echo.Context) error {
	userID := c.Param("userID")
	tagID := c.Param("tagID")

	tag, err := model.GetTag(userID, tagID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Failed to get tag")
	}

	if err := tag.Delete(); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to delete tag")
	}

	go notification.Send(events.UserTagsUpdated, events.UserEvent{ID: userID})
	return c.NoContent(http.StatusNoContent)
}

func getUserTags(ID string) ([]*TagForResponse, error) {
	tagList, err := model.GetUserTagsByUserID(ID)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusNotFound, "Tags are not found")
	}

	var res []*TagForResponse
	for _, v := range tagList {
		t, err := formatTag(v)
		if err != nil {
			return nil, err
		}
		res = append(res, t)
	}
	return res, nil

}

func formatTag(userTag *model.UsersTag) (*TagForResponse, error) {
	tag, err := model.GetTagByID(userTag.TagID)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Tag is not found")
	}
	return &TagForResponse{
		ID:       tag.ID,
		Tag:      tag.Name,
		IsLocked: userTag.IsLocked,
	}, nil
}
