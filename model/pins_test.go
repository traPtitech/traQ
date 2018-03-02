package model

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPinTableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "pins", (&Pin{}).TableName())
}

func beforePinTest(t *testing.T, userID, channelID string) *Pin {
	testMessage := mustMakeMessage(t, userID, channelID)
	testPin := &Pin{
		UserID:    userID,
		MessageID: testMessage.ID,
		ChannelID: channelID,
	}
	return testPin
}

func TestPinCreate(t *testing.T) {
	assert, require, user, channel := beforeTest(t)
	testPin := beforePinTest(t, user.ID, channel.ID)

	//正常系
	assert.NoError(testPin.Create())
	pins, err := GetPinsByChannelID(testPin.ChannelID)
	require.NoError(err)
	assert.Len(pins, 1)
	testPin.CreatedAt = pins[0].CreatedAt
	assert.Equal(*pins[0], *testPin)
}

func TestGetPin(t *testing.T) {
	assert, require, user, channel := beforeTest(t)
	testPin := beforePinTest(t, user.ID, channel.ID)

	//正常系
	require.NoError(testPin.Create())
	pin, err := GetPin(testPin.ID)
	assert.NoError(err)
	testPin.CreatedAt = pin.CreatedAt
	assert.Equal(*pin, *testPin)
}

func TestGetPinsByChannelID(t *testing.T) {
	assert, require, user, channel := beforeTest(t)
	testPin := beforePinTest(t, user.ID, channel.ID)

	//正常系
	require.NoError(testPin.Create())
	pins, err := GetPinsByChannelID(testPin.ChannelID)
	assert.NoError(err)
	assert.Len(pins, 1)
	testPin.CreatedAt = pins[0].CreatedAt
	assert.Equal(*pins[0], *testPin)
}

func TestPinDelete(t *testing.T) {
	assert, require, user, channel := beforeTest(t)
	testPin := beforePinTest(t, user.ID, channel.ID)

	//正常系
	require.NoError(testPin.Create())
	assert.NoError(testPin.Delete())
	pins, err := GetPinsByChannelID(testPin.ChannelID)
	require.NoError(err)
	assert.Len(pins, 0)
}
