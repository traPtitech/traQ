package v3

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	vd "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/router/utils"
	"github.com/traPtitech/traQ/service/file"
	"github.com/traPtitech/traQ/utils/optional"
)

// GetFilesRequest GET /files 用リクエストクエリ
type GetFilesRequest struct {
	Limit     int                    `query:"limit"`
	Offset    int                    `query:"offset"`
	Since     optional.Of[time.Time] `query:"since"`
	Until     optional.Of[time.Time] `query:"until"`
	Inclusive bool                   `query:"inclusive"`
	Order     string                 `query:"order"`
	ChannelID uuid.UUID              `query:"channelId"`
	Mine      bool                   `query:"mine"`
}

func (q *GetFilesRequest) Validate() error {
	if q.Limit == 0 {
		q.Limit = 20
	}
	return vd.ValidateStruct(q,
		vd.Field(&q.Limit, vd.Min(1), vd.Max(200)),
		vd.Field(&q.Offset, vd.Min(0)),
		vd.Field(&q.Mine, vd.When(q.ChannelID == uuid.Nil, vd.Required)),
	)
}

// GetFiles GET /files
func (h *Handlers) GetFiles(c echo.Context) error {
	var req GetFilesRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	q := repository.FilesQuery{
		Since:     req.Since,
		Until:     req.Until,
		Inclusive: req.Inclusive,
		Limit:     req.Limit,
		Offset:    req.Offset,
		Asc:       strings.ToLower(req.Order) == "asc",
		Type:      model.FileTypeUserFile,
	}

	if req.Mine {
		q.UploaderID = optional.From(getRequestUserID(c))
	}
	if req.ChannelID != uuid.Nil {
		// チャンネルアクセス権確認
		if ok, err := h.ChannelManager.IsChannelAccessibleToUser(getRequestUserID(c), req.ChannelID); err != nil {
			return herror.InternalServerError(err)
		} else if !ok {
			return herror.BadRequest("invalid channelId")
		}
		q.ChannelID = optional.From(req.ChannelID)
	}

	files, more, err := h.FileManager.List(q)
	if err != nil {
		return herror.InternalServerError(err)
	}
	c.Response().Header().Set(consts.HeaderMore, strconv.FormatBool(more))
	return c.JSON(http.StatusOK, formatFileInfos(files))
}

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

	args := file.SaveArgs{
		FileName:  uploadedFile.Filename,
		FileSize:  uploadedFile.Size,
		MimeType:  uploadedFile.Header.Get(echo.HeaderContentType),
		FileType:  model.FileTypeUserFile,
		CreatorID: optional.From(userID),
		Src:       src,
	}

	// チャンネルアクセス権確認
	channelID := uuid.FromStringOrNil(c.FormValue("channelId"))
	if ok, err := h.ChannelManager.IsChannelAccessibleToUser(userID, channelID); err != nil {
		return herror.InternalServerError(err)
	} else if !ok {
		return herror.BadRequest("invalid channelId")
	}
	ch, err := h.ChannelManager.GetChannel(channelID)
	if err != nil {
		return herror.InternalServerError(err)
	}
	if ch.IsArchived() {
		return herror.BadRequest(fmt.Sprintf("channel #%s has been archived", h.ChannelManager.PublicChannelTree().GetChannelPath(ch.ID)))
	}
	if !ch.IsPublic {
		// アクセスコントロール設定
		members, err := h.ChannelManager.GetDMChannelMembers(ch.ID)
		if err != nil {
			return herror.InternalServerError(err)
		}
		for _, v := range members {
			args.ACLAllow(v)
		}
	}
	args.ChannelID = optional.From(channelID)

	// 保存
	file, err := h.FileManager.Save(args)
	if err != nil {
		return herror.InternalServerError(err)
	}
	return c.JSON(http.StatusCreated, formatFileInfo(file))
}

// GetFileMeta GET /files/:fileID/meta
func (h *Handlers) GetFileMeta(c echo.Context) error {
	c.Response().Header().Set(consts.HeaderCacheControl, "private, max-age=86400") // 1日キャッシュ
	return c.JSON(http.StatusOK, formatFileInfo(getParamFile(c)))
}

// GetThumbnailImage GET /files/:fileID/thumbnail
func (h *Handlers) GetThumbnailImage(c echo.Context) error {
	return utils.ServeFileThumbnail(c, getParamFile(c), h.Repo, h.Logger)
}

// GetFile GET /files/:fileID
func (h *Handlers) GetFile(c echo.Context) error {
	return utils.ServeFile(c, getParamFile(c))
}

// DeleteFile DELETE /files/:fileID
func (h *Handlers) DeleteFile(c echo.Context) error {
	f := getParamFile(c)

	if !f.GetCreatorID().Valid || f.GetFileType() != model.FileTypeUserFile {
		return herror.Forbidden()
	}

	if f.GetCreatorID().V != getRequestUserID(c) { // TODO 管理者権限
		return herror.Forbidden()
	}

	if err := h.FileManager.Delete(f.GetID()); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}
