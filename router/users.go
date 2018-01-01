package router

import (
	"fmt"
	"net/http"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo"
	"github.com/labstack/echo-contrib/session"
	"github.com/traPtitech/traQ/model"
)

type loginRequestBody struct {
	Name string `json:"name" form:"name"`
	Pass string `json:"pass" form:"pass"`
}

// PostLogin Post /login のハンドラ
func PostLogin(c echo.Context) error {
	requestBody := &loginRequestBody{}
	err := c.Bind(requestBody)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprint(err))
	}

	user := &model.User{
		Name: requestBody.Name,
	}
	err = user.Authorization(requestBody.Pass)
	if err != nil {
		return echo.NewHTTPError(http.StatusForbidden, fmt.Sprint(err))
	}

	sess, err := session.Get("sessions", c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("an error occurrerd while getting session: %v", err))
	}

	sess.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 14,
		HttpOnly: true,
	}

	sess.Values["userID"] = user.ID
	sess.Save(c.Request(), c.Response())
	return c.NoContent(http.StatusOK)
}

// PostLogout Post /logout のハンドラ
func PostLogout(c echo.Context) error {
	sess, err := session.Get("sessions", c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "an error occurrerd while getting session")
	}

	sess.Values["userID"] = nil
	sess.Save(c.Request(), c.Response())
	return c.NoContent(http.StatusOK)
}
