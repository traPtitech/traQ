package router

import (
	"fmt"
	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/rbac/permission"
	"github.com/traPtitech/traQ/repository"
	"gopkg.in/guregu/null.v3"
	"net/http"
)

// GetRoles GET /authority/roles
func (h *Handlers) GetRoles(c echo.Context) error {
	rs, err := h.Repo.GetAllRoles()
	if err != nil {
		return internalServerError(err, h.requestContextLogger(c))
	}
	return c.JSON(http.StatusOK, formatRoles(rs))
}

// PostRoles POST /authority/roles
func (h *Handlers) PostRoles(c echo.Context) error {
	var req struct {
		Name string `json:"name"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return badRequest(err)
	}

	if err := h.Repo.CreateRole(req.Name); err != nil {
		switch {
		case repository.IsArgError(err):
			return badRequest(err)
		case err == repository.ErrAlreadyExists:
			return conflict()
		default:
			return internalServerError(err, h.requestContextLogger(c))
		}
	}
	return c.NoContent(http.StatusCreated)
}

// GetRole GET /authority/roles/:role
func (h *Handlers) GetRole(c echo.Context) error {
	role := c.Param("role")
	r, err := h.Repo.GetRole(role)
	if err != nil {
		switch err {
		case repository.ErrNotFound:
			return notFound()
		default:
			return internalServerError(err, h.requestContextLogger(c))
		}
	}
	return c.JSON(http.StatusOK, formatRole(r))
}

// PatchRole PATCH /authority/roles/:role
func (h *Handlers) PatchRole(c echo.Context) error {
	var req struct {
		Permissions  []string  `json:"permissions"`
		Inheritances []string  `json:"inheritances"`
		OAuth2Scope  null.Bool `json:"oauth2Scope"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return badRequest(err)
	}

	role := c.Param("role")
	args := repository.UpdateRoleArgs{
		Permissions:  req.Permissions,
		Inheritances: req.Inheritances,
		OAuth2Scope:  req.OAuth2Scope,
	}

	if err := h.Repo.UpdateRole(role, args); err != nil {
		switch {
		case repository.IsArgError(err):
			return badRequest(err)
		case err == repository.ErrNotFound:
			return notFound()
		default:
			return internalServerError(err, h.requestContextLogger(c))
		}
	}
	return c.NoContent(http.StatusNoContent)
}

// GetPermissions GET /authority/permissions
func (h *Handlers) GetPermissions(c echo.Context) error {
	return c.JSON(http.StatusOK, permission.List.Array())
}

// GetAuthorityReload GET /authority/reload
func (h *Handlers) GetAuthorityReload(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{"reload": h.RBAC.LastReloadTime()})
}

// PostAuthorityReload POST /authority/reload
func (h *Handlers) PostAuthorityReload(c echo.Context) error {
	if err := h.RBAC.Reload(); err != nil {
		return internalServerError(fmt.Errorf("rbac reloading failed: %v", err), h.requestContextLogger(c))
	}
	return c.NoContent(http.StatusNoContent)
}
