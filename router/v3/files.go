package v3

import (
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/router/utils"
	"net/http"
)

// PostFile POST /files
func (h *Handlers) PostFile(c echo.Context) error {
	userID := getRequestUserID(c)

	// ファイルチェック
	src, uploadedFile, err := c.Request().FormFile("file")
	if err != nil {
		return herror.BadRequest(err)
	}
	defer src.Close()
	if uploadedFile.Size == 0 {
		return herror.BadRequest("non-empty file is required")
	}

	// チャンネルアクセス権確認
	channelId := uuid.FromStringOrNil(c.FormValue("channelId"))
	if ok, err := h.Repo.IsChannelAccessibleToUser(userID, channelId); err != nil {
		return herror.InternalServerError(err)
	} else if !ok {
		return herror.BadRequest("invalid channelId")
	}
	ch, err := h.Repo.GetChannel(channelId)
	if err != nil {
		return herror.InternalServerError(err)
	}

	args := repository.SaveFileArgs{
		FileName: uploadedFile.Filename,
		FileSize: uploadedFile.Size,
		MimeType: uploadedFile.Header.Get(echo.HeaderContentType),
		FileType: model.FileTypeUserFile,
		Src:      src,
	}
	args.SetCreator(userID)
	args.SetChannel(channelId)
	if !ch.IsPublic {
		members, err := h.Repo.GetPrivateChannelMemberIDs(ch.ID)
		if err != nil {
			return herror.InternalServerError(err)
		}
		for _, v := range members {
			args.ACLAllow(v)
		}
	}

	file, err := h.Repo.SaveFile(args)
	if err != nil {
		return herror.InternalServerError(err)
	}
	return c.JSON(http.StatusCreated, formatFileInfo(file))
}

// GetFileMeta GET /files/:fileID/meta
func (h *Handlers) GetFileMeta(c echo.Context) error {
	return c.JSON(http.StatusOK, formatFileInfo(getParamFile(c)))
}

// GetThumbnailImage GET /files/:fileID/thumbnail
func (h *Handlers) GetThumbnailImage(c echo.Context) error {
	return utils.ServeFileThumbnail(c, getParamFile(c))
}

// GetFile GET /files/:fileID
func (h *Handlers) GetFile(c echo.Context) error {
	return utils.ServeFile(c, getParamFile(c))
}

// DeleteFile DELETE /files/:fileID
func (h *Handlers) DeleteFile(c echo.Context) error {
	userID := getRequestUserID(c)
	f := getParamFile(c)

	if !f.GetCreatorID().Valid || f.GetFileType() != model.FileTypeUserFile {
		return herror.Forbidden()
	}

	if f.GetCreatorID().UUID != userID { // TODO 管理者権限
		return herror.Forbidden()
	}

	if err := h.Repo.DeleteFile(f.GetID()); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}
