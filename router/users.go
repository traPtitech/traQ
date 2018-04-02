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
	"github.com/traPtitech/traQ/rbac/role"
)

// UserForResponse クライアントに返す形のユーザー構造体
type UserForResponse struct {
	UserID      string `json:"userId"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	IconID      string `json:"iconFileId"`
	Bot         bool   `json:"bot"`
}

// UserDetailForResponse クライアントに返す形の詳細ユーザー構造体
type UserDetailForResponse struct {
	UserID      string            `json:"userId"`
	Name        string            `json:"name"`
	DisplayName string            `json:"displayName"`
	IconID      string            `json:"iconFileId"`
	Bot         bool              `json:"bot"`
	TagList     []*TagForResponse `json:"tagList"`
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
	return c.NoContent(http.StatusNoContent)
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
	me := c.Get("user").(*model.User)
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

	return c.Redirect(http.StatusFound, "/api/1.0/files/"+user.Icon)
}

// GetMyIcon GET /users/me/icon のハンドラ
func GetMyIcon(c echo.Context) error {
	user := c.Get("user").(*model.User)
	return c.Redirect(http.StatusFound, "/api/1.0/files/"+user.Icon)
}

// PutMyIcon Post /users/me/icon のハンドラ
func PutMyIcon(c echo.Context) error {
	user := c.Get("user").(*model.User)

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

	go notification.Send(events.UserIconUpdated, events.UserEvent{ID: user.ID})
	return c.NoContent(http.StatusOK)
}

// PatchMe PUT /users/me
func PatchMe(c echo.Context) error {
	user := c.Get("user").(*model.User)

	req := struct {
		DisplayName *string `json:"displayName"`
	}{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}
	if req.DisplayName != nil {
		if err := user.UpdateDisplayName(*req.DisplayName); err != nil {
			switch err {
			case model.ErrUserInvalidDisplayName:
				return echo.NewHTTPError(http.StatusBadRequest, err)
			default:
				c.Logger().Error(err)
				return echo.NewHTTPError(http.StatusInternalServerError)
			}
		}
	}

	go notification.Send(events.UserUpdated, events.UserEvent{ID: user.ID})
	return c.NoContent(http.StatusNoContent)
}

// PostUsers Post /users のハンドラ
func PostUsers(c echo.Context) error {
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
		Role:  role.User.ID(),
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
	res := &UserForResponse{
		UserID:      user.ID,
		Name:        user.Name,
		DisplayName: user.DisplayName,
		IconID:      user.Icon,
		Bot:         user.Bot,
	}
	if len(res.DisplayName) == 0 {
		res.DisplayName = res.Name
	}
	return res
}

func formatUserDetail(user *model.User, tagList []*model.UsersTag) (*UserDetailForResponse, error) {
	userDetail := &UserDetailForResponse{
		UserID:      user.ID,
		Name:        user.Name,
		DisplayName: user.DisplayName,
		IconID:      user.Icon,
		Bot:         user.Bot,
	}
	if len(userDetail.DisplayName) == 0 {
		userDetail.DisplayName = userDetail.Name
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

func validateUserID(userID string) (*model.User, error) {
	u, err := model.GetUser(userID)
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return nil, echo.NewHTTPError(http.StatusNotFound, "This user dosen't exist")
		default:
			log.Errorf("failed to get usee: %v", err)
			return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to get user")
		}
	}
	return u, nil
}
