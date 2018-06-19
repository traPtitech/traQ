package router

import (
	"github.com/gorilla/sessions"
	"github.com/labstack/echo"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/gommon/log"
	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/rbac/role"
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

// PostLogin Post /login のハンドラ
func PostLogin(c echo.Context) error {
	requestBody := &struct {
		Name string `json:"name" form:"name"`
		Pass string `json:"pass" form:"pass"`
	}{}

	if err := c.Bind(requestBody); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	user := &model.User{
		Name: requestBody.Name,
	}

	if err := user.Authorization(requestBody.Pass); err != nil {
		return echo.NewHTTPError(http.StatusForbidden, err)
	}

	sess, err := session.Get("sessions", c)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, "an error occurrerd while getting session")
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
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	res := make([]*UserForResponse, len(users))
	for i, user := range users {
		res[i] = formatUser(user)
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
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
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

// GetUserIcon GET /users/{userID}/icon のハンドラ
func GetUserIcon(c echo.Context) error {
	userID := c.Param("userID")

	user, err := model.GetUser(userID)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	if _, ok := c.QueryParams()["thumb"]; ok {
		return c.Redirect(http.StatusFound, "/api/1.0/files/"+user.Icon+"/thumbnail")
	}

	return c.Redirect(http.StatusFound, "/api/1.0/files/"+user.Icon)
}

// GetMyIcon GET /users/me/icon のハンドラ
func GetMyIcon(c echo.Context) error {
	user := c.Get("user").(*model.User)
	if _, ok := c.QueryParams()["thumb"]; ok {
		return c.Redirect(http.StatusFound, "/api/1.0/files/"+user.Icon+"/thumbnail")
	}
	return c.Redirect(http.StatusFound, "/api/1.0/files/"+user.Icon)
}

// PutMyIcon PUT /users/me/icon のハンドラ
func PutMyIcon(c echo.Context) error {
	user := c.Get("user").(*model.User)

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
	if err := user.UpdateIconID(iconID.String()); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	go event.Emit(event.UserIconUpdated, event.UserEvent{ID: user.ID})
	return c.NoContent(http.StatusOK)
}

// PatchMe PUT /users/me
func PatchMe(c echo.Context) error {
	user := c.Get("user").(*model.User)

	req := struct {
		ExPassword  string `json:"exPassword"`
		DisplayName string `json:"displayName"`
		Email       string `json:"email"`
		Password    string `json:"password"`
		TwitterID   string `json:"twitterId"`
	}{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request")
	}

	if req.Email == "" && req.Password == "" {
		user.DisplayName = req.DisplayName
		if req.TwitterID != "" {
			user.TwitterID = req.TwitterID
		}
		if err := user.Update(); err != nil {
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update user. Please check the format of displayName")
		}
		return c.NoContent(http.StatusNoContent)
	}

	if err := user.Authorization(req.ExPassword); err != nil {
		return c.JSON(http.StatusUnauthorized, "Password is wrong")
	}

	if req.DisplayName != "" {
		user.DisplayName = req.DisplayName
	}
	if req.TwitterID != "" {
		user.TwitterID = req.TwitterID
	}
	if req.Email != "" {
		user.Email = req.Email
	}
	if req.Password != "" {
		if err := user.SetPassword(req.Password); err != nil {
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update password")
		}
	}

	if err := user.Update(); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update user. Please check the format of email, password or displayName")
	}

	go event.Emit(event.UserUpdated, event.UserEvent{ID: user.ID})
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

	go event.Emit(event.UserJoined, event.UserEvent{ID: newUser.ID})
	return c.NoContent(http.StatusCreated)
}

func formatUser(user *model.User) *UserForResponse {
	res := &UserForResponse{
		UserID:      user.ID,
		Name:        user.Name,
		DisplayName: user.DisplayName,
		IconID:      user.Icon,
		Bot:         user.Bot,
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

	for _, tag := range tagList {
		formattedTag, err := formatTag(tag)
		if err != nil {
			return nil, err
		}
		res.TagList = append(res.TagList, formattedTag)
	}
	return res, nil
}

func validateUserID(userID string) (*model.User, error) {
	u, err := model.GetUser(userID)
	if err != nil {
		if err != model.ErrNotFound {
			log.Errorf("failed to get user: %v", err)
		}
		return nil, err
	}
	return u, nil
}
