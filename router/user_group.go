package router

import (
	"github.com/gofrs/uuid"
	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/rbac/role"
	"github.com/traPtitech/traQ/repository"
	"gopkg.in/guregu/null.v3"
	"net/http"
)

// GetUserGroups GET /groups
func (h *Handlers) GetUserGroups(c echo.Context) error {
	gs, err := h.Repo.GetAllUserGroups()
	if err != nil {
		return internalServerError(err, h.requestContextLogger(c))
	}

	res, err := h.formatUserGroups(gs)
	if err != nil {
		return internalServerError(err, h.requestContextLogger(c))
	}

	return c.JSON(http.StatusOK, res)
}

// PostUserGroups POST /groups
func (h *Handlers) PostUserGroups(c echo.Context) error {
	reqUserID := getRequestUserID(c)

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Type        string `json:"type"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return badRequest(err)
	}

	if req.Type == "grade" {
		// 学年グループは権限が必要
		if getRequestUser(c).Role != role.Admin.ID() {
			return forbidden("you are not permitted to create groups of this type")
		}
	}

	g, err := h.Repo.CreateUserGroup(req.Name, req.Description, req.Type, reqUserID)
	if err != nil {
		switch {
		case err == repository.ErrAlreadyExists:
			return conflict("the name's group has already existed")
		case repository.IsArgError(err):
			return badRequest(err)
		default:
			return internalServerError(err, h.requestContextLogger(c))
		}
	}

	res, _ := h.formatUserGroup(g)
	return c.JSON(http.StatusCreated, res)
}

// GetUserGroup GET /groups/:groupID
func (h *Handlers) GetUserGroup(c echo.Context) error {
	g := getGroupFromContext(c)

	res, err := h.formatUserGroup(g)
	if err != nil {
		return internalServerError(err, h.requestContextLogger(c))
	}

	return c.JSON(http.StatusOK, res)
}

// PatchUserGroup PATCH /groups/:groupID
func (h *Handlers) PatchUserGroup(c echo.Context) error {
	groupID := getRequestParamAsUUID(c, paramGroupID)
	reqUserID := getRequestUserID(c)
	g := getGroupFromContext(c)

	var req struct {
		Name        null.String   `json:"name"`
		Description null.String   `json:"description"`
		AdminUserID uuid.NullUUID `json:"adminUserId"`
		Type        null.String   `json:"type"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return badRequest(err)
	}

	// 管理者ユーザーかどうか
	if g.AdminUserID != reqUserID {
		return forbidden("you are not the group's admin")
	}

	if req.Type.ValueOrZero() == "grade" {
		// 学年グループは権限が必要
		if getRequestUser(c).Role != role.Admin.ID() {
			return forbidden("you are not permitted to create groups of this type")
		}
	}

	args := repository.UpdateUserGroupNameArgs{
		Name:        req.Name,
		Description: req.Description,
		AdminUserID: req.AdminUserID,
		Type:        req.Type,
	}
	if err := h.Repo.UpdateUserGroup(groupID, args); err != nil {
		switch {
		case err == repository.ErrAlreadyExists:
			return conflict("the name's group has already existed")
		case repository.IsArgError(err):
			return badRequest(err)
		default:
			return internalServerError(err, h.requestContextLogger(c))
		}
	}

	return c.NoContent(http.StatusNoContent)
}

// DeleteUserGroup DELETE /groups/:groupID
func (h *Handlers) DeleteUserGroup(c echo.Context) error {
	groupID := getRequestParamAsUUID(c, paramGroupID)
	userID := getRequestUserID(c)
	g := getGroupFromContext(c)

	// 管理者ユーザーかどうか
	if g.AdminUserID != userID {
		return forbidden("you are not the group's admin")
	}

	if err := h.Repo.DeleteUserGroup(groupID); err != nil {
		return internalServerError(err, h.requestContextLogger(c))
	}

	return c.NoContent(http.StatusNoContent)
}

// GetUserGroupMembers GET /groups/:groupID/members
func (h *Handlers) GetUserGroupMembers(c echo.Context) error {
	groupID := getRequestParamAsUUID(c, paramGroupID)

	res, err := h.Repo.GetUserGroupMemberIDs(groupID)
	if err != nil {
		return internalServerError(err, h.requestContextLogger(c))
	}

	return c.JSON(http.StatusOK, res)
}

// PostUserGroupMembers POST /groups/:groupID/members
func (h *Handlers) PostUserGroupMembers(c echo.Context) error {
	groupID := getRequestParamAsUUID(c, paramGroupID)
	reqUserID := getRequestUserID(c)
	g := getGroupFromContext(c)

	var req struct {
		UserID uuid.UUID `json:"userId"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return badRequest(err)
	}

	// 管理者ユーザーかどうか
	if g.AdminUserID != reqUserID {
		return forbidden("you are not the group's admin")
	}

	// ユーザーが存在するか
	if ok, err := h.Repo.UserExists(req.UserID); err != nil {
		return internalServerError(err, h.requestContextLogger(c))
	} else if !ok {
		return badRequest("this user doesn't exist")
	}

	if err := h.Repo.AddUserToGroup(req.UserID, groupID); err != nil {
		return internalServerError(err, h.requestContextLogger(c))
	}

	return c.NoContent(http.StatusNoContent)
}

// DeleteUserGroupMembers DELETE /groups/:groupID/members/:userID
func (h *Handlers) DeleteUserGroupMembers(c echo.Context) error {
	groupID := getRequestParamAsUUID(c, paramGroupID)
	userID := getRequestParamAsUUID(c, paramUserID)
	reqUserID := getRequestUserID(c)
	g := getGroupFromContext(c)

	// 管理者ユーザーかどうか
	if g.AdminUserID != reqUserID {
		return forbidden("you are not the group's admin")
	}

	if err := h.Repo.RemoveUserFromGroup(userID, groupID); err != nil {
		return internalServerError(err, h.requestContextLogger(c))
	}

	return c.NoContent(http.StatusNoContent)
}

// GetMyBelongingGroup GET /users/me/groups
func (h *Handlers) GetMyBelongingGroup(c echo.Context) error {
	userID := getRequestUserID(c)

	ids, err := h.Repo.GetUserBelongingGroupIDs(userID)
	if err != nil {
		return internalServerError(err, h.requestContextLogger(c))
	}

	return c.JSON(http.StatusOK, ids)
}

// GetUserBelongingGroup GET /users/:userID/groups
func (h *Handlers) GetUserBelongingGroup(c echo.Context) error {
	userID := getRequestParamAsUUID(c, paramUserID)

	ids, err := h.Repo.GetUserBelongingGroupIDs(userID)
	if err != nil {
		return internalServerError(err, h.requestContextLogger(c))
	}

	return c.JSON(http.StatusOK, ids)
}
