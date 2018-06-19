package router

import (
	"github.com/go-sql-driver/mysql"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/event"
	"net/http"
	"time"

	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/model"
)

type clipFolderForResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// GetClips GET /users/me/clips
func GetClips(c echo.Context) error {
	type clipMessageForResponse struct {
		FolderID  string              `json:"folderId"`
		ClipID    string              `json:"clipId"`
		ClippedAt time.Time           `json:"clippedAt"`
		Message   *MessageForResponse `json:"message"`
	}

	user := c.Get("user").(*model.User)

	// クリップ取得
	clips, err := model.GetClipMessagesByUser(user.GetUID())
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	// 整形
	res := make([]*clipMessageForResponse, len(clips))
	for i, v := range clips {
		res[i] = &clipMessageForResponse{
			FolderID:  v.Clip.FolderID,
			ClipID:    v.Clip.ID,
			ClippedAt: v.Clip.CreatedAt,
			Message:   formatMessage(v.Message),
		}
	}

	return c.JSON(http.StatusOK, res)
}

// PostClip POST /users/me/clips
func PostClip(c echo.Context) error {
	user := c.Get("user").(*model.User)

	// リクエスト検証
	req := struct {
		MessageID string `json:"messageId" validate:"uuid,required"`
		FolderID  string `json:"folderId"`
	}{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}
	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	// メッセージの存在と可用性を確認
	if _, err := validateMessageID(req.MessageID, user.ID); err != nil {
		return err
	}

	if len(req.FolderID) > 0 {
		// 保存先フォルダが指定されてる場合はフォルダの確認
		folder, err := model.GetClipFolder(uuid.FromStringOrNil(req.FolderID))
		if err != nil {
			switch err {
			case model.ErrNotFound:
				return echo.NewHTTPError(http.StatusBadRequest, "the folder is not found")
			default:
				c.Logger().Error(err)
				return echo.NewHTTPError(http.StatusInternalServerError)
			}
		}
		// フォルダがリクエストユーザーのものかを確認
		if folder.UserID != user.ID {
			return echo.NewHTTPError(http.StatusBadRequest, "the folder is not found")
		}
	} else {
		// 指定されていない場合はデフォルトフォルダを探す
		folders, err := model.GetClipFolders(user.GetUID())
		if err != nil {
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
		for _, v := range folders {
			if v.Name == "Default" {
				req.FolderID = v.ID
				break
			}
		}
		if len(req.FolderID) == 0 {
			// 存在しなかったのでデフォルトフォルダを作る
			folder, err := model.CreateClipFolder(user.GetUID(), "Default")
			if err != nil {
				c.Logger().Error(err)
				return echo.NewHTTPError(http.StatusInternalServerError)
			}
			go event.Emit(event.ClipFolderCreated, &event.ClipEvent{ID: folder.GetID(), UserID: user.GetUID()})
			req.FolderID = folder.ID
		}
	}

	// クリップ作成
	clip, err := model.CreateClip(uuid.Must(uuid.FromString(req.MessageID)), uuid.Must(uuid.FromString(req.FolderID)), user.GetUID())
	if err != nil {
		switch e := err.(type) {
		case *mysql.MySQLError:
			if e.Number == errMySQLDuplicatedRecord {
				return echo.NewHTTPError(http.StatusBadRequest, "already clipped")
			}
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	go event.Emit(event.ClipCreated, &event.ClipEvent{ID: clip.GetID(), UserID: clip.GetUID()})
	return c.JSON(http.StatusCreated, struct {
		ID string `json:"id"`
	}{clip.ID})
}

// GetClip GET /users/me/clips/:clipID
func GetClip(c echo.Context) error {
	clipID := c.Param("clipID")
	user := c.Get("user").(*model.User)

	// クリップ取得
	clip, err := model.GetClipMessage(uuid.FromStringOrNil(clipID))
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}
	// クリップがリクエストユーザーのものかを確認
	if clip.Clip.UserID != user.ID {
		return echo.NewHTTPError(http.StatusNotFound)
	}

	return c.JSON(http.StatusOK, formatMessage(clip.Message))
}

// DeleteClip DELETE /users/me/clips/:clipID
func DeleteClip(c echo.Context) error {
	clipID := c.Param("clipID")
	user := c.Get("user").(*model.User)

	// クリップ取得
	clip, err := model.GetClipMessage(uuid.FromStringOrNil(clipID))
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}
	// クリップがリクエストユーザーのものかを確認
	if clip.Clip.UserID != user.ID {
		return echo.NewHTTPError(http.StatusNotFound)
	}

	// クリップ削除
	if err := model.DeleteClip(clip.Clip.GetID()); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	go event.Emit(event.ClipDeleted, &event.ClipEvent{ID: clip.Clip.GetID(), UserID: clip.Clip.GetUID()})
	return c.NoContent(http.StatusNoContent)
}

// GetClipsFolder GET /users/me/clips/:clipID/folder
func GetClipsFolder(c echo.Context) error {
	clipID := c.Param("clipID")
	user := c.Get("user").(*model.User)

	// クリップ取得
	clip, err := model.GetClipMessage(uuid.FromStringOrNil(clipID))
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}
	// クリップがリクエストユーザーのものかを確認
	if clip.Clip.UserID != user.ID {
		return echo.NewHTTPError(http.StatusNotFound)
	}

	// クリップのフォルダを取得
	folder, err := model.GetClipFolder(clip.GetFID())
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	return c.JSON(http.StatusOK, formatClipFolder(folder))
}

// PutClipsFolder PUT /users/me/clips/:clipID/folder
func PutClipsFolder(c echo.Context) error {
	clipID := c.Param("clipID")
	user := c.Get("user").(*model.User)

	// リクエスト検証
	req := struct {
		FolderID string `json:"folderId" validate:"uuid,required"`
	}{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}
	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	// クリップ取得
	clip, err := model.GetClipMessage(uuid.FromStringOrNil(clipID))
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}
	// クリップがリクエストユーザーのものかを確認
	if clip.Clip.UserID != user.ID {
		return echo.NewHTTPError(http.StatusNotFound)
	}

	// 変更先のクリップのフォルダを取得
	folder, err := model.GetClipFolder(uuid.FromStringOrNil(req.FolderID))
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusBadRequest, "the folder is not found")
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}
	// フォルダがリクエストユーザーのものかを確認
	if folder.UserID != user.ID {
		return echo.NewHTTPError(http.StatusBadRequest, "the folder is not found")
	}

	clip.Clip.FolderID = folder.ID

	// クリップを更新
	if err := model.UpdateClip(clip.Clip); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	go event.Emit(event.ClipMoved, &event.ClipEvent{ID: clip.Clip.GetID(), UserID: clip.Clip.GetUID()})
	return c.NoContent(http.StatusNoContent)
}

// GetClipFolders GET /users/me/clips/folders
func GetClipFolders(c echo.Context) error {
	user := c.Get("user").(*model.User)

	// フォルダ取得
	folders, err := model.GetClipFolders(user.GetUID())
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	// 整形
	res := make([]*clipFolderForResponse, len(folders))
	for i, v := range folders {
		res[i] = formatClipFolder(v)
	}

	return c.JSON(http.StatusOK, res)
}

// PostClipFolder POST /users/me/clips/folders
func PostClipFolder(c echo.Context) error {
	user := c.Get("user").(*model.User)

	// リクエスト検証
	req := struct {
		Name string `json:"name" validate:"required,max=30"`
	}{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}
	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	// フォルダ作成
	folder, err := model.CreateClipFolder(user.GetUID(), req.Name)
	if err != nil {
		switch e := err.(type) {
		case *mysql.MySQLError:
			if e.Number == errMySQLDuplicatedRecord {
				// フォルダ名が重複
				return echo.NewHTTPError(http.StatusConflict, "the name is duplicated")
			}
		}
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	go event.Emit(event.ClipFolderCreated, &event.ClipEvent{ID: folder.GetID(), UserID: folder.GetUID()})
	return c.JSON(http.StatusCreated, formatClipFolder(folder))
}

// GetClipFolder GET /users/me/clips/folders/:folderID
func GetClipFolder(c echo.Context) error {
	type clipMessageForResponse struct {
		ClipID    string              `json:"clipId"`
		ClippedAt time.Time           `json:"clippedAt"`
		Message   *MessageForResponse `json:"message"`
	}

	folderID := c.Param("folderID")
	user := c.Get("user").(*model.User)

	// フォルダ取得
	folder, err := model.GetClipFolder(uuid.FromStringOrNil(folderID))
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}
	// フォルダがリクエストユーザーのものかを確認
	if folder.UserID != user.ID {
		return echo.NewHTTPError(http.StatusNotFound)
	}

	// クリップ取得
	clips, err := model.GetClipMessages(folder.GetID())
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	// 整形
	res := make([]*clipMessageForResponse, len(clips))
	for i, v := range clips {
		res[i] = &clipMessageForResponse{
			ClipID:    v.Clip.ID,
			ClippedAt: v.Clip.CreatedAt,
			Message:   formatMessage(v.Message),
		}
	}

	return c.JSON(http.StatusOK, res)
}

// PatchClipFolder PATCH /users/me/clips/folders/:folderID
func PatchClipFolder(c echo.Context) error {
	folderID := c.Param("folderID")
	user := c.Get("user").(*model.User)

	// リクエスト検証
	req := struct {
		Name string `json:"name" validate:"required,max=30"`
	}{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}
	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	// フォルダ取得
	folder, err := model.GetClipFolder(uuid.FromStringOrNil(folderID))
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}
	// フォルダがリクエストユーザーのものかを確認
	if folder.UserID != user.ID {
		return echo.NewHTTPError(http.StatusNotFound)
	}

	folder.Name = req.Name

	// フォルダ更新
	if err := model.UpdateClipFolder(folder); err != nil {
		switch e := err.(type) {
		case *mysql.MySQLError:
			if e.Number == errMySQLDuplicatedRecord {
				// フォルダ名が重複
				return echo.NewHTTPError(http.StatusConflict, "the name is duplicated")
			}
		}
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	go event.Emit(event.ClipFolderUpdated, &event.ClipEvent{ID: folder.GetID(), UserID: folder.GetUID()})
	return c.NoContent(http.StatusNoContent)
}

// DeleteClipFolder DELETE /users/me/clips/folders/:folderID
func DeleteClipFolder(c echo.Context) error {
	folderID := c.Param("folderID")
	user := c.Get("user").(*model.User)

	// フォルダ取得
	folder, err := model.GetClipFolder(uuid.FromStringOrNil(folderID))
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}
	// フォルダがリクエストユーザーのものかを確認
	if folder.UserID != user.ID {
		return echo.NewHTTPError(http.StatusNotFound)
	}

	// フォルダ削除
	if err := model.DeleteClipFolder(folder.GetID()); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	go event.Emit(event.ClipFolderDeleted, &event.ClipEvent{ID: folder.GetID(), UserID: folder.GetUID()})
	return c.NoContent(http.StatusNoContent)
}

func formatClipFolder(raw *model.ClipFolder) *clipFolderForResponse {
	return &clipFolderForResponse{
		ID:   raw.ID,
		Name: raw.Name,
	}
}
