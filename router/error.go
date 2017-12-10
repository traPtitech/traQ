package router

import (
	"github.com/labstack/echo"
)

type errorResponse struct {
	message string
}

func errorMessageResponse(c echo.Context, code int, message string) {
	res := errorResponse{}
	res.message = message
	c.JSON(code, res)
}
