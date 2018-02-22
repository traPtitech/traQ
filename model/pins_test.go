package model

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func beforePinTest(t *testing.T) *Pin {
	testMessage := mustMakeMessage(t)
	testChannel := mustMakeChannel(t, "pin")
	testPin := &Pin{
		UserID:    testUserID,
		MessageID: testMessage.ID,
		ChannelID: testChannel.ID,
	}
	return testPin
}

func TestPinTableName(t *testing.T) {
	assert.Equal(t, "pins", (&Pin{}).TableName())
}

func TestPinCreate(t *testing.T) {
	beforeTest(t)
	testPin := beforePinTest(t)

	//正常系
	assert.NoError(t, testPin.Create())
	pins, err := GetPinsByChannelID(testPin.ChannelID)
	require.NoError(t, err)
	assert.Len(t, pins, 1)
	testPin.CreatedAt = pins[0].CreatedAt
	assert.Equal(t, *pins[0], *testPin)
}

func TestGetPin(t *testing.T) {
	beforeTest(t)
	testPin := beforePinTest(t)

	//正常系
	require.NoError(t, testPin.Create())
	pin, err := GetPin(testPin.ID)
	assert.NoError(t, err)
	testPin.CreatedAt = pin.CreatedAt
	assert.Equal(t, *pin, *testPin)
}

func TestGetPinsByChannelID(t *testing.T) {
	beforeTest(t)
	testPin := beforePinTest(t)

	//正常系
	require.NoError(t, testPin.Create())
	pins, err := GetPinsByChannelID(testPin.ChannelID)
	assert.NoError(t, err)
	assert.Len(t, pins, 1)
	testPin.CreatedAt = pins[0].CreatedAt
	assert.Equal(t, *pins[0], *testPin)
}

func TestPinDelete(t *testing.T) {
	beforeTest(t)
	testPin := beforePinTest(t)

	//正常系
	require.NoError(t, testPin.Create())
	assert.NoError(t, testPin.Delete())
	pins, err := GetPinsByChannelID(testPin.ChannelID)
	require.NoError(t, err)
	assert.Len(t, pins, 0)
}
