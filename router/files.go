package router

import (
	"fmt"
	"github.com/karixtech/zapdriver"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/repository"
	"go.uber.org/zap"
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/model"
)

// PostFile POST /files
func (h *Handlers) PostFile(c echo.Context) error {
	userID := getRequestUserID(c)

	uploadedFile, err := c.FormFile("file")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	// アクセスコントロールリスト作成
	aclRead := repository.ACL{}
	if s := c.FormValue("acl_readable"); len(s) != 0 && s != "all" {
		for _, v := range strings.Split(s, ",") {
			uid, err := uuid.FromString(v)
			if err != nil || uid == uuid.Nil {
				return echo.NewHTTPError(http.StatusBadRequest, err)
			}

			if ok, err := h.Repo.UserExists(uid); err != nil {
				h.Logger.Error(unexpectedError, zap.Error(err), zapdriver.HTTP(zapdriver.NewHTTP(c.Request(), nil)))
				return c.NoContent(http.StatusInternalServerError)
			} else if !ok {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("unknown acl user id: %s", uid))
			}
			aclRead[uid] = true
		}
	} else {
		aclRead[uuid.Nil] = true
	}

	src, err := uploadedFile.Open()
	if err != nil {
		h.Logger.Error(unexpectedError, zap.Error(err), zapdriver.HTTP(zapdriver.NewHTTP(c.Request(), nil)))
		return c.NoContent(http.StatusInternalServerError)
	}
	defer src.Close()

	file, err := h.Repo.SaveFileWithACL(uploadedFile.Filename, src, uploadedFile.Size, uploadedFile.Header.Get(echo.HeaderContentType), model.FileTypeUserFile, userID, aclRead)
	if err != nil {
		h.Logger.Error(unexpectedError, zap.Error(err), zapdriver.HTTP(zapdriver.NewHTTP(c.Request(), nil)))
		return c.NoContent(http.StatusInternalServerError)
	}
	return c.JSON(http.StatusCreated, file)
}

// GetFileByID GET /files/:fileID
func (h *Handlers) GetFileByID(c echo.Context) error {
	meta := getFileFromContext(c)
	dl := c.QueryParam("dl")

	c.Response().Header().Set(headerFileMetaType, meta.Type)
	switch meta.Type {
	case model.FileTypeStamp, model.FileTypeIcon:
		c.Response().Header().Set(headerCacheFile, "true")
	}

	// 直接アクセスURLが発行できる場合は、そっちにリダイレクト
	url, _ := h.Repo.GetFS().GenerateAccessURL(meta.GetKey())
	if len(url) > 0 {
		return c.Redirect(http.StatusFound, url)
	}

	file, err := h.Repo.GetFS().OpenFileByKey(meta.GetKey())
	if err != nil {
		h.Logger.Error(unexpectedError, zap.Error(err), zapdriver.HTTP(zapdriver.NewHTTP(c.Request(), nil)))
		return c.NoContent(http.StatusInternalServerError)
	}
	defer file.Close()

	c.Response().Header().Set(echo.HeaderContentType, meta.Mime)
	c.Response().Header().Set(headerETag, strconv.Quote(meta.Hash))
	c.Response().Header().Set(headerCacheControl, "private, max-age=31536000") //1年間キャッシュ
	if dl == "1" {
		c.Response().Header().Set(echo.HeaderContentDisposition, fmt.Sprintf("attachment; filename=%s", meta.Name))
	}

	http.ServeContent(c.Response(), c.Request(), meta.Name, meta.CreatedAt, file)
	return nil
}

// DeleteFileByID DELETE /files/:fileID
func (h *Handlers) DeleteFileByID(c echo.Context) error {
	fileID := getRequestParamAsUUID(c, paramFileID)

	if err := h.Repo.DeleteFile(fileID); err != nil {
		h.Logger.Error(unexpectedError, zap.Error(err), zapdriver.HTTP(zapdriver.NewHTTP(c.Request(), nil)))
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}

// GetMetaDataByFileID GET /files/:fileID/meta
func (h *Handlers) GetMetaDataByFileID(c echo.Context) error {
	meta := getFileFromContext(c)
	return c.JSON(http.StatusOK, meta)
}

// GetThumbnailByID GET /files/:fileID/thumbnail
func (h *Handlers) GetThumbnailByID(c echo.Context) error {
	meta := getFileFromContext(c)

	if !meta.HasThumbnail {
		return echo.NewHTTPError(http.StatusNotFound, "file is found, but thumbnail is not found")
	}

	c.Response().Header().Set(headerFileMetaType, meta.Type)
	c.Response().Header().Set(headerCacheFile, "true")

	// 直接アクセスURLが発行できる場合は、そっちにリダイレクト
	url, _ := h.Repo.GetFS().GenerateAccessURL(meta.GetThumbKey())
	if len(url) > 0 {
		return c.Redirect(http.StatusFound, url)
	}

	file, err := h.Repo.GetFS().OpenFileByKey(meta.GetThumbKey())
	if err != nil {
		h.Logger.Error(unexpectedError, zap.Error(err), zapdriver.HTTP(zapdriver.NewHTTP(c.Request(), nil)))
		return c.NoContent(http.StatusInternalServerError)
	}
	defer file.Close()

	c.Response().Header().Set(headerCacheControl, "private, max-age=31536000") //1年間キャッシュ
	return c.Stream(http.StatusOK, mimeImagePNG, file)
}
