package extension

import (
	vd "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/gofrs/uuid"
	jsonIter "github.com/json-iterator/go"
	"github.com/labstack/echo/v4"

	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/router/utils"
	"github.com/traPtitech/traQ/service/channel"
)

// Context echo.Contextのカスタム
type Context struct {
	echo.Context
}

// JSON encoding/jsonをjsonIter.ConfigCompatibleWithStandardLibraryに置換
func (c *Context) JSON(code int, i interface{}) (err error) {
	if _, pretty := c.QueryParams()["pretty"]; pretty {
		return c.Context.JSON(code, i)
	}
	return json(c, code, i, jsonIter.ConfigFastest)
}

func json(c echo.Context, code int, i interface{}, cfg jsonIter.API) error {
	stream := cfg.BorrowStream(c.Response())
	defer cfg.ReturnStream(stream)

	c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	c.Response().WriteHeader(code)
	stream.WriteVal(i)
	stream.WriteRaw("\n")
	return stream.Flush()
}

// Wrap カスタムコンテキストラッパー
func Wrap(repo repository.Repository, cm channel.Manager) echo.MiddlewareFunc {
	return func(n echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set(consts.KeyRepo, repo)
			c.Set(consts.KeyChannelManager, cm)
			return n(&Context{Context: c})
		}
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
	if err := vd.ValidateWithContext(utils.NewRequestValidateContext(c), i); err != nil {
		if e, ok := err.(vd.InternalError); ok {
			return herror.InternalServerError(e.InternalError())
		}
		return herror.BadRequest(err)
	}
	return nil
}
