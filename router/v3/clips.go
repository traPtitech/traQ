package v3

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// PostClipFolders POST /clip-folders
func (h *Handlers) CreateClipFolders(c echo.Context) error {
	userID := getRequestUserID(c)

	var req PostChannelRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, h.Realtime.OnlineCounter.GetOnlineUserIDs())
}

// GetClipFolders GET /clip-folders
func (h *Handlers) GetClipFolders(c echo.Context) error {
	return c.JSON(http.StatusOK, h.Realtime.OnlineCounter.GetOnlineUserIDs())
}

// GetClipFolder GET /clip-folders/:folderID
func (h *Handlers) GetClipFolder(c echo.Context) error {
	return c.JSON(http.StatusOK, h.Realtime.OnlineCounter.GetOnlineUserIDs())
}

// DeleteClipFolder DELETE /clip-folder/:folderID
func (h *Handlers) DeleteClipFolder(c echo.Context) error {
	return c.JSON(http.StatusOK, h.Realtime.OnlineCounter.GetOnlineUserIDs())
}

// EditClipFolder PATCH /clip-folders/:folderID
func (h *Handlers) EditClipFolder(c echo.Context) error {
	return c.JSON(http.StatusOK, h.Realtime.OnlineCounter.GetOnlineUserIDs())
}

// PostClipFolderMessage POST /clip-folders/:folderID/messages
func (h *Handlers) PostClipFolderMessages(c echo.Context) error {
	cf := getParamClipFolder(c)

	var req PostClipFolderMessageRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	m, err := h.Repo.AddClipFolderMessage(cf.ID, req.MessageID)
	if err != nil {
		return herror.InternalServerError(err)
	}

	return c.JSON(http.StatusOK, formatClipFolderMessage(cf.ID, m))
}

// GetFolderMessages GET /clip-folders/:folderID/messages
func (h *Handlers) GetClipFolderMessages(c echo.Context) error {
	return c.JSON(http.StatusOK, h.Realtime.OnlineCounter.GetOnlineUserIDs())
}

// DeleteFolderMessages DELETE /clip-folders/:folderID/messages/:messageID
func (h *Handlers) DeleteClipFolderMessages(c echo.Context) error {
	return c.JSON(http.StatusOK, h.Realtime.OnlineCounter.GetOnlineUserIDs())
}
