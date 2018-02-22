package router

import (
	"bytes"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetChannelPin(t *testing.T) {
	e, cookie, mw := beforeTest(t)
	testChannel := mustMakeChannel(t, testUser.ID, "pinChannel", true)
	testMessage := mustMakeMessage(t)

	//正常系
	testPin := mustMakePin(t, testChannel.ID, testUser.ID, testMessage.ID)
	c, rec := getContext(e, t, cookie, nil)
	c.SetPath("/channel/:channelID/pin")
	c.SetParamNames("channelID")
	c.SetParamValues(testChannel.ID)
	requestWithContext(t, mw(GetChannelPin), c)

	assert.EqualValues(t, http.StatusOK, rec.Code)
	var responseBody []*PinForResponse
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &responseBody))
	assert.Len(t, responseBody, 1)

	correctResponse, err := formatPin(testPin)
	require.NoError(t, err)

	correctResponse.Message.Datetime = parseDateTime(correctResponse.Message.Datetime)
	assert.Equal(t, *responseBody[0].Message, *correctResponse.Message)
	responseBody[0].Message, correctResponse.Message = nil, nil

	correctResponse.DateTime = parseDateTime(correctResponse.DateTime)
	assert.Equal(t, *responseBody[0], *correctResponse)
}

func TestGetPin(t *testing.T) {
	e, cookie, mw := beforeTest(t)
	testChannel := mustMakeChannel(t, testUser.ID, "pinChannel", true)
	testMessage := mustMakeMessage(t)

	//正常系
	testPin := mustMakePin(t, testChannel.ID, testUser.ID, testMessage.ID)
	c, rec := getContext(e, t, cookie, nil)
	c.SetPath("/pin/:pinID")
	c.SetParamNames("pinID")
	c.SetParamValues(testPin.ID)
	requestWithContext(t, mw(GetPin), c)

	assert.EqualValues(t, http.StatusOK, rec.Code)
	responseBody := &PinForResponse{}
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), responseBody))

	correctResponse, err := formatPin(testPin)
	require.NoError(t, err)

	correctResponse.Message.Datetime = parseDateTime(correctResponse.Message.Datetime)
	assert.Equal(t, *responseBody.Message, *correctResponse.Message)
	responseBody.Message, correctResponse.Message = nil, nil

	correctResponse.DateTime = parseDateTime(correctResponse.DateTime)
	assert.Equal(t, *responseBody, *correctResponse)
}

func TestPostPin(t *testing.T) {
	e, cookie, mw := beforeTest(t)
	testChannel := mustMakeChannel(t, testUser.ID, "pinChannel", true)
	testMessage := mustMakeMessage(t)

	//正常系
	post := struct {
		MessageID string `json:"messageId"`
	}{
		MessageID: testMessage.ID,
	}
	body, err := json.Marshal(post)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/", bytes.NewReader(body))
	c, rec := getContext(e, t, cookie, req)
	c.SetPath("/channels/:channelID/pin")
	c.SetParamNames("channelID")
	c.SetParamValues(testChannel.ID)
	requestWithContext(t, mw(PostPin), c)

	assert.EqualValues(t, http.StatusCreated, rec.Code)
	responseBody := &PinForResponse{}
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), responseBody))

	correctResponse, err := getChannelPinResponse(testChannel.ID)
	require.NoError(t, err)
	require.Len(t, correctResponse, 1)

	correctResponse[0].Message.Datetime = parseDateTime(correctResponse[0].Message.Datetime)
	assert.Equal(t, *responseBody.Message, *correctResponse[0].Message)
	responseBody.Message, correctResponse[0].Message = nil, nil

	correctResponse[0].DateTime = parseDateTime(correctResponse[0].DateTime)
	assert.Equal(t, *responseBody, *correctResponse[0])
}

func TestDeletePin(t *testing.T) {
	e, cookie, mw := beforeTest(t)
	testChannel := mustMakeChannel(t, testUser.ID, "pinChannel", true)
	testMessage := mustMakeMessage(t)

	//正常系
	testPin := mustMakePin(t, testChannel.ID, testUser.ID, testMessage.ID)
	req := httptest.NewRequest("DELETE", "/", nil)
	c, rec := getContext(e, t, cookie, req)
	c.SetPath("/pin/:pinID")
	c.SetParamNames("pinID")
	c.SetParamValues(testPin.ID)
	requestWithContext(t, mw(DeletePin), c)

	assert.EqualValues(t, http.StatusNoContent, rec.Code)
}
