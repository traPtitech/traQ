package router

import (
	"fmt"
	"github.com/go-sql-driver/mysql"
	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/logging"
	"github.com/traPtitech/traQ/repository"
	"go.uber.org/zap"
	"net/http"
	"runtime"
	"runtime/debug"
)

func isMySQLDuplicatedRecordErr(err error) bool {
	merr, ok := err.(*mysql.MySQLError)
	if !ok {
		return false
	}
	return merr.Number == errMySQLDuplicatedRecord
}

func notFound(err ...interface{}) error {
	return httpError(http.StatusNotFound, err)
}

func badRequest(err ...interface{}) error {
	return httpError(http.StatusBadRequest, err)
}

func forbidden(err ...interface{}) error {
	return httpError(http.StatusForbidden, err)
}

func conflict(err ...interface{}) error {
	return httpError(http.StatusConflict, err)
}

func internalServerError(err error, logger *zap.Logger) error {
	if logger != nil {
		logger.Error(fmt.Sprintf("%s\n%s", err.Error(), debug.Stack()), logging.ErrorReport(runtime.Caller(1)), zap.Error(err))
	}
	return echo.NewHTTPError(http.StatusInternalServerError)
}

func httpError(code int, err interface{}) error {
	switch v := err.(type) {
	case []interface{}:
		if len(v) > 0 {
			return httpError(code, v[0])
		}
		return httpError(code, nil)
	case string:
		return echo.NewHTTPError(code, v)
	case *repository.ArgumentError:
		return echo.NewHTTPError(code, v.Error())
	case nil:
		return echo.NewHTTPError(code)
	default:
		return echo.NewHTTPError(code, v)
	}
}
