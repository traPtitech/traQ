package v1

import (
	"fmt"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/sessions"
	"net/http"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestHandlers_PostFile(t *testing.T) {
	t.Parallel()
	repo, server, _, _, session, _, user, _ := setupWithUsers(t, common1)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.POST("/api/1.0/files").
			WithMultipart().
			WithFileBytes("file", "test.txt", []byte("aaa")).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Bad Request (No file)", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.POST("/api/1.0/files").
			WithCookie(sessions.CookieName, session).
			WithMultipart().
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("Bad Request (Wrong ACL)", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.POST("/api/1.0/files").
			WithCookie(sessions.CookieName, session).
			WithMultipart().
			WithFileBytes("file", "test.txt", []byte("aaa")).
			WithFormField("acl_readable", "bad acl").
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("Bad Request (Unknown User ACL Entry)", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.POST("/api/1.0/files").
			WithCookie(sessions.CookieName, session).
			WithMultipart().
			WithFileBytes("file", "test.txt", []byte("aaa")).
			WithFormField("acl_readable", uuid.Must(uuid.NewV4())).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("Success with No ACL", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		file := []byte("test file")
		obj := e.POST("/api/1.0/files").
			WithCookie(sessions.CookieName, session).
			WithMultipart().
			WithFileBytes("file", "test.txt", file).
			Expect().
			Status(http.StatusCreated).
			JSON().
			Object()

		obj.Value("fileId").String().NotEmpty()
		obj.Value("name").String().Equal("test.txt")
		obj.Value("size").Number().Equal(len(file))

		_, err := repo.GetFileMeta(uuid.FromStringOrNil(obj.Value("fileId").String().Raw()))
		require.NoError(t, err)
	})

	t.Run("Success with ACL", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		file := []byte("test file")
		obj := e.POST("/api/1.0/files").
			WithCookie(sessions.CookieName, session).
			WithMultipart().
			WithFileBytes("file", "test.txt", file).
			WithFormField("acl_readable", user.ID).
			Expect().
			Status(http.StatusCreated).
			JSON().
			Object()

		obj.Value("fileId").String().NotEmpty()
		obj.Value("name").String().Equal("test.txt")
		obj.Value("size").Number().Equal(len(file))

		f, err := repo.GetFileMeta(uuid.FromStringOrNil(obj.Value("fileId").String().Raw()))
		require.NoError(t, err)

		t.Run("granted user", func(t *testing.T) {
			t.Parallel()
			ok, err := repo.IsFileAccessible(f.ID, user.ID)
			require.NoError(t, err)
			assert.True(t, ok)
		})

		t.Run("not granted user", func(t *testing.T) {
			t.Parallel()
			user := mustMakeUser(t, repo, random)
			ok, err := repo.IsFileAccessible(f.ID, user.ID)
			require.NoError(t, err)
			assert.False(t, ok)
		})
	})
}

func TestHandlers_GetFileByID(t *testing.T) {
	t.Parallel()
	repo, server, _, require, session, _ := setup(t, common1)

	file := mustMakeFile(t, repo)
	grantedUser := mustMakeUser(t, repo, random)
	secureContent := "secure"
	secureFile, err := repo.SaveFileWithACL("secure", strings.NewReader(secureContent), int64(len(secureContent)), "text/plain", model.FileTypeUserFile, uuid.NullUUID{Valid: true, UUID: grantedUser.ID}, repository.ACL{})
	require.NoError(err)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/files/{fileID}", file.ID).
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
		e.GET("/api/1.0/files/{fileID}", secureFile.ID).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/files/{fileID}", file.ID).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusOK).
			Body().
			Equal("test message")
	})

	t.Run("Success with dl param", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		res := e.GET("/api/1.0/files/{fileID}", file.ID).
			WithCookie(sessions.CookieName, session).
			WithQuery("dl", 1).
			Expect().
			Status(http.StatusOK)
		res.Header(echo.HeaderContentDisposition).Equal(fmt.Sprintf("attachment; filename=%s", file.Name))
		res.Header(consts.HeaderCacheControl).Equal("private, max-age=31536000")
		res.Body().Equal("test message")
	})

	t.Run("Success with icon file", func(t *testing.T) {
		t.Parallel()
		iconFileID, err := repo.GenerateIconFile("test")
		require.NoError(err)
		iconFile, err := repo.GetFileMeta(iconFileID)
		require.NoError(err)

		e := makeExp(t, server)
		res := e.GET("/api/1.0/files/{fileID}", iconFile.ID).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusOK)
		res.ContentType(iconFile.Mime)
		res.Header(consts.HeaderCacheFile).Equal("true")
		res.Header(consts.HeaderFileMetaType).Equal("icon")
	})

	t.Run("Success With secure file", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/files/{fileID}", secureFile.ID).
			WithCookie(sessions.CookieName, generateSession(t, grantedUser.ID)).
			Expect().
			Status(http.StatusOK).
			Body().
			Equal(secureContent)
	})
}

func TestHandlers_DeleteFileByID(t *testing.T) {
	t.Parallel()
	repo, server, _, _, session, adminSession := setup(t, common1)

	file := mustMakeFile(t, repo)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.DELETE("/api/1.0/files/{fileID}", file.ID).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.DELETE("/api/1.0/files/{fileID}", file.ID).
			WithCookie(sessions.CookieName, adminSession).
			Expect().
			Status(http.StatusNoContent)

		_, err := repo.GetFileMeta(file.ID)
		require.Equal(t, repository.ErrNotFound, err)
	})

	t.Run("Failure1", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		file := mustMakeFile(t, repo)
		e.DELETE("/api/1.0/files/{fileID}", file.ID).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusForbidden)
	})
}

func TestHandlers_GetMetaDataByFileID(t *testing.T) {
	t.Parallel()
	repo, server, _, _, session, _ := setup(t, common1)

	file := mustMakeFile(t, repo)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/files/{fileID}/meta", file.ID).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		obj := e.GET("/api/1.0/files/{fileID}/meta", file.ID).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusOK).
			JSON().
			Object()

		obj.Value("fileId").String().Equal(file.ID.String())
		obj.Value("name").String().Equal(file.Name)
		obj.Value("mime").String().Equal(file.Mime)
		obj.Value("size").Number().Equal(file.Size)
		obj.Value("md5").String().Equal(file.Hash)
		obj.Value("hasThumb").Boolean().Equal(file.HasThumbnail)
	})
}

func TestHandlers_GetThumbnailByID(t *testing.T) {
	t.Parallel()
	repo, server, _, require, session, _ := setup(t, common1)

	file := mustMakeFile(t, repo)
	grantedUser := mustMakeUser(t, repo, random)
	secureContent := "secure"
	secureFile, err := repo.SaveFileWithACL("secure", strings.NewReader(secureContent), int64(len(secureContent)), "text/plain", model.FileTypeUserFile, uuid.NullUUID{Valid: true, UUID: grantedUser.ID}, repository.ACL{})
	require.NoError(err)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/files/{fileID}/thumbnail", file.ID).
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
		e.GET("/api/1.0/files/{fileID}/thumbnail", secureFile.ID).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("No Thumbnail", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/files/{fileID}/thumbnail", file.ID).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		iconFileID, err := repo.GenerateIconFile("test")
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
