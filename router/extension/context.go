package extension

import (
	jsoniter "github.com/json-iterator/go"
	"github.com/labstack/echo"
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
	return c.json(code, i, jsoniter.ConfigCompatibleWithStandardLibrary)
}

func (c *Context) json(code int, i interface{}, cfg jsoniter.API) error {
	stream := cfg.BorrowStream(c.Response())
	defer cfg.ReturnStream(stream)

	c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
	c.Response().WriteHeader(code)
	stream.WriteVal(i)
	return stream.Error
}
