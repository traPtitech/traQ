package model

import (
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPinTableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "pins", (&Pin{}).TableName())
}

func TestCreatePin(t *testing.T) {
	assert, _, user, channel := beforeTest(t)
	testMessage := mustMakeMessage(t, user.GetUID(), channel.GetCID())

	p, err := CreatePin(testMessage.GetID(), user.GetUID())
	if assert.NoError(err) {
		assert.NotEmpty(p)
	}

	_, err = CreatePin(testMessage.GetID(), user.GetUID())
	assert.Error(err)
}

func TestGetPin(t *testing.T) {
	assert, require, user, channel := beforeTest(t)

	testMessage := mustMakeMessage(t, user.GetUID(), channel.GetCID())
	p, err := CreatePin(testMessage.GetID(), user.GetUID())
	require.NoError(err)

	pin, err := GetPin(p)
	if assert.NoError(err) {
		assert.Equal(p.String(), pin.ID)
		assert.Equal(testMessage.ID, pin.MessageID)
		assert.Equal(user.ID, pin.UserID)
		assert.NotZero(pin.CreatedAt)
		assert.NotZero(pin.Message)
	}

	pin, err = GetPin(uuid.Nil)
	if assert.NoError(err) {
		assert.Nil(pin)
	}
}

func TestIsPinned(t *testing.T) {
	assert, require, user, channel := beforeTest(t)

	testMessage := mustMakeMessage(t, user.GetUID(), channel.GetCID())
	_, err := CreatePin(testMessage.GetID(), user.GetUID())
	require.NoError(err)

	ok, err := IsPinned(testMessage.GetID())
	if assert.NoError(err) {
		assert.True(ok)
	}

	ok, err = IsPinned(uuid.Nil)
	if assert.NoError(err) {
		assert.False(ok)
	}
}

func TestDeletePin(t *testing.T) {
	assert, require, user, channel := beforeTest(t)

	testMessage := mustMakeMessage(t, user.GetUID(), channel.GetCID())
	p, err := CreatePin(testMessage.GetID(), user.GetUID())
	require.NoError(err)

	if assert.NoError(DeletePin(p)) {
		pin, err := GetPin(uuid.Nil)
		require.NoError(err)
		assert.Nil(pin)
	}
}

func TestGetPinsByChannelID(t *testing.T) {
	assert, require, user, channel := beforeTest(t)

	testMessage := mustMakeMessage(t, user.GetUID(), channel.GetCID())
	_, err := CreatePin(testMessage.GetID(), user.GetUID())
	require.NoError(err)

	//正常系
	pins, err := GetPinsByChannelID(channel.GetCID())
	if assert.NoError(err) {
		if assert.Len(pins, 1) {
			pin, err := GetPin(uuid.Nil)
			require.NoError(err)
			assert.EqualValues(pin, pins[0])
		}
	}
}
