package v3

import (
	"github.com/stretchr/testify/require"
	"github.com/traPtitech/traQ/router/session"
	file2 "github.com/traPtitech/traQ/service/file"
	"net/http"
	"testing"
)

func TestHandlers_GetFileMeta(t *testing.T) {
	t.Parallel()
	path := "/api/v3/files/{fileID}/meta"
	env := Setup(t, common)
	s := env.S(t, env.CreateUser(t, rand).GetID())
	file := env.MakeFile(t)

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()

		e := env.R(t)
		e.GET(path, file.GetID()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		e := env.R(t)
		obj := e.GET(path, file.GetID()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusOK).
			JSON().
			Object()

		obj.Value("id").String().Equal(file.GetID().String())
		obj.Value("name").String().Equal(file.GetFileName())
		obj.Value("mime").String().Equal(file.GetMIMEType())
		obj.Value("size").Number().Equal(file.GetFileSize())
		obj.Value("md5").String().Equal(file.GetMD5Hash())
		obj.Value("thumbnails").Array().Length().Equal(0)
	})

	t.Run("success with image thumbnail", func(t *testing.T) {
		t.Parallel()

		iconFileID, err := file2.GenerateIconFile(env.FM, "test")
		require.NoError(t, err)

		e := env.R(t)
		obj := e.GET(path, iconFileID).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusOK).
			JSON().
			Object()

		obj.Value("id").String().Equal(iconFileID.String())
		thumbnails := obj.Value("thumbnails").Array()
		thumbnails.Length().Equal(1)
		thumbnail := thumbnails.First().Object()
		thumbnail.Value("type").Equal("image")
		thumbnail.Value("mime").Equal("image/png")
		thumbnail.Value("width").NotNull().NotEqual(0)
		thumbnail.Value("height").NotNull().NotEqual(0)
	})
}
