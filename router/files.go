package router

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/model"
)

// FileForResponse クライアントに返すファイル構造体
type FileForResponse struct {
	ID       string `json:"fileID"`
	Name     string `json:"name"`
	Mime     string `json:"mime"`
	Size     int64  `json:"size"`
	DateTime string `json:"datetime"`
}

// PostFile POST /file のハンドラ
func PostFile(c echo.Context) error {
	userID := c.Get("user").(*model.User).ID

	uploadedFile, err := c.FormFile("file")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Failed to upload file: %v", err))
	}

	file := &model.File{
		Name:      uploadedFile.Filename,
		Size:      uploadedFile.Size,
		CreatorID: userID,
	}

	src, err := uploadedFile.Open()
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Failed to open file")
	}

	if err := file.Create(src); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create file")
	}
	return c.JSON(http.StatusCreated, formatFile(file))
}

// GetFileByID GET /file/{fileID}
func GetFileByID(c echo.Context) error {
	ID := c.Param("fileID")

	file, err := model.OpenFileByID(ID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get file")
	}
	defer file.Close()

	f, err := model.GetMetaFileDataByID(ID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get File")
	}

	return c.Stream(http.StatusOK, f.Mime, file)
}

// DeleteFileByID DELETE /file/{fileID}
func DeleteFileByID(c echo.Context) error {
	ID := c.Param("fileID")

	file, err := model.GetMetaFileDataByID(ID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("fileID is wrong: %s", ID))
	}

	if err := file.Delete(); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to delete data")
	}
	return c.NoContent(http.StatusNoContent)
}

// GetMetaDataByFileID GET /file/{fileID}/meta
func GetMetaDataByFileID(c echo.Context) error {
	ID := c.Param("fileID")

	file, err := model.GetMetaFileDataByID(ID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "fileID is wrong")
	}
	return c.JSON(http.StatusOK, formatFile(file))
}

// TODO: そのうち実装
// GetThumnailByID GET /file/{fileID}/thumnail

func formatFile(f *model.File) *FileForResponse {
	return &FileForResponse{
		ID:       f.ID,
		Name:     f.Name,
		Mime:     f.Mime,
		Size:     f.Size,
		DateTime: f.CreatedAt.String(),
	}
}
