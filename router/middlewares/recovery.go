package middlewares

import (
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/router/extension"
	"github.com/traPtitech/traQ/router/extension/herror"
	"go.uber.org/zap"
	"net"
	"os"
	"strings"
)

// Recovery Recoveryミドルウェア
func Recovery(logger *zap.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			defer func() {
				if r := recover(); r != nil {
					pe, ok := r.(error)
					if !ok {
						pe = fmt.Errorf("%v", r)
					}

					if ne, ok := pe.(*net.OpError); ok {
						if se, ok := ne.Err.(*os.SyscallError); ok {
							if strings.Contains(strings.ToLower(se.Error()), "broken pipe") || strings.Contains(strings.ToLower(se.Error()), "connection reset by peer") {
								logger.Warn(pe.Error(),
									zap.String("requestId", extension.GetRequestID(c)),
									zap.Error(pe),
								)
								err = nil
								return
							}
						}
					}

					err = herror.Panic(pe)
				}
			}()
			return next(c)
		}
	}
}
