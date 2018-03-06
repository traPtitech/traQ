package router

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/model"
)

// FileForResponse クライアントに返すファイル構造体
type FileForResponse struct {
	ID       string `json:"fileId"`
	Name     string `json:"name"`
	Mime     string `json:"mime"`
	Size     int64  `json:"size"`
	DateTime string `json:"datetime"`
}

// PostFile POST /files のハンドラ
func PostFile(c echo.Context) error {
	userID := c.Get("user").(*model.User).ID

	uploadedFile, err := c.FormFile("file")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("failed to upload file: %v", err))
	}

	file := &model.File{
		Name:      uploadedFile.Filename,
		Size:      uploadedFile.Size,
		CreatorID: userID,
	}

	src, err := uploadedFile.Open()
	if err != nil {
		c.Echo().Logger.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}
	defer src.Close()

	if err := file.Create(src); err != nil {
		c.Echo().Logger.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}
	return c.JSON(http.StatusCreated, formatFile(file))
}

// GetFileByID GET /files/{fileID}
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
		c.Echo().Logger.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}
	defer file.Close()

	if dl == "1" {
		c.Response().Header().Set(echo.HeaderContentDisposition, fmt.Sprintf("attachment; filename=%s", meta.Name))
	}

	return c.Stream(http.StatusOK, meta.Mime, file)
}

// DeleteFileByID DELETE /files/{fileID}
func DeleteFileByID(c echo.Context) error {
	ID := c.Param("fileID")

	meta, err := validateFileID(ID)
	if err != nil {
		c.Echo().Logger.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}
	if file == nil {
		return echo.NewHTTPError(http.StatusNotFound)
	}

	if err := meta.Delete(); err != nil {
		c.Echo().Logger.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to delete data")
	}

	return c.NoContent(http.StatusNoContent)
}

// GetMetaDataByFileID GET /files/{fileID}/meta
func GetMetaDataByFileID(c echo.Context) error {
	ID := c.Param("fileID")

	meta, err := validateFileID(ID)
	if err != nil {
		c.Echo().Logger.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}
	return c.JSON(http.StatusOK, formatFile(meta))
}

// TODO: そのうち実装
// GetThumbnailByID GET /files/{fileID}/thumbnail

func formatFile(f *model.File) *FileForResponse {
	return &FileForResponse{
		ID:       f.ID,
		Name:     f.Name,
		Mime:     f.Mime,
		Size:     f.Size,
		DateTime: f.CreatedAt.String(),
	}
}

func validateFileID(fileID string) (*model.File, error) {
	f := &model.File{ID: fileID}
	ok, err := f.Exists()
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "An error occurred in the server while get file")

	}
	if !ok {
		return nil, echo.NewHTTPError(http.StatusNotFound, "The specified channel does not exist")
	}
	return f, nil
}
