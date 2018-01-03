package router

import (
	"net/http"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/labstack/echo"
	"github.com/labstack/echo-contrib/session"
	"github.com/traPtitech/traQ/model"
)

func TestGetUserInfo(t *testing.T) {
	e, cookie, mw := beforeTest(t)

	c, rec := getContext(e, t, cookie, nil)
	c.SetPath("/test")

	requestWithContext(t, mw(prepareUser(GetUserInfo(testGetUser))), c)

	if rec.Code != http.StatusOK {
		t.Log(rec.Code)
		t.Fatal(rec.Body.String())
	}
}

// userデータをセッションに保存する
func prepareUser(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		user := &model.User{
			Name:  "middlewareTest",
			Email: "middleware@trap.jp",
			Icon:  "empty",
		}
		if err := user.SetPassword("test"); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "an error occurred while setting password: %v", err)
		}
		if err := user.Create(); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "an error occurred while creating user: %v", err)
		}

		sess, err := session.Get("sessions", c)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "an error occurred while getting session: %v", err)
		}

		sess.Values["userID"] = user.ID
		sess.Save(c.Request(), c.Response())
		return next(c)
	}
}

func testGetUser(c echo.Context) error {
	user := c.Get("userID").(*model.User)

	type TestResponseUser struct {
		ID   string `json:"ID"`
		Name string `json:"Name"`
	}

	res := &TestResponseUser{user.ID, user.Name}
	return c.JSON(http.StatusOK, res)
}
