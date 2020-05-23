package v1

import (
	"github.com/labstack/echo/v4"
	"net/http"
	"strconv"
	"testing"
)

func TestHandlers_GetPublicUserIcon(t *testing.T) {
	t.Parallel()
	env, _, _, _, _, testUser, _ := setupWithUsers(t, common5)

	t.Run("No name", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.GET("/api/1.0/public/icon/").
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("No user", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.GET("/api/1.0/public/icon/no+user").
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		_, require := assertAndRequire(t)

		meta, err := env.Repository.GetFileMeta(testUser.GetIconFileID())
		require.NoError(err)
		e := env.makeExp(t)
		e.GET("/api/1.0/public/icon/{username}", testUser.GetName()).
			Expect().
			Status(http.StatusOK).
			Header(echo.HeaderContentLength).
			Equal(strconv.FormatInt(meta.GetFileSize(), 10))
	})

	t.Run("Success With 304", func(t *testing.T) {
		t.Parallel()
		_, require := assertAndRequire(t)

		meta, err := env.Repository.GetFileMeta(testUser.GetIconFileID())
		require.NoError(err)

		e := env.makeExp(t)
		e.GET("/api/1.0/public/icon/{username}", testUser.GetName()).
			WithHeader("If-None-Match", strconv.Quote(meta.GetMD5Hash())).
			Expect().
			Status(http.StatusNotModified)
	})
}
