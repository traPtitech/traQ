package extension

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gavv/httpexpect/v2"
	"github.com/labstack/echo/v4"

	"github.com/traPtitech/traQ/router/consts"
)

func TestCheckPreconditions(t *testing.T) {
	t.Parallel()

	modTime := time.Now()
	eTag := `"xyz"`

	e := echo.New()
	e.Any("/", func(c echo.Context) error {
		c.Response().Header().Set(consts.HeaderETag, eTag)
		SetLastModified(c, modTime)
		if ok, err := CheckPreconditions(c, modTime); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err)
		} else if ok {
			return nil
		}
		return c.String(http.StatusOK, "OK")
	})
	server := httptest.NewServer(e)

	exp := func(t *testing.T, server *httptest.Server) *httpexpect.Expect {
		t.Helper()
		return httpexpect.WithConfig(httpexpect.Config{
			BaseURL:  server.URL,
			Reporter: httpexpect.NewAssertReporter(t),
			Printers: []httpexpect.Printer{
				httpexpect.NewCurlPrinter(t),
				httpexpect.NewDebugPrinter(t, true),
			},
			Client: &http.Client{
				Jar:     nil, // クッキーは保持しない
				Timeout: time.Second * 30,
				CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
					return http.ErrUseLastResponse // リダイレクトを自動処理しない
				},
			},
		})
	}

	t.Run("OK", func(t *testing.T) {
		t.Parallel()
		e := exp(t, server)
		e.GET("/").
			Expect().
			Status(http.StatusOK)
	})

	t.Run("NotModified (If-Modified-Since)", func(t *testing.T) {
		t.Parallel()
		e := exp(t, server)
		e.GET("/").
			WithHeader(consts.HeaderIfModifiedSince, time.Now().UTC().Format(http.TimeFormat)).
			Expect().
			Status(http.StatusNotModified)
	})

	t.Run("OK (If-Modified-Since)", func(t *testing.T) {
		t.Parallel()
		e := exp(t, server)
		e.GET("/").
			WithHeader(consts.HeaderIfModifiedSince, modTime.Add(-1*time.Second).UTC().Format(http.TimeFormat)).
			Expect().
			Status(http.StatusOK)
	})

	t.Run("OK (Bad If-Modified-Since)", func(t *testing.T) {
		t.Parallel()
		e := exp(t, server)
		e.GET("/").
			WithHeader(consts.HeaderIfModifiedSince, "aoiuiouoijo").
			Expect().
			Status(http.StatusOK)
	})

	t.Run("OK (Bad Method If-Modified-Since)", func(t *testing.T) {
		t.Parallel()
		e := exp(t, server)
		e.PUT("/").
			WithHeader(consts.HeaderIfModifiedSince, modTime.UTC().Format(http.TimeFormat)).
			Expect().
			Status(http.StatusOK)
	})

	t.Run("OK (If-Unmodified-Since)", func(t *testing.T) {
		t.Parallel()
		e := exp(t, server)
		e.GET("/").
			WithHeader(consts.HeaderIfUnmodifiedSince, time.Now().UTC().Format(http.TimeFormat)).
			Expect().
			Status(http.StatusOK)
	})

	t.Run("OK (Bad If-Unmodified-Since)", func(t *testing.T) {
		t.Parallel()
		e := exp(t, server)
		e.GET("/").
			WithHeader(consts.HeaderIfUnmodifiedSince, "aoiuiouoijo").
			Expect().
			Status(http.StatusOK)
	})

	t.Run("PreconditionFailed (If-Unmodified-Since)", func(t *testing.T) {
		t.Parallel()
		e := exp(t, server)
		e.GET("/").
			WithHeader(consts.HeaderIfUnmodifiedSince, modTime.Add(-1*time.Second).UTC().Format(http.TimeFormat)).
			Expect().
			Status(http.StatusPreconditionFailed)
	})

	t.Run("OK (If-None-Match)", func(t *testing.T) {
		t.Parallel()
		e := exp(t, server)
		e.GET("/").
			WithHeader(consts.HeaderIfNoneMatch, `"abc", W/"def", "`).
			Expect().
			Status(http.StatusOK)
	})

	t.Run("NotModified (If-None-Match)", func(t *testing.T) {
		t.Parallel()
		e := exp(t, server)
		e.GET("/").
			WithHeader(consts.HeaderIfNoneMatch, `"xyz"`).
			Expect().
			Status(http.StatusNotModified)
	})

	t.Run("NotModified (GET * If-None-Match)", func(t *testing.T) {
		t.Parallel()
		e := exp(t, server)
		e.GET("/").
			WithHeader(consts.HeaderIfNoneMatch, `*`).
			Expect().
			Status(http.StatusNotModified)
	})

	t.Run("PreconditionFailed (POST * If-None-Match)", func(t *testing.T) {
		t.Parallel()
		e := exp(t, server)
		e.POST("/").
			WithHeader(consts.HeaderIfNoneMatch, `*`).
			Expect().
			Status(http.StatusPreconditionFailed)
	})

	t.Run("OK (If-Match)", func(t *testing.T) {
		t.Parallel()
		e := exp(t, server)
		e.GET("/").
			WithHeader(consts.HeaderIfMatch, `W/"abc", "xyz"`).
			Expect().
			Status(http.StatusOK)
	})

	t.Run("OK (* If-Match)", func(t *testing.T) {
		t.Parallel()
		e := exp(t, server)
		e.GET("/").
			WithHeader(consts.HeaderIfMatch, `*`).
			Expect().
			Status(http.StatusOK)
	})

	t.Run("PreconditionFailed (If-Match)", func(t *testing.T) {
		t.Parallel()
		e := exp(t, server)
		e.GET("/").
			WithHeader(consts.HeaderIfMatch, `"abc", "`).
			Expect().
			Status(http.StatusPreconditionFailed)
	})

	t.Run("Bad ETag", func(t *testing.T) {
		t.Parallel()
		e := exp(t, server)
		e.GET("/").
			WithHeader(consts.HeaderIfMatch, `"abc", "a`).
			Expect().
			Status(http.StatusPreconditionFailed)
	})
}
