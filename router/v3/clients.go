package v3

import (
	vd "github.com/go-ozzo/ozzo-validation"
	"github.com/go-ozzo/ozzo-validation/is"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/utils"
	"net/http"
)

// PostClientsRequest POST /clients リクエストボディ
type PostClientsRequest struct {
	Name        string             `json:"name"`
	Description string             `json:"description"`
	CallbackURL string             `json:"callbackUrl"`
	Scopes      model.AccessScopes `json:"scopes"`
}

func (r PostClientsRequest) Validate() error {
	return vd.ValidateStruct(&r,
		vd.Field(&r.Name, vd.Required, vd.RuneLength(1, 32)),
		vd.Field(&r.Description, vd.Required, vd.RuneLength(0, 1000)),
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
		ID:           utils.RandAlphabetAndNumberString(36),
		Name:         req.Name,
		Description:  req.Description,
		Confidential: false,
		CreatorID:    userID,
		RedirectURI:  req.CallbackURL,
		Secret:       utils.RandAlphabetAndNumberString(36),
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
		if oc.CreatorID != getRequestUserID(c) { // TODO 管理者権限
			return herror.Forbidden()
		}
		return c.JSON(http.StatusOK, formatOAuth2ClientDetail(oc))
	}

	return c.JSON(http.StatusOK, formatOAuth2Client(oc))
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
