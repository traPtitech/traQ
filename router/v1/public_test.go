package v1

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/service/file"
	"github.com/traPtitech/traQ/service/rbac/role"
	"github.com/traPtitech/traQ/utils/random"
)

func TestHandlers_GetPublicUserIcon(t *testing.T) {
	t.Parallel()
	env, _, require, _, _ := setup(t, common2)

	fid, err := file.GenerateIconFile(env.FileManager, "test")
	require.NoError(err)

	testUser, err := env.Repository.CreateUser(repository.CreateUserArgs{
		Name:       random.AlphaNumeric(32),
		Role:       role.User,
		IconFileID: fid,
	})
	require.NoError(err)

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

		meta, err := env.FileManager.Get(testUser.GetIconFileID())
		require.NoError(err)
		e := env.makeExp(t)
		e.GET("/api/1.0/public/icon/{username}", testUser.GetName()).
			Expect().
			Status(http.StatusOK).
			Header(echo.HeaderContentLength).
			IsEqual(strconv.FormatInt(meta.GetFileSize(), 10))
	})

	t.Run("Success With 304", func(t *testing.T) {
		t.Parallel()
		_, require := assertAndRequire(t)

		meta, err := env.FileManager.Get(testUser.GetIconFileID())
		require.NoError(err)

		e := env.makeExp(t)
		e.GET("/api/1.0/public/icon/{username}", testUser.GetName()).
			WithHeader("If-None-Match", strconv.Quote(meta.GetMD5Hash())).
			Expect().
			Status(http.StatusNotModified)
	})
}
