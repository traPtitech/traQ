package router

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/traPtitech/traQ/model"
)

func TestPostUserTags(t *testing.T) {
	e, cookie, mw := beforeTest(t)
	tagText := "post test"

	// 正常系
	post := struct {
		Tag string `json:"tag"`
	}{
		Tag: tagText,
	}
	body, err := json.Marshal(post)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("POST", "http://test", bytes.NewReader(body))
	c, rec := getContext(e, t, cookie, req)
	c.SetPath("/users/:userID/tags")
	c.SetParamNames("userID")
	c.SetParamValues(testUser.ID)
	requestWithContext(t, mw(PostUserTag), c)

	var responseBody []*TagForResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &responseBody); err != nil {
		t.Fatalf("Response body can't unmarshal: %v", err)
	}

	if rec.Code != http.StatusCreated {
		t.Fatalf("Response code wrong. want: %d, actual: %d", http.StatusCreated, rec.Code)
	}
	if responseBody[0].Tag != tagText {
		t.Errorf("Tag is wrong. want: %s, actual: %s", tagText, responseBody[0].Tag)
	}
}

func TestGetUserTags(t *testing.T) {
	e, cookie, mw := beforeTest(t)
	for i := 0; i < 5; i++ {
		tagText := model.CreateUUID()
		if _, err := makeTag(testUser.ID, tagText); err != nil {
			t.Fatal(err)
		}
	}

	// 正常系
	c, rec := getContext(e, t, cookie, nil)
	c.SetPath("/users/:userID/tags/")
	c.SetParamNames("userID")
	c.SetParamValues(testUser.ID)
	requestWithContext(t, mw(GetUserTags), c)

	var responseBody []TagForResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &responseBody); err != nil {
		t.Fatalf("Response body can't unmarshal: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Response code is wrong. want: %d, actual:%d", http.StatusOK, rec.Code)
	}
	if len(responseBody) != 5 {
		t.Errorf("Length of response tags is wrong. want: 5, actual: %d", len(responseBody))
	}
}

func TestPutUserTags(t *testing.T) {
	e, cookie, mw := beforeTest(t)
	tagText := "put test"

	// 正常系
	tag, err := makeTag(testUser.ID, tagText)
	if err != nil {
		t.Fatal(err)
	}

	post := struct {
		IsLocked bool `json:"isLocked"`
	}{
		IsLocked: true,
	}
	body, err := json.Marshal(post)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("PUT", "http://test", bytes.NewReader(body))
	c, rec := getContext(e, t, cookie, req)
	c.SetPath("/users/:userID/tags/:tagID")
	c.SetParamNames("userID", "tagID")
	c.SetParamValues(testUser.ID, tag.ID)
	requestWithContext(t, mw(PutUserTag), c)

	var responseBody []*TagForResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &responseBody); err != nil {
		t.Fatalf("Response body can't unmarshal: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Response code wrong. want: %d, actual: %d", http.StatusOK, rec.Code)
	}
	if responseBody[0].IsLocked != true {
		t.Errorf("Response isLocked is wrong. want: true, actual: %v", responseBody[0].IsLocked)
	}
}

func TestDeleteUserTags(t *testing.T) {
	e, cookie, mw := beforeTest(t)
	tagText := "Delete test"

	// 正常系
	tag, err := makeTag(testUser.ID, tagText)
	if err != nil {
		t.Fatal(err)
	}

	c, rec := getContext(e, t, cookie, nil)
	c.SetPath("/users/:userID/tags/:tagID")
	c.SetParamNames("userID", "tagID")
	c.SetParamValues(testUser.ID, tag.ID)
	requestWithContext(t, mw(DeleteUserTag), c)

	if rec.Code != http.StatusNoContent {
		t.Errorf("Response code wrong. want: %d, actual: %d", http.StatusNoContent, rec.Code)
	}
}
