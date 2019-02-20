package router

import (
	"fmt"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/sessions"
	"net/http"
	"testing"

	"github.com/labstack/echo"
)

func TestHandlers_PostFile(t *testing.T) {
	t.Parallel()
	repo, server, _, _, session, _ := setup(t, common1)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.POST("/api/1.0/files").
			WithMultipart().
			WithFileBytes("file", "test.txt", []byte("aaa")).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Successful1", func(t *testing.T) {
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
}

func TestHandlers_GetFileByID(t *testing.T) {
	t.Parallel()
	repo, server, _, _, session, _ := setup(t, common1)

	file := mustMakeFile(t, repo, uuid.Nil)

	t.Run("NotLoggedIn", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/files/{fileID}", file.ID).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("Successful1", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/files/{fileID}", file.ID).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusOK).
			Body().
			Equal("test message")
	})

	t.Run("Successful2", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		res := e.GET("/api/1.0/files/{fileID}", file.ID).
			WithCookie(sessions.CookieName, session).
			WithQuery("dl", 1).
			Expect().
			Status(http.StatusOK)
		res.Header(echo.HeaderContentDisposition).Equal(fmt.Sprintf("attachment; filename=%s", file.Name))
		res.Header(headerCacheControl).Equal("private, max-age=31536000")
		res.Body().Equal("test message")
	})
}

func TestHandlers_DeleteFileByID(t *testing.T) {
	t.Parallel()
	repo, server, _, _, session, adminSession := setup(t, common1)

	file := mustMakeFile(t, repo, uuid.Nil)

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
		file := mustMakeFile(t, repo, uuid.Nil)
		e.DELETE("/api/1.0/files/{fileID}", file.ID).
			WithCookie(sessions.CookieName, session).
			Expect().
			Status(http.StatusForbidden)
	})
}

func TestHandlers_GetMetaDataByFileID(t *testing.T) {
	t.Parallel()
	repo, server, _, _, session, _ := setup(t, common1)

	file := mustMakeFile(t, repo, uuid.Nil)

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
