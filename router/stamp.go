package router

import (
	"bytes"
	"context"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/external/imagemagick"
	"github.com/traPtitech/traQ/rbac"
	"github.com/traPtitech/traQ/rbac/permission"
	"github.com/traPtitech/traQ/utils/thumb"
	"github.com/traPtitech/traQ/utils/validator"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/model"
)

const (
	stampMaxWidth  = 128
	stampMaxHeight = 128
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

	// name確認
	name := c.FormValue("name")
	if !validator.NameRegex.MatchString(name) {
		return echo.NewHTTPError(http.StatusBadRequest, "name must be 1-32 characters of a-zA-Z0-9_-")
	}

	// file確認
	uploadedFile, err := c.FormFile("file")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	// ファイルサイズ制限1MB
	if uploadedFile.Size > 1024*1024 {
		return echo.NewHTTPError(http.StatusBadRequest, "too big image file")
	}

	// ファイルタイプ確認・必要があればリサイズ
	b := &bytes.Buffer{}
	src, err := uploadedFile.Open()
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}
	defer src.Close()
	switch uploadedFile.Header.Get(echo.HeaderContentType) {
	case "image/png":
		img, err := png.Decode(src)
		if err != nil {
			// 不正なpngである
			return echo.NewHTTPError(http.StatusBadRequest, "bad png file")
		}
		if img.Bounds().Size().X > stampMaxWidth || img.Bounds().Size().Y > stampMaxHeight {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) //10秒以内に終わらないファイルは無効
			defer cancel()
			img, err = thumb.Resize(ctx, img, stampMaxWidth, stampMaxHeight)
			if err != nil {
				switch err {
				case context.DeadlineExceeded:
					// リサイズタイムアウト
					return echo.NewHTTPError(http.StatusBadRequest, "bad png file (resize timeout)")
				default:
					// 予期しないエラー
					c.Logger().Error(err)
					return echo.NewHTTPError(http.StatusInternalServerError)
				}
			}
		}

		// bytesに戻す
		if b, err = thumb.EncodeToPNG(img); err != nil {
			// 予期しないエラー
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}

	case "image/jpeg":
		img, err := jpeg.Decode(src)
		if err != nil {
			// 不正なjpgである
			return echo.NewHTTPError(http.StatusBadRequest, "bad jpg file")
		}
		if img.Bounds().Size().X > stampMaxWidth || img.Bounds().Size().Y > stampMaxHeight {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) //10秒以内に終わらないファイルは無効
			defer cancel()
			img, err = thumb.Resize(ctx, img, stampMaxWidth, stampMaxHeight)
			if err != nil {
				switch err {
				case context.DeadlineExceeded:
					// リサイズタイムアウト
					return echo.NewHTTPError(http.StatusBadRequest, "bad jpg file (resize timeout)")
				default:
					// 予期しないエラー
					c.Logger().Error(err)
					return echo.NewHTTPError(http.StatusInternalServerError)
				}
			}
		}

		// PNGに変換
		if b, err = thumb.EncodeToPNG(img); err != nil {
			// 予期しないエラー
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}

	case "image/gif":
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) //10秒以内に終わらないファイルは無効
		defer cancel()
		b, err = imagemagick.ResizeAnimationGIF(ctx, src, stampMaxWidth, stampMaxHeight, false)
		if err != nil {
			switch err {
			case imagemagick.ErrUnavailable:
				// gifは一時的にサポートされていない
				return echo.NewHTTPError(http.StatusBadRequest, "gif file is temporarily unsupported")
			case imagemagick.ErrUnsupportedType:
				// 不正なgifである
				return echo.NewHTTPError(http.StatusBadRequest, "bad gif file")
			case context.DeadlineExceeded:
				// リサイズタイムアウト
				return echo.NewHTTPError(http.StatusBadRequest, "bad gif file (resize timeout)")
			default:
				// 予期しないエラー
				c.Logger().Error(err)
				return echo.NewHTTPError(http.StatusInternalServerError)
			}
		}

	case "image/svg+xml":
		// TODO svgバリデーション
		io.Copy(b, src)

	default:
		return echo.NewHTTPError(http.StatusBadRequest, "invalid image file")
	}

	// スタンプ画像保存
	file := &model.File{
		Name:      uploadedFile.Filename,
		Size:      int64(b.Len()),
		CreatorID: userID,
	}
	if err := file.Create(b); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	// スタンプ作成
	s, err := model.CreateStamp(name, file.ID, userID)
	if err != nil {
		switch err {
		case model.ErrStampInvalidName: //起こらないはず
			return echo.NewHTTPError(http.StatusBadRequest, err)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	go event.Emit(event.StampCreated, event.StampEvent{ID: uuid.Must(uuid.FromString(s.ID))})
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
	user := c.Get("user").(*model.User)
	r := c.Get("rbac").(*rbac.RBAC)
	stampID := c.Param("stampID")

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
		stamp.Name = name
	}

	// 画像変更
	uploadedFile, err := c.FormFile("file")
	if err == nil {
		// ファイルサイズ制限1MB
		if uploadedFile.Size > 1024*1024 {
			return echo.NewHTTPError(http.StatusBadRequest, "too big image file")
		}

		// ファイルタイプ確認・必要があればリサイズ
		b := &bytes.Buffer{}
		src, err := uploadedFile.Open()
		if err != nil {
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
		defer src.Close()
		switch uploadedFile.Header.Get(echo.HeaderContentType) {
		case "image/png":
			img, err := png.Decode(src)
			if err != nil {
				// 不正なpngである
				return echo.NewHTTPError(http.StatusBadRequest, "bad png file")
			}
			if img.Bounds().Size().X > stampMaxWidth || img.Bounds().Size().Y > stampMaxHeight {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) //10秒以内に終わらないファイルは無効
				defer cancel()
				img, err = thumb.Resize(ctx, img, stampMaxWidth, stampMaxHeight)
				if err != nil {
					switch err {
					case context.DeadlineExceeded:
						// リサイズタイムアウト
						return echo.NewHTTPError(http.StatusBadRequest, "bad png file (resize timeout)")
					default:
						// 予期しないエラー
						c.Logger().Error(err)
						return echo.NewHTTPError(http.StatusInternalServerError)
					}
				}
			}

			// bytesに戻す
			if b, err = thumb.EncodeToPNG(img); err != nil {
				// 予期しないエラー
				c.Logger().Error(err)
				return echo.NewHTTPError(http.StatusInternalServerError)
			}

		case "image/jpeg":
			img, err := jpeg.Decode(src)
			if err != nil {
				// 不正なjpgである
				return echo.NewHTTPError(http.StatusBadRequest, "bad jpg file")
			}
			if img.Bounds().Size().X > stampMaxWidth || img.Bounds().Size().Y > stampMaxHeight {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) //10秒以内に終わらないファイルは無効
				defer cancel()
				img, err = thumb.Resize(ctx, img, stampMaxWidth, stampMaxHeight)
				if err != nil {
					switch err {
					case context.DeadlineExceeded:
						// リサイズタイムアウト
						return echo.NewHTTPError(http.StatusBadRequest, "bad jpg file (resize timeout)")
					default:
						// 予期しないエラー
						c.Logger().Error(err)
						return echo.NewHTTPError(http.StatusInternalServerError)
					}
				}
			}

			// PNGに変換
			if b, err = thumb.EncodeToPNG(img); err != nil {
				// 予期しないエラー
				c.Logger().Error(err)
				return echo.NewHTTPError(http.StatusInternalServerError)
			}

		case "image/gif":
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) //10秒以内に終わらないファイルは無効
			defer cancel()
			b, err = imagemagick.ResizeAnimationGIF(ctx, src, stampMaxWidth, stampMaxHeight, false)
			if err != nil {
				switch err {
				case imagemagick.ErrUnavailable:
					// gifは一時的にサポートされていない
					return echo.NewHTTPError(http.StatusBadRequest, "gif file is temporarily unsupported")
				case imagemagick.ErrUnsupportedType:
					// 不正なgifである
					return echo.NewHTTPError(http.StatusBadRequest, "bad gif file")
				case context.DeadlineExceeded:
					// リサイズタイムアウト
					return echo.NewHTTPError(http.StatusBadRequest, "bad gif file (resize timeout)")
				default:
					// 予期しないエラー
					c.Logger().Error(err)
					return echo.NewHTTPError(http.StatusInternalServerError)
				}
			}

		case "image/svg+xml":
			// TODO svgバリデーション
			io.Copy(b, src)

		default:
			return echo.NewHTTPError(http.StatusBadRequest, "invalid image file")
		}

		// スタンプ画像保存
		file := &model.File{
			Name:      uploadedFile.Filename,
			Size:      int64(b.Len()),
			CreatorID: user.ID,
		}
		if err := file.Create(b); err != nil {
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}

		stamp.FileID = file.ID
	} else if err != http.ErrMissingFile {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	// 更新
	if err := stamp.Update(); err != nil {
		switch err {
		case model.ErrStampInvalidName: //起こらないはず
			return echo.NewHTTPError(http.StatusBadRequest, err)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	go event.Emit(event.StampModified, event.StampEvent{ID: uuid.Must(uuid.FromString(stamp.ID))})
	return c.NoContent(http.StatusNoContent)
}

// DeleteStamp : DELETE /stamps/:stampID
func DeleteStamp(c echo.Context) error {
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

	if err = model.DeleteStamp(stampID); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	go event.Emit(event.StampDeleted, event.StampEvent{ID: uuid.Must(uuid.FromString(stamp.ID))})
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
	user := c.Get("user").(*model.User)
	messageID := c.Param("messageID")
	stampID := c.Param("stampID")

	// メッセージ存在の確認
	message, err := model.GetMessageByID(messageID)
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
	channel, err := model.GetChannelByID(user.ID, message.ChannelID)
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
		if ok, err := channel.Exists(user.ID); err != nil {
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		} else if !ok {
			return echo.NewHTTPError(http.StatusNotFound)
		}
	}

	ms, err := model.AddStampToMessage(messageID, stampID, user.ID)
	if err != nil {
		if errSQL, ok := err.(*mysql.MySQLError); ok {
			if errSQL.Number == 1452 { //外部キー制約
				return echo.NewHTTPError(http.StatusBadRequest)
			}
		}
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	go event.Emit(event.MessageStamped, event.MessageStampEvent{
		ID:        uuid.Must(uuid.FromString(messageID)),
		ChannelID: uuid.Must(uuid.FromString(message.ChannelID)),
		StampID:   uuid.Must(uuid.FromString(stampID)),
		UserID:    user.GetUID(),
		Count:     ms.Count,
		CreatedAt: ms.CreatedAt,
	})
	return c.NoContent(http.StatusNoContent)
}

// DeleteMessageStamp : DELETE /messages/:messageID/stamps/:stampID
func DeleteMessageStamp(c echo.Context) error {
	user := c.Get("user").(*model.User)
	messageID := c.Param("messageID")
	stampID := c.Param("stampID")

	// メッセージ存在の確認
	message, err := model.GetMessageByID(messageID)
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
	channel, err := model.GetChannelByID(user.ID, message.ChannelID)
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
		if ok, err := channel.Exists(user.ID); err != nil {
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		} else if !ok {
			return echo.NewHTTPError(http.StatusNotFound)
		}
	}

	if err := model.RemoveStampFromMessage(messageID, stampID, user.ID); err != nil {
		if errSQL, ok := err.(*mysql.MySQLError); ok {
			if errSQL.Number == 1452 { //外部キー制約
				return echo.NewHTTPError(http.StatusBadRequest)
			}
		}
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	go event.Emit(event.MessageUnstamped, event.MessageStampEvent{
		ID:        uuid.Must(uuid.FromString(messageID)),
		ChannelID: uuid.Must(uuid.FromString(message.ChannelID)),
		StampID:   uuid.Must(uuid.FromString(stampID)),
		UserID:    user.GetUID(),
	})
	return c.NoContent(http.StatusNoContent)
}

// GetMyStampHistory GET /users/me/stamp-history
func GetMyStampHistory(c echo.Context) error {
	userID := c.Get("user").(*model.User).ID

	h, err := model.GetUserStampHistory(userID)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, h)
}
