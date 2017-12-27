package router

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

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
