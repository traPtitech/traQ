package router

import (
	"encoding/base64"
	"github.com/labstack/echo"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/oauth2"
	"github.com/traPtitech/traQ/oauth2/scope"
	"net/http"
	"regexp"
	"time"
)

var uriRegex = regexp.MustCompile(`^([a-z0-9+.-]+):(?://(?:((?:[a-z0-9-._~!$&'()*+,;=:]|%[0-9A-F]{2})*)@)?((?:[a-z0-9-._~!$&'()*+,;=]|%[0-9A-F]{2})*)(?::(\d*))?(/(?:[a-z0-9-._~!$&'()*+,;=:@/]|%[0-9A-F]{2})*)?|(/?(?:[a-z0-9-._~!$&'()*+,;=:@]|%[0-9A-F]{2})+(?:[a-z0-9-._~!$&'()*+,;=:@/]|%[0-9A-F]{2})*)?)(?:\?((?:[a-z0-9-._~!$&'()*+,;=:/?@]|%[0-9A-F]{2})*))?(?:#((?:[a-z0-9-._~!$&'()*+,;=:/?@]|%[0-9A-F]{2})*))?$`)

// ClientInfo レスポンス用クライアント情報構造体
type ClientInfo struct {
	ClientID    string `json:"clientId"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CreatorID   string `json:"creatorId"`
}

// OwnedClientInfo レスポンス用クライアント情報構造体
type OwnedClientInfo struct {
	ClientID    string             `json:"clientId"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	CreatorID   string             `json:"creatorId"`
	Scopes      scope.AccessScopes `json:"scopes"`
	RedirectURI string             `json:"redirectUri"`
	Secret      string             `json:"secret"`
}

// AllowedClientInfo レスポンス用クライアント情報構造体
type AllowedClientInfo struct {
	TokenID     string             `json:"tokenId"`
	ClientID    string             `json:"clientId"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	CreatorID   string             `json:"creatorId"`
	Scopes      scope.AccessScopes `json:"scopes"`
	ApprovedAt  time.Time          `json:"approvedAt"`
}

// GetMyTokens GET /users/me/tokens
func (h *Handlers) GetMyTokens(c echo.Context) error {
	userID := getRequestUserID(c)

	ot, err := h.OAuth2.GetTokensByUser(userID)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	res := make([]*AllowedClientInfo, len(ot))
	for i, v := range ot {
		oc, err := h.OAuth2.GetClient(v.ClientID)
		if err != nil {
			switch err {
			case oauth2.ErrClientNotFound:
				continue
			default:
				c.Logger().Error(err)
				return echo.NewHTTPError(http.StatusInternalServerError)
			}
		}
		res[i] = &AllowedClientInfo{
			TokenID:     v.ID.String(),
			ClientID:    v.ClientID,
			Name:        oc.Name,
			Description: oc.Description,
			CreatorID:   oc.CreatorID.String(),
			Scopes:      v.Scopes,
			ApprovedAt:  v.CreatedAt,
		}
	}

	return c.JSON(http.StatusOK, res)
}

// DeleteMyToken DELETE /users/me/tokens/:tokenID
func (h *Handlers) DeleteMyToken(c echo.Context) error {
	tokenID := getRequestParamAsUUID(c, paramTokenID)
	userID := getRequestUserID(c)

	ot, err := h.OAuth2.GetTokenByID(tokenID)
	if err != nil {
		switch err {
		case oauth2.ErrTokenNotFound:
			return echo.NewHTTPError(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}
	if ot.UserID != userID {
		return echo.NewHTTPError(http.StatusNotFound)
	}

	if err := h.OAuth2.DeleteTokenByAccess(ot.AccessToken); err != nil {
		switch err {
		case oauth2.ErrTokenNotFound:
			return echo.NewHTTPError(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	return c.NoContent(http.StatusNoContent)
}

// GetClients GET /clients
func (h *Handlers) GetClients(c echo.Context) error {
	userID := getRequestUserID(c)

	oc, err := h.OAuth2.GetClientsByUser(userID)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	res := make([]*OwnedClientInfo, len(oc))
	for i, v := range oc {
		res[i] = &OwnedClientInfo{
			ClientID:    v.ID,
			Name:        v.Name,
			Description: v.Description,
			CreatorID:   v.CreatorID.String(),
			Scopes:      v.Scopes,
			RedirectURI: v.RedirectURI,
			Secret:      v.Secret,
		}
	}

	return c.JSON(http.StatusOK, res)
}

// PostClients POST /clients
func (h *Handlers) PostClients(c echo.Context) error {
	userID := getRequestUserID(c)

	req := struct {
		Name        string   `json:"name"        validate:"required,max=32"`
		Description string   `json:"description" validate:"required"`
		RedirectURI string   `json:"redirectUri" validate:"uri,required"`
		Scopes      []string `json:"scopes"      validate:"unique,dive,required"`
	}{}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	scopes := scope.AccessScopes{}
	for _, v := range req.Scopes {
		s := scope.AccessScope(v)
		if !scope.Valid(s) {
			return echo.NewHTTPError(http.StatusBadRequest)
		}
		scopes = append(scopes, s)
	}

	client := &oauth2.Client{
		ID:           uuid.NewV4().String(),
		Name:         req.Name,
		Description:  req.Description,
		Confidential: false,
		CreatorID:    userID,
		RedirectURI:  req.RedirectURI,
		Secret:       base64.RawURLEncoding.EncodeToString(uuid.NewV4().Bytes()),
		Scopes:       scopes,
	}
	if err := h.OAuth2.SaveClient(client); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, &OwnedClientInfo{
		ClientID:    client.ID,
		Name:        client.Name,
		Description: client.Description,
		CreatorID:   client.CreatorID.String(),
		Scopes:      client.Scopes,
		RedirectURI: client.RedirectURI,
		Secret:      client.Secret,
	})
}

// GetClient GET /clients/:clientID
func (h *Handlers) GetClient(c echo.Context) error {
	clientID := c.Param("clientID")

	oc, err := h.OAuth2.GetClient(clientID)
	if err != nil {
		switch err {
		case oauth2.ErrClientNotFound:
			return echo.NewHTTPError(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	return c.JSON(http.StatusOK, &ClientInfo{
		ClientID:    oc.ID,
		Name:        oc.Name,
		Description: oc.Description,
		CreatorID:   oc.CreatorID.String(),
	})
}

// PatchClient PATCH /clients/:clientID
func (h *Handlers) PatchClient(c echo.Context) error {
	clientID := c.Param("clientID")
	userID := getRequestUserID(c)

	oc, err := h.OAuth2.GetClient(clientID)
	if err != nil {
		switch err {
		case oauth2.ErrClientNotFound:
			return echo.NewHTTPError(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}
	if oc.CreatorID != userID {
		return echo.NewHTTPError(http.StatusForbidden)
	}

	req := struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		RedirectURI string `json:"redirectUri"`
	}{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	if len(req.Name) > 0 {
		if len(req.Name) > 32 {
			return echo.NewHTTPError(http.StatusBadRequest)
		}
		oc.Name = req.Name
	}

	if len(req.Description) > 0 {
		oc.Description = req.Description
	}

	if len(req.RedirectURI) > 0 {
		if !uriRegex.MatchString(req.RedirectURI) {
			return echo.NewHTTPError(http.StatusBadRequest)
		}
		oc.RedirectURI = req.RedirectURI
	}

	if err := h.OAuth2.UpdateClient(oc); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}

// DeleteClient DELETE /clients/:clientID
func (h *Handlers) DeleteClient(c echo.Context) error {
	clientID := c.Param("clientID")
	userID := getRequestUserID(c)

	oc, err := h.OAuth2.GetClient(clientID)
	if err != nil {
		switch err {
		case oauth2.ErrClientNotFound:
			return echo.NewHTTPError(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}
	if oc.CreatorID != userID {
		return echo.NewHTTPError(http.StatusForbidden)
	}

	// revoke tokens
	if err := h.OAuth2.DeleteTokenByClient(clientID); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	// delete client
	if err := h.OAuth2.DeleteClient(clientID); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}
