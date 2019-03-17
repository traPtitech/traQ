package router

import (
	"github.com/labstack/echo"
	"net/http"
	"net/textproto"
	"strings"
	"time"
)

func scanETag(s string) (etag string, remain string) {
	s = textproto.TrimString(s)
	start := 0
	if strings.HasPrefix(s, weakPrefix) {
		start = 2
	}
	if len(s[start:]) < 2 || s[start] != '"' {
		return "", ""
	}
	for i := start + 1; i < len(s); i++ {
		c := s[i]
		if c == '"' {
			return s[:i+1], s[i+1:]
		}
	}
	return "", ""
}

func etagStrongMatch(a, b string) bool {
	return a == b && a != "" && a[0] == '"'
}

func etagWeakMatch(a, b string) bool {
	return strings.TrimPrefix(a, weakPrefix) == strings.TrimPrefix(b, weakPrefix)
}

type condResult int

const (
	weakPrefix = "W/"

	condNone condResult = iota
	condTrue
	condFalse
)

func checkIfMatch(c echo.Context) condResult {
	im := c.Request().Header.Get(headerIfMatch)
	if im == "" {
		return condNone
	}
	for {
		im = textproto.TrimString(im)
		if len(im) == 0 {
			break
		}
		if im[0] == ',' {
			im = im[1:]
			continue
		}
		if im[0] == '*' {
			return condTrue
		}
		etag, remain := scanETag(im)
		if etag == "" {
			break
		}
		if etagStrongMatch(etag, c.Response().Header().Get(headerETag)) {
			return condTrue
		}
		im = remain
	}

	return condFalse
}

func checkIfNoneMatch(c echo.Context) condResult {
	inm := c.Request().Header.Get(headerIfNoneMatch)
	if inm == "" {
		return condNone
	}
	buf := inm
	for {
		buf = textproto.TrimString(buf)
		if len(buf) == 0 {
			break
		}
		if buf[0] == ',' {
			buf = buf[1:]
		}
		if buf[0] == '*' {
			return condFalse
		}
		etag, remain := scanETag(buf)
		if etag == "" {
			break
		}
		if etagWeakMatch(etag, c.Response().Header().Get(headerETag)) {
			return condFalse
		}
		buf = remain
	}
	return condTrue
}

func checkIfModifiedSince(c echo.Context, modtime time.Time) condResult {
	if m := c.Request().Method; m != http.MethodGet && m != http.MethodHead {
		return condNone
	}
	ims := c.Request().Header.Get(headerIfModifiedSince)
	if ims == "" || isZeroTime(modtime) {
		return condNone
	}
	if t, err := http.ParseTime(ims); err == nil {
		if modtime.Before(t.Add(1 * time.Second)) {
			return condFalse
		}
		return condTrue
	}
	return condNone
}

func checkIfUnmodifiedSince(c echo.Context, modtime time.Time) condResult {
	ius := c.Request().Header.Get(headerIfUnmodifiedSince)
	if ius == "" || isZeroTime(modtime) {
		return condNone
	}
	if t, err := http.ParseTime(ius); err == nil {
		if modtime.Before(t.Add(1 * time.Second)) {
			return condTrue
		}
		return condFalse
	}
	return condNone
}

var unixEpochTime = time.Unix(0, 0)

func isZeroTime(t time.Time) bool {
	return t.IsZero() || t.Equal(unixEpochTime)
}

func setLastModified(c echo.Context, modtime time.Time) {
	if !isZeroTime(modtime) {
		c.Response().Header().Set(echo.HeaderLastModified, modtime.UTC().Format(http.TimeFormat))
	}
}

func writeNotModified(c echo.Context) error {
	h := c.Response().Header()
	delete(h, echo.HeaderContentType)
	delete(h, echo.HeaderContentLength)
	if h.Get(headerETag) != "" {
		delete(h, echo.HeaderLastModified)
	}
	return c.NoContent(http.StatusNotModified)
}

func checkPreconditions(c echo.Context, modtime time.Time) (done bool, err error) {
	setLastModified(c, modtime)
	ch := checkIfMatch(c)
	if ch == condNone {
		ch = checkIfUnmodifiedSince(c, modtime)
	}
	if ch == condFalse {
		return true, c.NoContent(http.StatusPreconditionFailed)
	}

	switch checkIfNoneMatch(c) {
	case condFalse:
		if m := c.Request().Method; m == http.MethodGet || m == http.MethodHead {
			return true, writeNotModified(c)
		}
		return true, c.NoContent(http.StatusPreconditionFailed)
	case condNone:
		if checkIfModifiedSince(c, modtime) == condFalse {
			return true, writeNotModified(c)
		}
	}

	return false, nil
}
