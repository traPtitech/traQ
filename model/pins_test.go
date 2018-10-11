package model

import (
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/traPtitech/traQ/utils"
	"testing"
)

func TestPinTableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "pins", (&Pin{}).TableName())
}

// TestParallelGroup8 並列テストグループ8 競合がないようなサブテストにすること
func TestParallelGroup8(t *testing.T) {
	assert, require, user, channel := beforeTest(t)

	// CreatePin
	t.Run("TestCreatePin", func(t *testing.T) {
		t.Parallel()

		testMessage := mustMakeMessage(t, user.GetUID(), channel.ID)

		p, err := CreatePin(testMessage.GetID(), user.GetUID())
		if assert.NoError(err) {
			assert.NotEmpty(p)
		}

		_, err = CreatePin(testMessage.GetID(), user.GetUID())
		assert.Error(err)
	})

	// GetPin
	t.Run("TestGetPin", func(t *testing.T) {
		t.Parallel()

		testMessage := mustMakeMessage(t, user.GetUID(), channel.ID)
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
	})

	// IsPinned
	t.Run("TestIsPinned", func(t *testing.T) {
		t.Parallel()

		testMessage := mustMakeMessage(t, user.GetUID(), channel.ID)
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
	})

	// DeletePin
	t.Run("TestDeletePin", func(t *testing.T) {
		t.Parallel()

		testMessage := mustMakeMessage(t, user.GetUID(), channel.ID)
		p, err := CreatePin(testMessage.GetID(), user.GetUID())
		require.NoError(err)

		if assert.NoError(DeletePin(p)) {
			pin, err := GetPin(uuid.Nil)
			require.NoError(err)
			assert.Nil(pin)
		}
	})

	// GetPinsByChannelID
	t.Run("TestGetPinsByChannelID", func(t *testing.T) {
		t.Parallel()

		channel := mustMakeChannelDetail(t, user.GetUID(), utils.RandAlphabetAndNumberString(20), "")
		testMessage := mustMakeMessage(t, user.GetUID(), channel.ID)
		_, err := CreatePin(testMessage.GetID(), user.GetUID())
		require.NoError(err)

		pins, err := GetPinsByChannelID(channel.ID)
		if assert.NoError(err) {
			assert.Len(pins, 1)
		}
	})
}
