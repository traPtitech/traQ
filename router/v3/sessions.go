package v3

import (
	vd "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/utils/validator"
	"go.uber.org/zap"
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
		// MEMO 旧パスワード者のためにバリデーションを消している
		// vd.Field(&r.Password, validator.PasswordRuleRequired...),
		vd.Field(&r.Password, vd.Required),
	)
}

// Login POST /login
func (h *Handlers) Login(c echo.Context) error {
	var req PostLoginRequest
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
	if err := user.Authenticate(req.Password); err != nil {
		h.L(c).Info("an api login attempt failed: wrong password", zap.String("username", req.Name))
		return echo.NewHTTPError(http.StatusUnauthorized, err)
	}
	h.L(c).Info("an api login attempt succeeded", zap.String("username", req.Name))

	if _, err := h.SessStore.RenewSession(c, user.GetID()); err != nil {
		return herror.InternalServerError(err)
	}

	if redirect := c.QueryParam("redirect"); len(redirect) > 0 {
		return c.Redirect(http.StatusFound, redirect)
	}
	return c.NoContent(http.StatusNoContent)
}

// Logout POST /logout
func (h *Handlers) Logout(c echo.Context) error {
	sess, err := h.SessStore.GetSession(c, false)
	if err != nil {
		return herror.InternalServerError(err)
	}
	if sess != nil && isTrue(c.QueryParam("all")) && sess.LoggedIn() {
		if err := h.SessStore.RevokeSessionsByUserID(sess.UserID()); err != nil {
			return herror.InternalServerError(err)
		}
	}

	if err := h.SessStore.RevokeSession(c); err != nil {
		return herror.InternalServerError(err)
	}

	if redirect := c.QueryParam("redirect"); len(redirect) > 0 {
		return c.Redirect(http.StatusFound, redirect)
	}
	return c.NoContent(http.StatusNoContent)
}

// GetMySessions GET /users/me/sessions
func (h *Handlers) GetMySessions(c echo.Context) error {
	userID := getRequestUserID(c)

	ses, err := h.SessStore.GetSessionsByUserID(userID)
	if err != nil {
		return herror.InternalServerError(err)
	}

	type response struct {
		ID       uuid.UUID `json:"id"`
		IssuedAt time.Time `json:"issuedAt"`
	}

	res := make([]response, len(ses))
	for k, v := range ses {
		res[k] = response{
			ID:       v.RefID(),
			IssuedAt: v.CreatedAt(),
		}
	}

	return c.JSON(http.StatusOK, res)
}

// RevokeMySession DELETE /users/me/sessions/:referenceID
func (h *Handlers) RevokeMySession(c echo.Context) error {
	referenceID := getParamAsUUID(c, consts.ParamReferenceID)

	if err := h.SessStore.RevokeSessionByRefID(referenceID); err != nil {
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

// GetMyExternalAccounts GET /users/me/ex-accounts
func (h *Handlers) GetMyExternalAccounts(c echo.Context) error {
	links, err := h.Repo.GetLinkedExternalUserAccounts(getRequestUserID(c))
	if err != nil {
		return herror.InternalServerError(err)
	}

	type response struct {
		ProviderName string    `json:"providerName"`
		ExternalName string    `json:"externalName"`
		LinkedAt     time.Time `json:"linkedAt"`
	}
	res := make([]response, len(links))
	for i, link := range links {
		res[i] = response{
			ProviderName: link.ProviderName,
			LinkedAt:     link.CreatedAt,
		}
		if exName, ok := link.Extra["externalName"]; ok {
			res[i].ExternalName = exName.(string)
		} else {
			res[i].ExternalName = link.ExternalID
		}
	}

	return c.JSON(http.StatusOK, res)
}

// PostLinkExternalAccount POST /users/me/ex-accounts/link リクエストボディ
type PostLinkExternalAccount struct {
	ProviderName string `json:"providerName"`
}

func (r PostLinkExternalAccount) Validate() error {
	return vd.ValidateStruct(&r,
		vd.Field(&r.ProviderName, vd.Required),
	)
}

// UnlinkExternalAccount POST /users/me/ex-accounts/link
func (h *Handlers) LinkExternalAccount(c echo.Context) error {
	var req PostLinkExternalAccount
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	if !h.EnabledExternalAccountProviders[req.ProviderName] {
		return herror.BadRequest("invalid provider name")
	}

	links, err := h.Repo.GetLinkedExternalUserAccounts(getRequestUserID(c))
	if err != nil {
		return herror.InternalServerError(err)
	}
	for _, link := range links {
		if link.ProviderName == req.ProviderName {
			return herror.BadRequest("already linked")
		}
	}

	return c.Redirect(http.StatusFound, "/api/auth/"+req.ProviderName+"?link=1")
}

// PostUnlinkExternalAccount POST /users/me/ex-accounts/unlink リクエストボディ
type PostUnlinkExternalAccount struct {
	ProviderName string `json:"providerName"`
}

func (r PostUnlinkExternalAccount) Validate() error {
	return vd.ValidateStruct(&r,
		vd.Field(&r.ProviderName, vd.Required),
	)
}

// UnlinkExternalAccount POST /users/me/ex-accounts/unlink
func (h *Handlers) UnlinkExternalAccount(c echo.Context) error {
	var req PostUnlinkExternalAccount
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	if err := h.Repo.UnlinkExternalUserAccount(getRequestUserID(c), req.ProviderName); err != nil {
		switch err {
		case repository.ErrNotFound:
			return herror.BadRequest("invalid provider name")
		default:
			return herror.InternalServerError(err)
		}
	}

	return c.NoContent(http.StatusNoContent)
}
