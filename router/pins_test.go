package router

import (
	"bytes"
	"testing"
	"encoding/json"
	"net/http/httptest"
)

func TestGetPin(t *testing.T) {
	e, cookie, mw := beforeTest(t)

	message := makeMessage()
	if err := pinMessage(message.ChannelID, message.ID, message.UserID); err != nil{
		t.Fatalf("Fail to pin mesasge: %v", nil)
	}
	rec := request(e, t, mw(GetPin(c)), cookie, nil)

	if req.Code != 200 {
		t.Fatalf("Response code wring: want 200, actual %d", req.Code)
	}

	var responseBody []MessageForResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &responseBody); err !=  nil {
		t.Fatalf("Response body can't unmarshal: %v", err)
	}

	if len(responseBody) != 1 {
		t.Fatalf("Response messages length wrong: want 1, actual %d", len(responseBody))
	}

	if responseBody[0].Content != "text message"{
		t.Fatalf("message texr is wrong: want %v, actual %v", "text message", responseBody[0].Context)
	}
}

func TestPutPin(t *testing.T) {
	e, cookie, mw := beforeTest(t)
	message := makeMessage()

	post := struct {
		MessageID string `josn:"messageId"`
	}{
		MessageID: message.ID,
	}

	body, err := json.Marshal(put)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("PUT", "http://test", bytes.NewReader(body))
	rec := request(e, t, mw(PutPin), cookie, req)

	if req.Code != 200 {
		t.Fatalf("Respnse code wrong: want 200, actual %d", rec.Code)
	}

	var responseBody []MessageForResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &responseBody); err != nil {
		t.Fatalf("Response body can't unmarshal: %v", err)
	}

	if len(responseBody) != 1 {
		t.Fatalf("Response message length wrong: want 1, actual %d", len(responseBody))
	}

	if responseBody[0].Content != "text message" {
		t.Fatalf("message text is wrong: want %v, actual %v", "text message", responseBody[0].Content)
	}
}

func TestDeletePin(t *testing.T) {
	e, cookie, mw := beforeTest(t)

	message := makeMessage()
	if err := pinMessage(message.ChannelID, message.ID, testUser.ID); err != nil {
		t.Fatalf("failed to pin message: %v", err)
	}

	post := struct {
		MessageID string `josn:"messageId"`
	}{
		MessageID: message.ID
	}

	body, err := json.Marshal(post)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("DELETE", "http://test", bytes.NewReader(body))
	rec := request(e, t, mw(DeletePin), cookie, req)

	if rec.Code != 204 {
		t.Fatalf("Respomse code wrong: want 204, actual %d", rec.Code)
	}

}
