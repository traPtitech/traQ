package v1

import (
	"net/http"
	"time"

	vd "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/utils/optional"
	"github.com/traPtitech/traQ/utils/random"
)

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
		return herror.InternalServerError(err)
	}

	res := make([]AllowedClientInfo, len(ot))
	for i, v := range ot {
		oc, err := h.Repo.GetClient(v.ClientID)
		if err != nil {
			return herror.InternalServerError(err)
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
	tokenID := getRequestParamAsUUID(c, consts.ParamTokenID)
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

// GetClients GET /clients
func (h *Handlers) GetClients(c echo.Context) error {
	oc, err := h.Repo.GetClients(repository.GetClientsQuery{}.IsDevelopedBy(getRequestUserID(c)))
	if err != nil {
		return herror.InternalServerError(err)
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

// PostClientsRequest POST /clients リクエストボディ
type PostClientsRequest struct {
	Name        string             `json:"name"`
	Description string             `json:"description"`
	RedirectURI string             `json:"redirectUri"`
	Scopes      model.AccessScopes `json:"scopes"`
}

func (r PostClientsRequest) Validate() error {
	return vd.ValidateStruct(&r,
		vd.Field(&r.Name, vd.Required, vd.RuneLength(1, 32)),
		vd.Field(&r.Description, vd.Required, vd.RuneLength(0, 1000)),
		vd.Field(&r.RedirectURI, vd.Required, is.URL),
		vd.Field(&r.Scopes, vd.Required),
	)
}

// PostClients POST /clients
func (h *Handlers) PostClients(c echo.Context) error {
	userID := getRequestUserID(c)

	var req PostClientsRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	client := &model.OAuth2Client{
		ID:           random.SecureAlphaNumeric(36),
		Name:         req.Name,
		Description:  req.Description,
		Confidential: false,
		CreatorID:    userID,
		RedirectURI:  req.RedirectURI,
		Secret:       random.SecureAlphaNumeric(36),
		Scopes:       req.Scopes,
	}
	if err := h.Repo.SaveClient(client); err != nil {
		return herror.InternalServerError(err)
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

// PatchClientRequest PATCH /clients/:clientID リクエストボディ
type PatchClientRequest struct {
	Name        optional.String `json:"name"`
	Description optional.String `json:"description"`
	RedirectURI optional.String `json:"redirectUri"`
}

func (r PatchClientRequest) Validate() error {
	return vd.ValidateStruct(&r,
		vd.Field(&r.Name, vd.RuneLength(1, 32)),
		vd.Field(&r.Description, vd.RuneLength(1, 1000)),
		vd.Field(&r.RedirectURI, is.URL),
	)
}

// PatchClient PATCH /clients/:clientID
func (h *Handlers) PatchClient(c echo.Context) error {
	oc := getClientFromContext(c)

	var req PatchClientRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	args := repository.UpdateClientArgs{
		Name:        req.Name,
		Description: req.Description,
		CallbackURL: req.RedirectURI,
	}
	if err := h.Repo.UpdateClient(oc.ID, args); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}

// DeleteClient DELETE /clients/:clientID
func (h *Handlers) DeleteClient(c echo.Context) error {
	clientID := c.Param(consts.ParamClientID)

	// delete client
	if err := h.Repo.DeleteClient(clientID); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}

// GetClientDetail GET /client/:clientID/detail
func (h *Handlers) GetClientDetail(c echo.Context) error {
	oc := getClientFromContext(c)

	return c.JSON(http.StatusOK, &OwnedClientInfo{
		ClientID:    oc.ID,
		Name:        oc.Name,
		Description: oc.Description,
		CreatorID:   oc.CreatorID,
		Scopes:      oc.Scopes,
		RedirectURI: oc.RedirectURI,
		Secret:      oc.Secret,
	})
}
