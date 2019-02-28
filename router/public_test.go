package router

import (
	"github.com/labstack/echo"
	"io/ioutil"
	"net/http"
	"strconv"
	"testing"
)

func TestHandlers_GetPublicUserIcon(t *testing.T) {
	t.Parallel()
	repo, server, _, _, _, _, testUser, _ := setupWithUsers(t, common5)

	t.Run("No name", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/public/icon/").
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("No user", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/public/icon/no+user").
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		_, require := assertAndRequire(t)

		_, src, err := repo.OpenFile(testUser.Icon)
		require.NoError(err)
		i, err := ioutil.ReadAll(src)
		require.NoError(err)

		e := makeExp(t, server)
		e.GET("/api/1.0/public/icon/{username}", testUser.Name).
			Expect().
			Status(http.StatusOK).
			Header(echo.HeaderContentLength).
			Equal(strconv.Itoa(len(i)))
	})

	t.Run("Success with thumbnail", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/public/icon/{username}", testUser.Name).
			WithQuery("thumb", "").
			Expect().
			Status(http.StatusOK)
	})
}
