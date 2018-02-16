package router

import (
	"fmt"
	"net/http"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo"
	"github.com/labstack/echo-contrib/session"
	"github.com/traPtitech/traQ/model"
)

// UserForResponse クライアントに返す形のユーザー構造体
type UserForResponse struct {
	UserID string `json:"userId"`
	Name   string `json:"name"`
}

// UserDetailForResponse クライアントに返す形の詳細ユーザー構造体
type UserDetailForResponse struct {
	UserID  string            `json:"userId"`
	Name    string            `json:"name"`
	TagList []*TagForResponse `json:"tagList"`
}

type loginRequestBody struct {
	Name string `json:"name" form:"name"`
	Pass string `json:"pass" form:"pass"`
}

// PostLogin Post /login のハンドラ
func PostLogin(c echo.Context) error {
	requestBody := &loginRequestBody{}
	err := c.Bind(requestBody)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprint(err))
	}

	user := &model.User{
		Name: requestBody.Name,
	}
	err = user.Authorization(requestBody.Pass)
	if err != nil {
		return echo.NewHTTPError(http.StatusForbidden, fmt.Sprint(err))
	}

	sess, err := session.Get("sessions", c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("an error occurrerd while getting session: %v", err))
	}

	sess.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 14,
		HttpOnly: true,
	}

	sess.Values["userID"] = user.ID
	sess.Save(c.Request(), c.Response())
	return c.NoContent(http.StatusOK)
}

// PostLogout Post /logout のハンドラ
func PostLogout(c echo.Context) error {
	sess, err := session.Get("sessions", c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "an error occurrerd while getting session")
	}

	sess.Values["userID"] = nil
	sess.Save(c.Request(), c.Response())
	return c.NoContent(http.StatusOK)
}

// GetUsers GET /users のハンドラ
func GetUsers(c echo.Context) error {
	users, err := model.GetUsers()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Can't get Users")
	}

	res := make([]*UserForResponse, 0)
	for _, user := range users {
		res = append(res, formatUser(user))
	}
	return c.JSON(http.StatusOK, res)
}

// GetMe GET /users/me のハンドラ
func GetMe(c echo.Context) error {
	userID := c.Get("user").(*model.User).ID

	me, err := model.GetUser(userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Can't get you")
	}
	return c.JSON(http.StatusOK, formatUser(me))
}

// GetUserByID /GET /users/{userID} のハンドラ
func GetUserByID(c echo.Context) error {
	userID := c.Param("userID")
	user, err := model.GetUser(userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	tagList, err := model.GetUserTagsByUserID(userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	userDetail, err := formatUserDetail(user, tagList)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, userDetail)
}

func formatUser(user *model.User) *UserForResponse {
	return &UserForResponse{
		UserID: user.ID,
		Name:   user.Name,
	}
}

func formatUserDetail(user *model.User, tagList []*model.UsersTag) (*UserDetailForResponse, error) {
	userDetail := &UserDetailForResponse{
		UserID: user.ID,
		Name:   user.Name,
	}
	for _, tag := range tagList {
		formatedTag, err := formatTag(tag)
		if err != nil {
			return nil, err
		}
		userDetail.TagList = append(userDetail.TagList, formatedTag)
	}
	return userDetail, nil
}
