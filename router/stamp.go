package router

import (
	"github.com/go-sql-driver/mysql"
	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/notification"
	"github.com/traPtitech/traQ/notification/events"
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

	s, err := model.CreateStamp(name, file.ID, userID)
	if err != nil {
		switch err {
		case model.ErrStampInvalidName:
			return echo.NewHTTPError(http.StatusBadRequest, err)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	go notification.Send(events.StampCreated, events.StampEvent{ID: s.ID})
	return c.NoContent(http.StatusCreated)
}

// GetStamp : GET /stamps/:stampID
func GetStamp(c echo.Context) error {
	stampID := c.Param("stampID")

	stamp, err := model.GetStamp(stampID)
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	return c.JSON(http.StatusOK, stamp)
}

// PatchStamp : PATCH /stamps/:stampID
func PatchStamp(c echo.Context) error {
	userID := c.Get("user").(*model.User).ID
	stampID := c.Param("stampID")

	stamp, err := model.GetStamp(stampID)
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
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
		switch err {
		case model.ErrStampInvalidName:
			return echo.NewHTTPError(http.StatusBadRequest, err)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	go notification.Send(events.StampModified, events.StampEvent{ID: stamp.ID})
	return c.NoContent(http.StatusNoContent)
}

// DeleteStamp : DELETE /stamps/:stampID
func DeleteStamp(c echo.Context) error {
	userID := c.Get("user").(*model.User).ID
	stampID := c.Param("stampID")

	stamp, err := model.GetStamp(stampID)
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}
	if stamp.CreatorID != userID {
		return echo.NewHTTPError(http.StatusForbidden)
	}

	if err = model.DeleteStamp(stampID); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	go notification.Send(events.StampDeleted, events.StampEvent{ID: stamp.ID})
	return c.NoContent(http.StatusNoContent)
}

// GetMessageStamps : GET /messages/:messageID/stamps
func GetMessageStamps(c echo.Context) error {
	userID := c.Get("user").(*model.User).ID
	messageID := c.Param("messageID")

	// Privateチャンネルの確認
	channel, err := model.GetChannelByMessageID(messageID)
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}
	if !channel.IsPublic {
		if ok, err := channel.Exists(userID); err != nil {
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		} else if !ok {
			return echo.NewHTTPError(http.StatusNotFound)
		}
	}

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

	// メッセージ存在の確認
	message, err := model.GetMessage(messageID)
	if err != nil {
		switch err {
		case model.ErrNotFound, model.ErrMessageAlreadyDeleted:
			return echo.NewHTTPError(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	// Privateチャンネルの確認
	channel, err := model.GetChannelByID(userID, message.ChannelID)
	if err != nil {
		switch err {
		case model.ErrNotFoundOrForbidden:
			return echo.NewHTTPError(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}
	if !channel.IsPublic {
		if ok, err := channel.Exists(userID); err != nil {
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		} else if !ok {
			return echo.NewHTTPError(http.StatusNotFound)
		}
	}

	count, err := model.AddStampToMessage(messageID, stampID, userID)
	if err != nil {
		if errSQL, ok := err.(*mysql.MySQLError); ok {
			if errSQL.Number == 1452 { //外部キー制約
				return echo.NewHTTPError(http.StatusBadRequest)
			}
		}
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	go notification.Send(events.MessageStamped, events.MessageStampEvent{
		ID:        messageID,
		ChannelID: message.ChannelID,
		StampID:   stampID,
		UserID:    userID,
		Count:     count,
	})
	return c.NoContent(http.StatusNoContent)
}

// DeleteMessageStamp : DELETE /messages/:messageID/stamps/:stampID
func DeleteMessageStamp(c echo.Context) error {
	userID := c.Get("user").(*model.User).ID
	messageID := c.Param("messageID")
	stampID := c.Param("stampID")

	// メッセージ存在の確認
	message, err := model.GetMessage(messageID)
	if err != nil {
		switch err {
		case model.ErrNotFound, model.ErrMessageAlreadyDeleted:
			return echo.NewHTTPError(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	// Privateチャンネルの確認
	channel, err := model.GetChannelByID(userID, message.ChannelID)
	if err != nil {
		switch err {
		case model.ErrNotFoundOrForbidden:
			return echo.NewHTTPError(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}
	if !channel.IsPublic {
		if ok, err := channel.Exists(userID); err != nil {
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		} else if !ok {
			return echo.NewHTTPError(http.StatusNotFound)
		}
	}

	if err := model.RemoveStampFromMessage(messageID, stampID, userID); err != nil {
		if errSQL, ok := err.(*mysql.MySQLError); ok {
			if errSQL.Number == 1452 { //外部キー制約
				return echo.NewHTTPError(http.StatusBadRequest)
			}
		}
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	go notification.Send(events.MessageUnstamped, events.MessageStampEvent{
		ID:        messageID,
		ChannelID: message.ChannelID,
		StampID:   stampID,
		UserID:    userID,
	})
	return c.NoContent(http.StatusNoContent)
}
