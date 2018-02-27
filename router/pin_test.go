package router

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetChannelPin(t *testing.T) {
	e, cookie, mw, assert, require := beforeTest(t)
	testChannel := mustMakeChannel(t, testUser.ID, "pinChannel", true)
	testMessage := mustMakeMessage(t, testUser.ID, testChannel.ID)

	//正常系
	testPin := mustMakePin(t, testChannel.ID, testUser.ID, testMessage.ID)
	c, rec := getContext(e, t, cookie, nil)
	c.SetPath("/channel/:channelID/pin")
	c.SetParamNames("channelID")
	c.SetParamValues(testChannel.ID)
	requestWithContext(t, mw(GetChannelPin), c)

	assert.EqualValues(http.StatusOK, rec.Code)
	var responseBody []*PinForResponse
	assert.NoError(json.Unmarshal(rec.Body.Bytes(), &responseBody))
	assert.Len(responseBody, 1)

	correctResponse, err := formatPin(testPin)
	require.NoError(err)

	correctResponse.Message.Datetime = parseDateTime(correctResponse.Message.Datetime)
	assert.Equal(*responseBody[0].Message, *correctResponse.Message)
	responseBody[0].Message, correctResponse.Message = nil, nil

	correctResponse.DateTime = parseDateTime(correctResponse.DateTime)
	assert.Equal(*responseBody[0], *correctResponse)
}

func TestGetPin(t *testing.T) {
	e, cookie, mw, assert, require := beforeTest(t)
	testChannel := mustMakeChannel(t, testUser.ID, "pinChannel", true)
	testMessage := mustMakeMessage(t, testUser.ID, testChannel.ID)

	//正常系
	testPin := mustMakePin(t, testChannel.ID, testUser.ID, testMessage.ID)
	c, rec := getContext(e, t, cookie, nil)
	c.SetPath("/pin/:pinID")
	c.SetParamNames("pinID")
	c.SetParamValues(testPin.ID)
	requestWithContext(t, mw(GetPin), c)

	assert.EqualValues(http.StatusOK, rec.Code)
	responseBody := &PinForResponse{}
	assert.NoError(json.Unmarshal(rec.Body.Bytes(), responseBody))

	correctResponse, err := formatPin(testPin)
	require.NoError(err)

	correctResponse.Message.Datetime = parseDateTime(correctResponse.Message.Datetime)
	assert.Equal(*responseBody.Message, *correctResponse.Message)
	responseBody.Message, correctResponse.Message = nil, nil

	correctResponse.DateTime = parseDateTime(correctResponse.DateTime)
	assert.Equal(*responseBody, *correctResponse)
}

func TestPostPin(t *testing.T) {
	e, cookie, mw, assert, require := beforeTest(t)
	testChannel := mustMakeChannel(t, testUser.ID, "pinChannel", true)
	testMessage := mustMakeMessage(t, testUser.ID, testChannel.ID)

	//正常系
	post := struct {
		MessageID string `json:"messageId"`
	}{
		MessageID: testMessage.ID,
	}
	body, err := json.Marshal(post)
	require.NoError(err)

	req := httptest.NewRequest("POST", "/", bytes.NewReader(body))
	c, rec := getContext(e, t, cookie, req)
	c.SetPath("/channels/:channelID/pin")
	c.SetParamNames("channelID")
	c.SetParamValues(testChannel.ID)
	requestWithContext(t, mw(PostPin), c)

	assert.EqualValues(http.StatusCreated, rec.Code)
	responseBody := &PinForResponse{}
	assert.NoError(json.Unmarshal(rec.Body.Bytes(), responseBody))

	correctResponse, err := getChannelPinResponse(testChannel.ID)
	require.NoError(err)
	require.Len(correctResponse, 1)

	correctResponse[0].Message.Datetime = parseDateTime(correctResponse[0].Message.Datetime)
	assert.Equal(*responseBody.Message, *correctResponse[0].Message)
	responseBody.Message, correctResponse[0].Message = nil, nil

	correctResponse[0].DateTime = parseDateTime(correctResponse[0].DateTime)
	assert.Equal(*responseBody, *correctResponse[0])
}

func TestDeletePin(t *testing.T) {
	e, cookie, mw, assert, _ := beforeTest(t)
	testChannel := mustMakeChannel(t, testUser.ID, "pinChannel", true)
	testMessage := mustMakeMessage(t, testUser.ID, testChannel.ID)

	//正常系
	testPin := mustMakePin(t, testChannel.ID, testUser.ID, testMessage.ID)
	req := httptest.NewRequest("DELETE", "/", nil)
	c, rec := getContext(e, t, cookie, req)
	c.SetPath("/pin/:pinID")
	c.SetParamNames("pinID")
	c.SetParamValues(testPin.ID)
	requestWithContext(t, mw(DeletePin), c)

	assert.EqualValues(http.StatusNoContent, rec.Code)
}
