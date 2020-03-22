package v3

import (
	"net/http"
	"strconv"

	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
)

type PostClipFolderRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// PostClipFolders POST /clip-folders
func (h *Handlers) CreateClipFolders(c echo.Context) error {
	userID := getRequestUserID(c)

	var req PostClipFolderRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	cf, err := h.Repo.CreateClipFolder(userID, req.Name, req.Description)
	if err != nil {
		return herror.InternalServerError(err)
	}

	return c.JSON(http.StatusCreated, formatClipFolder(cf))

}

// GetClipFolders GET /clip-folders
func (h *Handlers) GetClipFolders(c echo.Context) error {
	userID := getRequestUserID(c)

	cfs, err := h.Repo.GetClipFoldersByUserID(userID)
	if err != nil {
		return herror.InternalServerError(err)
	}

	return c.JSON(http.StatusOK, formatClipFolders(cfs))
}

// GetClipFolder GET /clip-folders/:folderID
func (h *Handlers) GetClipFolder(c echo.Context) error {
	return c.JSON(http.StatusOK, formatClipFolder(getParamClipFolder(c)))
}

// DeleteClipFolder DELETE /clip-folder/:folderID
func (h *Handlers) DeleteClipFolder(c echo.Context) error {
	folderID := getParamAsUUID(c, consts.ParamClipFolderID)

	if err := h.Repo.DeleteClipFolder(folderID); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

// EditClipFolder PATCH /clip-folders/:folderID
func (h *Handlers) EditClipFolder(c echo.Context) error {
	cf := getParamClipFolder(c)

	var req PostClipFolderRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	if err := h.Repo.UpdateClipFolder(cf.ID, req.Name, req.Description); err != nil {
		return herror.InternalServerError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

type PostClipFolderMessageRequest struct {
	MessageID uuid.UUID
}

// PostClipFolderMessage POST /clip-folders/:folderID/messages
func (h *Handlers) PostClipFolderMessages(c echo.Context) error {
	cf := getParamClipFolder(c)

	var req PostClipFolderMessageRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	m, err := h.Repo.GetMessageByID(req.MessageID)
	if err != nil {
		return herror.InternalServerError(err)
	}

	var cfm *model.ClipFolderMessage
	cfm, err = h.Repo.AddClipFolderMessage(cf.ID, req.MessageID)
	if err != nil {
		return herror.InternalServerError(err)
	}

	return c.JSON(http.StatusOK, formatClipFolderMessage(cfm.ClippedAt, m))
}

// GetFolderMessages GET /clip-folders/:folderID/messages
func (h *Handlers) GetClipFolderMessages(c echo.Context) error {
	cf := getParamClipFolder(c)

	var req repository.ClipFolderMessageQuery
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	messages, more, err := h.Repo.GetClipFolderMessages(cf.ID, req)
	if err != nil {
		return herror.InternalServerError(err)
	}

	c.Response().Header().Set(consts.HeaderMore, strconv.FormatBool(more))

	return c.JSON(http.StatusOK, messages)
}

// DeleteFolderMessages DELETE /clip-folders/:folderID/messages/:messageID
func (h *Handlers) DeleteClipFolderMessages(c echo.Context) error {
	m := getParamMessage(c)
	cf := getParamClipFolder(c)
	if err := h.Repo.DeleteClipFolderMessage(cf.ID, m.ID); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}
