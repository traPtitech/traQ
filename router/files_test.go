package router

import (
	"fmt"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/sessions"
	"net/http"
	"testing"

	"github.com/labstack/echo"
)

func TestGroup_Files(t *testing.T) {
	_, require, session, adminSession := beforeTest(t)

	t.Run("TestPostFile", func(t *testing.T) {
		t.Parallel()

		t.Run("NotLoggedIn", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.POST("/api/1.0/files").
				WithMultipart().
				WithFileBytes("file", "test.txt", []byte("aaa")).
				Expect().
				Status(http.StatusForbidden)
		})

		t.Run("Successful1", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
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

			_, err := model.GetMetaFileDataByID(uuid.FromStringOrNil(obj.Value("fileId").String().Raw()))
			require.NoError(err)
		})
	})

	t.Run("TestGetFileByID", func(t *testing.T) {
		t.Parallel()

		file := mustMakeFile(t)

		t.Run("NotLoggedIn", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.GET("/api/1.0/files/{fileID}", file.ID).
				Expect().
				Status(http.StatusForbidden)
		})

		t.Run("Successful1", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.GET("/api/1.0/files/{fileID}", file.ID).
				WithCookie(sessions.CookieName, session).
				Expect().
				Status(http.StatusOK).
				Body().
				Equal("test message")
		})

		t.Run("Successful2", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			res := e.GET("/api/1.0/files/{fileID}", file.ID).
				WithCookie(sessions.CookieName, session).
				WithQuery("dl", 1).
				Expect().
				Status(http.StatusOK)
			res.Header(echo.HeaderContentDisposition).Equal(fmt.Sprintf("attachment; filename=%s", file.Name))
			res.Header(headerCacheControl).Equal("private, max-age=31536000")
			res.Body().Equal("test message")
		})
	})

	t.Run("TestDeleteFileByID", func(t *testing.T) {
		t.Parallel()

		file := mustMakeFile(t)

		t.Run("NotLoggedIn", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.DELETE("/api/1.0/files/{fileID}", file.ID).
				Expect().
				Status(http.StatusForbidden)
		})

		t.Run("Successful1", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.DELETE("/api/1.0/files/{fileID}", file.ID).
				WithCookie(sessions.CookieName, adminSession).
				Expect().
				Status(http.StatusNoContent)

			_, err := model.GetMetaFileDataByID(file.ID)
			require.Equal(model.ErrNotFound, err)
		})

		t.Run("Failure1", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			file := mustMakeFile(t)
			e.DELETE("/api/1.0/files/{fileID}", file.ID).
				WithCookie(sessions.CookieName, session).
				Expect().
				Status(http.StatusForbidden)
		})
	})

	t.Run("TestGetMetaDataByFileID", func(t *testing.T) {
		t.Parallel()

		file := mustMakeFile(t)

		t.Run("NotLoggedIn", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
			e.GET("/api/1.0/files/{fileID}/meta", file.ID).
				Expect().
				Status(http.StatusForbidden)
		})

		t.Run("Successful1", func(t *testing.T) {
			t.Parallel()
			e := makeExp(t)
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
	})
}
