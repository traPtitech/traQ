package router

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/traPtitech/traQ/model"
)

func TestGetClips(t *testing.T) {
	e, cookie, mw := beforeTest(t)

	message := makeMessage()

	if err := clipMessage(testUser.ID, message.ID); err != nil {
		t.Fatalf("failed to clip message: %v", err)
	}

	rec := request(e, t, mw(GetClips), cookie, nil)

	if rec.Code != 200 {
		t.Fatalf("Response code wrong: want 200, actual %d", rec.Code)
	}

	var responseBody []MessageForResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &responseBody); err != nil {
		t.Fatalf("Response body can't unmarshal: %v", err)
	}

	if len(responseBody) != 1 {
		t.Fatalf("Response messages length wrong: want 1, actual %d", len(responseBody))
	}

	if responseBody[0].Content != message.Text {
		t.Fatalf("message text is wrong: want %v, actual %v", message.Text, responseBody[0].Content)
	}
}

func TestPostClips(t *testing.T) {
	e, cookie, mw := beforeTest(t)

	message := makeMessage()

	post := struct {
		MessageID string `json:"messageId"`
	}{
		MessageID: message.ID,
	}

	body, err := json.Marshal(post)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("POST", "http://test", bytes.NewReader(body))
	rec := request(e, t, mw(PostClips), cookie, req)

	if rec.Code != 201 {
		t.Fatalf("Response code wrong: want 200, actual %d", rec.Code)
	}

	var responseBody []MessageForResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &responseBody); err != nil {
		t.Fatalf("Response body can't unmarshal: %v", err)
	}

	if len(responseBody) != 1 {
		t.Fatalf("Response messages length wrong: want 1, actual %d", len(responseBody))
	}

	if responseBody[0].Content != message.Text {
		t.Fatalf("message text is wrong: want %v, actual %v", message.Text, responseBody[0].Content)
	}
}

func TestDeleteClips(t *testing.T) {
	e, cookie, mw := beforeTest(t)

	message := makeMessage()
	if err := clipMessage(testUser.ID, message.ID); err != nil {
		t.Fatalf("failed to clip message: %v", err)
	}

	post := struct {
		MessageID string `json:"messageId"`
	}{
		MessageID: message.ID,
	}

	body, err := json.Marshal(post)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("DELETE", "http://test", bytes.NewReader(body))
	rec := request(e, t, mw(DeleteClips), cookie, req)

	if rec.Code != 200 {
		t.Fatalf("Response code wrong: want 200, actual %d", rec.Code)
	}

	var responseBody []MessageForResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &responseBody); err != nil {
		t.Fatalf("Response body can't unmarshal: %v", err)
	}

	if len(responseBody) != 0 {
		t.Fatalf("Response messages length wrong: want 1, actual %d", len(responseBody))
	}
}

func clipMessage(userID, messageID string) error {
	clip := &model.Clip{
		UserID:    userID,
		MessageID: messageID,
	}

	return clip.Create()
}
