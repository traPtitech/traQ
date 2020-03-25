package v3

import (
	vd "github.com/go-ozzo/ozzo-validation"
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/router/utils"
	"gopkg.in/guregu/null.v3"
	"net/http"
	"strconv"
	"strings"
)

// GetFilesRequest GET /files 用リクエストクエリ
type GetFilesRequest struct {
	Limit     int           `query:"limit"`
	Offset    int           `query:"offset"`
	Since     null.Time     `query:"since"`
	Until     null.Time     `query:"until"`
	Inclusive bool          `query:"inclusive"`
	Order     string        `query:"order"`
	ChannelID uuid.NullUUID `query:"channelId"`
	Mine      bool          `query:"mine"`
}

func (q *GetFilesRequest) Validate() error {
	if q.Limit == 0 {
		q.Limit = 20
	}
	if !q.ChannelID.Valid && !q.Mine {
		q.Mine = true
	}
	return vd.ValidateStruct(q,
		vd.Field(&q.Limit, vd.Min(1), vd.Max(200)),
		vd.Field(&q.Offset, vd.Min(0)),
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
		Type:      null.StringFrom(model.FileTypeUserFile),
	}

	if req.Mine {
		q.UploaderID = uuid.NullUUID{Valid: true, UUID: getRequestUserID(c)}
	}
	if req.ChannelID.Valid {
		// チャンネルアクセス権確認
		if ok, err := h.Repo.IsChannelAccessibleToUser(getRequestUserID(c), req.ChannelID.UUID); err != nil {
			return herror.InternalServerError(err)
		} else if !ok {
			return herror.BadRequest("invalid channelId")
		}
		q.ChannelID = req.ChannelID
	}

	files, more, err := h.Repo.GetFiles(q)
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
