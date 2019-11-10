package router

import (
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/skip2/go-qrcode"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/rbac/role"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/sessions"
	"github.com/traPtitech/traQ/utils"
	"gopkg.in/guregu/null.v3"
)

// PostLogin POST /login
func (h *Handlers) PostLogin(c echo.Context) error {
	var req struct {
		Name string `json:"name" form:"name"`
		Pass string `json:"pass" form:"pass"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return badRequest(err)
	}

	user, err := h.Repo.GetUserByName(req.Name)
	if err != nil {
		switch err {
		case repository.ErrNotFound:
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid name")
		default:
			return internalServerError(err, h.requestContextLogger(c))
		}
	}

	// ユーザーのアカウント状態の確認
	switch user.Status {
	case model.UserAccountStatusDeactivated, model.UserAccountStatusSuspended:
		return forbidden("this account is currently suspended")
	case model.UserAccountStatusActive:
		break
	}

	// パスワード検証
	if err := model.AuthenticateUser(user, req.Pass); err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err)
	}

	sess, err := sessions.Get(c.Response(), c.Request(), true)
	if err != nil {
		return internalServerError(err, h.requestContextLogger(c))
	}

	if err := sess.SetUser(user.ID); err != nil {
		return internalServerError(err, h.requestContextLogger(c))
	}

	if redirect := c.QueryParam("redirect"); len(redirect) > 0 {
		return c.Redirect(http.StatusFound, redirect)
	}
	return c.NoContent(http.StatusNoContent)
}

// PostLogout POST /logout
func (h *Handlers) PostLogout(c echo.Context) error {
	sess, err := sessions.Get(c.Response(), c.Request(), false)
	if err != nil {
		return internalServerError(err, h.requestContextLogger(c))
	}
	if sess != nil {
		if err := sess.Destroy(c.Response(), c.Request()); err != nil {
			return internalServerError(err, h.requestContextLogger(c))
		}
	}

	if redirect := c.QueryParam("redirect"); len(redirect) > 0 {
		return c.Redirect(http.StatusFound, redirect)
	}
	return c.NoContent(http.StatusNoContent)
}

// GetUsers GET /users
func (h *Handlers) GetUsers(c echo.Context) error {
	res, err, _ := h.getUsersResponseCacheGroup.Do("", func() (interface{}, error) {
		users, err := h.Repo.GetUsers()
		if err != nil {
			return nil, err
		}
		return json.Marshal(h.formatUsers(users))
	})
	if err != nil {
		return internalServerError(err, h.requestContextLogger(c))
	}
	return c.JSONBlob(http.StatusOK, res.([]byte))
}

// GetMe GET /users/me
func (h *Handlers) GetMe(c echo.Context) error {
	me := getRequestUser(c)
	return c.JSON(http.StatusOK, h.formatMe(me))
}

// GetUserByID GET /users/:userID
func (h *Handlers) GetUserByID(c echo.Context) error {
	userID := getRequestParamAsUUID(c, paramUserID)
	user := getUserFromContext(c)

	tagList, err := h.Repo.GetUserTagsByUserID(userID)
	if err != nil {
		return internalServerError(err, h.requestContextLogger(c))
	}

	userDetail, err := h.formatUserDetail(user, tagList)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, userDetail)
}

// PatchUserByID PATCH /users/:userID
func (h *Handlers) PatchUserByID(c echo.Context) error {
	userID := getRequestParamAsUUID(c, paramUserID)

	var req struct {
		DisplayName null.String `json:"displayName"`
		TwitterID   null.String `json:"twitterId"`
		Role        null.String `json:"role"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return badRequest(err)
	}

	if err := h.Repo.UpdateUser(userID, repository.UpdateUserArgs{DisplayName: req.DisplayName, TwitterID: req.TwitterID, Role: req.Role}); err != nil {
		switch {
		case repository.IsArgError(err):
			return badRequest(err)
		default:
			return internalServerError(err, h.requestContextLogger(c))
		}
	}

	return c.NoContent(http.StatusNoContent)
}

// PutUserStatus PUT /users/:userID/status
func (h *Handlers) PutUserStatus(c echo.Context) error {
	userID := getRequestParamAsUUID(c, paramUserID)

	var req struct {
		Status int `json:"status"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return badRequest(err)
	}

	if err := h.Repo.ChangeUserAccountStatus(userID, model.UserAccountStatus(req.Status)); err != nil {
		switch {
		case repository.IsArgError(err):
			return badRequest(err)
		default:
			return internalServerError(err, h.requestContextLogger(c))
		}
	}

	return c.NoContent(http.StatusNoContent)
}

// PutUserPassword PUT /users/:userID/password
func (h *Handlers) PutUserPassword(c echo.Context) error {
	userID := getRequestParamAsUUID(c, paramUserID)

	var req struct {
		NewPassword string `json:"newPassword"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return badRequest(err)
	}

	if err := h.Repo.ChangeUserPassword(userID, req.NewPassword); err != nil {
		switch {
		case repository.IsArgError(err):
			return badRequest(err)
		default:
			return internalServerError(err, h.requestContextLogger(c))
		}
	}
	_ = sessions.DestroyByUserID(userID)

	return c.NoContent(http.StatusNoContent)
}

// GetUserIcon GET /users/:userID/icon
func (h *Handlers) GetUserIcon(c echo.Context) error {
	return h.getUserIcon(c, getUserFromContext(c))
}

// GetMyIcon GET /users/me/icon
func (h *Handlers) GetMyIcon(c echo.Context) error {
	return h.getUserIcon(c, getRequestUser(c))
}

// PutUserIcon PUT /users/:userID/icon
func (h *Handlers) PutUserIcon(c echo.Context) error {
	return h.putUserIcon(c, getRequestParamAsUUID(c, paramUserID))
}

// PutMyIcon PUT /users/me/icon
func (h *Handlers) PutMyIcon(c echo.Context) error {
	return h.putUserIcon(c, getRequestUserID(c))
}

// PatchMe PATCH /users/me
func (h *Handlers) PatchMe(c echo.Context) error {
	userID := getRequestUserID(c)

	var req struct {
		DisplayName null.String `json:"displayName"`
		TwitterID   null.String `json:"twitterId"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return badRequest(err)
	}

	if err := h.Repo.UpdateUser(userID, repository.UpdateUserArgs{DisplayName: req.DisplayName, TwitterID: req.TwitterID}); err != nil {
		switch {
		case repository.IsArgError(err):
			return badRequest(err)
		default:
			return internalServerError(err, h.requestContextLogger(c))
		}
	}

	return c.NoContent(http.StatusNoContent)
}

// PutPassword PUT /users/me/password
func (h *Handlers) PutPassword(c echo.Context) error {
	user := getRequestUser(c)

	var req struct {
		Old string `json:"password"`
		New string `json:"newPassword"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return badRequest(err)
	}

	if err := model.AuthenticateUser(user, req.Old); err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "current password is wrong")
	}

	if err := h.Repo.ChangeUserPassword(user.ID, req.New); err != nil {
		switch {
		case repository.IsArgError(err):
			return badRequest(err)
		default:
			return internalServerError(err, h.requestContextLogger(c))
		}
	}
	_ = sessions.DestroyByUserID(user.ID)

	return c.NoContent(http.StatusNoContent)
}

// GetMyQRCode GET /users/me/qr-code
func (h *Handlers) GetMyQRCode(c echo.Context) error {
	// UserForJWTClaim QRコードで表示するJWTのClaimの形のユーザー構造体
	type UserForJWTClaim struct {
		jwt.StandardClaims
		UserID      uuid.UUID `json:"userId"`
		Name        string    `json:"name"`
		DisplayName string    `json:"displayName"`
	}

	user := getRequestUser(c)

	now := time.Now()
	deadline := now.Add(10 * time.Minute)

	token, err := utils.Signer.Sign(&UserForJWTClaim{
		StandardClaims: jwt.StandardClaims{
			IssuedAt:  now.Unix(),
			ExpiresAt: deadline.Unix(),
		},
		UserID:      user.ID,
		Name:        user.Name,
		DisplayName: user.DisplayName,
	})
	if err != nil {
		return internalServerError(err, h.requestContextLogger(c))
	}

	png, err := qrcode.Encode(token, qrcode.Low, 512)
	if err != nil {
		return internalServerError(err, h.requestContextLogger(c))
	}

	return c.Blob(http.StatusOK, "image/png", png)
}

// PostUsers POST /users
func (h *Handlers) PostUsers(c echo.Context) error {
	var req struct {
		Name     string `json:"name"     validate:"name"`
		Password string `json:"password" validate:"password"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return badRequest(err)
	}

	if _, err := h.Repo.GetUserByName(req.Name); err != repository.ErrNotFound {
		if err != nil {
			return internalServerError(err, h.requestContextLogger(c))
		}
		return conflict("the name's user has already existed")
	}

	user, err := h.Repo.CreateUser(req.Name, req.Password, role.User)
	if err != nil {
		return internalServerError(err, h.requestContextLogger(c))
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{"id": user.ID})
}
