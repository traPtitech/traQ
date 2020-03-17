package v1

import (
	"fmt"
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/router/utils"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/model"
)

// PostFile POST /files
func (h *Handlers) PostFile(c echo.Context) error {
	userID := getRequestUserID(c)

	src, uploadedFile, err := c.Request().FormFile("file")
	if err != nil {
		return herror.BadRequest(err)
	}
	defer src.Close()

	if uploadedFile.Size == 0 {
		return herror.BadRequest("non-empty file is required")
	}

	args := repository.SaveFileArgs{
		FileName: uploadedFile.Filename,
		FileSize: uploadedFile.Size,
		MimeType: uploadedFile.Header.Get(echo.HeaderContentType),
		FileType: model.FileTypeUserFile,
		ACL:      repository.ACL{},
		Src:      src,
	}
	args.SetCreator(userID)

	// アクセスコントロールリスト作成
	if s := c.FormValue("acl_readable"); len(s) != 0 && s != "all" {
		for _, v := range strings.Split(s, ",") {
			uid, _ := uuid.FromString(v)
			if ok, err := h.Repo.UserExists(uid); err != nil {
				return herror.InternalServerError(err)
			} else if !ok {
				return herror.BadRequest(fmt.Sprintf("unknown acl user id: %s", uid))
			}
			args.ACLAllow(uid)
		}
	} else {
		args.ACLAllow(uuid.Nil)
	}

	file, err := h.Repo.SaveFile(args)
	if err != nil {
		return herror.InternalServerError(err)
	}

	return c.JSON(http.StatusCreated, formatFile(file))
}

// GetFileByID GET /files/:fileID
func (h *Handlers) GetFileByID(c echo.Context) error {
	return utils.ServeFile(c, getFileFromContext(c))
}

// DeleteFileByID DELETE /files/:fileID
func (h *Handlers) DeleteFileByID(c echo.Context) error {
	fileID := getRequestParamAsUUID(c, consts.ParamFileID)

	if err := h.Repo.DeleteFile(fileID); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}

// GetMetaDataByFileID GET /files/:fileID/meta
func (h *Handlers) GetMetaDataByFileID(c echo.Context) error {
	meta := getFileFromContext(c)
	c.Response().Header().Set(consts.HeaderCacheControl, "private, max-age=86400") // 1日キャッシュ
	return c.JSON(http.StatusOK, formatFile(meta))
}

// GetThumbnailByID GET /files/:fileID/thumbnail
func (h *Handlers) GetThumbnailByID(c echo.Context) error {
	return utils.ServeFileThumbnail(c, getFileFromContext(c))
}
