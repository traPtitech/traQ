package router

import (
	"github.com/gofrs/uuid"
	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/rbac/role"
	"github.com/traPtitech/traQ/repository"
	"go.uber.org/zap"
	"gopkg.in/guregu/null.v3"
	"net/http"
	"time"
)

type userGroupResponse struct {
	GroupID     uuid.UUID   `json:"groupId"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Type        string      `json:"type"`
	AdminUserID uuid.UUID   `json:"adminUserId"`
	Members     []uuid.UUID `json:"members"`
	CreatedAt   time.Time   `json:"createdAt"`
	UpdatedAt   time.Time   `json:"updatedAt"`
}

// GetUserGroups GET /groups
func (h *Handlers) GetUserGroups(c echo.Context) error {
	gs, err := h.Repo.GetAllUserGroups()
	if err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return c.NoContent(http.StatusInternalServerError)
	}

	res, err := h.formatUserGroups(gs)
	if err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return c.NoContent(http.StatusInternalServerError)
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
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	if req.Type == "grade" {
		// 学年グループは権限が必要
		if getRequestUser(c).Role != role.Admin.ID() {
			return c.NoContent(http.StatusForbidden)
		}
	}

	g, err := h.Repo.CreateUserGroup(req.Name, req.Description, req.Type, reqUserID)
	if err != nil {
		switch {
		case err == repository.ErrAlreadyExists:
			return echo.NewHTTPError(http.StatusConflict, "name conflicts")
		case repository.IsArgError(err):
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		default:
			h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
			return c.NoContent(http.StatusInternalServerError)
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
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return c.NoContent(http.StatusInternalServerError)
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
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	// 管理者ユーザーかどうか
	if g.AdminUserID != reqUserID {
		return c.NoContent(http.StatusForbidden)
	}

	if req.Type.ValueOrZero() == "grade" {
		// 学年グループは権限が必要
		if getRequestUser(c).Role != role.Admin.ID() {
			return c.NoContent(http.StatusForbidden)
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
			return echo.NewHTTPError(http.StatusConflict, "name conflicts")
		case repository.IsArgError(err):
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		default:
			h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
			return c.NoContent(http.StatusInternalServerError)
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
		return c.NoContent(http.StatusForbidden)
	}

	if err := h.Repo.DeleteUserGroup(groupID); err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}

// GetUserGroupMembers GET /groups/:groupID/members
func (h *Handlers) GetUserGroupMembers(c echo.Context) error {
	groupID := getRequestParamAsUUID(c, paramGroupID)

	res, err := h.Repo.GetUserGroupMemberIDs(groupID)
	if err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return c.NoContent(http.StatusInternalServerError)
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
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	// 管理者ユーザーかどうか
	if g.AdminUserID != reqUserID {
		return c.NoContent(http.StatusForbidden)
	}

	// ユーザーが存在するか
	if ok, err := h.Repo.UserExists(req.UserID); err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return c.NoContent(http.StatusInternalServerError)
	} else if !ok {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return c.NoContent(http.StatusBadRequest)
	}

	if err := h.Repo.AddUserToGroup(req.UserID, groupID); err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return c.NoContent(http.StatusInternalServerError)
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
		return c.NoContent(http.StatusForbidden)
	}

	// ユーザーが存在するか
	if ok, err := h.Repo.UserExists(userID); err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return c.NoContent(http.StatusInternalServerError)
	} else if !ok {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return c.NoContent(http.StatusBadRequest)
	}

	if err := h.Repo.RemoveUserFromGroup(userID, groupID); err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}

// GetMyBelongingGroup GET /users/me/groups
func (h *Handlers) GetMyBelongingGroup(c echo.Context) error {
	userID := getRequestUserID(c)

	ids, err := h.Repo.GetUserBelongingGroupIDs(userID)
	if err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, ids)
}

// GetUserBelongingGroup GET /users/:userID/groups
func (h *Handlers) GetUserBelongingGroup(c echo.Context) error {
	userID := getRequestParamAsUUID(c, paramUserID)

	ids, err := h.Repo.GetUserBelongingGroupIDs(userID)
	if err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, ids)
}

func (h *Handlers) formatUserGroup(g *model.UserGroup) (r *userGroupResponse, err error) {
	r = &userGroupResponse{
		GroupID:     g.ID,
		Name:        g.Name,
		Description: g.Description,
		Type:        g.Type,
		AdminUserID: g.AdminUserID,
		CreatedAt:   g.CreatedAt,
		UpdatedAt:   g.UpdatedAt,
	}
	r.Members, err = h.Repo.GetUserGroupMemberIDs(g.ID)
	return
}

func (h *Handlers) formatUserGroups(gs []*model.UserGroup) ([]*userGroupResponse, error) {
	arr := make([]*userGroupResponse, len(gs))
	for i, g := range gs {
		r, err := h.formatUserGroup(g)
		if err != nil {
			return nil, err
		}
		arr[i] = r
	}
	return arr, nil
}
