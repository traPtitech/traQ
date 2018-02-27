package router

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetClips(t *testing.T) {
	e, cookie, mw, assert, _ := beforeTest(t)
	channel := mustMakeChannel(t, testUser.ID, "test", true)
	message := mustMakeMessage(t, testUser.ID, channel.ID)
	mustClipMessage(t, testUser.ID, message.ID)

	rec := request(e, t, mw(GetClips), cookie, nil)
	if assert.EqualValues(http.StatusOK, rec.Code) {
		var responseBody []MessageForResponse
		if assert.NoError(json.Unmarshal(rec.Body.Bytes(), &responseBody)) {
			assert.Len(responseBody, 1)
			assert.Equal(message.Text, responseBody[0].Content)
		}
	}
}

func TestPostClips(t *testing.T) {
	e, cookie, mw, assert, require := beforeTest(t)
	channel := mustMakeChannel(t, testUser.ID, "test", true)
	message := mustMakeMessage(t, testUser.ID, channel.ID)

	post := struct {
		MessageID string `json:"messageId"`
	}{
		MessageID: message.ID,
	}

	body, err := json.Marshal(post)
	require.NoError(err)
	req := httptest.NewRequest("POST", "http://test", bytes.NewReader(body))
	rec := request(e, t, mw(PostClips), cookie, req)

	if assert.EqualValues(http.StatusCreated, rec.Code) {
		var responseBody []MessageForResponse
		if assert.NoError(json.Unmarshal(rec.Body.Bytes(), &responseBody)) {
			assert.Len(responseBody, 1)
			assert.Equal(message.Text, responseBody[0].Content)
		}
	}
}

func TestDeleteClips(t *testing.T) {
	e, cookie, mw, assert, require := beforeTest(t)
	channel := mustMakeChannel(t, testUser.ID, "test", true)
	message := mustMakeMessage(t, testUser.ID, channel.ID)
	mustClipMessage(t, testUser.ID, message.ID)

	post := struct {
		MessageID string `json:"messageId"`
	}{
		MessageID: message.ID,
	}

	body, err := json.Marshal(post)
	require.NoError(err)
	req := httptest.NewRequest("DELETE", "http://test", bytes.NewReader(body))
	rec := request(e, t, mw(DeleteClips), cookie, req)

	if assert.EqualValues(http.StatusOK, rec.Code) {
		var responseBody []MessageForResponse
		if assert.NoError(json.Unmarshal(rec.Body.Bytes(), &responseBody)) {
			assert.Len(responseBody, 0)
		}
	}
}
