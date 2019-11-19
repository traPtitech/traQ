package extension

import (
	jsoniter "github.com/json-iterator/go"
	"github.com/labstack/echo/v4"
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
	return c.json(code, i, jsoniter.ConfigFastest)
}

func (c *Context) json(code int, i interface{}, cfg jsoniter.API) error {
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

// CtxKey context.Context用のキータイプ
type CtxKey int

const (
	// CtxUserIDKey ユーザーUUIDキー
	CtxUserIDKey CtxKey = iota
)
