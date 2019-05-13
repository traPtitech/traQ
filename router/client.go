package router

import (
	"fmt"
	"github.com/gofrs/uuid"
	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/utils"
	"net/http"
	"regexp"
	"time"
)

var uriRegex = regexp.MustCompile(`^([a-z0-9+.-]+):(?://(?:((?:[a-z0-9-._~!$&'()*+,;=:]|%[0-9A-F]{2})*)@)?((?:[a-z0-9-._~!$&'()*+,;=]|%[0-9A-F]{2})*)(?::(\d*))?(/(?:[a-z0-9-._~!$&'()*+,;=:@/]|%[0-9A-F]{2})*)?|(/?(?:[a-z0-9-._~!$&'()*+,;=:@]|%[0-9A-F]{2})+(?:[a-z0-9-._~!$&'()*+,;=:@/]|%[0-9A-F]{2})*)?)(?:\?((?:[a-z0-9-._~!$&'()*+,;=:/?@]|%[0-9A-F]{2})*))?(?:#((?:[a-z0-9-._~!$&'()*+,;=:/?@]|%[0-9A-F]{2})*))?$`)

// ClientInfo レスポンス用クライアント情報構造体
type ClientInfo struct {
	ClientID    string    `json:"clientId"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatorID   uuid.UUID `json:"creatorId"`
}

// OwnedClientInfo レスポンス用クライアント情報構造体
type OwnedClientInfo struct {
	ClientID    string             `json:"clientId"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	CreatorID   uuid.UUID          `json:"creatorId"`
	Scopes      model.AccessScopes `json:"scopes"`
	RedirectURI string             `json:"redirectUri"`
	Secret      string             `json:"secret"`
}

// AllowedClientInfo レスポンス用クライアント情報構造体
type AllowedClientInfo struct {
	TokenID     uuid.UUID          `json:"tokenId"`
	ClientID    string             `json:"clientId"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	CreatorID   uuid.UUID          `json:"creatorId"`
	Scopes      model.AccessScopes `json:"scopes"`
	ApprovedAt  time.Time          `json:"approvedAt"`
}

// GetMyTokens GET /users/me/tokens
func (h *Handlers) GetMyTokens(c echo.Context) error {
	userID := getRequestUserID(c)

	ot, err := h.Repo.GetTokensByUser(userID)
	if err != nil {
		return internalServerError(err, h.requestContextLogger(c))
	}

	res := make([]AllowedClientInfo, len(ot))
	for i, v := range ot {
		oc, err := h.Repo.GetClient(v.ClientID)
		if err != nil {
			return internalServerError(err, h.requestContextLogger(c))
		}
		res[i] = AllowedClientInfo{
			TokenID:     v.ID,
			ClientID:    v.ClientID,
			Name:        oc.Name,
			Description: oc.Description,
			CreatorID:   oc.CreatorID,
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

	ot, err := h.Repo.GetTokenByID(tokenID)
	if err != nil {
		switch err {
		case repository.ErrNotFound:
			return notFound()
		default:
			return internalServerError(err, h.requestContextLogger(c))
		}
	}
	if ot.UserID != userID {
		return notFound()
	}

	if err := h.Repo.DeleteTokenByAccess(ot.AccessToken); err != nil {
		return internalServerError(err, h.requestContextLogger(c))
	}

	return c.NoContent(http.StatusNoContent)
}

// GetClients GET /clients
func (h *Handlers) GetClients(c echo.Context) error {
	userID := getRequestUserID(c)

	oc, err := h.Repo.GetClientsByUser(userID)
	if err != nil {
		return internalServerError(err, h.requestContextLogger(c))
	}

	res := make([]OwnedClientInfo, len(oc))
	for i, v := range oc {
		res[i] = OwnedClientInfo{
			ClientID:    v.ID,
			Name:        v.Name,
			Description: v.Description,
			CreatorID:   v.CreatorID,
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
		return badRequest(err)
	}

	scopes := model.AccessScopes{}
	for _, v := range req.Scopes {
		s := model.AccessScope(v)
		if !validScope(s) {
			return badRequest(fmt.Sprintf("invalid scope: %s", s))
		}
		scopes = append(scopes, s)
	}

	client := &model.OAuth2Client{
		ID:           utils.RandAlphabetAndNumberString(36),
		Name:         req.Name,
		Description:  req.Description,
		Confidential: false,
		CreatorID:    userID,
		RedirectURI:  req.RedirectURI,
		Secret:       utils.RandAlphabetAndNumberString(36),
		Scopes:       scopes,
	}
	if err := h.Repo.SaveClient(client); err != nil {
		return internalServerError(err, h.requestContextLogger(c))
	}

	return c.JSON(http.StatusOK, &OwnedClientInfo{
		ClientID:    client.ID,
		Name:        client.Name,
		Description: client.Description,
		CreatorID:   client.CreatorID,
		Scopes:      client.Scopes,
		RedirectURI: client.RedirectURI,
		Secret:      client.Secret,
	})
}

// GetClient GET /clients/:clientID
func (h *Handlers) GetClient(c echo.Context) error {
	oc := getClientFromContext(c)
	return c.JSON(http.StatusOK, &ClientInfo{
		ClientID:    oc.ID,
		Name:        oc.Name,
		Description: oc.Description,
		CreatorID:   oc.CreatorID,
	})
}

// PatchClient PATCH /clients/:clientID
func (h *Handlers) PatchClient(c echo.Context) error {
	oc := getClientFromContext(c)

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		RedirectURI string `json:"redirectUri"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return badRequest(err)
	}

	if len(req.Name) > 0 {
		if len(req.Name) > 32 {
			return badRequest("invalid name")
		}
		oc.Name = req.Name
	}

	if len(req.Description) > 0 {
		oc.Description = req.Description
	}

	if len(req.RedirectURI) > 0 {
		if !uriRegex.MatchString(req.RedirectURI) {
			return badRequest("invalid redirect uri")
		}
		oc.RedirectURI = req.RedirectURI
	}

	if err := h.Repo.UpdateClient(oc); err != nil {
		return internalServerError(err, h.requestContextLogger(c))
	}

	return c.NoContent(http.StatusNoContent)
}

// DeleteClient DELETE /clients/:clientID
func (h *Handlers) DeleteClient(c echo.Context) error {
	clientID := c.Param(paramClientID)

	// delete client
	if err := h.Repo.DeleteClient(clientID); err != nil {
		return internalServerError(err, h.requestContextLogger(c))
	}

	return c.NoContent(http.StatusNoContent)
}
