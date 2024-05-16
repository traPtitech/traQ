package v3

import (
	"net/http"

	vd "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/service/rbac/permission"
	"github.com/traPtitech/traQ/utils/optional"
	"github.com/traPtitech/traQ/utils/random"
	"github.com/traPtitech/traQ/utils/validator"
)

// GetClients GET /clients
func (h *Handlers) GetClients(c echo.Context) error {
	var q repository.GetClientsQuery

	if !isTrue(c.QueryParam("all")) {
		q = q.IsDevelopedBy(getRequestUserID(c))
	}

	ocs, err := h.Repo.GetClients(q)
	if err != nil {
		return herror.InternalServerError(err)
	}
	return c.JSON(http.StatusOK, formatOAuth2Clients(ocs))
}

// PostClientsRequest POST /clients リクエストボディ
type PostClientsRequest struct {
	Name         string             `json:"name"`
	Description  string             `json:"description"`
	CallbackURL  string             `json:"callbackUrl"`
	Scopes       model.AccessScopes `json:"scopes"`
	Confidential bool               `json:"confidential"` // default false (public client)
}

func (r PostClientsRequest) Validate() error {
	return vd.ValidateStruct(&r,
		vd.Field(&r.Name, vd.Required, vd.RuneLength(1, 32)),
		vd.Field(&r.Description, vd.Required, vd.RuneLength(1, 1000)),
		vd.Field(&r.CallbackURL, vd.Required, is.URL),
		vd.Field(&r.Scopes, vd.Required),
	)
}

// CreateClient POST /clients
func (h *Handlers) CreateClient(c echo.Context) error {
	userID := getRequestUserID(c)

	var req PostClientsRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	client := &model.OAuth2Client{
		ID:           random.SecureAlphaNumeric(36),
		Name:         req.Name,
		Description:  req.Description,
		Confidential: req.Confidential,
		CreatorID:    userID,
		RedirectURI:  req.CallbackURL,
		Secret:       random.SecureAlphaNumeric(36),
		Scopes:       req.Scopes,
	}
	if err := h.Repo.SaveClient(client); err != nil {
		return herror.InternalServerError(err)
	}

	return c.JSON(http.StatusCreated, formatOAuth2ClientDetail(client))
}

// GetClient GET /clients/:clientID
func (h *Handlers) GetClient(c echo.Context) error {
	oc := getParamClient(c)

	if isTrue(c.QueryParam("detail")) {
		user := getRequestUser(c)
		if !h.RBAC.IsGranted(user.GetRole(), permission.ManageOthersClient) && oc.CreatorID != user.GetID() {
			return herror.Forbidden()
		}
		return c.JSON(http.StatusOK, formatOAuth2ClientDetail(oc))
	}

	return c.JSON(http.StatusOK, formatOAuth2Client(oc))
}

// PatchClientRequest PATCH /clients/:clientID リクエストボディ
type PatchClientRequest struct {
	Name         optional.Of[string]    `json:"name"`
	Description  optional.Of[string]    `json:"description"`
	CallbackURL  optional.Of[string]    `json:"callbackUrl"`
	DeveloperID  optional.Of[uuid.UUID] `json:"developerId"`
	Confidential optional.Of[bool]      `json:"confidential"`
}

func (r PatchClientRequest) Validate() error {
	return vd.ValidateStruct(&r,
		vd.Field(&r.Name, validator.RequiredIfValid, vd.RuneLength(1, 32)),
		vd.Field(&r.Description, validator.RequiredIfValid, vd.RuneLength(1, 1000)),
		vd.Field(&r.CallbackURL, validator.RequiredIfValid, is.URL),
		vd.Field(&r.DeveloperID, validator.NotNilUUID),
	)
}

// EditClient PATCH /clients/:clientID
func (h *Handlers) EditClient(c echo.Context) error {
	oc := getParamClient(c)

	var req PatchClientRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	args := repository.UpdateClientArgs{
		Name:         req.Name,
		Description:  req.Description,
		DeveloperID:  req.DeveloperID,
		CallbackURL:  req.CallbackURL,
		Confidential: req.Confidential,
	}
	if err := h.Repo.UpdateClient(oc.ID, args); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}

// DeleteClient DELETE /clients/:clientID
func (h *Handlers) DeleteClient(c echo.Context) error {
	oc := getParamClient(c)

	// delete client
	if err := h.Repo.DeleteClient(oc.ID); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}
