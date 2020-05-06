package v1

import (
	"fmt"
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/sessions"
	"github.com/traPtitech/traQ/utils/optional"
	"net/http"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestHandlers_GetFileByID(t *testing.T) {
	t.Parallel()
	repo, server, _, require, session, _ := setup(t, common1)

	file := mustMakeFile(t, repo)
	grantedUser := mustMakeUser(t, repo, rand)
	secureContent := "secure"
	secureFile, err := repo.SaveFile(repository.SaveFileArgs{
		FileName:  "secure",
		FileSize:  int64(len(secureContent)),
		MimeType:  "text/plain",
		FileType:  model.FileTypeUserFile,
		CreatorID: optional.UUIDFrom(grantedUser.GetID()),
		ChannelID: optional.UUID{},
		ACL:       repository.ACL{},
		Src:       strings.NewReader(secureContent),
	})
	require.NoError(err)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/files/{fileID}", file.GetID()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Not Found", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/files/{fileID}", uuid.Must(uuid.NewV4())).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("Not Accessible", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/files/{fileID}", secureFile.GetID()).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/files/{fileID}", file.GetID()).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusOK).
			Body().
			Equal("test message")
	})

	t.Run("Success with dl param", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		res := e.GET("/api/1.0/files/{fileID}", file.GetID()).
			WithCookie(sessions.CookieName, session).
			WithQuery("dl", 1).
			Expect().
			Status(http.StatusOK)
		res.Header(echo.HeaderContentDisposition).Equal(fmt.Sprintf("attachment; filename=%s", file.GetFileName()))
		res.Header(consts.HeaderCacheControl).Equal("private, max-age=31536000")
		res.Body().Equal("test message")
	})

	t.Run("Success with icon file", func(t *testing.T) {
		t.Parallel()
		iconFileID, err := repository.GenerateIconFile(repo, "test")
		require.NoError(err)
		iconFile, err := repo.GetFileMeta(iconFileID)
		require.NoError(err)

		e := makeExp(t, server)
		res := e.GET("/api/1.0/files/{fileID}", iconFile.GetID()).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusOK)
		res.ContentType(iconFile.GetMIMEType())
		res.Header(consts.HeaderCacheFile).Equal("true")
		res.Header(consts.HeaderFileMetaType).Equal("icon")
	})

	t.Run("Success With secure file", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/files/{fileID}", secureFile.GetID()).
			WithCookie(sessions.CookieName, generateSession(t, grantedUser.GetID())).
			Expect().
			Status(http.StatusOK).
			Body().
			Equal(secureContent)
	})
}

func TestHandlers_GetMetaDataByFileID(t *testing.T) {
	t.Parallel()
	repo, server, _, _, session, _ := setup(t, common1)

	file := mustMakeFile(t, repo)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/files/{fileID}/meta", file.GetID()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		obj := e.GET("/api/1.0/files/{fileID}/meta", file.GetID()).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusOK).
			JSON().
			Object()

		obj.Value("fileId").String().Equal(file.GetID().String())
		obj.Value("name").String().Equal(file.GetFileName())
		obj.Value("mime").String().Equal(file.GetMIMEType())
		obj.Value("size").Number().Equal(file.GetFileSize())
		obj.Value("md5").String().Equal(file.GetMD5Hash())
		obj.Value("hasThumb").Boolean().Equal(file.HasThumbnail())
	})
}

func TestHandlers_GetThumbnailByID(t *testing.T) {
	t.Parallel()
	repo, server, _, require, session, _ := setup(t, common1)

	file := mustMakeFile(t, repo)
	grantedUser := mustMakeUser(t, repo, rand)
	secureContent := "secure"
	secureFile, err := repo.SaveFile(repository.SaveFileArgs{
		FileName:  "secure",
		FileSize:  int64(len(secureContent)),
		MimeType:  "text/plain",
		FileType:  model.FileTypeUserFile,
		CreatorID: optional.UUIDFrom(grantedUser.GetID()),
		ChannelID: optional.UUID{},
		ACL:       repository.ACL{},
		Src:       strings.NewReader(secureContent),
	})
	require.NoError(err)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/files/{fileID}/thumbnail", file.GetID()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Not Found", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/files/{fileID}/thumbnail", uuid.Must(uuid.NewV4())).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("Not Accessible", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/files/{fileID}/thumbnail", secureFile.GetID()).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("No Thumbnail", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/files/{fileID}/thumbnail", file.GetID()).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		iconFileID, err := repository.GenerateIconFile(repo, "test")
		require.NoError(err)

		e := makeExp(t, server)
		res := e.GET("/api/1.0/files/{fileID}/thumbnail", iconFileID).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusOK)
		res.Header(consts.HeaderCacheControl).Equal("private, max-age=31536000")
		res.ContentType(consts.MimeImagePNG)
	})
}
