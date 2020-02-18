package v3

import (
	vd "github.com/go-ozzo/ozzo-validation"
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension"
	"github.com/traPtitech/traQ/router/extension/herror"
	"net/http"
	"strconv"
)

// NotImplemented 未実装API. 501 NotImplementedを返す
func NotImplemented(c echo.Context) error {
	return echo.NewHTTPError(http.StatusNotImplemented)
}

// bindAndValidate 構造体iにFormDataまたはJsonをデシリアライズします
func bindAndValidate(c echo.Context, i interface{}) error {
	if err := c.Bind(i); err != nil {
		return err
	}
	if err := vd.Validate(i); err != nil {
		if e, ok := err.(vd.InternalError); ok {
			return herror.InternalServerError(e.InternalError())
		}
		return herror.BadRequest(err)
	}
	return nil
}

// isTrue 文字列sが"1", "t", "T", "true", "TRUE", "True"の場合にtrueを返す
func isTrue(s string) (b bool) {
	b, _ = strconv.ParseBool(s)
	return
}

// getRequestUser リクエストしてきたユーザーの情報を取得
func getRequestUser(c echo.Context) *model.User {
	return c.Get(consts.KeyUser).(*model.User)
}

// getRequestUserID リクエストしてきたユーザーUUIDを取得
func getRequestUserID(c echo.Context) uuid.UUID {
	return getRequestUser(c).ID
}

// getParamUser URLの:userIDに対応するユーザー構造体を取得
func getParamUser(c echo.Context) *model.User {
	return c.Get(consts.KeyParamUser).(*model.User)
}

// getParamWebhook URLの:webhookIDに対応するWebhookを取得
func getParamWebhook(c echo.Context) model.Webhook {
	return c.Get(consts.KeyParamWebhook).(model.Webhook)
}

// getParamBot URLの:botIDに対応するBotを取得
func getParamBot(c echo.Context) *model.Bot {
	return c.Get(consts.KeyParamBot).(*model.Bot)
}

// getParamAsUUID URLのnameパラメータの文字列をuuid.UUIDとして取得
func getParamAsUUID(c echo.Context, name string) uuid.UUID {
	return extension.GetRequestParamAsUUID(c, name)
}

// serveUserIcon userのアイコン画像ファイルをレスポンスとして返す
func serveUserIcon(c echo.Context, repo repository.Repository, user *model.User) error {
	// ファイルメタ取得
	meta, err := repo.GetFileMeta(user.Icon)
	if err != nil {
		return herror.InternalServerError(err)
	}

	// ファイルオープン
	file, err := repo.GetFS().OpenFileByKey(meta.GetKey(), meta.Type)
	if err != nil {
		return herror.InternalServerError(err)
	}
	defer file.Close()

	// レスポンスヘッダ設定
	c.Response().Header().Set(echo.HeaderContentType, meta.Mime)
	c.Response().Header().Set(consts.HeaderETag, strconv.Quote(meta.Hash))

	// ファイル送信
	http.ServeContent(c.Response(), c.Request(), meta.Name, meta.CreatedAt, file)
	return nil
}
