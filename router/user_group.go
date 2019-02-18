package router

import (
	"github.com/labstack/echo"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"net/http"
)

type userGroupResponse struct {
	GroupID     uuid.UUID   `json:"groupId"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	AdminUserId uuid.UUID   `json:"adminUserId"`
	Members     []uuid.UUID `json:"members"`
}

// GetUserGroups GET /groups
func (h *Handlers) GetUserGroups(c echo.Context) error {
	gs, err := h.Repo.GetAllUserGroups()
	if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	res, err := h.formatUserGroups(gs)
	if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, res)
}

// PostUserGroups POST /groups
func (h *Handlers) PostUserGroups(c echo.Context) error {
	reqUserID := getRequestUserID(c)

	var req struct {
		Name        string `json:"name" validate:"max=30,required"`
		Description string `json:"description"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	g, err := h.Repo.CreateUserGroup(req.Name, req.Description, reqUserID)
	if err != nil {
		switch err {
		case repository.ErrAlreadyExists:
			return c.NoContent(http.StatusConflict)
		default:
			c.Logger().Error(err)
			return c.NoContent(http.StatusInternalServerError)
		}
	}

	res, err := h.formatUserGroup(g)
	if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusCreated, res)
}

// GetUserGroup GET /groups/:groupID
func (h *Handlers) GetUserGroup(c echo.Context) error {
	groupID := getRequestParamAsUUID(c, paramGroupID)

	g, err := h.Repo.GetUserGroup(groupID)
	if err != nil {
		switch err {
		case repository.ErrNotFound:
			return c.NoContent(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return c.NoContent(http.StatusInternalServerError)
		}
	}

	res, err := h.formatUserGroup(g)
	if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, res)
}

// PatchUserGroup PATCH /groups/:groupID
func (h *Handlers) PatchUserGroup(c echo.Context) error {
	groupID := getRequestParamAsUUID(c, paramGroupID)
	reqUserID := getRequestUserID(c)

	var req struct {
		Name        string     `json:"name" validate:"max=30"`
		Description *string    `json:"description"`
		AdminUserID *uuid.UUID `json:"adminUserId"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	g, err := h.Repo.GetUserGroup(groupID)
	if err != nil {
		switch err {
		case repository.ErrNotFound:
			return c.NoContent(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return c.NoContent(http.StatusInternalServerError)
		}
	}

	// 管理者ユーザーかどうか
	if g.AdminUserID != reqUserID {
		return c.NoContent(http.StatusForbidden)
	}

	args := repository.UpdateUserGroupNameArgs{
		Name: req.Name,
	}
	if req.Description != nil {
		args.Description.Valid = true
		args.Description.String = *req.Description
	}
	if req.AdminUserID != nil {
		// ユーザーが存在するか
		if ok, err := h.Repo.UserExists(*req.AdminUserID); err != nil {
			c.Logger().Error(err)
			return c.NoContent(http.StatusInternalServerError)
		} else if !ok {
			c.Logger().Error(err)
			return c.NoContent(http.StatusBadRequest)
		}
		args.AdminUserID.Valid = true
		args.AdminUserID.UUID = *req.AdminUserID
	}

	if err := h.Repo.UpdateUserGroup(groupID, args); err != nil {
		switch err {
		case repository.ErrAlreadyExists:
			return c.NoContent(http.StatusConflict)
		default:
			c.Logger().Error(err)
			return c.NoContent(http.StatusInternalServerError)
		}
	}

	return c.NoContent(http.StatusNoContent)
}

// DeleteUserGroup DELETE /groups/:groupID
func (h *Handlers) DeleteUserGroup(c echo.Context) error {
	groupID := getRequestParamAsUUID(c, paramGroupID)
	userID := getRequestUserID(c)

	g, err := h.Repo.GetUserGroup(groupID)
	if err != nil {
		switch err {
		case repository.ErrNotFound:
			return c.NoContent(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return c.NoContent(http.StatusInternalServerError)
		}
	}

	// 管理者ユーザーかどうか
	if g.AdminUserID != userID {
		return c.NoContent(http.StatusForbidden)
	}

	if err := h.Repo.DeleteUserGroup(groupID); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}

// GetUserGroupMembers GET /groups/:groupID/members
func (h *Handlers) GetUserGroupMembers(c echo.Context) error {
	groupID := getRequestParamAsUUID(c, paramGroupID)

	res, err := h.Repo.GetUserGroupMemberIDs(groupID)
	if err != nil {
		switch err {
		case repository.ErrNotFound:
			return c.NoContent(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return c.NoContent(http.StatusInternalServerError)
		}
	}

	return c.JSON(http.StatusOK, res)
}

// PostUserGroupMembers POST /groups/:groupID/members
func (h *Handlers) PostUserGroupMembers(c echo.Context) error {
	groupID := getRequestParamAsUUID(c, paramGroupID)
	reqUserID := getRequestUserID(c)

	var req struct {
		UserID uuid.UUID `json:"userId"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	g, err := h.Repo.GetUserGroup(groupID)
	if err != nil {
		switch err {
		case repository.ErrNotFound:
			return c.NoContent(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return c.NoContent(http.StatusInternalServerError)
		}
	}

	// 管理者ユーザーかどうか
	if g.AdminUserID != reqUserID {
		return c.NoContent(http.StatusForbidden)
	}

	// ユーザーが存在するか
	if ok, err := h.Repo.UserExists(req.UserID); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	} else if !ok {
		c.Logger().Error(err)
		return c.NoContent(http.StatusBadRequest)
	}

	if err := h.Repo.AddUserToGroup(req.UserID, groupID); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}

// DeleteUserGroupMembers DELETE /groups/:groupID/members/:userID
func (h *Handlers) DeleteUserGroupMembers(c echo.Context) error {
	groupID := getRequestParamAsUUID(c, paramGroupID)
	userID := getRequestParamAsUUID(c, paramUserID)
	reqUserID := getRequestUserID(c)

	g, err := h.Repo.GetUserGroup(groupID)
	if err != nil {
		switch err {
		case repository.ErrNotFound:
			return c.NoContent(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return c.NoContent(http.StatusInternalServerError)
		}
	}

	// 管理者ユーザーかどうか
	if g.AdminUserID != reqUserID {
		return c.NoContent(http.StatusForbidden)
	}

	// ユーザーが存在するか
	if ok, err := h.Repo.UserExists(userID); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	} else if !ok {
		c.Logger().Error(err)
		return c.NoContent(http.StatusBadRequest)
	}

	if err := h.Repo.RemoveUserFromGroup(userID, groupID); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}

// GetMyBelongingGroup GET /users/me/groups
func (h *Handlers) GetMyBelongingGroup(c echo.Context) error {
	userID := getRequestUserID(c)

	ids, err := h.Repo.GetUserBelongingGroupIDs(userID)
	if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, ids)
}

// GetUserBelongingGroup GET /users/:userID/groups
func (h *Handlers) GetUserBelongingGroup(c echo.Context) error {
	userID := getRequestParamAsUUID(c, paramUserID)

	// ユーザーが存在するか
	if ok, err := h.Repo.UserExists(userID); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	} else if !ok {
		c.Logger().Error(err)
		return c.NoContent(http.StatusNotFound)
	}

	ids, err := h.Repo.GetUserBelongingGroupIDs(userID)
	if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, ids)
}

func (h *Handlers) formatUserGroup(g *model.UserGroup) (r *userGroupResponse, err error) {
	r = &userGroupResponse{
		GroupID:     g.ID,
		Name:        g.Name,
		Description: g.Description,
		AdminUserId: g.AdminUserID,
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
