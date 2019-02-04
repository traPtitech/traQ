package router

import (
	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/rbac/permission"
	"github.com/traPtitech/traQ/utils/validator"
	"net/http"
)

// GetStamps GET /stamps
func GetStamps(c echo.Context) error {
	stamps, err := model.GetAllStamps()
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, stamps)
}

// PostStamp POST /stamps
func PostStamp(c echo.Context) error {
	userID := getRequestUserID(c)

	// name確認
	name := c.FormValue("name")
	if !validator.NameRegex.MatchString(name) {
		return echo.NewHTTPError(http.StatusBadRequest, "name must be 1-32 characters of a-zA-Z0-9_-")
	}

	// スタンプ名の重複を確認
	if dup, err := model.IsStampNameDuplicate(name); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	} else if dup {
		return echo.NewHTTPError(http.StatusConflict, "this name has already been used")
	}

	// file確認
	uploadedFile, err := c.FormFile("file")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	// file処理
	fileID, err := processMultipartFormStampUpload(c, uploadedFile)
	if err != nil {
		return err
	}

	// スタンプ作成
	s, err := model.CreateStamp(name, fileID.String(), userID.String())
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	go event.Emit(event.StampCreated, &event.StampEvent{ID: s.GetID()})
	return c.NoContent(http.StatusCreated)
}

// GetStamp GET /stamps/:stampID
func GetStamp(c echo.Context) error {
	stampID := getRequestParamAsUUID(c, paramStampID)

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

// PatchStamp PATCH /stamps/:stampID
func PatchStamp(c echo.Context) error {
	user := getRequestUser(c)
	stampID := getRequestParamAsUUID(c, paramStampID)
	r := getRBAC(c)

	// スタンプの存在確認
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

	// ユーザー確認
	if stamp.CreatorID != user.ID && !r.IsGranted(user.GetUID(), user.Role, permission.EditStampCreatedByOthers) {
		return echo.NewHTTPError(http.StatusForbidden, "you are not permitted to edit stamp created by others")
	}

	data := model.Stamp{}
	// 名前変更
	name := c.FormValue("name")
	if len(name) > 0 {
		// 権限確認
		if !r.IsGranted(user.GetUID(), user.Role, permission.EditStampName) {
			return echo.NewHTTPError(http.StatusForbidden, "you are not permitted to change stamp name")
		}
		// 名前を検証
		if !validator.NameRegex.MatchString(name) {
			return echo.NewHTTPError(http.StatusBadRequest, "name must be 1-32 characters of a-zA-Z0-9_-")
		}
		// スタンプ名の重複を確認
		if dup, err := model.IsStampNameDuplicate(name); err != nil {
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		} else if dup {
			return echo.NewHTTPError(http.StatusConflict, "this name has already been used")
		}
		data.Name = name
	}

	// 画像変更
	uploadedFile, err := c.FormFile("file")
	if err == nil {
		fileID, err := processMultipartFormStampUpload(c, uploadedFile)
		if err != nil {
			return err
		}
		data.FileID = fileID.String()
	} else if err != http.ErrMissingFile {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	// 更新
	if err := model.UpdateStamp(stampID, data); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	go event.Emit(event.StampModified, &event.StampEvent{ID: stampID})
	return c.NoContent(http.StatusNoContent)
}

// DeleteStamp DELETE /stamps/:stampID
func DeleteStamp(c echo.Context) error {
	stampID := getRequestParamAsUUID(c, paramStampID)

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

	if err = model.DeleteStamp(stampID); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	go event.Emit(event.StampDeleted, &event.StampEvent{ID: stamp.GetID()})
	return c.NoContent(http.StatusNoContent)
}

// GetMessageStamps GET /messages/:messageID/stamps
func GetMessageStamps(c echo.Context) error {
	userID := getRequestUserID(c)
	messageID := getRequestParamAsUUID(c, paramMessageID)

	// メッセージ存在の確認
	message, err := model.GetMessageByID(messageID)
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}
	channelID := message.ChannelID

	// ユーザーからアクセス可能なチャンネルかどうか
	if ok, err := model.IsChannelAccessibleToUser(userID, channelID); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	} else if !ok {
		return echo.NewHTTPError(http.StatusNotFound)
	}

	stamps, err := model.GetMessageStamps(messageID)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, stamps)
}

// PostMessageStamp POST /messages/:messageID/stamps/:stampID
func PostMessageStamp(c echo.Context) error {
	userID := getRequestUserID(c)
	messageID := getRequestParamAsUUID(c, paramMessageID)
	stampID := getRequestParamAsUUID(c, paramStampID)

	// メッセージ存在の確認
	message, err := model.GetMessageByID(messageID)
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}
	channelID := message.ChannelID

	// ユーザーからアクセス可能なチャンネルかどうか
	if ok, err := model.IsChannelAccessibleToUser(userID, channelID); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	} else if !ok {
		return echo.NewHTTPError(http.StatusNotFound)
	}

	// スタンプの存在を確認
	if ok, err := model.StampExists(stampID); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	} else if !ok {
		return echo.NewHTTPError(http.StatusNotFound)
	}

	// スタンプをメッセージに押す
	ms, err := model.AddStampToMessage(messageID, stampID, userID)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	go event.Emit(event.MessageStamped, &event.MessageStampEvent{
		ID:        messageID,
		ChannelID: channelID,
		StampID:   stampID,
		UserID:    userID,
		Count:     ms.Count,
		CreatedAt: ms.CreatedAt,
	})
	return c.NoContent(http.StatusNoContent)
}

// DeleteMessageStamp DELETE /messages/:messageID/stamps/:stampID
func DeleteMessageStamp(c echo.Context) error {
	userID := getRequestUserID(c)
	messageID := getRequestParamAsUUID(c, paramMessageID)
	stampID := getRequestParamAsUUID(c, paramStampID)

	// メッセージ存在の確認
	message, err := model.GetMessageByID(messageID)
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}
	channelID := message.ChannelID

	// ユーザーからアクセス可能なチャンネルかどうか
	if ok, err := model.IsChannelAccessibleToUser(userID, channelID); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	} else if !ok {
		return echo.NewHTTPError(http.StatusNotFound)
	}

	// スタンプの存在を確認
	if ok, err := model.StampExists(stampID); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	} else if !ok {
		return echo.NewHTTPError(http.StatusNotFound)
	}

	// スタンプをメッセージから削除
	if err := model.RemoveStampFromMessage(messageID, stampID, userID); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	go event.Emit(event.MessageUnstamped, &event.MessageStampEvent{
		ID:        messageID,
		ChannelID: channelID,
		StampID:   stampID,
		UserID:    userID,
	})
	return c.NoContent(http.StatusNoContent)
}

// GetMyStampHistory GET /users/me/stamp-history
func GetMyStampHistory(c echo.Context) error {
	userID := getRequestUserID(c)

	h, err := model.GetUserStampHistory(userID)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, h)
}
