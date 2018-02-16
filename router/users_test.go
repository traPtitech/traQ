package router

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"net/http"

	"github.com/labstack/echo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestGetUsers(t *testing.T) {
	e, cookie, mw := beforeTest(t)
	assert := assert.New(t)

	rec := request(e, t, mw(GetUsers), cookie, nil)
	if assert.EqualValues(http.StatusOK, rec.Code, rec.Body.String()) {
		var responseBody []UserForResponse
		if assert.NoError(json.Unmarshal(rec.Body.Bytes(), &responseBody)) {
			// testUser traq
			assert.Len(responseBody, 2)
		}
	}
}

func TestGetMe(t *testing.T) {
	e, cookie, mw := beforeTest(t)
	assert := assert.New(t)

	rec := request(e, t, mw(GetMe), cookie, nil)
	if assert.EqualValues(http.StatusOK, rec.Code, rec.Body.String()) {
		var me UserForResponse
		if assert.NoError(json.Unmarshal(rec.Body.Bytes(), &me)) {
			assert.Equal(testUser.ID, me.UserID)
		}
	}
}

func TestGetUserByID(t *testing.T) {
	e, cookie, mw := beforeTest(t)
	assert := assert.New(t)

	c, rec := getContext(e, t, cookie, nil)
	c.SetPath("/users/:userID")
	c.SetParamNames("userID")
	c.SetParamValues(testUser.ID)

	requestWithContext(t, mw(GetUserByID), c)

	if assert.EqualValues(http.StatusOK, rec.Code, rec.Body.String()) {
		var user UserDetailForResponse
		if assert.NoError(json.Unmarshal(rec.Body.Bytes(), &user)) {
			assert.Equal(testUser.ID, user.UserID)
		}
	}
}
