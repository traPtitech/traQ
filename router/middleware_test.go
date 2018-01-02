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

	// userの追加
	user := &model.User{
		Name:  "middlewareTest",
		Email: "middleware@trap.jp",
		Icon:  "empty",
	}
	if err := user.SetPassword("test"); err != nil {
		t.Fatal(err)
	}
	if err := user.Create(); err != nil {
		t.Fatal(err)
	}

	sess, err := session.Get("sessions", c)
	if err != nil {
		t.Fatalf("an error occurred while getting session: %v", err)
	}

	sess.Values["userID"] = user.ID
	sess.Save(c.Request(), c.Response())

	requestWithContext(t, mw(GetUserInfo(testGetUser)), c)

	if rec.Code != http.StatusOK {
		t.Log(rec.Code)
		t.Fatal(rec.Body.String())
	}
}

func testGetUser(c echo.Context) error {
	user := c.Get("userID").(model.User)

	type TestResponseUser struct {
		ID   string `json:"ID"`
		Name string `json:"Name"`
	}

	res := &TestResponseUser{user.ID, user.Name}
	return c.JSON(http.StatusOK, res)
}
