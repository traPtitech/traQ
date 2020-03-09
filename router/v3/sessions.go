package v3

import (
	vd "github.com/go-ozzo/ozzo-validation"
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/router/sessions"
	"github.com/traPtitech/traQ/utils/validator"
	"net/http"
	"time"
)

// PostLoginRequest POST /login リクエストボディ
type PostLoginRequest struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

func (r PostLoginRequest) Validate() error {
	return vd.ValidateStruct(&r,
		vd.Field(&r.Name, validator.UserNameRuleRequired...),
		vd.Field(&r.Password, validator.PasswordRuleRequired...),
	)
}

// Login POST /login
func (h *Handlers) Login(c echo.Context) error {
	var req PostLoginRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	user, err := h.Repo.GetUserByName(req.Name)
	if err != nil {
		switch err {
		case repository.ErrNotFound:
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid name")
		default:
			return herror.InternalServerError(err)
		}
	}

	// ユーザーのアカウント状態の確認
	switch user.Status {
	case model.UserAccountStatusDeactivated, model.UserAccountStatusSuspended:
		return herror.Forbidden("this account is currently suspended")
	case model.UserAccountStatusActive:
		break
	}

	// パスワード検証
	if err := model.AuthenticateUser(user, req.Password); err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err)
	}

	sess, err := sessions.Get(c.Response(), c.Request(), true)
	if err != nil {
		return herror.InternalServerError(err)
	}

	if err := sess.SetUser(user.ID); err != nil {
		return herror.InternalServerError(err)
	}

	if redirect := c.QueryParam("redirect"); len(redirect) > 0 {
		return c.Redirect(http.StatusFound, redirect)
	}
	return c.NoContent(http.StatusNoContent)
}

// Logout POST /logout
func (h *Handlers) Logout(c echo.Context) error {
	sess, err := sessions.Get(c.Response(), c.Request(), false)
	if err != nil {
		return herror.InternalServerError(err)
	}
	if sess != nil {
		if isTrue(c.QueryParam("all")) {
			uid := sess.GetUserID()
			if uid == uuid.Nil {
				if err := sess.Destroy(c.Response(), c.Request()); err != nil {
					return herror.InternalServerError(err)
				}
			} else {
				if err := sessions.DestroyByUserID(uid); err != nil {
					return herror.InternalServerError(err)
				}
			}
		} else {
			if err := sess.Destroy(c.Response(), c.Request()); err != nil {
				return herror.InternalServerError(err)
			}
		}
	}

	if redirect := c.QueryParam("redirect"); len(redirect) > 0 {
		return c.Redirect(http.StatusFound, redirect)
	}
	return c.NoContent(http.StatusNoContent)
}

// GetMySessions GET /users/me/sessions
func (h *Handlers) GetMySessions(c echo.Context) error {
	userID := getRequestUserID(c)

	ses, err := sessions.GetByUserID(userID)
	if err != nil {
		return herror.InternalServerError(err)
	}

	type response struct {
		ID         string    `json:"id"`
		IP         string    `json:"ip"`
		UA         string    `json:"ua"`
		LastAccess time.Time `json:"lastAccess"`
		IssuedAt   time.Time `json:"issuedAt"`
	}

	res := make([]response, len(ses))
	for k, v := range ses {
		referenceID, created, lastAccess, lastIP, lastUserAgent := v.GetSessionInfo()
		res[k] = response{
			ID:         referenceID.String(),
			IP:         lastIP,
			UA:         lastUserAgent,
			LastAccess: lastAccess,
			IssuedAt:   created,
		}
	}

	return c.JSON(http.StatusOK, res)
}

// RevokeMySession DELETE /users/me/sessions/:referenceID
func (h *Handlers) RevokeMySession(c echo.Context) error {
	userID := getRequestUserID(c)
	referenceID := getParamAsUUID(c, consts.ParamReferenceID)

	err := sessions.DestroyByReferenceID(userID, referenceID)
	if err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}

// GetMyTokens GET /users/me/tokens
func (h *Handlers) GetMyTokens(c echo.Context) error {
	userID := getRequestUserID(c)

	ot, err := h.Repo.GetTokensByUser(userID)
	if err != nil {
		return herror.InternalServerError(err)
	}

	type response struct {
		ID       uuid.UUID          `json:"id"`
		ClientID string             `json:"clientId"`
		Scopes   model.AccessScopes `json:"scopes"`
		IssuedAt time.Time          `json:"issuedAt"`
	}

	res := make([]response, len(ot))
	for i, v := range ot {
		res[i] = response{
			ID:       v.ID,
			ClientID: v.ClientID,
			Scopes:   v.Scopes,
			IssuedAt: v.CreatedAt,
		}
	}

	return c.JSON(http.StatusOK, res)
}

// RevokeMyToken DELETE /users/me/tokens/:tokenID
func (h *Handlers) RevokeMyToken(c echo.Context) error {
	tokenID := getParamAsUUID(c, consts.ParamTokenID)
	userID := getRequestUserID(c)

	ot, err := h.Repo.GetTokenByID(tokenID)
	if err != nil {
		switch err {
		case repository.ErrNotFound:
			return herror.NotFound()
		default:
			return herror.InternalServerError(err)
		}
	}
	if ot.UserID != userID {
		return herror.NotFound()
	}

	if err := h.Repo.DeleteTokenByAccess(ot.AccessToken); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}
