package router

import (
	"github.com/labstack/echo"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCheckPreconditions(t *testing.T) {
	t.Parallel()

	modTime := time.Now()
	eTag := `"xyz"`

	e := echo.New()
	e.Any("/", func(c echo.Context) error {
		c.Response().Header().Set(headerETag, eTag)
		setLastModified(c, modTime)
		if ok, err := checkPreconditions(c, modTime); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err)
		} else if ok {
			return nil
		}
		return c.String(http.StatusOK, "OK")
	})
	server := httptest.NewServer(e)

	t.Run("OK", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/").
			Expect().
			Status(http.StatusOK)
	})

	t.Run("NotModified (If-Modified-Since)", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/").
			WithHeader(headerIfModifiedSince, time.Now().UTC().Format(http.TimeFormat)).
			Expect().
			Status(http.StatusNotModified)
	})

	t.Run("OK (If-Modified-Since)", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/").
			WithHeader(headerIfModifiedSince, modTime.Add(-1*time.Second).UTC().Format(http.TimeFormat)).
			Expect().
			Status(http.StatusOK)
	})

	t.Run("OK (Bad If-Modified-Since)", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/").
			WithHeader(headerIfModifiedSince, "aoiuiouoijo").
			Expect().
			Status(http.StatusOK)
	})

	t.Run("OK (Bad Method If-Modified-Since)", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.PUT("/").
			WithHeader(headerIfModifiedSince, modTime.UTC().Format(http.TimeFormat)).
			Expect().
			Status(http.StatusOK)
	})

	t.Run("OK (If-Unmodified-Since)", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/").
			WithHeader(headerIfUnmodifiedSince, time.Now().UTC().Format(http.TimeFormat)).
			Expect().
			Status(http.StatusOK)
	})

	t.Run("OK (Bad If-Unmodified-Since)", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/").
			WithHeader(headerIfUnmodifiedSince, "aoiuiouoijo").
			Expect().
			Status(http.StatusOK)
	})

	t.Run("PreconditionFailed (If-Unmodified-Since)", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/").
			WithHeader(headerIfUnmodifiedSince, modTime.Add(-1*time.Second).UTC().Format(http.TimeFormat)).
			Expect().
			Status(http.StatusPreconditionFailed)
	})

	t.Run("OK (If-None-Match)", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/").
			WithHeader(headerIfNoneMatch, `"abc", W/"def", "`).
			Expect().
			Status(http.StatusOK)
	})

	t.Run("NotModified (If-None-Match)", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/").
			WithHeader(headerIfNoneMatch, `"xyz"`).
			Expect().
			Status(http.StatusNotModified)
	})

	t.Run("NotModified (GET * If-None-Match)", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/").
			WithHeader(headerIfNoneMatch, `*`).
			Expect().
			Status(http.StatusNotModified)
	})

	t.Run("PreconditionFailed (POST * If-None-Match)", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.POST("/").
			WithHeader(headerIfNoneMatch, `*`).
			Expect().
			Status(http.StatusPreconditionFailed)
	})

	t.Run("OK (If-Match)", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/").
			WithHeader(headerIfMatch, `W/"abc", "xyz"`).
			Expect().
			Status(http.StatusOK)
	})

	t.Run("OK (* If-Match)", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/").
			WithHeader(headerIfMatch, `*`).
			Expect().
			Status(http.StatusOK)
	})

	t.Run("PreconditionFailed (If-Match)", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/").
			WithHeader(headerIfMatch, `"abc", "`).
			Expect().
			Status(http.StatusPreconditionFailed)
	})

	t.Run("Bad ETag", func(t *testing.T) {
		t.Parallel()
		e := makeExp(t, server)
		e.GET("/").
			WithHeader(headerIfMatch, `"abc", "a`).
			Expect().
			Status(http.StatusPreconditionFailed)
	})
}
