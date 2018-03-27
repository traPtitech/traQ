package oauth2

import (
	"encoding/json"
	"github.com/labstack/echo"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestHandler_TokenEndpointHandler_Failure1(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)
	e := echo.New()
	h := &Handler{Store: NewStoreMock()}

	f := url.Values{}
	f.Set("grant_type", "ああああ")

	req := httptest.NewRequest(echo.POST, "/", strings.NewReader(f.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if assert.NoError(h.TokenEndpointHandler(c)) {
		assert.Equal(http.StatusBadRequest, rec.Code)
		assert.Equal("no-store", rec.Header().Get("Cache-Control"))
		assert.Equal("no-cache", rec.Header().Get("Pragma"))

		res := errorResponse{}
		if assert.NoError(json.NewDecoder(rec.Body).Decode(&res)) {
			assert.Equal(errUnsupportedGrantType, res.ErrorType)
		}
	}
}

func TestHandler_TokenEndpointHandler_Failure2(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)
	e := echo.New()
	h := &Handler{Store: NewStoreMock()}

	req := httptest.NewRequest(echo.POST, "/", nil)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if assert.NoError(h.TokenEndpointHandler(c)) {
		assert.Equal(http.StatusBadRequest, rec.Code)
		assert.Equal("no-store", rec.Header().Get("Cache-Control"))
		assert.Equal("no-cache", rec.Header().Get("Pragma"))

		res := errorResponse{}
		if assert.NoError(json.NewDecoder(rec.Body).Decode(&res)) {
			assert.Equal(errInvalidRequest, res.ErrorType)
		}
	}
}
