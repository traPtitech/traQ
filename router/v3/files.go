package v3

import (
	"fmt"
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"net/http"
	"strconv"
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
	meta := getParamFile(c)

	if !meta.HasThumbnail() {
		return herror.NotFound()
	}

	file, err := meta.OpenThumbnail()
	if err != nil {
		return herror.InternalServerError(err)
	}
	defer file.Close()

	c.Response().Header().Set(consts.HeaderFileMetaType, meta.GetFileType())
	c.Response().Header().Set(consts.HeaderCacheFile, "true")
	c.Response().Header().Set(consts.HeaderCacheControl, "private, max-age=31536000") // 1年間キャッシュ
	return c.Stream(http.StatusOK, meta.GetThumbnailMIMEType(), file)
}

// GetFile GET /files/:fileID
func (h *Handlers) GetFile(c echo.Context) error {
	meta := getParamFile(c)

	c.Response().Header().Set(consts.HeaderFileMetaType, meta.GetFileType())
	switch meta.GetFileType() {
	case model.FileTypeStamp, model.FileTypeIcon:
		c.Response().Header().Set(consts.HeaderCacheFile, "true")
	}

	// 直接アクセスURLが発行できる場合は、そっちにリダイレクト
	url, _ := h.Repo.GetFS().GenerateAccessURL(meta.GetID().String(), meta.GetFileType())
	if len(url) > 0 {
		return c.Redirect(http.StatusFound, url)
	}

	file, err := meta.Open()
	if err != nil {
		return herror.InternalServerError(err)
	}
	defer file.Close()

	c.Response().Header().Set(echo.HeaderContentType, meta.GetMIMEType())
	c.Response().Header().Set(consts.HeaderETag, strconv.Quote(meta.GetMD5Hash()))
	c.Response().Header().Set(consts.HeaderCacheControl, "private, max-age=31536000") // 1年間キャッシュ
	if isTrue(c.QueryParam("dl")) {
		c.Response().Header().Set(echo.HeaderContentDisposition, fmt.Sprintf("attachment; filename=%s", meta.GetFileName()))
	}

	http.ServeContent(c.Response(), c.Request(), meta.GetFileName(), meta.GetCreatedAt(), file)
	return nil
}
