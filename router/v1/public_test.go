package v1

import (
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/router/consts"
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

		meta, err := repo.GetFileMeta(testUser.GetIconFileID())
		require.NoError(err)
		e := makeExp(t, server)
		e.GET("/api/1.0/public/icon/{username}", testUser.GetName()).
			Expect().
			Status(http.StatusOK).
			Header(echo.HeaderContentLength).
			Equal(strconv.FormatInt(meta.Size, 10))
	})

	t.Run("Success With 304", func(t *testing.T) {
		t.Parallel()
		_, require := assertAndRequire(t)

		meta, err := repo.GetFileMeta(testUser.GetIconFileID())
		require.NoError(err)

		e := makeExp(t, server)
		e.GET("/api/1.0/public/icon/{username}", testUser.GetName()).
			WithHeader("If-None-Match", strconv.Quote(meta.Hash)).
			Expect().
			Status(http.StatusNotModified)
	})
}

func TestHandlers_GetPublicEmojiJSON(t *testing.T) {
	t.Parallel()
	repo, server, _, _, _, _ := setup(t, s3)

	var stamps []interface{}
	for i := 0; i < 10; i++ {
		s := mustMakeStamp(t, repo, random, uuid.Nil)
		stamps = append(stamps, s.Name)
	}

	e := makeExp(t, server)
	res := e.GET("/api/1.0/public/emoji.json").
		Expect().
		Status(http.StatusOK)

	res.JSON().
		Object().
		Value("all").
		Array().
		ContainsOnly(stamps...)

	res.Header(echo.HeaderLastModified).
		NotEmpty()

	t.Run("304", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/public/emoji.json").
			WithHeader(consts.HeaderIfModifiedSince, res.Header(echo.HeaderLastModified).Raw()).
			Expect().
			Status(http.StatusNotModified)
	})

	t.Run("Return cache", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/public/emoji.json").
			Expect().
			Status(http.StatusOK)
	})
}

func TestHandlers_GetPublicEmojiCSS(t *testing.T) {
	t.Parallel()
	repo, server, _, _, _, _ := setup(t, s4)

	for i := 0; i < 10; i++ {
		mustMakeStamp(t, repo, random, uuid.Nil)
	}

	e := makeExp(t, server)
	res := e.GET("/api/1.0/public/emoji.css").
		Expect().
		Status(http.StatusOK)

	res.ContentType("text/css")
	res.Header(echo.HeaderLastModified).NotEmpty()

	t.Run("304", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/public/emoji.css").
			WithHeader(consts.HeaderIfModifiedSince, res.Header(echo.HeaderLastModified).Raw()).
			Expect().
			Status(http.StatusNotModified)
	})

	t.Run("Return cache", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/public/emoji.css").
			Expect().
			Status(http.StatusOK)
	})
}

func TestHandlers_GetPublicEmojiImage(t *testing.T) {
	t.Parallel()
	repo, server, _, _, _, _ := setup(t, common5)

	s := mustMakeStamp(t, repo, random, uuid.Nil)

	t.Run("Not Found", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/public/emoji/{stampID}", uuid.Must(uuid.NewV4())).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/api/1.0/public/emoji/{stampID}", s.ID).
			Expect().
			Status(http.StatusOK)
	})

	t.Run("Success With 304", func(t *testing.T) {
		t.Parallel()
		_, require := assertAndRequire(t)

		meta, err := repo.GetFileMeta(s.FileID)
		require.NoError(err)

		e := makeExp(t, server)
		e.GET("/api/1.0/public/emoji/{stampID}", s.ID).
			WithHeader("If-None-Match", strconv.Quote(meta.Hash)).
			Expect().
			Status(http.StatusNotModified)
	})
}
