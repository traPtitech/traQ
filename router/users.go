package router

import (
	"fmt"
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo"
	"github.com/labstack/echo-contrib/session"
	"github.com/traPtitech/traQ/model"
)

type loginRequesBoty struct {
	Name string `json:"name"`
	Pass string `json:"pass"`
}

// PostLogin Post /login のハンドラ
func PostLogin(c echo.Context) error {
	requestBody := &loginRequesBoty{}
	c.Bind(requestBody)

	user := &model.User{
		Name: requestBody.Name,
	}
	ok, err := user.Authorization(requestBody.Pass)
	if !ok {
		return echo.NewHTTPError(http.StatusForbidden, err)
	}

	sess, err := session.Get("sessions", c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "an error occurrerd while getting session")
	}

	sess.Options = &sessions.Options{
		Path:   "/",
		MaxAge: 86400 * 14,
	}

	sess.Values["userID"] = user.ID
	fmt.Println(sess)
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
