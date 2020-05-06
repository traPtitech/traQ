package v1

import (
	vd "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/rbac/permission"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/utils/optional"
	"net/http"
)

// GetUserGroups GET /groups
func (h *Handlers) GetUserGroups(c echo.Context) error {
	gs, err := h.Repo.GetAllUserGroups()
	if err != nil {
		return herror.InternalServerError(err)
	}

	res, err := h.formatUserGroups(gs)
	if err != nil {
		return herror.InternalServerError(err)
	}

	return c.JSON(http.StatusOK, res)
}

// PostUserGroupsRequest POST /groups リクエストボディ
type PostUserGroupsRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
}

func (r PostUserGroupsRequest) Validate() error {
	return vd.ValidateStruct(&r,
		vd.Field(&r.Name, vd.Required, vd.RuneLength(1, 30)),
		vd.Field(&r.Description, vd.RuneLength(0, 100)),
		vd.Field(&r.Type, vd.RuneLength(0, 30)),
	)
}

// PostUserGroups POST /groups
func (h *Handlers) PostUserGroups(c echo.Context) error {
	reqUserID := getRequestUserID(c)

	var req PostUserGroupsRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	if req.Type == "grade" && !h.RBAC.IsGranted(getRequestUser(c).GetRole(), permission.CreateSpecialUserGroup) {
		// 学年グループは権限が必要
		return herror.Forbidden("you are not permitted to create groups of this type")
	}

	g, err := h.Repo.CreateUserGroup(req.Name, req.Description, req.Type, reqUserID)
	if err != nil {
		switch {
		case err == repository.ErrAlreadyExists:
			return herror.Conflict("the name's group has already existed")
		case repository.IsArgError(err):
			return herror.BadRequest(err)
		default:
			return herror.InternalServerError(err)
		}
	}

	return c.JSON(http.StatusCreated, formatUserGroup(g))
}

// GetUserGroup GET /groups/:groupID
func (h *Handlers) GetUserGroup(c echo.Context) error {
	g := getGroupFromContext(c)
	return c.JSON(http.StatusOK, formatUserGroup(g))
}

// PatchUserGroupRequest PATCH /groups/:groupID リクエストボディ
type PatchUserGroupRequest struct {
	Name        optional.String `json:"name"`
	Description optional.String `json:"description"`
	Type        optional.String `json:"type"`
}

func (r PatchUserGroupRequest) Validate() error {
	return vd.ValidateStruct(&r,
		vd.Field(&r.Name, vd.RuneLength(1, 30)),
		vd.Field(&r.Description, vd.RuneLength(0, 100)),
		vd.Field(&r.Type, vd.RuneLength(0, 30)),
	)
}

// PatchUserGroup PATCH /groups/:groupID
func (h *Handlers) PatchUserGroup(c echo.Context) error {
	groupID := getRequestParamAsUUID(c, consts.ParamGroupID)
	reqUserID := getRequestUserID(c)
	g := getGroupFromContext(c)

	var req PatchUserGroupRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	// 管理者ユーザーかどうか
	if !g.IsAdmin(reqUserID) {
		return herror.Forbidden("you are not the group's admin")
	}

	if req.Type.ValueOrZero() == "grade" && !h.RBAC.IsGranted(getRequestUser(c).GetRole(), permission.CreateSpecialUserGroup) {
		// 学年グループは権限が必要
		return herror.Forbidden("you are not permitted to create groups of this type")
	}

	args := repository.UpdateUserGroupNameArgs{
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
	}
	if err := h.Repo.UpdateUserGroup(groupID, args); err != nil {
		switch {
		case err == repository.ErrAlreadyExists:
			return herror.Conflict("the name's group has already existed")
		case repository.IsArgError(err):
			return herror.BadRequest(err)
		default:
			return herror.InternalServerError(err)
		}
	}

	return c.NoContent(http.StatusNoContent)
}

// DeleteUserGroup DELETE /groups/:groupID
func (h *Handlers) DeleteUserGroup(c echo.Context) error {
	groupID := getRequestParamAsUUID(c, consts.ParamGroupID)
	userID := getRequestUserID(c)
	g := getGroupFromContext(c)

	// 管理者ユーザーかどうか
	if !g.IsAdmin(userID) {
		return herror.Forbidden("you are not the group's admin")
	}

	if err := h.Repo.DeleteUserGroup(groupID); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}

// GetUserGroupMembers GET /groups/:groupID/members
func (h *Handlers) GetUserGroupMembers(c echo.Context) error {
	groupID := getRequestParamAsUUID(c, consts.ParamGroupID)

	res, err := h.Repo.GetUserIDs(repository.UsersQuery{}.GMemberOf(groupID))
	if err != nil {
		return herror.InternalServerError(err)
	}

	return c.JSON(http.StatusOK, res)
}

// PostUserGroupMembersRequest POST /groups/:groupID/members リクエストボディ
type PostUserGroupMembersRequest struct {
	UserID uuid.UUID `json:"userId"`
}

// PostUserGroupMembers POST /groups/:groupID/members
func (h *Handlers) PostUserGroupMembers(c echo.Context) error {
	groupID := getRequestParamAsUUID(c, consts.ParamGroupID)
	reqUserID := getRequestUserID(c)
	g := getGroupFromContext(c)

	var req PostUserGroupMembersRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	// 管理者ユーザーかどうか
	if !g.IsAdmin(reqUserID) {
		return herror.Forbidden("you are not the group's admin")
	}

	// ユーザーが存在するか
	if ok, err := h.Repo.UserExists(req.UserID); err != nil {
		return herror.InternalServerError(err)
	} else if !ok {
		return herror.BadRequest("this user doesn't exist")
	}

	if err := h.Repo.AddUserToGroup(req.UserID, groupID, ""); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}

// DeleteUserGroupMembers DELETE /groups/:groupID/members/:userID
func (h *Handlers) DeleteUserGroupMembers(c echo.Context) error {
	groupID := getRequestParamAsUUID(c, consts.ParamGroupID)
	userID := getRequestParamAsUUID(c, consts.ParamUserID)
	reqUserID := getRequestUserID(c)
	g := getGroupFromContext(c)

	// 管理者ユーザーかどうか
	if !g.IsAdmin(reqUserID) {
		return herror.Forbidden("you are not the group's admin")
	}

	if err := h.Repo.RemoveUserFromGroup(userID, groupID); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}

// GetMyBelongingGroup GET /users/me/groups
func (h *Handlers) GetMyBelongingGroup(c echo.Context) error {
	userID := getRequestUserID(c)

	ids, err := h.Repo.GetUserBelongingGroupIDs(userID)
	if err != nil {
		return herror.InternalServerError(err)
	}

	return c.JSON(http.StatusOK, ids)
}

// GetUserBelongingGroup GET /users/:userID/groups
func (h *Handlers) GetUserBelongingGroup(c echo.Context) error {
	userID := getRequestParamAsUUID(c, consts.ParamUserID)

	ids, err := h.Repo.GetUserBelongingGroupIDs(userID)
	if err != nil {
		return herror.InternalServerError(err)
	}

	return c.JSON(http.StatusOK, ids)
}
