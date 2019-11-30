package extension

import (
	jsoniter "github.com/json-iterator/go"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/router/extension/herror"
	"go.uber.org/zap"
	"net/http"
)

// ErrorHandler カスタムエラーハンドラ
func ErrorHandler(logger *zap.Logger) echo.HTTPErrorHandler {
	return func(e error, c echo.Context) {
		var (
			code int
			body interface{}
		)

		switch err := e.(type) {
		case nil:
			return
		case *echo.HTTPError:
			if err.Internal != nil {
				if herr, ok := err.Internal.(*echo.HTTPError); ok {
					err = herr
				}
			}
			if m, ok := err.Message.(string); ok {
				body = echo.Map{"message": m}
			} else if e, ok := err.Message.(error); ok {
				body = echo.Map{"message": e.Error()}
			}

			code = err.Code
		case *herror.InternalError:
			logger.Error(err.Error(), append(err.Fields, zap.String("logging.googleapis.com/trace", GetTraceID(c)))...)
			code = http.StatusInternalServerError
			body = echo.Map{"message": http.StatusText(http.StatusInternalServerError)}
		default:
			logger.Error(err.Error(), zap.String("logging.googleapis.com/trace", GetTraceID(c)))
			code = http.StatusInternalServerError
			body = echo.Map{"message": http.StatusText(http.StatusInternalServerError)}
		}

		if !c.Response().Committed {
			if c.Request().Method == http.MethodHead {
				e = c.NoContent(code)
			} else {
				e = json(c, code, body, jsoniter.ConfigFastest)
			}
			if e != nil {
				logger.Warn("failed to send error response", zap.Error(e), zap.String("logging.googleapis.com/trace", GetTraceID(c)))
			}
		}
	}
}
