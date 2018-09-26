package router

import (
	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/rbac/role"
	"github.com/traPtitech/traQ/sessions"
	"net/http"
	"time"
)

// UserForResponse クライアントに返す形のユーザー構造体
type UserForResponse struct {
	UserID      string     `json:"userId"`
	Name        string     `json:"name"`
	DisplayName string     `json:"displayName"`
	IconID      string     `json:"iconFileId"`
	Bot         bool       `json:"bot"`
	TwitterID   string     `json:"twitterId"`
	LastOnline  *time.Time `json:"lastOnline"`
	IsOnline    bool       `json:"isOnline"`
}

// UserDetailForResponse クライアントに返す形の詳細ユーザー構造体
type UserDetailForResponse struct {
	UserID      string            `json:"userId"`
	Name        string            `json:"name"`
	DisplayName string            `json:"displayName"`
	IconID      string            `json:"iconFileId"`
	Bot         bool              `json:"bot"`
	TwitterID   string            `json:"twitterId"`
	LastOnline  *time.Time        `json:"lastOnline"`
	IsOnline    bool              `json:"isOnline"`
	TagList     []*TagForResponse `json:"tagList"`
}

// PostLogin POST /login
func PostLogin(c echo.Context) error {
	req := struct {
		Name string `json:"name" form:"name" validate:"required"`
		Pass string `json:"pass" form:"pass" validate:"required"`
	}{}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	user, err := model.GetUserByName(req.Name)
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusBadRequest, "name or password is wrong")
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}
	if err := model.AuthenticateUser(user, req.Pass); err != nil {
		return echo.NewHTTPError(http.StatusForbidden, err)
	}

	sess, err := sessions.Get(c.Response(), c.Request(), true)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	if err := sess.SetUser(user.GetUID()); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}
	return c.NoContent(http.StatusNoContent)
}

// PostLogout POST /logout
func PostLogout(c echo.Context) error {
	sess, err := sessions.Get(c.Response(), c.Request(), false)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}
	if sess != nil {
		if err := sess.Destroy(c.Response(), c.Request()); err != nil {
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	return c.NoContent(http.StatusNoContent)
}

// GetUsers GET /users
func GetUsers(c echo.Context) error {
	users, err := model.GetUsers()
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	res := make([]*UserForResponse, len(users))
	for i, user := range users {
		res[i] = formatUser(user)
	}
	return c.JSON(http.StatusOK, res)
}

// GetMe GET /users/me
func GetMe(c echo.Context) error {
	me := getRequestUser(c)
	return c.JSON(http.StatusOK, formatUser(me))
}

// GetUserByID GET /users/:userID
func GetUserByID(c echo.Context) error {
	userID := getRequestParamAsUUID(c, paramUserID)

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

	tagList, err := model.GetUserTagsByUserID(userID)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	userDetail, err := formatUserDetail(user, tagList)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, userDetail)
}

// GetUserIcon GET /users/:userID/icon
func GetUserIcon(c echo.Context) error {
	userID := getRequestParamAsUUID(c, paramUserID)

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

	if _, ok := c.QueryParams()["thumb"]; ok {
		return c.Redirect(http.StatusFound, "/api/1.0/files/"+user.Icon+"/thumbnail")
	}

	return c.Redirect(http.StatusFound, "/api/1.0/files/"+user.Icon)
}

// GetMyIcon GET /users/me/icon
func GetMyIcon(c echo.Context) error {
	user := getRequestUser(c)
	if _, ok := c.QueryParams()["thumb"]; ok {
		return c.Redirect(http.StatusFound, "/api/1.0/files/"+user.Icon+"/thumbnail")
	}
	return c.Redirect(http.StatusFound, "/api/1.0/files/"+user.Icon)
}

// PutMyIcon PUT /users/me/icon
func PutMyIcon(c echo.Context) error {
	userID := getRequestUserID(c)

	// file確認
	uploadedFile, err := c.FormFile("file")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	iconID, err := processMultipartFormIconUpload(c, uploadedFile)
	if err != nil {
		return err
	}

	// アイコン変更
	if err := model.ChangeUserIcon(userID, iconID); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	go event.Emit(event.UserIconUpdated, &event.UserEvent{ID: userID.String()})
	return c.NoContent(http.StatusOK)
}

// PatchMe PATCH /users/me
func PatchMe(c echo.Context) error {
	userID := getRequestUserID(c)

	req := struct {
		DisplayName string `json:"displayName" validate:"max=32"`
		TwitterID   string `json:"twitterId"   validate:"twitterid"`
	}{}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	if len(req.DisplayName) > 0 {
		if err := model.ChangeUserDisplayName(userID, req.DisplayName); err != nil {
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	if len(req.TwitterID) > 0 {
		if err := model.ChangeUserTwitterID(userID, req.TwitterID); err != nil {
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	go event.Emit(event.UserUpdated, &event.UserEvent{ID: userID.String()})
	return c.NoContent(http.StatusNoContent)
}

// PutPassword PUT /users/me/password
func PutPassword(c echo.Context) error {
	user := getRequestUser(c)

	req := struct {
		Old string `json:"password"    validate:"password"`
		New string `json:"newPassword" validate:"password"`
	}{}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	if err := model.AuthenticateUser(user, req.Old); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "password is wrong")
	}

	if err := model.ChangeUserPassword(user.GetUID(), req.New); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}

// PostUsers POST /users
func PostUsers(c echo.Context) error {
	req := struct {
		Name     string `json:"name"     validate:"name"`
		Password string `json:"password" validate:"password"`
		Email    string `json:"email"    validate:"email"`
	}{}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	if _, err := model.GetUserByName(req.Name); err != model.ErrNotFound {
		if err != nil {
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
		return echo.NewHTTPError(http.StatusBadRequest, "the name's user has already existed")
	}

	u, err := model.CreateUser(req.Name, req.Email, req.Password, role.User)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	go event.Emit(event.UserJoined, &event.UserEvent{ID: u.ID})
	return c.NoContent(http.StatusCreated)
}

func formatUser(user *model.User) *UserForResponse {
	res := &UserForResponse{
		UserID:      user.ID,
		Name:        user.Name,
		DisplayName: user.DisplayName,
		IconID:      user.Icon,
		Bot:         user.Bot,
		TwitterID:   user.TwitterID,
		IsOnline:    user.IsOnline(),
	}
	if t := user.GetLastOnline(); !t.IsZero() {
		res.LastOnline = &t
	}
	if len(res.DisplayName) == 0 {
		res.DisplayName = res.Name
	}
	return res
}

func formatUserDetail(user *model.User, tagList []*model.UsersTag) (*UserDetailForResponse, error) {
	res := &UserDetailForResponse{
		UserID:      user.ID,
		Name:        user.Name,
		DisplayName: user.DisplayName,
		IconID:      user.Icon,
		Bot:         user.Bot,
		TwitterID:   user.TwitterID,
		IsOnline:    user.IsOnline(),
	}
	if t := user.GetLastOnline(); !t.IsZero() {
		res.LastOnline = &t
	}
	if len(res.DisplayName) == 0 {
		res.DisplayName = res.Name
	}

	res.TagList = make([]*TagForResponse, len(tagList))
	for i, tag := range tagList {
		res.TagList[i] = formatTag(tag)
	}
	return res, nil
}
