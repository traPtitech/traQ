package router

import (
	"fmt"
	"github.com/satori/go.uuid"
	"net/http"

	"github.com/labstack/echo"
	"github.com/labstack/gommon/log"
	"github.com/traPtitech/traQ/model"
)

// FileForResponse クライアントに返すファイル構造体
type FileForResponse struct {
	ID          string `json:"fileId"`
	Name        string `json:"name"`
	Mime        string `json:"mime"`
	Size        int64  `json:"size"`
	DateTime    string `json:"datetime"`
	HasThumb    bool   `json:"hasThumb"`
	ThumbWidth  int    `json:"thumbWidth,omitempty"`
	ThumbHeight int    `json:"thumbHeight,omitempty"`
}

// PostFile POST /files
func PostFile(c echo.Context) error {
	userID := c.Get("user").(*model.User).ID

	uploadedFile, err := c.FormFile("file")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	file := &model.File{
		Name:      uploadedFile.Filename,
		Size:      uploadedFile.Size,
		CreatorID: userID,
	}

	src, err := uploadedFile.Open()
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}
	defer src.Close()

	if err := file.Create(src); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}
	return c.JSON(http.StatusCreated, formatFile(file))
}

// GetFileByID GET /files/:fileID
func GetFileByID(c echo.Context) error {
	ID := c.Param("fileID")
	dl := c.QueryParam("dl")

	meta, err := validateFileID(ID)
	if err != nil {
		return err
	}

	url := meta.GetRedirectURL()
	if len(url) > 0 {
		return c.Redirect(http.StatusFound, url) //オブジェクトストレージで直接アクセス出来る場合はリダイレクトする
	}

	file, err := meta.Open()
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}
	defer file.Close()

	if dl == "1" {
		c.Response().Header().Set(echo.HeaderContentDisposition, fmt.Sprintf("attachment; filename=%s", meta.Name))
	}

	c.Response().Header().Set("Cache-Control", "private, max-age=31536000") //1年間キャッシュ

	return c.Stream(http.StatusOK, meta.Mime, file)
}

// DeleteFileByID DELETE /files/:fileID
func DeleteFileByID(c echo.Context) error {
	meta, err := validateFileID(c.Param("fileID"))
	if err != nil {
		return err
	}

	if err := model.DeleteFile(meta.GetID()); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}

// GetMetaDataByFileID GET /files/:fileID/meta
func GetMetaDataByFileID(c echo.Context) error {
	meta, err := validateFileID(c.Param("fileID"))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, formatFile(meta))
}

// GetThumbnailByID GET /files/:fileID/thumbnail
func GetThumbnailByID(c echo.Context) error {
	meta, err := validateFileID(c.Param("fileID"))
	if err != nil {
		return err
	}
	if !meta.HasThumbnail {
		return echo.NewHTTPError(http.StatusNotFound, "The specified file exists, but its thumbnail doesn't.")
	}

	file, err := meta.OpenThumbnail()
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}
	defer file.Close()
	c.Response().Header().Set("Cache-Control", "private, max-age=31536000") //1年間キャッシュ

	return c.Stream(http.StatusOK, "image/png", file)
}

func formatFile(f *model.File) *FileForResponse {
	return &FileForResponse{
		ID:          f.ID,
		Name:        f.Name,
		Mime:        f.Mime,
		Size:        f.Size,
		DateTime:    f.CreatedAt.String(),
		HasThumb:    f.HasThumbnail,
		ThumbWidth:  f.ThumbnailWidth,
		ThumbHeight: f.ThumbnailHeight,
	}
}

func validateFileID(fileID string) (*model.File, error) {
	f, err := model.GetMetaFileDataByID(uuid.FromStringOrNil(fileID))
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return nil, echo.NewHTTPError(http.StatusNotFound, "The specified file does not exist")
		default:
			log.Error(err)
			return nil, echo.NewHTTPError(http.StatusInternalServerError, "An error occurred in the server while get file")
		}
	}
	return f, nil
}
