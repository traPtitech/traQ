package v1

import (
	"fmt"
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"net/http"
	"strconv"
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
	return c.JSON(http.StatusCreated, file)
}

// GetFileByID GET /files/:fileID
func (h *Handlers) GetFileByID(c echo.Context) error {
	meta := getFileFromContext(c)
	dl := c.QueryParam("dl")

	c.Response().Header().Set(consts.HeaderFileMetaType, meta.Type)
	switch meta.Type {
	case model.FileTypeStamp, model.FileTypeIcon:
		c.Response().Header().Set(consts.HeaderCacheFile, "true")
	}

	// 直接アクセスURLが発行できる場合は、そっちにリダイレクト
	url, _ := h.Repo.GetFS().GenerateAccessURL(meta.GetKey(), meta.Type)
	if len(url) > 0 {
		return c.Redirect(http.StatusFound, url)
	}

	file, err := h.Repo.GetFS().OpenFileByKey(meta.GetKey(), meta.Type)
	if err != nil {
		return herror.InternalServerError(err)
	}
	defer file.Close()

	c.Response().Header().Set(echo.HeaderContentType, meta.Mime)
	c.Response().Header().Set(consts.HeaderETag, strconv.Quote(meta.Hash))
	c.Response().Header().Set(consts.HeaderCacheControl, "private, max-age=31536000") // 1年間キャッシュ
	if dl == "1" {
		c.Response().Header().Set(echo.HeaderContentDisposition, fmt.Sprintf("attachment; filename=%s", meta.Name))
	}

	http.ServeContent(c.Response(), c.Request(), meta.Name, meta.CreatedAt, file)
	return nil
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
	return c.JSON(http.StatusOK, meta)
}

// GetThumbnailByID GET /files/:fileID/thumbnail
func (h *Handlers) GetThumbnailByID(c echo.Context) error {
	meta := getFileFromContext(c)

	if !meta.HasThumbnail {
		return herror.NotFound("file is found, but thumbnail is not found")
	}

	c.Response().Header().Set(consts.HeaderFileMetaType, meta.Type)
	c.Response().Header().Set(consts.HeaderCacheFile, "true")

	// 直接アクセスURLが発行できる場合は、そっちにリダイレクト
	url, _ := h.Repo.GetFS().GenerateAccessURL(meta.GetThumbKey(), model.FileTypeThumbnail)
	if len(url) > 0 {
		return c.Redirect(http.StatusFound, url)
	}

	file, err := h.Repo.GetFS().OpenFileByKey(meta.GetThumbKey(), model.FileTypeThumbnail)
	if err != nil {
		return herror.InternalServerError(err)
	}
	defer file.Close()

	c.Response().Header().Set(consts.HeaderCacheControl, "private, max-age=31536000") // 1年間キャッシュ
	return c.Stream(http.StatusOK, consts.MimeImagePNG, file)
}
