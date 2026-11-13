package extension

import (
	vd "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v5"

	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/router/utils"
	"github.com/traPtitech/traQ/service/channel"
	"github.com/traPtitech/traQ/utils/jsonutil"
)

// JSONSerializer jsonutilを用いるecho.JSONSerializerの実装。
// `?pretty` クエリが指定された場合はインデント付きで出力する。
type JSONSerializer struct{}

// Serialize iをJSONとしてレスポンスに書き込む。
// echoのContext.jsonがContent-Typeとステータスコードを設定するため、ここでは本体のみ書き込む。
func (JSONSerializer) Serialize(c *echo.Context, i any, indent string) error {
	if indent == "" {
		if _, pretty := c.QueryParams()["pretty"]; pretty {
			indent = "  "
		}
	}
	if indent != "" {
		b, err := jsonutil.MarshalIndent(i, indent)
		if err != nil {
			return err
		}
		_, err = c.Response().Write(append(b, '\n'))
		return err
	}
	return jsonutil.Encode(c.Response(), i)
}

// Deserialize リクエストボディのJSONをiにデシリアライズする。
func (JSONSerializer) Deserialize(c *echo.Context, i any) error {
	if err := jsonutil.Decode(c.Request().Body, i); err != nil {
		return echo.ErrBadRequest.Wrap(err)
	}
	return nil
}

// json codeでヘッダーを書き込んだ上でiを書き込む（エラーハンドラ用）。
func json(c *echo.Context, code int, i interface{}) error {
	c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	c.Response().WriteHeader(code)
	return jsonutil.Encode(c.Response(), i)
}

// Wrap repositoryとchannel.Managerをコンテキストにセットするミドルウェア
func Wrap(repo repository.Repository, cm channel.Manager) echo.MiddlewareFunc {
	return func(n echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			c.Set(consts.KeyRepo, repo)
			c.Set(consts.KeyChannelManager, cm)
			return n(c)
		}
	}
}

// GetRequestParamAsUUID 指定したリクエストパラメーターをUUIDとして取得します
func GetRequestParamAsUUID(c *echo.Context, name string) uuid.UUID {
	return uuid.FromStringOrNil(c.Param(name))
}

// BindAndValidate 構造体iにFormDataまたはJsonをデシリアライズします
func BindAndValidate(c *echo.Context, i interface{}) error {
	if err := c.Bind(i); err != nil {
		return err
	}
	if err := vd.ValidateWithContext(utils.NewRequestValidateContext(c), i); err != nil {
		if e, ok := err.(vd.InternalError); ok {
			return herror.InternalServerError(e.InternalError())
		}
		return herror.BadRequest(err)
	}
	return nil
}
