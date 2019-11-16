package herror

import (
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/repository"
	"net/http"
)

func NotFound(err ...interface{}) error {
	return HttpError(http.StatusNotFound, err)
}

func BadRequest(err ...interface{}) error {
	return HttpError(http.StatusBadRequest, err)
}

func Forbidden(err ...interface{}) error {
	return HttpError(http.StatusForbidden, err)
}

func Conflict(err ...interface{}) error {
	return HttpError(http.StatusConflict, err)
}

func Unauthorized(err ...interface{}) error {
	return HttpError(http.StatusUnauthorized, err)
}

func HttpError(code int, err interface{}) error {
	switch v := err.(type) {
	case []interface{}:
		if len(v) > 0 {
			return HttpError(code, v[0])
		}
		return HttpError(code, nil)
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
