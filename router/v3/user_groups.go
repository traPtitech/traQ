package v3

import (
	"context"
	"net/http"

	vd "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"

	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/router/utils"
	file2 "github.com/traPtitech/traQ/service/file"
	"github.com/traPtitech/traQ/service/rbac/permission"
	"github.com/traPtitech/traQ/utils/optional"
	"github.com/traPtitech/traQ/utils/validator"
)

// GetUserGroups GET /groups
func (h *Handlers) GetUserGroups(c echo.Context) error {
	gs, err := h.Repo.GetAllUserGroups()
	if err != nil {
		return herror.InternalServerError(err)
	}
	return extension.ServeJSONWithETag(c, formatUserGroups(gs))
}

// PostUserGroupRequest POST /groups リクエストボディ
type PostUserGroupRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
}

func (r PostUserGroupRequest) Validate() error {
	return vd.ValidateStruct(&r,
		vd.Field(&r.Name, validator.UserGroupNameRuleRequired...),
		vd.Field(&r.Description, vd.RuneLength(0, 100)),
		vd.Field(&r.Type, vd.RuneLength(0, 30)),
	)
}

// PostUserGroups POST /groups
func (h *Handlers) PostUserGroups(c echo.Context) error {
	reqUserID := getRequestUserID(c)

	var req PostUserGroupRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	if req.Type == "grade" && !h.RBAC.IsGranted(getRequestUser(c).GetRole(), permission.CreateSpecialUserGroup) {
		// 学年グループは権限が必要
		return herror.Forbidden("you are not permitted to create groups of this type")
	}

	iconFileID, err := file2.GenerateIconFile(h.FileManager, req.Name)
	if err != nil {
		return herror.InternalServerError(err)
	}

	g, err := h.Repo.CreateUserGroup(req.Name, req.Description, req.Type, reqUserID, iconFileID)
	if err != nil {
		if err == repository.ErrAlreadyExists {
			return herror.Conflict("group with the name already exists")
		}
		return herror.InternalServerError(err)
	}

	return c.JSON(http.StatusCreated, formatUserGroup(g))
}

// GetUserGroup GET /groups/:groupID
func (h *Handlers) GetUserGroup(c echo.Context) error {
	return c.JSON(http.StatusOK, formatUserGroup(getParamGroup(c)))
}

// PatchUserGroupRequest PATCH /groups/:groupID リクエストボディ
type PatchUserGroupRequest struct {
	Name        optional.Of[string] `json:"name"`
	Description optional.Of[string] `json:"description"`
	Type        optional.Of[string] `json:"type"`
}

func (r PatchUserGroupRequest) Validate() error {
	return vd.ValidateStruct(&r,
		vd.Field(&r.Name, append(validator.UserGroupNameRule, validator.RequiredIfValid)...),
		vd.Field(&r.Description, vd.RuneLength(0, 100)),
		vd.Field(&r.Type, vd.RuneLength(0, 30)),
	)
}

// EditUserGroup PATCH /groups/:groupID
func (h *Handlers) EditUserGroup(c echo.Context) error {
	g := getParamGroup(c)

	var req PatchUserGroupRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	if req.Type.ValueOrZero() == "grade" && !h.RBAC.IsGranted(getRequestUser(c).GetRole(), permission.CreateSpecialUserGroup) {
		// 学年グループは権限が必要
		return herror.Forbidden("you are not permitted to create groups of this type")
	}

	args := repository.UpdateUserGroupArgs{
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
	}
	if err := h.Repo.UpdateUserGroup(g.ID, args); err != nil {
		if err == repository.ErrAlreadyExists {
			return herror.Conflict("group with the name already exists")
		}
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}

// PutUserGroupIcon PUT /groups/:groupID/icon
func (h *Handlers) PutUserGroupIcon(c echo.Context) error {
	g := getParamGroup(c)

	fileID, err := utils.SaveUploadIconImage(h.Imaging, c, h.FileManager, "file")
	if err != nil {
		return err
	}

	if err := h.Repo.UpdateUserGroup(g.ID, repository.UpdateUserGroupArgs{Icon: optional.From(fileID)}); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}

// DeleteUserGroup DELETE /groups/:groupID
func (h *Handlers) DeleteUserGroup(c echo.Context) error {
	g := getParamGroup(c)

	if err := h.Repo.DeleteUserGroup(g.ID); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}

// GetUserGroupMembers GET /groups/:groupID/members
func (h *Handlers) GetUserGroupMembers(c echo.Context) error {
	return c.JSON(http.StatusOK, formatUserGroupMembers(getParamGroup(c).Members))
}

// PostUserGroupMemberRequest POST /groups/:groupID/members リクエストボディ
type PostUserGroupMemberRequest struct {
	ID   uuid.UUID `json:"id"`
	Role string    `json:"role"`
}

func (r PostUserGroupMemberRequest) ValidateWithContext(ctx context.Context) error {
	return vd.ValidateStructWithContext(ctx, &r,
		vd.Field(&r.ID, vd.Required, validator.NotNilUUID, utils.IsUserID, utils.IsNotWebhookUserID),
		vd.Field(&r.Role, vd.RuneLength(0, 100)),
	)
}

// AddUserGroupMember POST /groups/:groupID/members
func (h *Handlers) AddUserGroupMember(c echo.Context) error {
	g := getParamGroup(c)

	var reqs []PostUserGroupMemberRequest
	if err := bindAndValidate(c, &reqs); err != nil {
		return err
	}
    if len(reqs) == 1 {
        req := reqs[0]
        if err := h.Repo.AddUserToGroup(req.ID, g.ID, req.Role); err != nil {
            return herror.InternalServerError(err)
        }
	} else if len(reqs) >= 2 {
		users := make([]User, len(reqs))
		for i, req := range reqs {
			users[i] = User{userID: req.ID, role: req.Role}
		}
		if err := h.Repo.AddUsersToGroup(users, g.ID); err != nil {
			return herror.InternalServerError(err)
		}
	}
	return c.NoContent(http.StatusNoContent)
}

// PatchUserGroupMemberRequest PATCH /groups/:groupID/members リクエストボディ
type PatchUserGroupMemberRequest struct {
	Role string `json:"role"`
}

func (r PatchUserGroupMemberRequest) Validate() error {
	return vd.ValidateStruct(&r,
		vd.Field(&r.Role, vd.RuneLength(0, 100)),
	)
}

// EditUserGroupMember POST /groups/:groupID/members/:userID
func (h *Handlers) EditUserGroupMember(c echo.Context) error {
	g := getParamGroup(c)
	uid := getParamAsUUID(c, consts.ParamUserID)

	var req PatchUserGroupMemberRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	// ユーザーがグループに存在するか
	if !g.IsMember(uid) {
		return herror.BadRequest("this user is not this group's member")
	}

	if err := h.Repo.AddUserToGroup(uid, g.ID, req.Role); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}

// RemoveUserGroupMember DELETE /groups/:groupID/members/:userID
func (h *Handlers) RemoveUserGroupMember(c echo.Context) error {
	userID := getParamAsUUID(c, consts.ParamUserID)
	g := getParamGroup(c)

	if err := h.Repo.RemoveUserFromGroup(userID, g.ID); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}

// GetUserGroupAdmins GET /groups/:groupID/admins
func (h *Handlers) GetUserGroupAdmins(c echo.Context) error {
	return c.JSON(http.StatusOK, getParamGroup(c).AdminIDArray())
}

// PostUserGroupAdminRequest POST /groups/:groupID/admins リクエストボディ
type PostUserGroupAdminRequest struct {
	ID uuid.UUID `json:"id"`
}

func (r PostUserGroupAdminRequest) ValidateWithContext(ctx context.Context) error {
	return vd.ValidateStructWithContext(ctx, &r,
		vd.Field(&r.ID, vd.Required, validator.NotNilUUID, utils.IsUserID, utils.IsNotWebhookUserID),
	)
}

// AddUserGroupAdmin POST /groups/:groupID/admins
func (h *Handlers) AddUserGroupAdmin(c echo.Context) error {
	g := getParamGroup(c)

	var req PostUserGroupAdminRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	if err := h.Repo.AddUserToGroupAdmin(req.ID, g.ID); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}

// RemoveUserGroupAdmin DELETE /groups/:groupID/admins/:userID
func (h *Handlers) RemoveUserGroupAdmin(c echo.Context) error {
	userID := getParamAsUUID(c, consts.ParamUserID)
	g := getParamGroup(c)

	if err := h.Repo.RemoveUserFromGroupAdmin(userID, g.ID); err != nil {
		if err == repository.ErrForbidden {
			return herror.BadRequest()
		}
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}
