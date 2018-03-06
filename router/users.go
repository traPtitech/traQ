package router

import (
	"fmt"
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/gommon/log"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/notification"
	"github.com/traPtitech/traQ/notification/events"
)

// UserForResponse クライアントに返す形のユーザー構造体
type UserForResponse struct {
	UserID string `json:"userId"`
	Name   string `json:"name"`
	IconID string `json:"iconFileId"`
}

// UserDetailForResponse クライアントに返す形の詳細ユーザー構造体
type UserDetailForResponse struct {
	UserID  string            `json:"userId"`
	Name    string            `json:"name"`
	IconID  string            `json:"iconFileId"`
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
		return echo.NewHTTPError(http.StatusInternalServerError, "an error occurred while getting session")
	}

	sess.Values["userID"] = nil
	sess.Save(c.Request(), c.Response())
	return c.NoContent(http.StatusNoContent)
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

// GetUserIcon GET /users/{userID}/icon のハンドラ
func GetUserIcon(c echo.Context) error {
	userID := c.Param("userID")

	user, err := model.GetUser(userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	file, err := model.OpenFileByID(user.Icon)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}
	defer file.Close()

	meta, err := model.GetMetaFileDataByID(user.Icon)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.Stream(http.StatusOK, meta.Mime, file)
}

// GetMyIcon GET /users/me/icon のハンドラ
func GetMyIcon(c echo.Context) error {
	userID := c.Get("user").(*model.User).ID

	user, err := model.GetUser(userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	file, err := model.OpenFileByID(user.Icon)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}
	defer file.Close()

	meta, err := model.GetMetaFileDataByID(user.Icon)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.Stream(http.StatusOK, meta.Mime, file)
}

// PutMyIcon Post /users/me/icon のハンドラ
func PutMyIcon(c echo.Context) error {
	userID := c.Get("user").(*model.User).ID

	user, err := model.GetUser(userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	uploadedFile, err := c.FormFile("file")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Failed to upload file: %v", err))
	}

	switch uploadedFile.Header.Get(echo.HeaderContentType) {
	case "image/png", "image/jpeg", "image/gif", "image/svg+xml":
		break
	default:
		return echo.NewHTTPError(http.StatusBadRequest, "invalid image file")
	}

	if uploadedFile.Size > 1024*1024 {
		return echo.NewHTTPError(http.StatusBadRequest, "too big image file")
	}

	file := &model.File{
		Name:      uploadedFile.Filename,
		Size:      uploadedFile.Size,
		CreatorID: user.ID,
	}

	src, err := uploadedFile.Open()
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Failed to open file")
	}
	defer src.Close()

	if err := file.Create(src); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create file")
	}

	if err := user.UpdateIconID(file.ID); err != nil {
		log.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	go notification.Send(events.UserIconUpdated, events.UserEvent{ID: userID})
	return c.NoContent(http.StatusOK)
}

// PostUsers Post /users のハンドラ
// TODO 暫定的仕様
func PostUsers(c echo.Context) error {
	user := c.Get("user").(*model.User)

	if user.Name != "traq" { //TODO 権限をちゃんとする
		return echo.NewHTTPError(http.StatusForbidden)
	}

	req := struct {
		Name     string `json:"name"`
		Password string `json:"password"`
		Email    string `json:"email"`
	}{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	newUser := &model.User{
		Name:  req.Name,
		Email: req.Email,
	}
	if err := newUser.SetPassword(req.Password); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}
	if err := newUser.Create(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	go notification.Send(events.UserJoined, events.UserEvent{ID: newUser.ID})
	return c.NoContent(http.StatusCreated)
}

func formatUser(user *model.User) *UserForResponse {
	return &UserForResponse{
		UserID: user.ID,
		Name:   user.Name,
		IconID: user.Icon,
	}
}

func formatUserDetail(user *model.User, tagList []*model.UsersTag) (*UserDetailForResponse, error) {
	userDetail := &UserDetailForResponse{
		UserID: user.ID,
		Name:   user.Name,
		IconID: user.Icon,
	}
	for _, tag := range tagList {
		formattedTag, err := formatTag(tag)
		if err != nil {
			return nil, err
		}
		userDetail.TagList = append(userDetail.TagList, formattedTag)
	}
	return userDetail, nil
}
