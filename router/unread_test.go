package router

import (
	"bytes"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGetUnread(t *testing.T) {
	e, cookie, mw := beforeTest(t)
	testMessage := mustMakeMessage(t)

	// 正常系
	mustMakeUnread(t, testUser.ID, testMessage.ID)
	c, rec := getContext(e, t, cookie, nil)
	c.SetPath("/users/me/unread")
	requestWithContext(t, mw(GetUnread), c)

	assert.EqualValues(t, http.StatusOK, rec.Code)
	var responseBody []*MessageForResponse
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &responseBody))
	assert.Len(t, responseBody, 1)
	correctResponse := formatMessage(testMessage)
	correctResponse.Datetime = correctResponse.Datetime.Truncate(time.Second) // DBは秒未満を切り捨てるので
	correctResponse.Datetime = correctResponse.Datetime.In(time.UTC)          // DBはタイムゾーン情報を保存しないので
	assert.Equal(t, *responseBody[0], *correctResponse)
}

func TestDeleteUnread(t *testing.T) {
	e, cookie, mw := beforeTest(t)
	testMessage := mustMakeMessage(t)

	// 正常系
	mustMakeUnread(t, testUser.ID, testMessage.ID)
	post := []string{testMessage.ID}
	body, err := json.Marshal(post)
	require.NoError(t, err)

	req := httptest.NewRequest("DELETE", "http://test", bytes.NewReader(body))
	c, rec := getContext(e, t, cookie, req)
	c.SetPath("/users/me/unread")
	requestWithContext(t, mw(DeleteUnread), c)

	assert.EqualValues(t, http.StatusNoContent, rec.Code)
}
