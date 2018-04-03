package router

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/traPtitech/traQ/model"

	"net/http"

	"github.com/labstack/echo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostLogin(t *testing.T) {
	e, mw := beforeLoginTest(t)
	mustCreateUser(t, "PostLogin")

	type requestJSON struct {
		Name string `json:"name"`
		Pass string `json:"pass"`
	}

	requestBody := &requestJSON{"PostLogin", "test"}

	body, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "http://test", bytes.NewReader(body))
	rec := request(e, t, mw(PostLogin), nil, req)

	assert.EqualValues(t, http.StatusNoContent, rec.Code)

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
	e, cookie, mw, assert, _ := beforeTest(t)

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
	e, cookie, mw, assert, _ := beforeTest(t)

	rec := request(e, t, mw(GetMe), cookie, nil)
	if assert.EqualValues(http.StatusOK, rec.Code, rec.Body.String()) {
		var me UserForResponse
		if assert.NoError(json.Unmarshal(rec.Body.Bytes(), &me)) {
			assert.Equal(testUser.ID, me.UserID)
		}
	}
}

func TestPatchMe(t *testing.T) {
	e, cookie, mw, assert, require := beforeTest(t)

	type requestJSON struct {
		ExPassword  string `json:"exPassword"`
		DisplayName string `json:"displayName"`
		Email       string `json:"email"`
		Password    string `json:"password"`
	}

	// 正常系
	post := requestJSON{
		ExPassword:  "test",
		DisplayName: "renamed",
		Email:       "example@gmail.com",
		Password:    "renamed",
	}
	body, err := json.Marshal(post)
	require.NoError(err)

	req := httptest.NewRequest("PATCH", "http://test", bytes.NewReader(body))
	rec := request(e, t, mw(PatchMe), cookie, req)

	if assert.EqualValues(http.StatusNoContent, rec.Code, rec.Body.String()) {
		updatedUser, err := model.GetUser(testUser.ID)
		require.NoError(err)
		assert.Equal(post.DisplayName, updatedUser.DisplayName)
	}

	// 正常系: displayNameのみの変更の場合
	post = requestJSON{
		DisplayName: "display-only",
	}
	body, err = json.Marshal(post)
	require.NoError(err)

	req = httptest.NewRequest("PATCH", "http://test", bytes.NewReader(body))
	rec = request(e, t, mw(PatchMe), cookie, req)

	if assert.EqualValues(http.StatusNoContent, rec.Code, rec.Body.String()) {
		updatedUser, err := model.GetUser(testUser.ID)
		require.NoError(err)
		assert.Equal(post.DisplayName, updatedUser.DisplayName)
	}

	// 正常系: emailのみの変更の場合
	post = requestJSON{
		ExPassword: "renamed",
		Email:      "popopo@gmail.com",
	}
	body, err = json.Marshal(post)
	require.NoError(err)

	req = httptest.NewRequest("PATCH", "http://test", bytes.NewReader(body))
	rec = request(e, t, mw(PatchMe), cookie, req)

	if assert.EqualValues(http.StatusNoContent, rec.Code, rec.Body.String()) {
		updatedUser, err := model.GetUser(testUser.ID)
		require.NoError(err)
		assert.Equal(post.Email, updatedUser.Email)
	}

	// 異常系: passwordが誤っている場合
	post = requestJSON{
		ExPassword:  "wrong-password",
		DisplayName: "display-only",
		Email:       "hogehoge@yahoo.co.jp",
		Password:    "password",
	}
	body, err = json.Marshal(post)
	require.NoError(err)

	req = httptest.NewRequest("PATCH", "http://test", bytes.NewReader(body))
	rec = request(e, t, mw(PatchMe), cookie, req)

	assert.EqualValues(http.StatusUnauthorized, rec.Code, rec.Body.String())
}

func TestGetUserByID(t *testing.T) {
	e, cookie, mw, assert, _ := beforeTest(t)

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
