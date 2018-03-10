package router

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func TestGetChannelVisibility(t *testing.T) {
	e, cookie, mw, assert, require := beforeTest(t)

	for i := 0; i < 5; i++ {
		mustMakeInvisibleChannel(t, testUser.ID, "test-"+strconv.Itoa(i), true)
	}

	rec := request(e, t, mw(GetChannelsVisibility), cookie, nil)

	require.EqualValues(http.StatusOK, rec.Code, rec.Body.String())

	var res Visibility
	if assert.NoError(json.Unmarshal(rec.Body.Bytes(), &res)) {
		assert.Equal(5, len(res.Hidden))
	}
}

func TestPutChannelVisibility(t *testing.T) {
	e, cookie, mw, assert, require := beforeTest(t)
	ch := mustMakeChannel(t, testUser.ID, "test", true)

	jsonBody := &Visibility{
		Visible: []string{},
		Hidden:  []string{ch.ID},
	}
	body, err := json.Marshal(jsonBody)
	require.NoError(err)

	req := httptest.NewRequest("PUT", "http://test", bytes.NewReader(body))
	c, rec := getContext(e, t, cookie, req)
	requestWithContext(t, mw(PutChannelsVisibility), c)

	require.EqualValues(http.StatusOK, rec.Code, rec.Body.String())

	var res Visibility
	if assert.NoError(json.Unmarshal(rec.Body.Bytes(), &res)) {
		assert.Equal(len(jsonBody.Hidden), len(res.Hidden))
	}

}
