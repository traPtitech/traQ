package extension

import (
	vd "github.com/go-ozzo/ozzo-validation"
	"github.com/gofrs/uuid"
	jsoniter "github.com/json-iterator/go"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/router/extension/herror"
)

// CtxKey context.Context用のキータイプ
type CtxKey int

const (
	// CtxUserIDKey ユーザーUUIDキー
	CtxUserIDKey CtxKey = iota
)

// Context echo.Contextのカスタム
type Context struct {
	echo.Context
}

// JSON encoding/jsonをjsoniter.ConfigCompatibleWithStandardLibraryに置換
func (c *Context) JSON(code int, i interface{}) (err error) {
	if _, pretty := c.QueryParams()["pretty"]; pretty {
		return c.Context.JSON(code, i)
	}
	return json(c, code, i, jsoniter.ConfigFastest)
}

func json(c echo.Context, code int, i interface{}, cfg jsoniter.API) error {
	stream := cfg.BorrowStream(c.Response())
	defer cfg.ReturnStream(stream)

	c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
	c.Response().WriteHeader(code)
	stream.WriteVal(i)
	stream.WriteRaw("\n")
	return stream.Flush()
}

// Wrap カスタムコンテキストラッパー
func Wrap() echo.MiddlewareFunc {
	return func(n echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error { return n(&Context{Context: c}) }
	}
}

// GetRequestParamAsUUID 指定したリクエストパラメーターをUUIDとして取得します
func GetRequestParamAsUUID(c echo.Context, name string) uuid.UUID {
	return uuid.FromStringOrNil(c.Param(name))
}

// BindAndValidate 構造体iにFormDataまたはJsonをデシリアライズします
func BindAndValidate(c echo.Context, i interface{}) error {
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
