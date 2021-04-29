package v1

import (
	"fmt"
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/session"
	file2 "github.com/traPtitech/traQ/service/file"
	"github.com/traPtitech/traQ/utils/optional"
	"net/http"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestHandlers_GetFileByID(t *testing.T) {
	t.Parallel()
	env, _, require, s, _ := setup(t, common1)

	file := env.mustMakeFile(t)
	grantedUser := env.mustMakeUser(t, rand)
	secureContent := "secure"
	secureFile, err := env.FileManager.Save(file2.SaveArgs{
		FileName:  "secure",
		FileSize:  int64(len(secureContent)),
		MimeType:  "text/plain",
		FileType:  model.FileTypeUserFile,
		CreatorID: optional.UUIDFrom(grantedUser.GetID()),
		ChannelID: optional.UUID{},
		ACL:       file2.ACL{},
		Src:       strings.NewReader(secureContent),
	})
	require.NoError(err)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.GET("/api/1.0/files/{fileID}", file.GetID()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Not Found", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.GET("/api/1.0/files/{fileID}", uuid.Must(uuid.NewV4())).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("Not Accessible", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.GET("/api/1.0/files/{fileID}", secureFile.GetID()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.GET("/api/1.0/files/{fileID}", file.GetID()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusOK).
			Body().
			Equal("test message")
	})

	t.Run("Success with dl param", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		res := e.GET("/api/1.0/files/{fileID}", file.GetID()).
			WithCookie(session.CookieName, s).
			WithQuery("dl", 1).
			Expect().
			Status(http.StatusOK)
		res.Header(echo.HeaderContentDisposition).Equal(fmt.Sprintf("attachment; filename=%s", file.GetFileName()))
		res.Header(consts.HeaderCacheControl).Equal("private, max-age=31536000")
		res.Body().Equal("test message")
	})

	t.Run("Success with icon file", func(t *testing.T) {
		t.Parallel()
		iconFileID, err := file2.GenerateIconFile(env.FileManager, "test")
		require.NoError(err)
		iconFile, err := env.FileManager.Get(iconFileID)
		require.NoError(err)

		e := env.makeExp(t)
		res := e.GET("/api/1.0/files/{fileID}", iconFile.GetID()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusOK)
		res.ContentType(iconFile.GetMIMEType())
		res.Header(consts.HeaderCacheFile).Equal("true")
		res.Header(consts.HeaderFileMetaType).Equal("icon")
	})

	t.Run("Success With secure file", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.GET("/api/1.0/files/{fileID}", secureFile.GetID()).
			WithCookie(session.CookieName, env.generateSession(t, grantedUser.GetID())).
			Expect().
			Status(http.StatusOK).
			Body().
			Equal(secureContent)
	})
}

func TestHandlers_GetMetaDataByFileID(t *testing.T) {
	t.Parallel()
	env, _, _, s, _ := setup(t, common1)

	file := env.mustMakeFile(t)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.GET("/api/1.0/files/{fileID}/meta", file.GetID()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		obj := e.GET("/api/1.0/files/{fileID}/meta", file.GetID()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusOK).
			JSON().
			Object()

		obj.Value("fileId").String().Equal(file.GetID().String())
		obj.Value("name").String().Equal(file.GetFileName())
		obj.Value("mime").String().Equal(file.GetMIMEType())
		obj.Value("size").Number().Equal(file.GetFileSize())
		obj.Value("md5").String().Equal(file.GetMD5Hash())
		hasThumb, _ := file.GetThumbnail(model.ThumbnailTypeImage)
		obj.Value("hasThumb").Boolean().Equal(hasThumb)
	})
}

func TestHandlers_GetThumbnailByID(t *testing.T) {
	t.Parallel()
	env, _, require, s, _ := setup(t, common1)

	file := env.mustMakeFile(t)
	grantedUser := env.mustMakeUser(t, rand)
	secureContent := "secure"
	secureFile, err := env.FileManager.Save(file2.SaveArgs{
		FileName:  "secure",
		FileSize:  int64(len(secureContent)),
		MimeType:  "text/plain",
		FileType:  model.FileTypeUserFile,
		CreatorID: optional.UUIDFrom(grantedUser.GetID()),
		ChannelID: optional.UUID{},
		ACL:       file2.ACL{},
		Src:       strings.NewReader(secureContent),
	})
	require.NoError(err)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.GET("/api/1.0/files/{fileID}/thumbnail", file.GetID()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Not Found", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.GET("/api/1.0/files/{fileID}/thumbnail", uuid.Must(uuid.NewV4())).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("Not Accessible", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.GET("/api/1.0/files/{fileID}/thumbnail", secureFile.GetID()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("No Thumbnail", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.GET("/api/1.0/files/{fileID}/thumbnail", file.GetID()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		iconFileID, err := file2.GenerateIconFile(env.FileManager, "test")
		require.NoError(err)

		e := env.makeExp(t)
		res := e.GET("/api/1.0/files/{fileID}/thumbnail", iconFileID).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusOK)
		res.Header(consts.HeaderCacheControl).Equal("private, max-age=31536000")
		res.ContentType(consts.MimeImagePNG)
	})
}
