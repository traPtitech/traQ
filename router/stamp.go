package router

import (
	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/model"
	"net/http"
)

// GetStamps : GET /stamps
func GetStamps(c echo.Context) error {
	stamps, err := model.GetAllStamps()
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, stamps)
}

// PostStamp : POST /stamps
func PostStamp(c echo.Context) error {
	userID := c.Get("user").(*model.User).ID

	name := c.FormValue("name")
	if len(name) == 0 || len(name) > 32 {
		return echo.NewHTTPError(http.StatusBadRequest)
	}
	uploadedFile, err := c.FormFile("file")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	contentType := uploadedFile.Header.Get(echo.HeaderContentType)
	switch contentType {
	case "image/png", "image/jpeg", "image/gif", "image/svg+xml":
		break
	default:
		return echo.NewHTTPError(http.StatusBadRequest, "invalid image file")
	}

	if uploadedFile.Size > 1024*1024 {
		return echo.NewHTTPError(http.StatusBadRequest, "too big image file")
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

	if _, err := model.CreateStamp(name, file.ID, userID); err != nil {
		if err == model.ErrStampInvalidName {
			return echo.NewHTTPError(http.StatusBadRequest, err)
		}
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusCreated)
}

// PatchStamp : PATCH /stamps/:stampID
func PatchStamp(c echo.Context) error {
	userID := c.Get("user").(*model.User).ID
	stampID := c.Param("stampID")

	stamp, err := model.GetStamp(stampID)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}
	if stamp == nil {
		return echo.NewHTTPError(http.StatusNotFound)
	}
	if stamp.CreatorID != userID {
		return echo.NewHTTPError(http.StatusForbidden)
	}

	name := c.FormValue("name")
	if len(name) > 0 {
		stamp.Name = name
	}
	uploadedFile, err := c.FormFile("file")
	if err != http.ErrMissingFile {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}
	if err == nil {
		contentType := uploadedFile.Header.Get(echo.HeaderContentType)
		switch contentType {
		case "image/png", "image/jpeg", "image/gif", "image/svg+xml":
			break
		default:
			return echo.NewHTTPError(http.StatusBadRequest, "invalid image file")
		}

		if uploadedFile.Size > 1024*1024 {
			return echo.NewHTTPError(http.StatusBadRequest, "too big image file")
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
		stamp.FileID = file.ID
	}

	if err := stamp.Update(); err != nil {
		if err == model.ErrStampInvalidName {
			return echo.NewHTTPError(http.StatusBadRequest, err)
		}
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}

// DeleteStamp : DELETE /stamps/:stampID
func DeleteStamp(c echo.Context) error {
	userID := c.Get("user").(*model.User).ID
	stampID := c.Param("stampID")

	stamp, err := model.GetStamp(stampID)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}
	if stamp == nil {
		return echo.NewHTTPError(http.StatusNotFound)
	}
	if stamp.CreatorID != userID {
		return echo.NewHTTPError(http.StatusForbidden)
	}

	if err = model.DeleteStamp(stampID); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}

// GetMessageStamps : GET /messages/:messageID/stamps
func GetMessageStamps(c echo.Context) error {
	messageID := c.Param("messageID")

	//TODO 見れないメッセージ(プライベートチャンネル)に対して404にする
	stamps, err := model.GetMessageStamps(messageID)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, stamps)
}

// PostMessageStamp : POST /messages/:messageID/stamps/:stampID
func PostMessageStamp(c echo.Context) error {
	userID := c.Get("user").(*model.User).ID
	messageID := c.Param("messageID")
	stampID := c.Param("stampID")

	//TODO 見れないメッセージ(プライベートチャンネル)に対して404にする
	err := model.AddStampToMessage(messageID, stampID, userID)
	if err != nil {
		//TODO エラーの種類で400,404,500に分岐
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	return c.NoContent(http.StatusNoContent)
}

// DeleteMessageStamp : DELETE /messages/:messageID/stamps/:stampID
func DeleteMessageStamp(c echo.Context) error {
	userID := c.Get("user").(*model.User).ID
	messageID := c.Param("messageID")
	stampID := c.Param("stampID")

	//TODO 見れないメッセージ(プライベートチャンネル)に対して404にする
	err := model.RemoveStampFromMessage(messageID, stampID, userID)
	if err != nil {
		//TODO エラーの種類で400,404,500に分岐
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	return c.NoContent(http.StatusNoContent)
}
