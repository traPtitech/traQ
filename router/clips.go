package router

import (
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/event"
	"net/http"
	"time"

	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/model"
)

// GetClips GET /users/me/clips
func GetClips(c echo.Context) error {
	type clipMessageForResponse struct {
		FolderID  string              `json:"folderId"`
		ClipID    string              `json:"clipId"`
		ClippedAt time.Time           `json:"clippedAt"`
		Message   *MessageForResponse `json:"message"`
	}

	userID := getRequestUserID(c)

	// クリップ取得
	clips, err := model.GetClipMessagesByUser(userID)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	// 整形
	res := make([]*clipMessageForResponse, len(clips))
	for i, v := range clips {
		res[i] = &clipMessageForResponse{
			FolderID:  v.FolderID,
			ClipID:    v.ID,
			ClippedAt: v.CreatedAt,
			Message:   formatMessage(&v.Message),
		}
	}

	return c.JSON(http.StatusOK, res)
}

// PostClip POST /users/me/clips
func PostClip(c echo.Context) error {
	userID := getRequestUserID(c)

	// リクエスト検証
	req := struct {
		MessageID string `json:"messageId" validate:"uuid,required"`
		FolderID  string `json:"folderId"`
	}{}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	// メッセージの存在と可用性を確認
	if _, err := validateMessageID(uuid.FromStringOrNil(req.MessageID), userID); err != nil {
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
		if folder.GetUID() != userID {
			return echo.NewHTTPError(http.StatusBadRequest, "the folder is not found")
		}
	} else {
		// 指定されていない場合はデフォルトフォルダを探す
		folders, err := model.GetClipFolders(userID)
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
			folder, err := model.CreateClipFolder(userID, "Default")
			if err != nil {
				c.Logger().Error(err)
				return echo.NewHTTPError(http.StatusInternalServerError)
			}
			go event.Emit(event.ClipFolderCreated, &event.ClipEvent{ID: folder.GetID(), UserID: userID})
			req.FolderID = folder.ID
		}
	}

	// クリップ作成
	clip, err := model.CreateClip(uuid.Must(uuid.FromString(req.MessageID)), uuid.Must(uuid.FromString(req.FolderID)), userID)
	if err != nil {
		if isMySQLDuplicatedRecordErr(err) {
			return echo.NewHTTPError(http.StatusBadRequest, "already clipped")
		}
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	go event.Emit(event.ClipCreated, &event.ClipEvent{ID: clip.GetID(), UserID: clip.GetUID()})
	return c.JSON(http.StatusCreated, struct {
		ID string `json:"id"`
	}{clip.ID})
}

// GetClip GET /users/me/clips/:clipID
func GetClip(c echo.Context) error {
	clipID := getRequestParamAsUUID(c, paramClipID)

	// クリップ取得
	clip, err := getClip(c, clipID, true)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, formatMessage(&clip.Message))
}

// DeleteClip DELETE /users/me/clips/:clipID
func DeleteClip(c echo.Context) error {
	clipID := getRequestParamAsUUID(c, paramClipID)

	// クリップ取得
	clip, err := getClip(c, clipID, true)
	if err != nil {
		return err
	}

	// クリップ削除
	if err := model.DeleteClip(clipID); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	go event.Emit(event.ClipDeleted, &event.ClipEvent{ID: clipID, UserID: clip.GetUID()})
	return c.NoContent(http.StatusNoContent)
}

// GetClipsFolder GET /users/me/clips/:clipID/folder
func GetClipsFolder(c echo.Context) error {
	clipID := getRequestParamAsUUID(c, paramClipID)

	// クリップ取得
	clip, err := getClip(c, clipID, true)
	if err != nil {
		return err
	}

	// クリップのフォルダを取得
	folder, err := getClipFolder(c, clip.GetFID(), false)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, folder)
}

// PutClipsFolder PUT /users/me/clips/:clipID/folder
func PutClipsFolder(c echo.Context) error {
	userID := getRequestUserID(c)
	clipID := getRequestParamAsUUID(c, paramClipID)

	// リクエスト検証
	req := struct {
		FolderID string `json:"folderId" validate:"uuid,required"`
	}{}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	// クリップ取得
	clip, err := getClip(c, clipID, true)
	if err != nil {
		return err
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
	if folder.GetUID() != userID {
		return echo.NewHTTPError(http.StatusBadRequest, "the folder is not found")
	}

	// クリップを更新
	if err := model.ChangeClipFolder(clipID, folder.GetID()); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	go event.Emit(event.ClipMoved, &event.ClipEvent{ID: clipID, UserID: clip.GetUID()})
	return c.NoContent(http.StatusNoContent)
}

// GetClipFolders GET /users/me/clips/folders
func GetClipFolders(c echo.Context) error {
	userID := getRequestUserID(c)

	// フォルダ取得
	folders, err := model.GetClipFolders(userID)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, folders)
}

// PostClipFolder POST /users/me/clips/folders
func PostClipFolder(c echo.Context) error {
	userID := getRequestUserID(c)

	// リクエスト検証
	req := struct {
		Name string `json:"name" validate:"required,max=30"`
	}{}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	// フォルダ作成
	folder, err := model.CreateClipFolder(userID, req.Name)
	if err != nil {
		if isMySQLDuplicatedRecordErr(err) {
			// フォルダ名が重複
			return echo.NewHTTPError(http.StatusConflict, "the name is duplicated")
		}
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	go event.Emit(event.ClipFolderCreated, &event.ClipEvent{ID: folder.GetID(), UserID: folder.GetUID()})
	return c.JSON(http.StatusCreated, folder)
}

// GetClipFolder GET /users/me/clips/folders/:folderID
func GetClipFolder(c echo.Context) error {
	type clipMessageForResponse struct {
		ClipID    string              `json:"clipId"`
		ClippedAt time.Time           `json:"clippedAt"`
		Message   *MessageForResponse `json:"message"`
	}

	folderID := getRequestParamAsUUID(c, paramFolderID)

	// フォルダ取得
	folder, err := getClipFolder(c, folderID, true)
	if err != nil {
		return err
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
			ClipID:    v.ID,
			ClippedAt: v.CreatedAt,
			Message:   formatMessage(&v.Message),
		}
	}

	return c.JSON(http.StatusOK, res)
}

// PatchClipFolder PATCH /users/me/clips/folders/:folderID
func PatchClipFolder(c echo.Context) error {
	folderID := getRequestParamAsUUID(c, paramFolderID)

	// リクエスト検証
	req := struct {
		Name string `json:"name" validate:"required,max=30"`
	}{}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	// フォルダ取得
	folder, err := getClipFolder(c, folderID, true)
	if err != nil {
		return err
	}

	// フォルダ更新
	if err := model.UpdateClipFolderName(folderID, req.Name); err != nil {
		if isMySQLDuplicatedRecordErr(err) {
			// フォルダ名が重複
			return echo.NewHTTPError(http.StatusConflict, "the name is duplicated")
		}
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	go event.Emit(event.ClipFolderUpdated, &event.ClipEvent{ID: folderID, UserID: folder.GetUID()})
	return c.NoContent(http.StatusNoContent)
}

// DeleteClipFolder DELETE /users/me/clips/folders/:folderID
func DeleteClipFolder(c echo.Context) error {
	folderID := getRequestParamAsUUID(c, paramFolderID)

	// フォルダ取得
	folder, err := getClipFolder(c, folderID, true)
	if err != nil {
		return err
	}

	// フォルダ削除
	if err := model.DeleteClipFolder(folderID); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	go event.Emit(event.ClipFolderDeleted, &event.ClipEvent{ID: folderID, UserID: folder.GetUID()})
	return c.NoContent(http.StatusNoContent)
}

func getClip(c echo.Context, clipID uuid.UUID, restrict bool) (*model.Clip, error) {
	userID := getRequestUserID(c)
	// クリップ取得
	clip, err := model.GetClipMessage(clipID)
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return nil, echo.NewHTTPError(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return nil, echo.NewHTTPError(http.StatusInternalServerError)
		}
	}
	// クリップがリクエストユーザーのものかを確認
	if restrict && clip.GetUID() != userID {
		return nil, echo.NewHTTPError(http.StatusNotFound)
	}
	return clip, nil
}

func getClipFolder(c echo.Context, folderID uuid.UUID, restrict bool) (*model.ClipFolder, error) {
	userID := getRequestUserID(c)
	// フォルダ取得
	folder, err := model.GetClipFolder(folderID)
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return nil, echo.NewHTTPError(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return nil, echo.NewHTTPError(http.StatusInternalServerError)
		}
	}
	// フォルダがリクエストユーザーのものかを確認
	if restrict && folder.GetUID() != userID {
		return nil, echo.NewHTTPError(http.StatusNotFound)
	}
	return folder, nil
}
