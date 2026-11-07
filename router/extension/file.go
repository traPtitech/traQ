package extension

import (
	"errors"

	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v5"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/service/file"
)

// GetAccessibleFile URLの:fileIDに対応するFileを取得し、userIDのユーザーがアクセス可能かを確認する
//
// v1とv3の両方から使う。
// ファイルが存在しない場合は404、アクセスできない場合は403を返す。
// スタンプ画像とアイコン画像は誰でもアクセスできる。
func GetAccessibleFile(c *echo.Context, fm file.Manager, userID uuid.UUID) (model.File, error) {
	ctx := c.Request().Context()

	id, err := uuid.FromString(c.Param(consts.ParamFileID))
	if err != nil || id == uuid.Nil {
		return nil, herror.NotFound()
	}

	f, err := fm.Get(ctx, id)
	if err != nil {
		if errors.Is(err, file.ErrNotFound) {
			return nil, herror.NotFound()
		}
		return nil, herror.InternalServerError(err)
	}

	if t := f.GetFileType(); t == model.FileTypeIcon || t == model.FileTypeStamp {
		return f, nil
	}

	ok, err := fm.Accessible(ctx, f.GetID(), userID)
	if err != nil {
		return nil, herror.InternalServerError(err)
	}
	if !ok {
		return nil, herror.Forbidden()
	}

	return f, nil
}
