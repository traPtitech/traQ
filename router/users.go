package router

import (
	"fmt"
	"github.com/labstack/echo"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/rbac/role"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/sessions"
	"net/http"
	"time"
)

// UserForResponse クライアントに返す形のユーザー構造体
type UserForResponse struct {
	UserID      uuid.UUID  `json:"userId"`
	Name        string     `json:"name"`
	DisplayName string     `json:"displayName"`
	IconID      uuid.UUID  `json:"iconFileId"`
	Bot         bool       `json:"bot"`
	TwitterID   string     `json:"twitterId"`
	LastOnline  *time.Time `json:"lastOnline"`
	IsOnline    bool       `json:"isOnline"`
	Suspended   bool       `json:"suspended"`
}

// UserDetailForResponse クライアントに返す形の詳細ユーザー構造体
type UserDetailForResponse struct {
	UserID      uuid.UUID         `json:"userId"`
	Name        string            `json:"name"`
	DisplayName string            `json:"displayName"`
	IconID      uuid.UUID         `json:"iconFileId"`
	Bot         bool              `json:"bot"`
	TwitterID   string            `json:"twitterId"`
	LastOnline  *time.Time        `json:"lastOnline"`
	IsOnline    bool              `json:"isOnline"`
	Suspended   bool              `json:"suspended"`
	TagList     []*TagForResponse `json:"tagList"`
}

// PostLogin POST /login
func (h *Handlers) PostLogin(c echo.Context) error {
	req := struct {
		Name string `json:"name" form:"name" validate:"required"`
		Pass string `json:"pass" form:"pass" validate:"required"`
	}{}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	user, err := h.Repo.GetUserByName(req.Name)
	if err != nil {
		switch err {
		case repository.ErrNotFound:
			return c.NoContent(http.StatusUnauthorized)
		default:
			c.Logger().Error(err)
			return c.NoContent(http.StatusInternalServerError)
		}
	}
	if err := model.AuthenticateUser(user, req.Pass); err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err)
	}

	// ユーザーのアカウント状態の確認
	switch user.Status {
	case model.UserAccountStatusSuspended:
		return echo.NewHTTPError(http.StatusForbidden, "this account is currently suspended")
	case model.UserAccountStatusValid:
		break
	}

	sess, err := sessions.Get(c.Response(), c.Request(), true)
	if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	if err := sess.SetUser(user.ID); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}

// PostLogout POST /logout
func (h *Handlers) PostLogout(c echo.Context) error {
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
func (h *Handlers) GetUsers(c echo.Context) error {
	users, err := h.Repo.GetUsers()
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	res := make([]*UserForResponse, len(users))
	for i, user := range users {
		res[i] = h.formatUser(user)
	}
	return c.JSON(http.StatusOK, res)
}

// GetMe GET /users/me
func (h *Handlers) GetMe(c echo.Context) error {
	me := getRequestUser(c)
	return c.JSON(http.StatusOK, h.formatUser(me))
}

// GetUserByID GET /users/:userID
func (h *Handlers) GetUserByID(c echo.Context) error {
	userID := getRequestParamAsUUID(c, paramUserID)
	user := getUserFromContext(c)

	tagList, err := h.Repo.GetUserTagsByUserID(userID)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	userDetail, err := h.formatUserDetail(user, tagList)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, userDetail)
}

// GetUserIcon GET /users/:userID/icon
func (h *Handlers) GetUserIcon(c echo.Context) error {
	user := getUserFromContext(c)

	if _, ok := c.QueryParams()["thumb"]; ok {
		return c.Redirect(http.StatusFound, fmt.Sprintf("/api/1.0/files/%s/thumbnail", user.Icon))
	}

	return c.Redirect(http.StatusFound, fmt.Sprintf("/api/1.0/files/%s", user.Icon))
}

// GetMyIcon GET /users/me/icon
func (h *Handlers) GetMyIcon(c echo.Context) error {
	user := getRequestUser(c)
	if _, ok := c.QueryParams()["thumb"]; ok {
		return c.Redirect(http.StatusFound, fmt.Sprintf("/api/1.0/files/%s/thumbnail", user.Icon))
	}
	return c.Redirect(http.StatusFound, fmt.Sprintf("/api/1.0/files/%s", user.Icon))
}

// PutMyIcon PUT /users/me/icon
func (h *Handlers) PutMyIcon(c echo.Context) error {
	userID := getRequestUserID(c)

	// file確認
	uploadedFile, err := c.FormFile("file")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	iconID, err := h.processMultipartFormIconUpload(c, uploadedFile)
	if err != nil {
		return err
	}

	// アイコン変更
	if err := h.Repo.ChangeUserIcon(userID, iconID); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusOK)
}

// PatchMe PATCH /users/me
func (h *Handlers) PatchMe(c echo.Context) error {
	userID := getRequestUserID(c)

	req := struct {
		DisplayName string `json:"displayName" validate:"max=32"`
		TwitterID   string `json:"twitterId"   validate:"twitterid"`
	}{}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	if len(req.DisplayName) > 0 {
		if err := h.Repo.ChangeUserDisplayName(userID, req.DisplayName); err != nil {
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	if len(req.TwitterID) > 0 {
		if err := h.Repo.ChangeUserTwitterID(userID, req.TwitterID); err != nil {
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	return c.NoContent(http.StatusNoContent)
}

// PutPassword PUT /users/me/password
func (h *Handlers) PutPassword(c echo.Context) error {
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

	if err := h.Repo.ChangeUserPassword(user.ID, req.New); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}

// PostUsers POST /users
func (h *Handlers) PostUsers(c echo.Context) error {
	req := struct {
		Name     string `json:"name"     validate:"name"`
		Password string `json:"password" validate:"password"`
		Email    string `json:"email"    validate:"email"`
	}{}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	if _, err := h.Repo.GetUserByName(req.Name); err != repository.ErrNotFound {
		if err != nil {
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
		return echo.NewHTTPError(http.StatusBadRequest, "the name's user has already existed")
	}

	if _, err := h.Repo.CreateUser(req.Name, req.Email, req.Password, role.User); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusCreated)
}

func (h *Handlers) formatUser(user *model.User) *UserForResponse {
	res := &UserForResponse{
		UserID:      user.ID,
		Name:        user.Name,
		DisplayName: user.DisplayName,
		IconID:      user.Icon,
		Bot:         user.Bot,
		TwitterID:   user.TwitterID,
		IsOnline:    h.Repo.IsUserOnline(user.ID),
		Suspended:   user.Status != model.UserAccountStatusValid,
	}
	if t, err := h.Repo.GetUserLastOnline(user.ID); err == nil && !t.IsZero() {
		res.LastOnline = &t
	}
	if len(res.DisplayName) == 0 {
		res.DisplayName = res.Name
	}
	return res
}

func (h *Handlers) formatUserDetail(user *model.User, tagList []*model.UsersTag) (*UserDetailForResponse, error) {
	res := &UserDetailForResponse{
		UserID:      user.ID,
		Name:        user.Name,
		DisplayName: user.DisplayName,
		IconID:      user.Icon,
		Bot:         user.Bot,
		TwitterID:   user.TwitterID,
		IsOnline:    h.Repo.IsUserOnline(user.ID),
		Suspended:   user.Status != model.UserAccountStatusValid,
	}
	if t, err := h.Repo.GetUserLastOnline(user.ID); err == nil && !t.IsZero() {
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
