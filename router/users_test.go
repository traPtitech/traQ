package router

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
)

func TestPostLogin(t *testing.T) {
	e, mw := beforeLoginTest(t)
	mustCreateUser(t)

	type requestJSON struct {
		Name string `json:"name"`
		Pass string `json:"pass"`
	}

	requestBody := &requestJSON{"PostLogin", "test"}

	body, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "http://test", bytes.NewReader(body))
	rec := request(e, t, mw(PostLogin), nil, req)

	assert.EqualValues(t, http.StatusOK, rec.Code)

	requestBody2 := &requestJSON{"PostLogin", "wrong_password"}
	body2, err := json.Marshal(requestBody2)
	require.NoError(t, err)

	req2 := httptest.NewRequest("POST", "http://test", bytes.NewReader(body2))
	req2.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec2 := httptest.NewRecorder()
	c := e.NewContext(req2, rec2)
	err2 := mw(PostLogin)(c).(*echo.HTTPError)

	if assert.Error(t, err2) {
		assert.EqualValues(t, http.StatusForbidden, err2.Code)
	}
}
