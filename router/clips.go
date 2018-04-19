package router

import (
	"github.com/go-sql-driver/mysql"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/notification"
	"github.com/traPtitech/traQ/notification/events"
	"net/http"
	"time"

	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/model"
)

type clipFolderForResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// GetClips GET /users/me/clips
func GetClips(c echo.Context) error {
	type clipMessageForResponse struct {
		FolderID  string              `json:"folderId"`
		ClipID    string              `json:"clipId"`
		ClippedAt time.Time           `json:"clippedAt"`
		Message   *MessageForResponse `json:"message"`
	}

	user := c.Get("user").(*model.User)

	clips, err := model.GetClipMessagesByUser(user.GetUID())
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	res := make([]*clipMessageForResponse, len(clips))
	for i, v := range clips {
		res[i] = &clipMessageForResponse{
			FolderID:  v.Clip.FolderID,
			ClipID:    v.Clip.ID,
			ClippedAt: v.Clip.CreatedAt,
			Message:   formatMessage(v.Message),
		}
	}

	return c.JSON(http.StatusOK, res)
}

// PostClip POST /users/me/clips
func PostClip(c echo.Context) error {
	user := c.Get("user").(*model.User)

	req := struct {
		MessageID string `json:"messageId" validate:"uuid,required"`
		FolderID  string `json:"folderId"`
	}{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}
	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	if _, err := validateMessageID(req.MessageID, user.ID); err != nil {
		return err
	}

	if len(req.FolderID) > 0 {
		_, err := model.GetClipFolder(uuid.FromStringOrNil(req.FolderID))
		if err != nil {
			switch err {
			case model.ErrNotFound:
				return echo.NewHTTPError(http.StatusBadRequest, "the folder is not found")
			default:
				c.Logger().Error(err)
				return echo.NewHTTPError(http.StatusInternalServerError)
			}
		}
	} else {
		folders, err := model.GetClipFolders(user.GetUID())
		if err != nil {
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
		for _, v := range folders {
			if v.Name == "Default" {
				req.FolderID = v.ID
				break
			}
		}
		if len(req.FolderID) == 0 {
			folder, err := model.CreateClipFolder(user.GetUID(), "Default")
			if err != nil {
				c.Logger().Error(err)
				return echo.NewHTTPError(http.StatusInternalServerError)
			}
			go notification.Send(events.ClipFolderCreated, events.ClipEvent{ID: folder.ID, UserID: user.ID})
			req.FolderID = folder.ID
		}
	}

	clip, err := model.CreateClip(uuid.Must(uuid.FromString(req.MessageID)), uuid.Must(uuid.FromString(req.FolderID)), user.GetUID())
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	go notification.Send(events.ClipCreated, events.ClipEvent{ID: clip.ID, UserID: clip.UserID})
	return c.JSON(http.StatusCreated, struct {
		ID string `json:"id"`
	}{clip.ID})
}

// GetClip GET /users/me/clips/:clipID
func GetClip(c echo.Context) error {
	clipID := c.Param("clipID")
	user := c.Get("user").(*model.User)

	clip, err := model.GetClipMessage(uuid.FromStringOrNil(clipID))
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}
	if clip.Clip.UserID != user.ID {
		return echo.NewHTTPError(http.StatusNotFound)
	}

	return c.JSON(http.StatusOK, formatMessage(clip.Message))
}

// DeleteClip DELETE /users/me/clips/:clipID
func DeleteClip(c echo.Context) error {
	clipID := c.Param("clipID")
	user := c.Get("user").(*model.User)

	clip, err := model.GetClipMessage(uuid.FromStringOrNil(clipID))
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}
	if clip.Clip.UserID != user.ID {
		return echo.NewHTTPError(http.StatusNotFound)
	}

	if err := model.DeleteClip(uuid.FromStringOrNil(clip.Clip.ID)); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	go notification.Send(events.ClipDeleted, events.ClipEvent{ID: clip.Clip.ID, UserID: clip.Clip.UserID})
	return c.NoContent(http.StatusNoContent)
}

// GetClipsFolder GET /users/me/clips/:clipID/folder
func GetClipsFolder(c echo.Context) error {
	clipID := c.Param("clipID")
	user := c.Get("user").(*model.User)

	clip, err := model.GetClipMessage(uuid.FromStringOrNil(clipID))
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}
	if clip.Clip.UserID != user.ID {
		return echo.NewHTTPError(http.StatusNotFound)
	}

	folder, err := model.GetClipFolder(uuid.FromStringOrNil(clip.FolderID))
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	return c.JSON(http.StatusOK, formatClipFolder(folder))
}

// PutClipsFolder PUT /users/me/clips/:clipID/folder
func PutClipsFolder(c echo.Context) error {
	clipID := c.Param("clipID")
	user := c.Get("user").(*model.User)

	req := struct {
		FolderID string `json:"folderId" validate:"uuid,required"`
	}{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}
	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	clip, err := model.GetClipMessage(uuid.FromStringOrNil(clipID))
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}
	if clip.Clip.UserID != user.ID {
		return echo.NewHTTPError(http.StatusNotFound)
	}

	folder, err := model.GetClipFolder(uuid.FromStringOrNil(req.FolderID))
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusBadRequest, "the folder is not found")
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	clip.Clip.FolderID = folder.ID

	if err := model.UpdateClip(clip.Clip); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	go notification.Send(events.ClipMoved, events.ClipEvent{ID: clip.Clip.ID, UserID: clip.Clip.UserID})
	return c.NoContent(http.StatusNoContent)
}

// GetClipFolders GET /users/me/clips/folders
func GetClipFolders(c echo.Context) error {
	user := c.Get("user").(*model.User)

	folders, err := model.GetClipFolders(user.GetUID())
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	res := make([]*clipFolderForResponse, len(folders))
	for i, v := range folders {
		res[i] = formatClipFolder(v)
	}

	return c.JSON(http.StatusOK, res)
}

// PostClipFolder POST /users/me/clips/folders
func PostClipFolder(c echo.Context) error {
	user := c.Get("user").(*model.User)

	req := struct {
		Name string `json:"name" validate:"required,max=30"`
	}{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}
	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	folder, err := model.CreateClipFolder(user.GetUID(), req.Name)
	if err != nil {
		switch e := err.(type) {
		case *mysql.MySQLError:
			if e.Number == 1062 {
				return echo.NewHTTPError(http.StatusConflict, "the name is duplicated")
			}
		}
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	go notification.Send(events.ClipFolderCreated, events.ClipEvent{ID: folder.ID, UserID: folder.UserID})
	return c.JSON(http.StatusCreated, formatClipFolder(folder))
}

// GetClipFolder GET /users/me/clips/folders/:folderID
func GetClipFolder(c echo.Context) error {
	type clipMessageForResponse struct {
		ClipID    string              `json:"clipId"`
		ClippedAt time.Time           `json:"clippedAt"`
		Message   *MessageForResponse `json:"message"`
	}

	folderID := c.Param("folderID")
	user := c.Get("user").(*model.User)

	folder, err := model.GetClipFolder(uuid.FromStringOrNil(folderID))
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}
	if folder.UserID != user.ID {
		return echo.NewHTTPError(http.StatusNotFound)
	}

	clips, err := model.GetClipMessages(uuid.FromStringOrNil(folderID))
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	res := make([]*clipMessageForResponse, len(clips))
	for i, v := range clips {
		res[i] = &clipMessageForResponse{
			ClipID:    v.Clip.ID,
			ClippedAt: v.Clip.CreatedAt,
			Message:   formatMessage(v.Message),
		}
	}

	return c.JSON(http.StatusOK, res)
}

// PatchClipFolder PATCH /users/me/clips/folders/:folderID
func PatchClipFolder(c echo.Context) error {
	folderID := c.Param("folderID")
	user := c.Get("user").(*model.User)

	req := struct {
		Name string `json:"name" validate:"required,max=30"`
	}{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}
	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	folder, err := model.GetClipFolder(uuid.FromStringOrNil(folderID))
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}
	if folder.UserID != user.ID {
		return echo.NewHTTPError(http.StatusNotFound)
	}

	folder.Name = req.Name

	if err := model.UpdateClipFolder(folder); err != nil {
		switch e := err.(type) {
		case *mysql.MySQLError:
			if e.Number == 1062 {
				return echo.NewHTTPError(http.StatusConflict, "the name is duplicated")
			}
		}
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	go notification.Send(events.ClipFolderUpdated, events.ClipEvent{ID: folder.ID, UserID: folder.UserID})
	return c.NoContent(http.StatusNoContent)
}

// DeleteClipFolder DELETE /users/me/clips/folders/:folderID
func DeleteClipFolder(c echo.Context) error {
	folderID := c.Param("folderID")
	user := c.Get("user").(*model.User)

	folder, err := model.GetClipFolder(uuid.FromStringOrNil(folderID))
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}
	if folder.UserID != user.ID {
		return echo.NewHTTPError(http.StatusNotFound)
	}

	if err := model.DeleteClipFolder(uuid.FromStringOrNil(folderID)); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	go notification.Send(events.ClipFolderDeleted, events.ClipEvent{ID: folder.ID, UserID: folder.UserID})
	return c.NoContent(http.StatusNoContent)
}

func formatClipFolder(raw *model.ClipFolder) *clipFolderForResponse {
	return &clipFolderForResponse{
		ID:   raw.ID,
		Name: raw.Name,
	}
}
