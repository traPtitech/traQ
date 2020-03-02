package v3

import (
	vd "github.com/go-ozzo/ozzo-validation"
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/router/sessions"
	"github.com/traPtitech/traQ/utils/validator"
	"net/http"
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
