package extension

import (
	"errors"
	"net"
	"net/http"

	jsonIter "github.com/json-iterator/go"
	"github.com/labstack/echo/v5"
	"go.uber.org/zap"

	"github.com/traPtitech/traQ/router/extension/herror"
)

// ErrorHandler カスタムエラーハンドラ
func ErrorHandler(logger *zap.Logger) echo.HTTPErrorHandler {
	return func(c *echo.Context, e error) {
		var (
			code int
			body interface{}
		)

		switch err := e.(type) {
		case nil:
			return
		case *herror.InternalError:
			logger.Error(err.Error(), append(err.Fields, zap.String("requestId", GetRequestID(c)))...)
			code = http.StatusInternalServerError
			body = map[string]any{"message": http.StatusText(http.StatusInternalServerError)}
		case *net.OpError:
			logger.Warn("network error", zap.Error(err), zap.String("requestId", GetRequestID(c)))
			code = http.StatusBadGateway
			body = map[string]any{"message": http.StatusText(http.StatusBadGateway)}
		default:
			var he *echo.HTTPError
			switch {
			case errors.As(e, &he):
				// Internalに更にHTTPErrorがラップされている場合はそちらを優先する
				if inner, ok := he.Unwrap().(*echo.HTTPError); ok {
					he = inner
				}
				code = he.Code
				msg := he.Message
				if msg == "" {
					msg = http.StatusText(he.Code)
				}
				body = map[string]any{"message": msg}
			case echo.StatusCode(e) != 0:
				// echoの定義済みエラー(*httpError)など、HTTPStatusCoderを実装するエラー
				code = echo.StatusCode(e)
				body = map[string]any{"message": http.StatusText(code)}
			default:
				logger.Error(e.Error(), zap.String("requestId", GetRequestID(c)))
				code = http.StatusInternalServerError
				body = map[string]any{"message": http.StatusText(http.StatusInternalServerError)}
			}
		}

		res, _ := echo.UnwrapResponse(c.Response())
		if res == nil || !res.Committed {
			if c.Request().Method == http.MethodHead {
				e = c.NoContent(code)
			} else {
				e = json(c, code, body, jsonIter.ConfigFastest)
			}
			if e != nil {
				logger.Warn("failed to send error response", zap.Error(e), zap.String("requestId", GetRequestID(c)))
			}
		}
	}
}
