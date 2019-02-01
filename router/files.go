package router

import (
	"fmt"
	"github.com/satori/go.uuid"
	"net/http"
	"strconv"

	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/model"
)

// PostFile POST /files
func PostFile(c echo.Context) error {
	userID := getRequestUserID(c)

	uploadedFile, err := c.FormFile("file")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	src, err := uploadedFile.Open()
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}
	defer src.Close()

	file := &model.File{
		Name:      uploadedFile.Filename,
		Size:      uploadedFile.Size,
		Mime:      uploadedFile.Header.Get(echo.HeaderContentType),
		Type:      model.FileTypeUserFile,
		CreatorID: userID,
	}
	if err := file.Create(src); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}
	return c.JSON(http.StatusCreated, file)
}

// GetFileByID GET /files/:fileID
func GetFileByID(c echo.Context) error {
	fileID := getRequestParamAsUUID(c, paramFileID)
	dl := c.QueryParam("dl")

	meta, err := validateFileID(c, fileID)
	if err != nil {
		return err
	}

	file, err := meta.Open()
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}
	defer file.Close()

	c.Response().Header().Set(echo.HeaderContentLength, strconv.FormatInt(meta.Size, 10))
	c.Response().Header().Set(headerCacheControl, "private, max-age=31536000") //1年間キャッシュ
	c.Response().Header().Set(headerFileMetaType, meta.Type)

	switch meta.Type {
	case model.FileTypeStamp, model.FileTypeIcon:
		c.Response().Header().Set(headerCacheFile, "true")
	}

	if dl == "1" {
		c.Response().Header().Set(echo.HeaderContentDisposition, fmt.Sprintf("attachment; filename=%s", meta.Name))
	}

	return c.Stream(http.StatusOK, meta.Mime, file)
}

// DeleteFileByID DELETE /files/:fileID
func DeleteFileByID(c echo.Context) error {
	fileID := getRequestParamAsUUID(c, paramFileID)
	_, err := validateFileID(c, fileID)
	if err != nil {
		return err
	}

	if err := model.DeleteFile(fileID); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}

// GetMetaDataByFileID GET /files/:fileID/meta
func GetMetaDataByFileID(c echo.Context) error {
	fileID := getRequestParamAsUUID(c, paramFileID)
	meta, err := validateFileID(c, fileID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, meta)
}

// GetThumbnailByID GET /files/:fileID/thumbnail
func GetThumbnailByID(c echo.Context) error {
	fileID := getRequestParamAsUUID(c, paramFileID)
	meta, err := validateFileID(c, fileID)
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
	c.Response().Header().Set(headerCacheControl, "private, max-age=31536000") //1年間キャッシュ

	return c.Stream(http.StatusOK, mimeImagePNG, file)
}

func validateFileID(c echo.Context, fileID uuid.UUID) (*model.File, error) {
	f, err := model.GetMetaFileDataByID(fileID)
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return nil, echo.NewHTTPError(http.StatusNotFound, "The specified file does not exist")
		default:
			c.Logger().Error(err)
			return nil, echo.NewHTTPError(http.StatusInternalServerError)
		}
	}
	return f, nil
}
