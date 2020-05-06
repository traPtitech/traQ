package v1

import (
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/router/utils"
	jwt2 "github.com/traPtitech/traQ/utils/jwt"
	"github.com/traPtitech/traQ/utils/validator"
	"go.uber.org/zap"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	vd "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/skip2/go-qrcode"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/rbac/role"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/sessions"
	"gopkg.in/guregu/null.v3"
)

// PostLogin POST /login
func (h *Handlers) PostLogin(c echo.Context) error {
	var req struct {
		Name string `json:"name" form:"name"`
		Pass string `json:"pass" form:"pass"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	user, err := h.Repo.GetUserByName(req.Name, false)
	if err != nil {
		switch err {
		case repository.ErrNotFound:
			h.L(c).Info("an api login attempt failed: unknown user", zap.String("username", req.Name))
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid name")
		default:
			return herror.InternalServerError(err)
		}
	}

	// ユーザーのアカウント状態の確認
	if !user.IsActive() {
		h.L(c).Info("an api login attempt failed: suspended user", zap.String("username", req.Name))
		return herror.Forbidden("this account is currently suspended")
	}

	// パスワード検証
	if err := user.Authenticate(req.Pass); err != nil {
		h.L(c).Info("an api login attempt failed: wrong password", zap.String("username", req.Name))
		return echo.NewHTTPError(http.StatusUnauthorized, err)
	}
	h.L(c).Info("an api login attempt succeeded", zap.String("username", req.Name))

	sess, err := sessions.Get(c.Response(), c.Request(), true)
	if err != nil {
		return herror.InternalServerError(err)
	}

	if err := sess.SetUser(user.GetID()); err != nil {
		return herror.InternalServerError(err)
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
		return herror.InternalServerError(err)
	}
	if sess != nil {
		if err := sess.Destroy(c.Response(), c.Request()); err != nil {
			return herror.InternalServerError(err)
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
		users, err := h.Repo.GetUsers(repository.UsersQuery{}.LoadProfile())
		if err != nil {
			return nil, err
		}
		return json.Marshal(h.formatUsers(users))
	})
	if err != nil {
		return herror.InternalServerError(err)
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
	userID := getRequestParamAsUUID(c, consts.ParamUserID)
	user := getUserFromContext(c)

	tagList, err := h.Repo.GetUserTagsByUserID(userID)
	if err != nil {
		return herror.InternalServerError(err)
	}

	userDetail, err := h.formatUserDetail(user, tagList)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, userDetail)
}

// PatchUserByIDRequest PATCH /users/:userID リクエストボディ
type PatchUserByIDRequest struct {
	DisplayName null.String `json:"displayName"`
	TwitterID   null.String `json:"twitterId"`
	Role        null.String `json:"role"`
}

func (r PatchUserByIDRequest) Validate() error {
	return vd.ValidateStruct(&r,
		vd.Field(&r.DisplayName, vd.RuneLength(0, 64)),
		vd.Field(&r.TwitterID, validator.TwitterIDRule...),
		vd.Field(&r.Role, vd.RuneLength(0, 30)),
	)
}

// PatchUserByID PATCH /users/:userID
func (h *Handlers) PatchUserByID(c echo.Context) error {
	userID := getRequestParamAsUUID(c, consts.ParamUserID)

	var req PatchUserByIDRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	if err := h.Repo.UpdateUser(userID, repository.UpdateUserArgs{DisplayName: req.DisplayName, TwitterID: req.TwitterID, Role: req.Role}); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}

// PutUserStatus PUT /users/:userID/status
func (h *Handlers) PutUserStatus(c echo.Context) error {
	userID := getRequestParamAsUUID(c, consts.ParamUserID)

	var req struct {
		Status int `json:"status"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	var args repository.UpdateUserArgs
	args.UserState.Valid = true
	args.UserState.State = model.UserAccountStatus(req.Status)

	if !args.UserState.State.Valid() {
		return herror.BadRequest("invalid status")
	}

	if err := h.Repo.UpdateUser(userID, args); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}

// PutUserPasswordRequest PUT /users/:userID/password リクエストボディ
type PutUserPasswordRequest struct {
	NewPassword string `json:"newPassword"`
}

func (r PutUserPasswordRequest) Validate() error {
	return vd.ValidateStruct(&r,
		vd.Field(&r.NewPassword, validator.PasswordRuleRequired...),
	)
}

// PutUserPassword PUT /users/:userID/password
func (h *Handlers) PutUserPassword(c echo.Context) error {
	var req PutUserPasswordRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	return utils.ChangeUserPassword(c, h.Repo, getRequestParamAsUUID(c, consts.ParamUserID), req.NewPassword)
}

// GetUserIcon GET /users/:userID/icon
func (h *Handlers) GetUserIcon(c echo.Context) error {
	return utils.ServeUserIcon(c, h.Repo, getUserFromContext(c))
}

// GetMyIcon GET /users/me/icon
func (h *Handlers) GetMyIcon(c echo.Context) error {
	return utils.ServeUserIcon(c, h.Repo, getRequestUser(c))
}

// PutUserIcon PUT /users/:userID/icon
func (h *Handlers) PutUserIcon(c echo.Context) error {
	return utils.ChangeUserIcon(h.Imaging, c, h.Repo, getRequestParamAsUUID(c, consts.ParamUserID))
}

// PutMyIcon PUT /users/me/icon
func (h *Handlers) PutMyIcon(c echo.Context) error {
	return utils.ChangeUserIcon(h.Imaging, c, h.Repo, getRequestUserID(c))
}

// PatchMeRequest PATCH /users/me リクエストボディ
type PatchMeRequest struct {
	DisplayName null.String `json:"displayName"`
	TwitterID   null.String `json:"twitterId"`
}

func (r PatchMeRequest) Validate() error {
	return vd.ValidateStruct(&r,
		vd.Field(&r.DisplayName, vd.RuneLength(0, 64)),
		vd.Field(&r.TwitterID, validator.TwitterIDRule...),
	)
}

// PatchMe PATCH /users/me
func (h *Handlers) PatchMe(c echo.Context) error {
	userID := getRequestUserID(c)

	var req PatchMeRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	if err := h.Repo.UpdateUser(userID, repository.UpdateUserArgs{DisplayName: req.DisplayName, TwitterID: req.TwitterID}); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}

// PutPasswordRequest PUT /users/me/password リクエストボディ
type PutPasswordRequest struct {
	Password    string `json:"password"`
	NewPassword string `json:"newPassword"`
}

func (r PutPasswordRequest) Validate() error {
	return vd.ValidateStruct(&r,
		vd.Field(&r.Password, vd.Required),
		vd.Field(&r.NewPassword, validator.PasswordRuleRequired...),
	)
}

// PutPassword PUT /users/me/password
func (h *Handlers) PutPassword(c echo.Context) error {
	user := getRequestUser(c)

	var req PutPasswordRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	if err := user.Authenticate(req.Password); err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "current password is wrong")
	}

	return utils.ChangeUserPassword(c, h.Repo, user.GetID(), req.NewPassword)
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

	token, err := jwt2.Sign(&UserForJWTClaim{
		StandardClaims: jwt.StandardClaims{
			IssuedAt:  now.Unix(),
			ExpiresAt: deadline.Unix(),
		},
		UserID:      user.GetID(),
		Name:        user.GetName(),
		DisplayName: user.GetDisplayName(),
	})
	if err != nil {
		return herror.InternalServerError(err)
	}

	png, err := qrcode.Encode(token, qrcode.Low, 512)
	if err != nil {
		return herror.InternalServerError(err)
	}

	return c.Blob(http.StatusOK, "image/png", png)
}

// PostUserRequest POST /users リクエストボディ
type PostUserRequest struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

func (r PostUserRequest) Validate() error {
	return vd.ValidateStruct(&r,
		vd.Field(&r.Name, validator.UserNameRuleRequired...),
		vd.Field(&r.Password, validator.PasswordRuleRequired...),
	)
}

// PostUsers POST /users
func (h *Handlers) PostUsers(c echo.Context) error {
	var req PostUserRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	if _, err := h.Repo.GetUserByName(req.Name, false); err != repository.ErrNotFound {
		if err != nil {
			return herror.InternalServerError(err)
		}
		return herror.Conflict("the name's user has already existed")
	}

	user, err := h.Repo.CreateUser(repository.CreateUserArgs{Name: req.Name, Password: req.Password, Role: role.User})
	if err != nil {
		return herror.InternalServerError(err)
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{"id": user.GetID()})
}
