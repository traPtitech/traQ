package router

import (
	"net/http"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/labstack/echo"
	"github.com/stretchr/testify/assert"
	"github.com/traPtitech/traQ/model"
)

func TestGetUserInfo(t *testing.T) {
	e, cookie, mw := beforeTest(t)

	c, rec := getContext(e, t, cookie, nil)
	c.SetPath("/test")

	requestWithContext(t, mw(testGetUser), c)

	assert.EqualValues(t, http.StatusOK, rec.Code, rec.Body.String())
}

func testGetUser(c echo.Context) error {
	user := c.Get("user").(*model.User)

	type TestResponseUser struct {
		ID   string `json:"ID"`
		Name string `json:"Name"`
	}

	res := &TestResponseUser{user.ID, user.Name}
	return c.JSON(http.StatusOK, res)
}
