package extension

import (
	stdjson "encoding/json"
	"io"

	vd "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/gofrs/uuid"
	jsonIter "github.com/json-iterator/go"
	"github.com/labstack/echo/v5"

	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/router/utils"
	"github.com/traPtitech/traQ/service/channel"
)

// JSONSerializer encoding/jsonの代わりにjsoniterを用いるecho.JSONSerializerの実装。
// `?pretty` クエリが指定された場合はインデント付きの標準エンコーダを使用する。
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
		enc := stdjson.NewEncoder(c.Response())
		enc.SetIndent("", indent)
		return enc.Encode(i)
	}
	return writeJSON(c.Response(), i, jsonIter.ConfigFastest)
}

// Deserialize リクエストボディのJSONをiにデシリアライズする。
func (JSONSerializer) Deserialize(c *echo.Context, i any) error {
	if err := stdjson.NewDecoder(c.Request().Body).Decode(i); err != nil {
		return echo.ErrBadRequest.Wrap(err)
	}
	return nil
}

// writeJSON iをjsoniterでwに書き込む（ヘッダー操作は行わない）。
func writeJSON(w io.Writer, i interface{}, cfg jsonIter.API) error {
	stream := cfg.BorrowStream(w)
	defer cfg.ReturnStream(stream)

	stream.WriteVal(i)
	stream.WriteRaw("\n")
	return stream.Flush()
}

// json codeでヘッダーを書き込んだ上でiをjsoniterで書き込む（エラーハンドラ用）。
func json(c *echo.Context, code int, i interface{}, cfg jsonIter.API) error {
	c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	c.Response().WriteHeader(code)
	return writeJSON(c.Response(), i, cfg)
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
