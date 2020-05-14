package v1

import (
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/service/rbac/permission"
	"github.com/traPtitech/traQ/service/rbac/role"
	"github.com/traPtitech/traQ/utils/optional"
	"net/http"
)

// GetRoles GET /authority/roles
func (h *Handlers) GetRoles(c echo.Context) error {
	rs, err := h.Repo.GetAllRoles()
	if err != nil {
		return herror.InternalServerError(err)
	}
	return c.JSON(http.StatusOK, formatRoles(rs))
}

// PostRoles POST /authority/roles
func (h *Handlers) PostRoles(c echo.Context) error {
	var req struct {
		Name string `json:"name"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	if err := h.Repo.CreateRole(req.Name); err != nil {
		switch {
		case repository.IsArgError(err):
			return herror.BadRequest(err)
		case err == repository.ErrAlreadyExists:
			return herror.Conflict()
		default:
			return herror.InternalServerError(err)
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
			return herror.NotFound()
		default:
			return herror.InternalServerError(err)
		}
	}
	return c.JSON(http.StatusOK, formatRole(r))
}

// PatchRole PATCH /authority/roles/:role
func (h *Handlers) PatchRole(c echo.Context) error {
	var req struct {
		Permissions  []string      `json:"permissions"`
		Inheritances []string      `json:"inheritances"`
		OAuth2Scope  optional.Bool `json:"oauth2Scope"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	r := c.Param("role")
	if r == role.Admin {
		return herror.Forbidden()
	}

	args := repository.UpdateRoleArgs{
		Permissions:  req.Permissions,
		Inheritances: req.Inheritances,
		OAuth2Scope:  req.OAuth2Scope,
	}

	if err := h.Repo.UpdateRole(r, args); err != nil {
		switch {
		case repository.IsArgError(err):
			return herror.BadRequest(err)
		case err == repository.ErrNotFound:
			return herror.NotFound()
		default:
			return herror.InternalServerError(err)
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
		return herror.InternalServerError(fmt.Errorf("rbac reloading failed: %v", err))
	}
	return c.NoContent(http.StatusNoContent)
}
