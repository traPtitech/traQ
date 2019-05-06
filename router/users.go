package router

import (
	"net/http"
	"strconv"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gofrs/uuid"
	"github.com/labstack/echo"
	"github.com/skip2/go-qrcode"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/rbac/role"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/sessions"
	"github.com/traPtitech/traQ/utils"
	"go.uber.org/zap"
	"gopkg.in/guregu/null.v3"
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
	Status      int        `json:"accountStatus"`
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
	Status      int               `json:"accountStatus"`
	TagList     []*TagForResponse `json:"tagList"`
}

// UserForJWTClaim QRコードで表示するJWTのClaimの形のユーザー構造体
type UserForJWTClaim struct {
	jwt.StandardClaims
	UserID      uuid.UUID `json:"userId"`
	Name        string    `json:"name"`
	DisplayName string    `json:"displayName"`
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
			h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
			return c.NoContent(http.StatusInternalServerError)
		}
	}
	if err := model.AuthenticateUser(user, req.Pass); err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err)
	}

	// ユーザーのアカウント状態の確認
	switch user.Status {
	case model.UserAccountStatusDeactivated, model.UserAccountStatusSuspended:
		return echo.NewHTTPError(http.StatusForbidden, "this account is currently suspended")
	case model.UserAccountStatusActive:
		break
	}

	sess, err := sessions.Get(c.Response(), c.Request(), true)
	if err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return c.NoContent(http.StatusInternalServerError)
	}

	if err := sess.SetUser(user.ID); err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return c.NoContent(http.StatusInternalServerError)
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
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return c.NoContent(http.StatusInternalServerError)
	}
	if sess != nil {
		if err := sess.Destroy(c.Response(), c.Request()); err != nil {
			h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
			return c.NoContent(http.StatusInternalServerError)
		}
	}

	if redirect := c.QueryParam("redirect"); len(redirect) > 0 {
		return c.Redirect(http.StatusFound, redirect)
	}
	return c.NoContent(http.StatusNoContent)
}

// GetUsers GET /users
func (h *Handlers) GetUsers(c echo.Context) error {
	users, err := h.Repo.GetUsers()
	if err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
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
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError)
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
		DisplayName null.String `json:"displayName" validate:"max=64"`
		TwitterID   null.String `json:"twitterId" validate:"twitterid"`
		Role        null.String `json:"role"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	if err := h.Repo.UpdateUser(userID, repository.UpdateUserArgs{DisplayName: req.DisplayName, TwitterID: req.TwitterID, Role: req.Role}); err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}

// PutUserStatus PUT /users/:userID/status
func (h *Handlers) PutUserStatus(c echo.Context) error {
	userID := getRequestParamAsUUID(c, paramUserID)

	var req struct {
		Status int `json:"status" validate:"min=0,max=2"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	if err := h.Repo.ChangeUserAccountStatus(userID, model.UserAccountStatus(req.Status)); err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}

// PutUserPassword PUT /users/:userID/password
func (h *Handlers) PutUserPassword(c echo.Context) error {
	userID := getRequestParamAsUUID(c, paramUserID)

	var req struct {
		NewPassword string `json:"newPassword" validate:"password"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	if err := h.Repo.ChangeUserPassword(userID, req.NewPassword); err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return c.NoContent(http.StatusInternalServerError)
	}

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

func (h *Handlers) getUserIcon(c echo.Context, user *model.User) error {
	// ファイルメタ取得
	meta, err := h.Repo.GetFileMeta(user.Icon)
	if err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	// ファイルオープン
	file, err := h.Repo.GetFS().OpenFileByKey(meta.GetKey())
	if err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return c.NoContent(http.StatusInternalServerError)
	}
	defer file.Close()

	c.Response().Header().Set(echo.HeaderContentType, meta.Mime)
	c.Response().Header().Set(headerETag, strconv.Quote(meta.Hash))
	http.ServeContent(c.Response(), c.Request(), meta.Name, meta.CreatedAt, file)
	return nil
}

// PutUserIcon PUT /users/:userID/icon
func (h *Handlers) PutUserIcon(c echo.Context) error {
	return h.putUserIcon(c, getRequestParamAsUUID(c, paramUserID))
}

// PutMyIcon PUT /users/me/icon
func (h *Handlers) PutMyIcon(c echo.Context) error {
	return h.putUserIcon(c, getRequestUserID(c))
}

func (h *Handlers) putUserIcon(c echo.Context, userID uuid.UUID) error {
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
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}

// PatchMe PATCH /users/me
func (h *Handlers) PatchMe(c echo.Context) error {
	userID := getRequestUserID(c)

	var req struct {
		DisplayName null.String `json:"displayName" validate:"max=32"`
		TwitterID   null.String `json:"twitterId"   validate:"twitterid"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	if err := h.Repo.UpdateUser(userID, repository.UpdateUserArgs{DisplayName: req.DisplayName, TwitterID: req.TwitterID}); err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}

// PutPassword PUT /users/me/password
func (h *Handlers) PutPassword(c echo.Context) error {
	user := getRequestUser(c)

	req := struct {
		Old string `json:"password"    validate:"required"`
		New string `json:"newPassword" validate:"password"`
	}{}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	if err := model.AuthenticateUser(user, req.Old); err != nil {
		return c.NoContent(http.StatusUnauthorized)
	}

	if err := h.Repo.ChangeUserPassword(user.ID, req.New); err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}

// GetMyQRCode GET /users/me/qr-code
func (h *Handlers) GetMyQRCode(c echo.Context) error {
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
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return c.NoContent(http.StatusInternalServerError)
	}

	png, err := qrcode.Encode(token, qrcode.Low, 512)
	if err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.Blob(http.StatusOK, "image/png", png)
}

// PostUsers POST /users
func (h *Handlers) PostUsers(c echo.Context) error {
	req := struct {
		Name     string `json:"name"     validate:"name"`
		Password string `json:"password" validate:"password"`
	}{}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	if _, err := h.Repo.GetUserByName(req.Name); err != repository.ErrNotFound {
		if err != nil {
			h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
		return echo.NewHTTPError(http.StatusBadRequest, "the name's user has already existed")
	}

	user, err := h.Repo.CreateUser(req.Name, req.Password, role.User)
	if err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{"id": user.ID})
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
		Suspended:   user.Status != model.UserAccountStatusActive,
		Status:      int(user.Status),
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
		Suspended:   user.Status != model.UserAccountStatusActive,
		Status:      int(user.Status),
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
