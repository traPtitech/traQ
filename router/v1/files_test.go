package v1

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/gofrs/uuid"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/session"
	file2 "github.com/traPtitech/traQ/service/file"
	"github.com/traPtitech/traQ/utils/optional"

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
		CreatorID: optional.From(grantedUser.GetID()),
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
			IsEqual("test message")
	})

	t.Run("Success with dl param", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		res := e.GET("/api/1.0/files/{fileID}", file.GetID()).
			WithCookie(session.CookieName, s).
			WithQuery("dl", 1).
			Expect().
			Status(http.StatusOK)
		res.Header(echo.HeaderContentDisposition).IsEqual(fmt.Sprintf("attachment; filename*=UTF-8''%s", url.PathEscape(file.GetFileName())))
		res.Header(consts.HeaderCacheControl).IsEqual("private, max-age=31536000")
		res.Body().IsEqual("test message")
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
		res.HasContentType(iconFile.GetMIMEType())
		res.Header(consts.HeaderCacheFile).IsEqual("true")
		res.Header(consts.HeaderFileMetaType).IsEqual("icon")
	})

	t.Run("Success With secure file", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.GET("/api/1.0/files/{fileID}", secureFile.GetID()).
			WithCookie(session.CookieName, env.generateSession(t, grantedUser.GetID())).
			Expect().
			Status(http.StatusOK).
			Body().
			IsEqual(secureContent)
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

		obj.Value("fileId").String().IsEqual(file.GetID().String())
		obj.Value("name").String().IsEqual(file.GetFileName())
		obj.Value("mime").String().IsEqual(file.GetMIMEType())
		obj.Value("size").Number().IsEqual(file.GetFileSize())
		obj.Value("md5").String().IsEqual(file.GetMD5Hash())
		hasThumb, _ := file.GetThumbnail(model.ThumbnailTypeImage)
		obj.Value("hasThumb").Boolean().IsEqual(hasThumb)
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
		CreatorID: optional.From(grantedUser.GetID()),
		ACL:       file2.ACL{},
		Src:       strings.NewReader(secureContent),
	})
	require.NoError(err)
	iconFileID, err := file2.GenerateIconFile(env.FileManager, "test")
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

	t.Run("Bad Thumbnail Type", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.GET("/api/1.0/files/{fileID}/thumbnail", iconFileID).
			WithQuery("type", "bad").
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("Thumbnail Type Not Found", func(t *testing.T) {
		t.Parallel()
		e := env.makeExp(t)
		e.GET("/api/1.0/files/{fileID}/thumbnail", iconFileID).
			WithQuery("type", "waveform").
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("Success (type=image)", func(t *testing.T) {
		t.Parallel()

		e := env.makeExp(t)
		res := e.GET("/api/1.0/files/{fileID}/thumbnail", iconFileID).
			WithQuery("type", "image").
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusOK)
		res.Header(consts.HeaderCacheControl).IsEqual("private, max-age=31536000")
		res.HasContentType(consts.MimeImagePNG)
	})

	t.Run("Success", func(t *testing.T) {
		t.Parallel()

		e := env.makeExp(t)
		res := e.GET("/api/1.0/files/{fileID}/thumbnail", iconFileID).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusOK)
		res.Header(consts.HeaderCacheControl).IsEqual("private, max-age=31536000")
		res.HasContentType(consts.MimeImagePNG)
	})
}
