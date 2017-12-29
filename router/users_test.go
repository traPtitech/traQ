package router

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/model"
)

func TestPostLogin(t *testing.T) {
	e, _, mw := beforeTest(t)
	createUser(t)

	type requestJSON struct {
		Name string `json:"name"`
		Pass string `json:"pass"`
	}

	requestBody := &requestJSON{"test", "test"}

	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("POST", "http://test", bytes.NewReader(body))
	rec := request(e, t, mw(PostLogin), nil, req)

	if rec.Code != 200 {
		t.Errorf("Status code wrong: want 200, actual %d", rec.Code)
	}

	requestBody2 := &requestJSON{"test", "wrong_password"}

	body2, err := json.Marshal(requestBody2)
	if err != nil {
		t.Fatal(err)
	}

	req2 := httptest.NewRequest("POST", "http://test", bytes.NewReader(body2))
	rec2 := httptest.NewRecorder()
	c := e.NewContext(req2, rec2)
	err2 := mw(PostLogin)(c).(*echo.HTTPError)

	if err2 == nil {
		t.Fatal("handler did not return error object")
	}

	if err2.Code != 403 {
		t.Errorf("Status code wrong: want 403, actual %d", err2.Code)
	}
}

func createUser(t *testing.T) {
	user := &model.User{
		Name:  "test",
		Email: "example@trap.jp",
		Icon:  "empty",
	}
	err := user.SetPassword("test")
	if err != nil {
		t.Fatal(err)
	}
	err = user.Create()
	if err != nil {
		t.Fatal(err)
	}
}
