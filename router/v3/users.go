package v3

import (
	"github.com/dgrijalva/jwt-go"
	vd "github.com/go-ozzo/ozzo-validation"
	"github.com/labstack/echo/v4"
	"github.com/skip2/go-qrcode"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/router/sessions"
	"github.com/traPtitech/traQ/utils"
	"github.com/traPtitech/traQ/utils/validator"
	"net/http"
	"time"
)

// PutMyPasswordRequest PUT /users/me/password リクエストボディ
type PutMyPasswordRequest struct {
	Password    string `json:"password"`
	NewPassword string `json:"newPassword"`
}

func (r PutMyPasswordRequest) Validate() error {
	return vd.ValidateStruct(&r,
		vd.Field(&r.Password, vd.Required),
		vd.Field(&r.NewPassword, validator.PasswordRuleRequired...),
	)
}

// ChangeMyPassword PUT /users/me/password
func (h *Handlers) PutMyPassword(c echo.Context) error {
	var req PutMyPasswordRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	user := getRequestUser(c)

	// パスワード認証
	if err := model.AuthenticateUser(user, req.Password); err != nil {
		return herror.Unauthorized("password is wrong")
	}

	// パスワード変更
	if err := h.Repo.ChangeUserPassword(user.ID, req.NewPassword); err != nil {
		return herror.InternalServerError(err)
	}
	_ = sessions.DestroyByUserID(user.ID) // 全セッションを破棄(強制ログアウト)
	return c.NoContent(http.StatusNoContent)
}

// GetMyQRCode GET /users/me/qr-code
func (h *Handlers) GetMyQRCode(c echo.Context) error {
	user := getRequestUser(c)

	// トークン生成
	now := time.Now()
	deadline := now.Add(5 * time.Minute)
	token, err := utils.Signer.Sign(jwt.MapClaims{
		"iat":         now.Unix(),
		"exp":         deadline.Unix(),
		"userId":      user.ID,
		"name":        user.Name,
		"displayName": user.DisplayName,
	})
	if err != nil {
		return herror.InternalServerError(err)
	}

	if isTrue(c.QueryParam("token")) {
		// 画像じゃなくて生のトークンを返す
		return c.String(http.StatusOK, token)
	}

	// QRコード画像生成
	png, err := qrcode.Encode(token, qrcode.Low, 512)
	if err != nil {
		return herror.InternalServerError(err)
	}
	return c.Blob(http.StatusOK, consts.MimeImagePNG, png)
}

// GetUserIcon GET /users/:userID/icon
func (h *Handlers) GetUserIcon(c echo.Context) error {
	return serveUserIcon(c, h.Repo, getParamUser(c))
}

// GetMyIcon GET /users/me/icon
func (h *Handlers) GetMyIcon(c echo.Context) error {
	return serveUserIcon(c, h.Repo, getRequestUser(c))
}

// GetMyStampHistory GET /users/me/stamp-history リクエストクエリ
type GetMyStampHistoryRequest struct {
	Limit int `query:"limit"`
}

func (r *GetMyStampHistoryRequest) Validate() error {
	if r.Limit == 0 {
		r.Limit = 100
	}
	return vd.ValidateStruct(r,
		vd.Field(&r.Limit, vd.Min(1), vd.Max(100)),
	)
}

// GetMyStampHistory GET /users/me/stamp-history
func (h *Handlers) GetMyStampHistory(c echo.Context) error {
	var req GetMyStampHistoryRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	userID := getRequestUserID(c)
	history, err := h.Repo.GetUserStampHistory(userID, req.Limit)
	if err != nil {
		return herror.InternalServerError(err)
	}

	return c.JSON(http.StatusOK, history)
}

// PostMyFCMDeviceRequest POST /users/me/fcm-device リクエストボディ
type PostMyFCMDeviceRequest struct {
	Token string `json:"token"`
}

func (r PostMyFCMDeviceRequest) Validate() error {
	return vd.ValidateStruct(&r,
		vd.Field(&r.Token, vd.Required, vd.RuneLength(1, 190)),
	)
}

// PostMyFCMDevice POST /users/me/fcm-device
func (h *Handlers) PostMyFCMDevice(c echo.Context) error {
	var req PostMyFCMDeviceRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	userID := getRequestUserID(c)
	if _, err := h.Repo.RegisterDevice(userID, req.Token); err != nil {
		switch {
		case repository.IsArgError(err):
			return herror.BadRequest(err)
		default:
			return herror.InternalServerError(err)
		}
	}

	return c.NoContent(http.StatusNoContent)
}
